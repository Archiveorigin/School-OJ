package runner

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"school-oj/apps/worker/internal/config"
	"school-oj/apps/worker/internal/models"

	"gorm.io/datatypes"
)

type JudgeRequest struct {
	SubmissionID uint
	Language     string
	SourceCode   string
	Problem      models.Problem
	Package      ProblemPackage
}

type JudgeResult struct {
	Status   models.SubmissionStatus
	Score    int
	TimeMS   int
	MemoryKB int
	Message  string
	Trace    datatypes.JSONMap
	Cases    []CaseResult
}

type CaseResult struct {
	Name     string
	Status   models.SubmissionStatus
	TimeMS   int
	MemoryKB int
	Message  string
}

type DockerRunner struct {
	Cfg config.Config
}

const bundledSeccompProfile = "/etc/oj-seccomp.json"

func (r DockerRunner) Judge(ctx context.Context, req JudgeRequest) JudgeResult {
	if err := r.Prepare(); err != nil {
		return systemError(err)
	}
	workDir, err := os.MkdirTemp(r.Cfg.SandboxWorkRoot, fmt.Sprintf("oj-%d-", req.SubmissionID))
	if err != nil {
		return systemError(err)
	}
	defer os.RemoveAll(workDir)
	if err := os.Chown(workDir, 65532, 65532); err != nil {
		return systemError(err)
	}
	if err := os.Chmod(workDir, 0o700); err != nil {
		return systemError(err)
	}
	spec, err := languageSpec(req.Language)
	if err != nil {
		return systemError(err)
	}
	sourcePath := filepath.Join(workDir, spec.Source)
	if err := os.WriteFile(sourcePath, []byte(req.SourceCode), 0o600); err != nil {
		return systemError(err)
	}
	if err := os.Chown(sourcePath, 65532, 65532); err != nil {
		return systemError(err)
	}
	limit := r.applyLimits(limits(req))
	if spec.Compile != "" {
		compileLimit := r.applyLimits(compileLimits(req, r.Cfg))
		out, status, ms := r.runContainer(ctx, workDir, spec.Image, spec.Compile, "", compileLimit)
		if status != models.StatusAccepted {
			return compileFailure(status, out, ms, compileLimit)
		}
	}
	finalStatus, totalScore, maxTime, cases := judgeCasesWithChecker(
		req.Package.Manifest.Cases,
		req.Package.Manifest.Checker,
		req.Package.CaseInput,
		req.Package.CaseOutput,
		func(input string) (string, models.SubmissionStatus, int) {
			return r.runContainer(ctx, workDir, spec.Image, runtimeCommand(spec.Run, limit), input, limit)
		},
	)
	message := "accepted"
	if finalStatus != models.StatusAccepted {
		message = "some test cases failed"
	}
	return JudgeResult{Status: finalStatus, Score: totalScore, TimeMS: maxTime, Message: message, Trace: trace(limit), Cases: cases}
}

func compileFailure(status models.SubmissionStatus, output string, elapsedMS int, limit sandboxLimits) JudgeResult {
	resultStatus := models.StatusCompileError
	if status == models.StatusSystemError {
		resultStatus = models.StatusSystemError
	}
	return JudgeResult{
		Status:  resultStatus,
		Message: failureMessage("compile", status, output),
		TimeMS:  elapsedMS,
		Trace:   trace(limit),
	}
}

type caseRunner func(input string) (string, models.SubmissionStatus, int)

func judgeCases(cases []Case, inputFor func(Case) string, outputFor func(Case) string, run caseRunner) (models.SubmissionStatus, int, int, []CaseResult) {
	return judgeCasesWithChecker(cases, Checker{Type: "exact"}, inputFor, outputFor, run)
}

func judgeCasesWithChecker(cases []Case, checker Checker, inputFor func(Case) string, outputFor func(Case) string, run caseRunner) (models.SubmissionStatus, int, int, []CaseResult) {
	totalWeight := 0
	for _, tc := range cases {
		totalWeight += tc.Weight
	}
	passedWeight := 0
	maxTime := 0
	finalStatus := models.StatusAccepted
	results := make([]CaseResult, 0, len(cases))
	for _, tc := range cases {
		expected := outputFor(tc)
		actual, status, ms := run(inputFor(tc))
		caseResult := CaseResult{Name: tc.Name, Status: status, TimeMS: ms}
		if ms > maxTime {
			maxTime = ms
		}
		if status == models.StatusAccepted {
			if compareOutput(checker, expected, actual) {
				passedWeight += tc.Weight
				caseResult.Message = "ok"
			} else {
				status = models.StatusWrongAnswer
				caseResult.Status = status
				caseResult.Message = diffMessage(expected, actual)
			}
		} else {
			caseResult.Message = actual
		}
		results = append(results, caseResult)
		if status != models.StatusAccepted {
			finalStatus = status
			break
		}
	}
	return finalStatus, weightedScore(passedWeight, totalWeight), maxTime, results
}

