package runner

import (
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
		limit,
	)
	if result.Status != models.StatusSystemError {
		t.Fatalf("compile infrastructure status = %s, want %s", result.Status, models.StatusSystemError)
	}
	if !IsInfrastructureError(result.Message) {
		t.Fatalf("compile infrastructure message is not retryable: %s", result.Message)
	}

	result = compileFailure(models.StatusRuntimeError, "compiler exited 1", 10, limit)
	if result.Status != models.StatusCompileError {
		t.Fatalf("ordinary compile failure status = %s, want %s", result.Status, models.StatusCompileError)
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
