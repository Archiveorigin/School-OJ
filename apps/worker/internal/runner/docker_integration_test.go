package runner

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"school-oj/apps/worker/internal/models"
)

// TestDockerSandboxMetricsIntegration is opt-in because it requires the Docker
// socket, sandbox images, and a host-visible work root shared with nested
// containers. The worker Dockerfile exposes an integration target for it.
func TestDockerSandboxMetricsIntegration(t *testing.T) {
	workRoot := os.Getenv("OJ_INTEGRATION_WORK_ROOT")
	if workRoot == "" {
		t.Skip("OJ_INTEGRATION_WORK_ROOT is not set")
	}
	workDir, err := os.MkdirTemp(workRoot, "metrics-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(workDir)
	if err := os.Chmod(workDir, 0o777); err != nil {
		t.Fatal(err)
	}

	seccomp := os.Getenv("OJ_INTEGRATION_SECCOMP")
	if seccomp == "" {
		seccomp = filepath.Join(workRoot, "seccomp", "oj-seccomp.json")
	}
	r := DockerRunner{}
	baseLimit := sandboxLimits{
		TimeLimitMS:   2000,
		MemoryMB:      64,
		OutputLimitKB: 64,
		CPU:           "1.0",
		Pids:          64,
		Seccomp:       seccomp,
	}

	out, status, timeMS, memoryKB := r.runContainer(
		context.Background(), workDir, "python:3.12-slim",
		`python3 -c "data = bytearray(8 * 1024 * 1024); print(len(data))"`, "", baseLimit,
	)
	if status != models.StatusAccepted || strings.TrimSpace(out) != "8388608" {
		t.Fatalf("accepted case: status=%s output=%q", status, out)
	}
	if timeMS <= 0 || memoryKB < 8*1024 {
		t.Fatalf("accepted metrics not captured: time_ms=%d memory_kb=%d", timeMS, memoryKB)
	}

	timeLimit := baseLimit
	timeLimit.TimeLimitMS = 100
	_, status, _, _ = r.runContainer(
		context.Background(), workDir, "python:3.12-slim",
		`python3 -c "while True: pass"`, "", timeLimit,
	)
	if status != models.StatusTimeLimit {
		t.Fatalf("CPU-bound timeout status=%s, want %s", status, models.StatusTimeLimit)
	}

	memoryLimit := baseLimit
	memoryLimit.MemoryMB = 32
	_, status, _, memoryKB = r.runContainer(
		context.Background(), workDir, "python:3.12-slim",
		`python3 -c "data = bytearray(64 * 1024 * 1024); print(len(data))"`, "", memoryLimit,
	)
	if status != models.StatusMemoryLimit {
		t.Fatalf("memory exhaustion status=%s memory_kb=%d, want %s", status, memoryKB, models.StatusMemoryLimit)
	}

	_, status, _, _ = r.runContainer(
		context.Background(), workDir, "python:3.12-slim",
		`python3 -c "raise RuntimeError('ordinary failure')"`, "", baseLimit,
	)
	if status != models.StatusRuntimeError {
		t.Fatalf("ordinary crash status=%s, want %s", status, models.StatusRuntimeError)
	}
}
