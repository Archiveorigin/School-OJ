package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"school-oj/apps/worker/internal/config"
	"school-oj/apps/worker/internal/models"
)

func TestPrepareHostVisibleRootCreatesAndRefreshesSeccompProfile(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "sandbox")
	profile := filepath.Join(root, "seccomp", "oj-seccomp.json")
	source := filepath.Join(base, "bundled.json")
	if err := os.WriteFile(source, []byte("profile-v1"), 0o644); err != nil {
		t.Fatal(err)
	}
	runner := DockerRunner{Cfg: config.Config{SandboxWorkRoot: root, SandboxSeccomp: profile}}
	if err := runner.prepareHostVisibleRoot(source); err != nil {
		t.Fatal(err)
	}
	assertFileBody(t, profile, "profile-v1")

	if err := os.WriteFile(source, []byte("profile-v2"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := runner.prepareHostVisibleRoot(source); err != nil {
		t.Fatal(err)
	}
	assertFileBody(t, profile, "profile-v2")
}

func TestPrepareHostVisibleRootRejectsRelativePaths(t *testing.T) {
	runner := DockerRunner{Cfg: config.Config{
		SandboxWorkRoot: "relative/root",
		SandboxSeccomp:  "relative/root/seccomp.json",
	}}
	if err := runner.prepareHostVisibleRoot("unused"); err == nil {
		t.Fatal("expected relative sandbox paths to be rejected")
	}
}

func TestLimitedBufferCancelsAtOutputLimit(t *testing.T) {
	cancelled := false
	buffer := limitedBuffer{
		limit:   4,
		onLimit: func() { cancelled = true },
	}
	written, err := buffer.Write([]byte("12345"))
	if err != nil {
		t.Fatal(err)
	}
	if written != 5 || buffer.String() != "1234" || !buffer.truncated || !cancelled {
		t.Fatalf("unexpected limited buffer state: written=%d body=%q truncated=%v cancelled=%v", written, buffer.String(), buffer.truncated, cancelled)
	}
}

func TestInfrastructureErrorClassification(t *testing.T) {
	for _, message := range []string{
		"mkdir /tmp/school-oj-worker/seccomp: no such file or directory",
		"Error response from daemon: error while creating mount source path '/var/lib/school-oj-worker/job': permission denied",
		"docker sandbox infrastructure is not ready",
		"unable to find image gcc:14-bookworm",
	} {
		if !IsInfrastructureError(message) {
			t.Fatalf("expected infrastructure error: %s", message)
		}
	}
	for _, message := range []string{
		"compile failed",
		"python: can't open file '/work/missing.py': no such file or directory",
	} {
		if IsInfrastructureError(message) {
			t.Fatalf("unexpected infrastructure error: %s", message)
		}
	}
}

func TestCompileFailurePreservesInfrastructureError(t *testing.T) {
	limit := sandboxLimits{TimeLimitMS: 30000, MemoryMB: 1024}
	result := compileFailure(
		models.StatusSystemError,
		"docker sandbox infrastructure is not ready",
		25,
		1024,
		limit,
	)
	if result.Status != models.StatusSystemError {
		t.Fatalf("compile infrastructure status = %s, want %s", result.Status, models.StatusSystemError)
	}
	if !IsInfrastructureError(result.Message) {
		t.Fatalf("compile infrastructure message is not retryable: %s", result.Message)
	}

	result = compileFailure(models.StatusRuntimeError, "compiler exited 1", 10, 2048, limit)
	if result.Status != models.StatusCompileError {
		t.Fatalf("ordinary compile failure status = %s, want %s", result.Status, models.StatusCompileError)
	}
}

func TestParseExecutionMetrics(t *testing.T) {
	stderr, metrics := parseExecutionMetrics("warning\n__SCHOOL_OJ_CPU_NS__=1250001\n__SCHOOL_OJ_WALL_NS__=4500001\n__SCHOOL_OJ_MEMORY_BYTES__=2097153\n__SCHOOL_OJ_OOM__=1\n")
	if stderr != "warning" {
		t.Fatalf("stderr = %q, want warning", stderr)
	}
	if metrics.TimeMS != 2 || metrics.WallTimeMS != 5 || metrics.MemoryKB != 2049 || !metrics.OOMKilled {
		t.Fatalf("unexpected metrics: %+v", metrics)
	}
}

func TestMemoryLimitClassificationUsesExplicitSignals(t *testing.T) {
	for _, message := range []string{"MemoryError", "java.lang.OutOfMemoryError: Java heap space", "terminate called after throwing std::bad_alloc"} {
		if !isExplicitMemoryLimitError(message) {
			t.Fatalf("expected memory limit marker: %s", message)
		}
	}
	for _, message := range []string{"Killed", "time limit exceeded", "memory usage report unavailable"} {
		if isExplicitMemoryLimitError(message) {
			t.Fatalf("unexpected memory limit marker: %s", message)
		}
	}
}

func TestExecutionClassificationPriority(t *testing.T) {
	runErr := fmt.Errorf("signal: killed")
	if got := classifyExecution("Killed", runErr, true, false, true); got != models.StatusTimeLimit {
		t.Fatalf("timeout plus OOM signal = %s, want %s", got, models.StatusTimeLimit)
	}
	if got := classifyExecution("Killed", runErr, false, false, false); got != models.StatusRuntimeError {
		t.Fatalf("plain killed process = %s, want %s", got, models.StatusRuntimeError)
	}
	if got := classifyExecution("MemoryError", runErr, false, false, false); got != models.StatusMemoryLimit {
		t.Fatalf("explicit allocation failure = %s, want %s", got, models.StatusMemoryLimit)
	}
	if got := classifyExecution("", nil, false, false, false); got != models.StatusAccepted {
		t.Fatalf("successful process = %s, want %s", got, models.StatusAccepted)
	}
}

func TestLimitsMatchDisplayedProblemConfiguration(t *testing.T) {
	req := JudgeRequest{
		Problem: models.Problem{TimeLimitMS: 1000, MemoryLimitMB: 128, OutputLimitKB: 512},
		Package: ProblemPackage{Manifest: Manifest{TimeLimitMS: 2000, MemoryLimitMB: 256, OutputLimitKB: 1024}},
	}
	got := limits(req)
	if got.TimeLimitMS != 1000 || got.MemoryMB != 128 || got.OutputLimitKB != 512 {
		t.Fatalf("limits do not match displayed problem configuration: %+v", got)
	}
	if configuredLimit(0, 2048) != 2048 {
		t.Fatal("legacy package limit was not used as fallback")
	}
}

func assertFileBody(t *testing.T, path string, want string) {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != want {
		t.Fatalf("file body = %q, want %q", body, want)
	}
}
