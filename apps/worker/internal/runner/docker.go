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
		out, status, ms, memoryKB := r.runContainer(ctx, workDir, spec.Image, spec.Compile, "", compileLimit)
		if status != models.StatusAccepted {
			return compileFailure(status, out, ms, memoryKB, compileLimit)
		}
	}
	finalStatus, totalScore, maxTime, maxMemory, cases := judgeCasesWithChecker(
		req.Package.Manifest.Cases,
		req.Package.Manifest.Checker,
		req.Package.CaseInput,
		req.Package.CaseOutput,
		func(input string) (string, models.SubmissionStatus, int, int) {
			return r.runContainer(ctx, workDir, spec.Image, runtimeCommand(spec.Run, limit), input, limit)
		},
	)
	message := "accepted"
	if finalStatus != models.StatusAccepted {
		message = "some test cases failed"
	}
	return JudgeResult{Status: finalStatus, Score: totalScore, TimeMS: maxTime, MemoryKB: maxMemory, Message: message, Trace: trace(limit), Cases: cases}
}

func compileFailure(status models.SubmissionStatus, output string, elapsedMS int, memoryKB int, limit sandboxLimits) JudgeResult {
	resultStatus := models.StatusCompileError
	if status == models.StatusSystemError {
		resultStatus = models.StatusSystemError
	}
	return JudgeResult{
		Status:   resultStatus,
		Message:  failureMessage("compile", status, output),
		TimeMS:   elapsedMS,
		MemoryKB: memoryKB,
		Trace:    trace(limit),
	}
}

type caseRunner func(input string) (string, models.SubmissionStatus, int, int)

func judgeCases(cases []Case, inputFor func(Case) string, outputFor func(Case) string, run caseRunner) (models.SubmissionStatus, int, int, int, []CaseResult) {
	return judgeCasesWithChecker(cases, Checker{Type: "exact"}, inputFor, outputFor, run)
}

