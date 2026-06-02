package runner

import (
	"archive/zip"
	"bytes"
	"testing"
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