func compareOutput(checker Checker, expected, actual string) bool {
	switch checker.Type {
	case "tokens":
		expectedTokens := strings.Fields(stripUTF8BOM(expected))
		actualTokens := strings.Fields(stripUTF8BOM(actual))
		if len(expectedTokens) != len(actualTokens) {
			return false
		}
		for i := range expectedTokens {
			if expectedTokens[i] != actualTokens[i] {
				return false
			}
		}
		return true
	case "float":
		return compareFloatTokens(checker, expected, actual)
	default:
		return normalize(actual) == normalize(expected)
	}
}

func compareFloatTokens(checker Checker, expected, actual string) bool {
	expectedTokens := strings.Fields(stripUTF8BOM(expected))
	actualTokens := strings.Fields(stripUTF8BOM(actual))
	if len(expectedTokens) != len(actualTokens) {
		return false
	}
	for i := range expectedTokens {
		expectedNumber, expectedErr := strconv.ParseFloat(expectedTokens[i], 64)
		actualNumber, actualErr := strconv.ParseFloat(actualTokens[i], 64)
		if expectedErr != nil {
			if expectedTokens[i] != actualTokens[i] {
				return false
			}
			continue
		}
		if actualErr != nil || math.IsNaN(expectedNumber) || math.IsInf(expectedNumber, 0) ||
			math.IsNaN(actualNumber) || math.IsInf(actualNumber, 0) {
			return false
		}
		difference := math.Abs(expectedNumber - actualNumber)
		scale := math.Max(math.Abs(expectedNumber), math.Abs(actualNumber))
		if difference > checker.AbsoluteTolerance+checker.RelativeTolerance*scale {
			return false
		}
	}
	return true
}

func weightedScore(passedWeight int, totalWeight int) int {
	if passedWeight <= 0 || totalWeight <= 0 {
		return 0
	}
	if passedWeight >= totalWeight {
		return 100
	}
	return (passedWeight*100 + totalWeight/2) / totalWeight
}

type spec struct {
	Source  string
	Image   string
	Compile string
	Run     string
}

func languageSpec(language string) (spec, error) {
	switch language {
	case "c":
		return spec{Source: "main.c", Image: "gcc:14-bookworm", Compile: "gcc /work/main.c -O2 -pipe -static -s -o /work/main", Run: "/work/main"}, nil
	case "cpp":
		return spec{Source: "main.cpp", Image: "gcc:14-bookworm", Compile: "g++ /work/main.cpp -std=c++17 -O2 -pipe -static -s -o /work/main", Run: "/work/main"}, nil
	case "python":
		return spec{Source: "main.py", Image: "python:3.12-slim", Run: "python3 /work/main.py"}, nil
	case "java":
		return spec{Source: "Main.java", Image: "eclipse-temurin:21-jdk", Compile: "javac /work/Main.java", Run: "java -Xmx{{JAVA_XMX_MB}}m -cp /work Main"}, nil
	default:
		return spec{}, fmt.Errorf("unsupported language: %s", language)
	}
}

func runtimeCommand(command string, limit sandboxLimits) string {
	if !strings.Contains(command, "{{JAVA_XMX_MB}}") {
		return command
	}
	heap := limit.MemoryMB * 3 / 4
	if heap < 16 {
		heap = 16
	}
	if limit.MemoryMB > 32 && heap > limit.MemoryMB-16 {
		heap = limit.MemoryMB - 16
	}
	return strings.ReplaceAll(command, "{{JAVA_XMX_MB}}", intString(heap))
}

type sandboxLimits struct {
	TimeLimitMS     int
	MemoryMB        int
	OutputLimitKB   int
	CPU             string
	Pids            int
	Seccomp         string
	WorkDirReadOnly bool
}

func limits(req JudgeRequest) sandboxLimits {
	return sandboxLimits{
		TimeLimitMS:     max(req.Problem.TimeLimitMS, req.Package.Manifest.TimeLimitMS),
		MemoryMB:        max(req.Problem.MemoryLimitMB, req.Package.Manifest.MemoryLimitMB),
		OutputLimitKB:   max(req.Problem.OutputLimitKB, req.Package.Manifest.OutputLimitKB),
		WorkDirReadOnly: true,
	}
}

func compileLimits(req JudgeRequest, cfg config.Config) sandboxLimits {
	return sandboxLimits{
		TimeLimitMS:   30000,
		MemoryMB:      max(cfg.SandboxMemory, 1024),
		OutputLimitKB: max(req.Problem.OutputLimitKB, 1024),
	}
}

