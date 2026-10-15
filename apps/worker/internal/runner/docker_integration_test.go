package runner

import (
	"context"
	"os"
	"path/filepath"
	"school-oj/apps/worker/internal/config"
	"strings"
	"testing"
	"time"

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
		context.Background(), workDir, PythonImage,
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
	_, status, timeMS, memoryKB = r.runContainer(
		context.Background(), workDir, PythonImage,
		`python3 -c "while True: pass"`, "", timeLimit,
	)
	if status != models.StatusTimeLimit || timeMS <= 0 || memoryKB <= 0 {
		t.Fatalf("CPU-bound timeout status=%s time_ms=%d memory_kb=%d, want %s with metrics", status, timeMS, memoryKB, models.StatusTimeLimit)
	}

	memoryLimit := baseLimit
	memoryLimit.MemoryMB = 32
	_, status, _, memoryKB = r.runContainer(
		context.Background(), workDir, PythonImage,
		`python3 -c "data = bytearray(64 * 1024 * 1024); print(len(data))"`, "", memoryLimit,
	)
	if status != models.StatusMemoryLimit {
		t.Fatalf("memory exhaustion status=%s memory_kb=%d, want %s", status, memoryKB, models.StatusMemoryLimit)
	}

	_, status, _, _ = r.runContainer(
		context.Background(), workDir, PythonImage,
		`python3 -c "raise RuntimeError('ordinary failure')"`, "", baseLimit,
	)
	if status != models.StatusRuntimeError {
		t.Fatalf("ordinary crash status=%s, want %s", status, models.StatusRuntimeError)
	}
}

func TestDockerSandboxLanguagesAndBudgetIntegration(t *testing.T) {
	root := os.Getenv("OJ_INTEGRATION_WORK_ROOT")
	if root == "" {
		t.Skip("requires host-visible sandbox root")
	}
	r := DockerRunner{Cfg: config.Config{SandboxWorkRoot: root, SandboxSeccomp: filepath.Join(root, "seccomp/oj-seccomp.json"), SandboxCPU: "1.0", SandboxMemory: 256, SandboxPids: 128, SubmissionTimeout: time.Minute}}
	if err := r.Prepare(); err != nil {
		t.Fatal(err)
	}
	if err := r.WarmImages(context.Background()); err != nil {
		t.Fatal(err)
	}
	pkg := ProblemPackage{Manifest: Manifest{TimeLimitMS: 3000, MemoryLimitMB: 256, OutputLimitKB: 64, Cases: []Case{{Name: "sum", Input: "tests/1.in", Output: "tests/1.out", Weight: 1}}}, Files: map[string][]byte{"tests/1.in": []byte("1 2\n"), "tests/1.out": []byte("3\n")}}
	for language, source := range map[string]string{
		"c":      "#include <stdio.h>\nint main(){int a,b;scanf(\"%d%d\",&a,&b);printf(\"%d\\n\",a+b);}",
		"cpp":    "#include <iostream>\nint main(){int a,b;std::cin>>a>>b;std::cout<<a+b<<std::endl;}",
		"python": "a,b=map(int,input().split()); print(a+b)",
		"java":   "import java.util.*; public class Main { public static void main(String[] args){ Scanner s=new Scanner(System.in); System.out.println(s.nextInt()+s.nextInt()); }}",
	} {
		t.Run(language, func(t *testing.T) {
			result := r.Judge(context.Background(), JudgeRequest{SubmissionID: 4242, Language: language, SourceCode: source, Package: pkg})
			if result.Status != models.StatusAccepted || result.Score != 100 {
				t.Fatalf("%s: %+v", language, result)
			}
		})
	}
	r.Cfg.SubmissionTimeout = 2 * time.Second
	pkg.Manifest.TimeLimitMS = 10000
	started := time.Now()
	result := r.Judge(context.Background(), JudgeRequest{SubmissionID: 4243, Language: "python", SourceCode: "import time; time.sleep(20)", Package: pkg})
	if result.Status != models.StatusTimeLimit || time.Since(started) > 12*time.Second {
		t.Fatalf("submission budget: %+v", result)
	}
	entries, err := filepath.Glob(filepath.Join(root, "oj-424*-*"))
	if err != nil || len(entries) > 0 {
		t.Fatalf("source directories leaked: %v", entries)
	}
}
