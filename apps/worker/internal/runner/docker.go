package runner

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

func (r DockerRunner) Judge(ctx context.Context, req JudgeRequest) JudgeResult {
	if err := r.prepareHostVisibleRoot(); err != nil {
		return systemError(err)
	}
	workDir, err := os.MkdirTemp(r.Cfg.SandboxWorkRoot, fmt.Sprintf("oj-%d-", req.SubmissionID))
	if err != nil {
		return systemError(err)
	}
	defer os.RemoveAll(workDir)
	if err := os.Chmod(workDir, 0o777); err != nil {
		return systemError(err)
	}
	spec, err := languageSpec(req.Language)
	if err != nil {
		return systemError(err)
	}
	if err := os.WriteFile(filepath.Join(workDir, spec.Source), []byte(req.SourceCode), 0o666); err != nil {
		return systemError(err)
	}
	limit := r.applyLimits(limits(req))
	if spec.Compile != "" {
		compileLimit := r.applyLimits(compileLimits(req, r.Cfg))
		out, status, ms := r.runContainer(ctx, workDir, spec.Image, spec.Compile, "", compileLimit)
		if status != models.StatusAccepted {
			return JudgeResult{Status: models.StatusCompileError, Message: failureMessage("compile", status, out), TimeMS: ms, Trace: trace(compileLimit)}
		}
	}
	var cases []CaseResult
	totalScore := 0
	maxTime := 0
	finalStatus := models.StatusAccepted
	for _, tc := range req.Package.Manifest.Cases {
		expected := normalize(req.Package.CaseOutput(tc))
		actual, status, ms := r.runContainer(ctx, workDir, spec.Image, spec.Run, req.Package.CaseInput(tc), limit)
		caseResult := CaseResult{Name: tc.Name, Status: status, TimeMS: ms}
		if ms > maxTime {
			maxTime = ms
		}
		if status == models.StatusAccepted {
			if normalize(actual) == expected {
				totalScore += tc.Weight
				caseResult.Message = "ok"
			} else {
				status = models.StatusWrongAnswer
				caseResult.Status = status
				caseResult.Message = diffMessage(expected, actual)
			}
		} else {
			caseResult.Message = actual
		}
		if finalStatus == models.StatusAccepted && status != models.StatusAccepted {
			finalStatus = status
		}
		cases = append(cases, caseResult)
	}
	if totalScore > 100 {
		totalScore = 100
	}
	message := "accepted"
	if finalStatus != models.StatusAccepted {
		message = "some test cases failed"
	}
	return JudgeResult{Status: finalStatus, Score: totalScore, TimeMS: maxTime, Message: message, Trace: trace(limit), Cases: cases}
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
		return spec{Source: "Main.java", Image: "eclipse-temurin:21-jdk", Compile: "javac /work/Main.java", Run: "java -Xmx192m -cp /work Main"}, nil
	default:
		return spec{}, fmt.Errorf("unsupported language: %s", language)
	}
}

type sandboxLimits struct {
	TimeLimitMS   int
	MemoryMB      int
	OutputLimitKB int
	CPU           string
	Pids          int
	Seccomp       string
}

