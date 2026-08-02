package runner

import (
	"archive/zip"
	"bytes"
	"strings"
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

func TestOutputCheckers(t *testing.T) {
	tests := []struct {
		name     string
		checker  Checker
		expected string
		actual   string
		match    bool
	}{
		{name: "exact normalizes line endings", checker: Checker{Type: "exact"}, expected: "a\nb\n", actual: "a\r\nb", match: true},
		{name: "tokens ignore whitespace", checker: Checker{Type: "tokens"}, expected: "1  2\n3", actual: "1\n2 3", match: true},
		{name: "tokens preserve token text", checker: Checker{Type: "tokens"}, expected: "01", actual: "1", match: false},
		{name: "float accepts tolerance", checker: Checker{Type: "float", AbsoluteTolerance: 1e-6, RelativeTolerance: 1e-6}, expected: "0.3", actual: "0.3000002", match: true},
		{name: "float rejects outside tolerance", checker: Checker{Type: "float", AbsoluteTolerance: 1e-9, RelativeTolerance: 1e-9}, expected: "0.3", actual: "0.31", match: false},
		{name: "float preserves labels", checker: Checker{Type: "float", AbsoluteTolerance: 1e-6, RelativeTolerance: 1e-6}, expected: "answer 1.0", actual: "answer 1.0000001", match: true},
		{name: "float rejects nan", checker: Checker{Type: "float", AbsoluteTolerance: 1, RelativeTolerance: 1}, expected: "0", actual: "NaN", match: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := compareOutput(test.checker, test.expected, test.actual); got != test.match {
				t.Fatalf("compareOutput() = %v, want %v", got, test.match)
			}
		})
	}
}

func TestParsePackageValidatesChecker(t *testing.T) {
	valid := testZip(t, map[string]string{
		"problem.yaml": "slug: p\ntitle: P\nchecker:\n  type: float\n  absolute_tolerance: 0.000001\ncases:\n  - input: tests/a.in\n    output: tests/a.out\n",
		"tests/a.in":   "1\n",
		"tests/a.out":  "1\n",
	})
	pkg, err := ParsePackage(valid)
	if err != nil {
		t.Fatal(err)
	}
	if pkg.Manifest.Checker.Type != "float" || pkg.Manifest.Checker.RelativeTolerance != 0 {
		t.Fatalf("unexpected checker: %+v", pkg.Manifest.Checker)
	}

	invalid := testZip(t, map[string]string{
		"problem.yaml": "slug: p\ntitle: P\nchecker:\n  type: script\ncases:\n  - input: tests/a.in\n    output: tests/a.out\n",
		"tests/a.in":   "1\n",
		"tests/a.out":  "1\n",
	})
	if _, err := ParsePackage(invalid); err == nil {
		t.Fatal("expected executable checker type to be rejected")
	}
}

func TestDiffMessageBoundsExpectedAndActual(t *testing.T) {
	message := diffMessage(strings.Repeat("e", 1000), strings.Repeat("a", 1000))
	if len([]rune(message)) > 1040 {
		t.Fatalf("diagnostic was not bounded: %d runes", len([]rune(message)))
	}
}

func TestJudgeCasesStopsAtFirstFailureAndScoresPrefix(t *testing.T) {
	var cases []Case
	for i := 0; i < 10; i++ {
		cases = append(cases, Case{Name: "case", Weight: 1})
	}
	calls := 0
	status, score, _, _, results := judgeCases(
		cases,
		func(Case) string { return "input\n" },
		func(Case) string { return "ok\n" },
		func(string) (string, models.SubmissionStatus, int, int) {
			calls++
			if calls == 5 {
				return "bad\n", models.StatusAccepted, 1, 1024
			}
			return "ok\n", models.StatusAccepted, 1, 1024
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
	_, score, _, _, results = judgeCases(
		cases,
		func(Case) string { return "input\n" },
		func(Case) string { return "ok\n" },
		func(string) (string, models.SubmissionStatus, int, int) {
			calls++
			return "bad\n", models.StatusAccepted, 1, 2048
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