func judgeCasesWithChecker(cases []Case, checker Checker, inputFor func(Case) string, outputFor func(Case) string, run caseRunner) (models.SubmissionStatus, int, int, int, []CaseResult) {
	totalWeight := 0
	for _, tc := range cases {
		totalWeight += tc.Weight
	}
	passedWeight := 0
	maxTime := 0
	maxMemory := 0
	finalStatus := models.StatusAccepted
	results := make([]CaseResult, 0, len(cases))
	for _, tc := range cases {
		expected := outputFor(tc)
		actual, status, ms, memoryKB := run(inputFor(tc))
		caseResult := CaseResult{Name: tc.Name, Status: status, TimeMS: ms, MemoryKB: memoryKB}
		if ms > maxTime {
			maxTime = ms
		}
		if memoryKB > maxMemory {
			maxMemory = memoryKB
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
	return finalStatus, weightedScore(passedWeight, totalWeight), maxTime, maxMemory, results
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
		TimeLimitMS:     configuredLimit(req.Problem.TimeLimitMS, req.Package.Manifest.TimeLimitMS),
		MemoryMB:        configuredLimit(req.Problem.MemoryLimitMB, req.Package.Manifest.MemoryLimitMB),
		OutputLimitKB:   configuredLimit(req.Problem.OutputLimitKB, req.Package.Manifest.OutputLimitKB),
		WorkDirReadOnly: true,
	}
}

// The database values are what the problem page shows to contestants, so they
// are authoritative. The package value is only a fallback for legacy records.
func configuredLimit(problemValue int, packageValue int) int {
	if problemValue > 0 {
		return problemValue
	}
	return packageValue
}

func compileLimits(req JudgeRequest, cfg config.Config) sandboxLimits {
	return sandboxLimits{
		TimeLimitMS:   30000,
		MemoryMB:      max(cfg.SandboxMemory, 1024),
		OutputLimitKB: max(req.Problem.OutputLimitKB, 1024),
	}
}

const (
	metricCPUNSPrefix      = "__SCHOOL_OJ_CPU_NS__="
	metricWallNSPrefix     = "__SCHOOL_OJ_WALL_NS__="
	metricMemoryBytePrefix = "__SCHOOL_OJ_MEMORY_BYTES__="
	metricOOMPrefix        = "__SCHOOL_OJ_OOM__="
)

type executionMetrics struct {
	TimeMS     int
	WallTimeMS int
	MemoryKB   int
	OOMKilled  bool
}

func (r DockerRunner) runContainer(ctx context.Context, workDir, image, command, input string, limit sandboxLimits) (string, models.SubmissionStatus, int, int) {
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
		"run",
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
		"sh", "-lc", instrumentedCommand(command),
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
	stateOOMKilled := inspectContainerOOM(containerName)
	r.cleanupContainer(containerName)
	cleanErr, metrics := parseExecutionMetrics(errOut.String())
	if metrics.TimeMS <= 0 {
		metrics.TimeMS = metrics.WallTimeMS
	}
	if metrics.WallTimeMS <= 0 {
		metrics.WallTimeMS = elapsed
	}
	combined := strings.TrimSpace(out.String() + "\n" + cleanErr)
	if out.truncated || errOut.truncated {
		return nonEmpty(combined, "output limit exceeded"), models.StatusOutputLimit, metrics.TimeMS, metrics.MemoryKB
	}
	timeExceeded := timeoutCtx.Err() == context.DeadlineExceeded || metrics.TimeMS > limit.TimeLimitMS
	status := classifyExecution(combined, err, timeExceeded, ctx.Err() != nil, stateOOMKilled || metrics.OOMKilled)
	switch status {
	case models.StatusTimeLimit:
		if metrics.TimeMS <= 0 {
			metrics.TimeMS = limit.TimeLimitMS
		}
		return nonEmpty(combined, "time limit exceeded"), models.StatusTimeLimit, metrics.TimeMS, metrics.MemoryKB
	case models.StatusSystemError:
		if IsInfrastructureError(combined) {
			return dockerInfraMessage(combined), models.StatusSystemError, metrics.TimeMS, metrics.MemoryKB
		}
		return nonEmpty(combined, "judge cancelled"), models.StatusSystemError, metrics.TimeMS, metrics.MemoryKB
	case models.StatusMemoryLimit:
		return nonEmpty(combined, "memory limit exceeded"), models.StatusMemoryLimit, metrics.TimeMS, metrics.MemoryKB
	case models.StatusRuntimeError:
		return nonEmpty(combined, err.Error()), models.StatusRuntimeError, metrics.TimeMS, metrics.MemoryKB
	}
	return out.String(), models.StatusAccepted, metrics.TimeMS, metrics.MemoryKB
}

func classifyExecution(output string, runErr error, timeExceeded bool, cancelled bool, oomKilled bool) models.SubmissionStatus {
	if timeExceeded {
		return models.StatusTimeLimit
	}
	if cancelled || IsInfrastructureError(output) {
		return models.StatusSystemError
	}
	if runErr == nil {
		return models.StatusAccepted
	}
	if oomKilled || isExplicitMemoryLimitError(output) {
		return models.StatusMemoryLimit
	}
	return models.StatusRuntimeError
}

func instrumentedCommand(command string) string {
	return `oj_started_ns=$(date +%s%N 2>/dev/null || true)
` + command + `
oj_exit=$?
oj_finished_ns=$(date +%s%N 2>/dev/null || true)
oj_cpu_ns=0
if [ -r /sys/fs/cgroup/cpu.stat ]; then
  while read -r oj_key oj_value; do
    case "$oj_key" in usage_usec) oj_cpu_ns=$((${oj_value:-0} * 1000)) ;; esac
  done < /sys/fs/cgroup/cpu.stat
elif [ -r /sys/fs/cgroup/cpuacct/cpuacct.usage ]; then
  oj_cpu_ns=$(cat /sys/fs/cgroup/cpuacct/cpuacct.usage 2>/dev/null || true)
fi
oj_peak=0
if [ -r /sys/fs/cgroup/memory.peak ]; then
  oj_peak=$(cat /sys/fs/cgroup/memory.peak 2>/dev/null || true)
elif [ -r /sys/fs/cgroup/memory/memory.max_usage_in_bytes ]; then
  oj_peak=$(cat /sys/fs/cgroup/memory/memory.max_usage_in_bytes 2>/dev/null || true)
fi
oj_oom=0
if [ -r /sys/fs/cgroup/memory.events ]; then
  while read -r oj_key oj_value; do
    case "$oj_key" in oom|oom_kill) [ "${oj_value:-0}" -gt 0 ] 2>/dev/null && oj_oom=1 ;; esac
  done < /sys/fs/cgroup/memory.events
elif [ -r /sys/fs/cgroup/memory/memory.oom_control ]; then
  while read -r oj_key oj_value; do
    case "$oj_key" in oom_kill) [ "${oj_value:-0}" -gt 0 ] 2>/dev/null && oj_oom=1 ;; esac
  done < /sys/fs/cgroup/memory/memory.oom_control
fi
oj_elapsed_ns=0
case "$oj_started_ns:$oj_finished_ns" in
  *[!0-9:]*|:*) ;;
  *) oj_elapsed_ns=$((oj_finished_ns - oj_started_ns)) ;;
esac
printf '\n__SCHOOL_OJ_CPU_NS__=%s\n__SCHOOL_OJ_WALL_NS__=%s\n__SCHOOL_OJ_MEMORY_BYTES__=%s\n__SCHOOL_OJ_OOM__=%s\n' "$oj_cpu_ns" "$oj_elapsed_ns" "$oj_peak" "$oj_oom" >&2
exit "$oj_exit"`
}

func parseExecutionMetrics(stderr string) (string, executionMetrics) {
	metrics := executionMetrics{}
	kept := make([]string, 0)
	for _, line := range strings.Split(stderr, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, metricCPUNSPrefix):
			if value, err := strconv.ParseInt(strings.TrimPrefix(trimmed, metricCPUNSPrefix), 10, 64); err == nil && value > 0 {
				metrics.TimeMS = int((value + int64(time.Millisecond) - 1) / int64(time.Millisecond))
			}
		case strings.HasPrefix(trimmed, metricWallNSPrefix):
			if value, err := strconv.ParseInt(strings.TrimPrefix(trimmed, metricWallNSPrefix), 10, 64); err == nil && value > 0 {
				metrics.WallTimeMS = int((value + int64(time.Millisecond) - 1) / int64(time.Millisecond))
			}
		case strings.HasPrefix(trimmed, metricMemoryBytePrefix):
			if value, err := strconv.ParseInt(strings.TrimPrefix(trimmed, metricMemoryBytePrefix), 10, 64); err == nil && value > 0 {
				metrics.MemoryKB = int((value + 1023) / 1024)
			}
		case strings.HasPrefix(trimmed, metricOOMPrefix):
			metrics.OOMKilled = strings.TrimPrefix(trimmed, metricOOMPrefix) == "1"
		default:
			kept = append(kept, line)
		}
	}
	return strings.TrimSpace(strings.Join(kept, "\n")), metrics
}

func inspectContainerOOM(name string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "docker", "inspect", "--format", "{{.State.OOMKilled}}", name).Output()
	return err == nil && strings.TrimSpace(string(out)) == "true"
}

func isExplicitMemoryLimitError(output string) bool {
	text := strings.ToLower(output)
	markers := []string{
		"cannot allocate memory",
		"java.lang.outofmemoryerror",
		"memoryerror",
		"std::bad_alloc",
	}
	for _, marker := range markers {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
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
			"time_measurement":     "cgroup_cpu_usage",
			"wall_time_guard_ms":   limit.TimeLimitMS + 1000,
			"output_limit_kb":      limit.OutputLimitKB,
			"memory_measurement":   "cgroup_peak",
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
