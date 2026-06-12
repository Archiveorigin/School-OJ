package runner

import (
	"archive/zip"
	"bytes"
	"testing"

	"school-oj/apps/worker/internal/models"
)

func TestParsePackage(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range map[string]string{
		"problem.yaml": "slug: p\ntitle: P\ncases:\n  - name: a\n    input: tests/a.in\n    output: tests/a.out\n    weight: 100\n",
		"tests/a.in":   "1 2\n",
		"tests/a.out":  "3\n",
	} {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(body))
	}
	_ = zw.Close()
	pkg, err := ParsePackage(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if got := pkg.CaseOutput(pkg.Manifest.Cases[0]); got != "3\n" {
		t.Fatalf("unexpected output %q", got)
	}
}

func TestParsePackageRejectsEmptyCases(t *testing.T) {
	body := testZip(t, map[string]string{
		"problem.yaml": "slug: p\ntitle: P\ncases: []\n",
	})
	if _, err := ParsePackage(body); err == nil {
		t.Fatal("expected empty cases to be rejected")
	}
}

func TestParsePackageRejectsUnsupportedExtraFile(t *testing.T) {
	body := testZip(t, map[string]string{
		"problem.yaml": "slug: p\ntitle: P\ncases:\n  - name: a\n    input: tests/a.in\n    output: tests/a.out\n    weight: 100\n",
		"tests/a.in":   "1 2\n",
		"tests/a.out":  "3\n",
		"tmp/junk.txt": "extra",
	})
	if _, err := ParsePackage(body); err == nil {
		t.Fatal("expected unsupported extra file to be rejected")
	}
}

func TestParsePackageRejectsWrongCaseExtension(t *testing.T) {
	body := testZip(t, map[string]string{
		"problem.yaml": "slug: p\ntitle: P\ncases:\n  - name: a\n    input: tests/a.out\n    output: tests/a.in\n    weight: 100\n",
		"tests/a.in":   "1 2\n",
		"tests/a.out":  "3\n",
	})
	if _, err := ParsePackage(body); err == nil {
		t.Fatal("expected wrong case extension to be rejected")
	}
}

func TestCaseIOAndNormalizeStripUTF8BOM(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range map[string]string{
		"problem.yaml": "slug: p\ntitle: P\ncases:\n  - name: a\n    input: tests/a.in\n    output: tests/a.out\n    weight: 100\n",
		"tests/a.in":   "\ufeff1 2\n",
		"tests/a.out":  "\ufeff3\n",
	} {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(body))
	}
	_ = zw.Close()
	pkg, err := ParsePackage(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if got := pkg.CaseInput(pkg.Manifest.Cases[0]); got != "1 2\n" {
		t.Fatalf("unexpected input %q", got)
	}
	if got := pkg.CaseOutput(pkg.Manifest.Cases[0]); got != "3\n" {
		t.Fatalf("unexpected output %q", got)
	}
	if got := normalize("\ufeffanswer\n"); got != "answer" {
		t.Fatalf("unexpected normalized output %q", got)
	}
}

func TestWeightedScoreNormalizesLargeCaseSets(t *testing.T) {
	if got := weightedScore(100, 200); got != 50 {
		t.Fatalf("expected 50, got %d", got)
	}
	if got := weightedScore(200, 200); got != 100 {
		t.Fatalf("expected 100, got %d", got)
	}
	if got := weightedScore(0, 200); got != 0 {
		t.Fatalf("expected 0, got %d", got)
	}
}

func TestJudgeCasesStopsAtFirstFailureAndScoresPrefix(t *testing.T) {
	var cases []Case
	for i := 0; i < 10; i++ {
		cases = append(cases, Case{Name: "case", Weight: 1})
	}
	calls := 0
	status, score, _, results := judgeCases(
		cases,
		func(Case) string { return "input\n" },
		func(Case) string { return "ok\n" },
		func(string) (string, models.SubmissionStatus, int) {
			calls++
			if calls == 5 {
				return "bad\n", models.StatusAccepted, 1
			}
			return "ok\n", models.StatusAccepted, 1
		},
	)
	if status != models.StatusWrongAnswer {
		t.Fatalf("expected wrong_answer, got %s", status)
	}
	if score != 40 {
		t.Fatalf("expected score 40, got %d", score)
	}
	if calls != 5 || len(results) != 5 {
		t.Fatalf("expected 5 executed cases, calls=%d results=%d", calls, len(results))
	}

	calls = 0
	_, score, _, results = judgeCases(
		cases,
		func(Case) string { return "input\n" },
		func(Case) string { return "ok\n" },
		func(string) (string, models.SubmissionStatus, int) {
			calls++
			return "bad\n", models.StatusAccepted, 1
		},
	)
	if score != 0 {
		t.Fatalf("expected score 0, got %d", score)
	}
	if calls != 1 || len(results) != 1 {
		t.Fatalf("expected 1 executed case, calls=%d results=%d", calls, len(results))
	}
}

func TestJavaRuntimeCommandUsesMemoryLimit(t *testing.T) {
	command := runtimeCommand("java -Xmx{{JAVA_XMX_MB}}m -cp /work Main", sandboxLimits{MemoryMB: 128})
	if command != "java -Xmx96m -cp /work Main" {
		t.Fatalf("unexpected java command: %s", command)
	}
}

func testZip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(body))
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