func limits(req JudgeRequest) sandboxLimits {
	return sandboxLimits{
		TimeLimitMS:   max(req.Problem.TimeLimitMS, req.Package.Manifest.TimeLimitMS),
		MemoryMB:      max(req.Problem.MemoryLimitMB, req.Package.Manifest.MemoryLimitMB),
		OutputLimitKB: max(req.Problem.OutputLimitKB, req.Package.Manifest.OutputLimitKB),
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
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	args := []string{
		"run", "--rm",
		"--network", "none",
		"--read-only",
		"--tmpfs", "/tmp:rw,nosuid,nodev,noexec,size=64m",
		"--user", "65532:65532",
		"--cap-drop", "ALL",
		"--security-opt", "no-new-privileges",
		"--security-opt", "seccomp=" + limit.Seccomp,
		"--pids-limit", strconv(limit.Pids),
		"--cpus", limit.CPU,
		"--memory", fmt.Sprintf("%dm", limit.MemoryMB),
		"--memory-swap", fmt.Sprintf("%dm", limit.MemoryMB),
		"--stop-timeout", "1",
		"-i",
		"-v", workDir + ":/work:rw",
		"-w", "/work",
		image,
		"sh", "-lc", command,
	}
	cmd := exec.CommandContext(runCtx, "docker", args...)
	cmd.Stdin = strings.NewReader(input)
	var out limitedBuffer
	out.limit = limit.OutputLimitKB * 1024
	var errOut limitedBuffer
	errOut.limit = limit.OutputLimitKB * 1024
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	start := time.Now()
	err := cmd.Run()
	elapsed := int(time.Since(start).Milliseconds())
	combined := strings.TrimSpace(out.String() + "\n" + errOut.String())
	if runCtx.Err() == context.DeadlineExceeded {
		return nonEmpty(combined, "time limit exceeded"), models.StatusTimeLimit, elapsed
	}
	if out.truncated || errOut.truncated {
		return nonEmpty(combined, "output limit exceeded"), models.StatusOutputLimit, elapsed
	}
	if err != nil {
		if isDockerInfraError(combined) {
			return dockerInfraMessage(combined), models.StatusSystemError, elapsed
		}
		if strings.Contains(combined, "Killed") || strings.Contains(combined, "memory") {
			return nonEmpty(combined, "memory limit exceeded"), models.StatusMemoryLimit, elapsed
		}
		return nonEmpty(combined, err.Error()), models.StatusRuntimeError, elapsed
	}
	return out.String(), models.StatusAccepted, elapsed
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
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if b.limit <= 0 {
		b.limit = 1024 * 1024
	}
	remaining := b.limit - b.Buffer.Len()
	if remaining <= 0 {
		b.truncated = true
		return len(p), nil
	}
	if len(p) > remaining {
		b.truncated = true
		_, _ = b.Buffer.Write(p[:remaining])
		return len(p), nil
	}
	return b.Buffer.Write(p)
}

func normalize(s string) string {
	lines := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " \t")
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func diffMessage(expected, actual string) string {
	if len(actual) > 500 {
		actual = actual[:500]
	}
	return fmt.Sprintf("expected %q, got %q", expected, normalize(actual))
}

func isDockerInfraError(output string) bool {
	text := strings.ToLower(output)
	markers := []string{
		"unable to find image",
		"no such image",
		"pull access denied",
		"manifest unknown",
		"error pulling image",
		"failed to resolve",
		"cannot connect to the docker daemon",
		"permission denied while trying to connect",
		"toomanyrequests",
	}
	for _, marker := range markers {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}

func dockerInfraMessage(output string) string {
	output = strings.TrimSpace(output)
	if len(output) > 800 {
		output = output[:800]
	}
	return "docker sandbox image or daemon is not ready; run ./scripts/pull_sandbox_images.sh on the host and restart worker. Details: " + output
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
			"network":            "none",
			"read_only_root":     true,
			"tmpfs":              "/tmp",
			"user":               "65532:65532",
			"cap_drop":           "ALL",
			"no_new_privileges":  true,
			"seccomp":            limit.Seccomp,
			"pids_limit":         limit.Pids,
			"cpu":                limit.CPU,
			"memory_mb":          limit.MemoryMB,
			"time_limit_ms":      limit.TimeLimitMS,
			"output_limit_kb":    limit.OutputLimitKB,
			"memory_swap_equals": true,
		},
	}
}

func systemError(err error) JudgeResult {
	return JudgeResult{Status: models.StatusSystemError, Message: err.Error()}
}

func (r DockerRunner) prepareHostVisibleRoot() error {
	if err := os.MkdirAll(r.Cfg.SandboxWorkRoot, 0o777); err != nil {
		return err
	}
	seccompDir := filepath.Dir(r.Cfg.SandboxSeccomp)
	if err := os.MkdirAll(seccompDir, 0o777); err != nil {
		return err
	}
	if _, err := os.Stat(r.Cfg.SandboxSeccomp); err == nil {
		return nil
	}
	body, err := os.ReadFile("/etc/oj-seccomp.json")
	if err != nil {
		return err
	}
	return os.WriteFile(r.Cfg.SandboxSeccomp, body, 0o644)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func strconv(v int) string {
	return fmt.Sprintf("%d", v)
}