func (r DockerRunner) runContainer(ctx context.Context, workDir, image, command, input string, limit sandboxLimits) (string, models.SubmissionStatus, int) {
	timeout := time.Duration(limit.TimeLimitMS+1000) * time.Millisecond
	timeoutCtx, cancelTimeout := context.WithTimeout(ctx, timeout)
	defer cancelTimeout()
	runCtx, cancelRun := context.WithCancel(timeoutCtx)
	defer cancelRun()
	containerName := fmt.Sprintf("school-oj-%s-%d", filepath.Base(workDir), time.Now().UnixNano())
	workMode := "rw"
	if limit.WorkDirReadOnly {
		workMode = "ro"
	}
	args := []string{
		"run", "--rm",
		"--name", containerName,
		"--log-driver", "none",
		"--network", "none",
		"--ipc", "none",
		"--read-only",
		"--tmpfs", "/tmp:rw,nosuid,nodev,noexec,size=64m",
		"--user", "65532:65532",
		"--cap-drop", "ALL",
		"--security-opt", "no-new-privileges",
		"--security-opt", "seccomp=" + limit.Seccomp,
		"--pids-limit", intString(limit.Pids),
		"--cpus", limit.CPU,
		"--memory", fmt.Sprintf("%dm", limit.MemoryMB),
		"--memory-swap", fmt.Sprintf("%dm", limit.MemoryMB),
		"--ulimit", "core=0:0",
		"--ulimit", "nofile=64:64",
		"--stop-timeout", "1",
		"--env", "PYTHONDONTWRITEBYTECODE=1",
		"-i",
		"-v", workDir + ":/work:" + workMode,
		"-w", "/work",
		image,
		"sh", "-lc", command,
	}
	cmd := exec.CommandContext(runCtx, "docker", args...)
	cmd.Stdin = strings.NewReader(input)
	var out limitedBuffer
	out.limit = limit.OutputLimitKB * 1024
	out.onLimit = cancelRun
	var errOut limitedBuffer
	errOut.limit = limit.OutputLimitKB * 1024
	errOut.onLimit = cancelRun
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	start := time.Now()
	err := cmd.Run()
	elapsed := int(time.Since(start).Milliseconds())
	r.cleanupContainer(containerName)
	combined := strings.TrimSpace(out.String() + "\n" + errOut.String())
	if out.truncated || errOut.truncated {
		return nonEmpty(combined, "output limit exceeded"), models.StatusOutputLimit, elapsed
	}
	if timeoutCtx.Err() == context.DeadlineExceeded {
		return nonEmpty(combined, "time limit exceeded"), models.StatusTimeLimit, elapsed
	}
	if err != nil {
		if IsInfrastructureError(combined) {
			return dockerInfraMessage(combined), models.StatusSystemError, elapsed
		}
		if strings.Contains(combined, "Killed") || strings.Contains(combined, "memory") {
			return nonEmpty(combined, "memory limit exceeded"), models.StatusMemoryLimit, elapsed
		}
		return nonEmpty(combined, err.Error()), models.StatusRuntimeError, elapsed
	}
	return out.String(), models.StatusAccepted, elapsed
}

func (r DockerRunner) cleanupContainer(name string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = exec.CommandContext(ctx, "docker", "rm", "-f", name).Run()
}

func (r DockerRunner) applyLimits(limit sandboxLimits) sandboxLimits {
	if limit.CPU == "" {
		limit.CPU = r.Cfg.SandboxCPU
	}
	if limit.MemoryMB <= 0 {
		limit.MemoryMB = r.Cfg.SandboxMemory
	}
	if limit.Pids <= 0 {
		limit.Pids = r.Cfg.SandboxPids
	}
	if limit.Seccomp == "" {
		limit.Seccomp = r.Cfg.SandboxSeccomp
	}
	return limit
}

type limitedBuffer struct {
	bytes.Buffer
	limit     int
	truncated bool
	onLimit   func()
	once      sync.Once
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if b.limit <= 0 {
		b.limit = 1024 * 1024
	}
	remaining := b.limit - b.Buffer.Len()
	if remaining <= 0 {
		b.truncated = true
		b.once.Do(func() {
			if b.onLimit != nil {
				b.onLimit()
			}
		})
		return len(p), nil
	}
	if len(p) > remaining {
		b.truncated = true
		_, _ = b.Buffer.Write(p[:remaining])
		b.once.Do(func() {
			if b.onLimit != nil {
				b.onLimit()
			}
		})
		return len(p), nil
	}
	return b.Buffer.Write(p)
}

