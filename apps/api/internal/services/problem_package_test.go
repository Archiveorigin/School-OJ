package services

import (
	"archive/zip"
	"bytes"
	"testing"
)

func TestParseProblemPackage(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	files := map[string]string{
		"problem.yaml": "slug: sum\ntitle: Sum\ncases:\n  - input: tests/1.in\n    output: tests/1.out\n    weight: 100\n",
		"tests/1.in":   "1 2\n",
		"tests/1.out":  "3\n",
	}
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	pkg, err := ParseProblemPackage(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if pkg.Manifest.Slug != "sum" || len(pkg.Manifest.Cases) != 1 {
		t.Fatalf("unexpected manifest: %+v", pkg.Manifest)
	}
}

func TestBuildProblemPackage(t *testing.T) {
	body, parsed, err := BuildProblemPackage(ProblemPackageDraft{
		Slug:          "form-sum",
		Title:         "Form Sum",
		Statement:     "sum two integers",
		TimeLimitMS:   1000,
		MemoryLimitMB: 128,
		OutputLimitKB: 64,
		Cases: []ProblemCaseDraft{
			{Name: "sample", Input: "1 2\n", Output: "3\n", Weight: 100},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(body) == 0 {
		t.Fatal("expected zip body")
	}
	if parsed.Manifest.Slug != "form-sum" {
		t.Fatalf("unexpected slug %s", parsed.Manifest.Slug)
	}
	if parsed.Manifest.Cases[0].Input != "tests/01.in" {
		t.Fatalf("unexpected input path %s", parsed.Manifest.Cases[0].Input)
	}
}