func normalize(s string) string {
	s = stripUTF8BOM(s)
	lines := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " \t")
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func diffMessage(expected, actual string) string {
	return fmt.Sprintf("expected %q, got %q", truncateDiagnostic(normalize(expected)), truncateDiagnostic(normalize(actual)))
}

func truncateDiagnostic(value string) string {
	runes := []rune(value)
	if len(runes) <= 500 {
		return value
	}
	return string(runes[:500]) + "…"
}

// IsInfrastructureError reports whether Docker failed before user code could run.
func IsInfrastructureError(output string) bool {
	text := strings.ToLower(output)
	markers := []string{
		"docker sandbox infrastructure is not ready",
		"unable to find image",
		"no such image",
		"pull access denied",
		"manifest unknown",
		"error pulling image",
		"failed to resolve",
		"cannot connect to the docker daemon",
		"permission denied while trying to connect",
		"toomanyrequests",
		"error while creating mount source path",
		"bind source path does not exist",
		"invalid mount config for type \"bind\"",
	}
	for _, marker := range markers {
		if strings.Contains(text, marker) {
			return true
		}
	}
	pathFailure := strings.Contains(text, "no such file or directory") ||
		strings.Contains(text, "permission denied") ||
		strings.Contains(text, "invalid seccomp")
	if pathFailure && (strings.Contains(text, "seccomp") || strings.Contains(text, "sandbox work root")) {
		return true
	}
	return false
}

func dockerInfraMessage(output string) string {
	output = strings.TrimSpace(output)
	if len(output) > 800 {
		output = output[:800]
	}
	return "docker sandbox infrastructure is not ready; verify sandbox images, Docker daemon access, and the host work root. Details: " + output
}

func failureMessage(phase string, status models.SubmissionStatus, output string) string {
	output = strings.TrimSpace(output)
	if output == "" {
		return fmt.Sprintf("%s failed: %s", phase, status)
	}
	return output
}

func nonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func trace(limit sandboxLimits) datatypes.JSONMap {
	return datatypes.JSONMap{
		"docker": map[string]any{
			"network":              "none",
			"read_only_root":       true,
			"tmpfs":                "/tmp",
			"user":                 "65532:65532",
			"cap_drop":             "ALL",
			"no_new_privileges":    true,
			"seccomp":              limit.Seccomp,
			"pids_limit":           limit.Pids,
			"cpu":                  limit.CPU,
			"memory_mb":            limit.MemoryMB,
			"time_limit_ms":        limit.TimeLimitMS,
			"output_limit_kb":      limit.OutputLimitKB,
			"memory_swap_equals":   true,
			"work_mount_read_only": limit.WorkDirReadOnly,
			"ipc":                  "none",
			"log_driver":           "none",
			"nofile_limit":         64,
		},
	}
}

func systemError(err error) JudgeResult {
	return JudgeResult{Status: models.StatusSystemError, Message: err.Error()}
}

// Prepare creates the host-visible sandbox root and installs the seccomp profile.
func (r DockerRunner) Prepare() error {
	return r.prepareHostVisibleRoot(bundledSeccompProfile)
}

func (r DockerRunner) prepareHostVisibleRoot(seccompSource string) error {
	workRoot := filepath.Clean(r.Cfg.SandboxWorkRoot)
	if !filepath.IsAbs(workRoot) {
		return fmt.Errorf("sandbox work root must be absolute: %s", r.Cfg.SandboxWorkRoot)
	}
	seccompPath := filepath.Clean(r.Cfg.SandboxSeccomp)
	if !filepath.IsAbs(seccompPath) {
		return fmt.Errorf("sandbox seccomp path must be absolute: %s", r.Cfg.SandboxSeccomp)
	}
	if err := os.MkdirAll(workRoot, 0o755); err != nil {
		return fmt.Errorf("create sandbox work root %s: %w", workRoot, err)
	}
	seccompDir := filepath.Dir(seccompPath)
	if err := os.MkdirAll(seccompDir, 0o755); err != nil {
		return fmt.Errorf("create seccomp directory %s: %w", seccompDir, err)
	}
	body, err := os.ReadFile(seccompSource)
	if err != nil {
		return fmt.Errorf("read bundled seccomp profile %s: %w", seccompSource, err)
	}
	if current, err := os.ReadFile(seccompPath); err == nil && bytes.Equal(current, body) {
		return nil
	} else if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read sandbox seccomp profile %s: %w", seccompPath, err)
	}
	if err := writeFileAtomic(seccompPath, body, 0o644); err != nil {
		return fmt.Errorf("write sandbox seccomp profile %s: %w", seccompPath, err)
	}
	return nil
}

func writeFileAtomic(path string, body []byte, mode os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".oj-seccomp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.Write(body); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func intString(v int) string {
	return fmt.Sprintf("%d", v)
}
