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

func TestBuildProblemPackageWithAsset(t *testing.T) {
	_, parsed, err := BuildProblemPackage(ProblemPackageDraft{
		Slug:      "image-sum",
		Title:     "Image Sum",
		Statement: "see ![sample](assets/sample.png)",
		Assets: []ProblemAssetDraft{
			{Name: "sample.png", Path: "assets/sample.png", ContentType: "image/png", Data: tinyPNGDataURL},
		},
		Cases: []ProblemCaseDraft{
			{Name: "sample", Input: "1 2\n", Output: "3\n", Weight: 100},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.Manifest.Assets) != 1 || len(parsed.Assets) != 1 {
		t.Fatalf("expected one asset, got manifest=%d parsed=%d", len(parsed.Manifest.Assets), len(parsed.Assets))
	}
	if parsed.Manifest.Assets[0].Path != "assets/sample.png" || parsed.Assets[0].ContentType != "image/png" {
		t.Fatalf("unexpected asset metadata: %+v %+v", parsed.Manifest.Assets[0], parsed.Assets[0])
	}
}

func TestBuildProblemPackageAllowsManyWeightedCases(t *testing.T) {
	cases := make([]ProblemCaseDraft, 0, 200)
	for i := 0; i < 200; i++ {
		cases = append(cases, ProblemCaseDraft{Name: "case", Input: "1\n", Output: "1\n", Weight: 1})
	}
	_, parsed, err := BuildProblemPackage(ProblemPackageDraft{
		Slug:  "many-cases",
		Title: "Many Cases",
		Cases: cases,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.Manifest.Cases) != 200 {
		t.Fatalf("expected 200 cases, got %d", len(parsed.Manifest.Cases))
	}
	if parsed.Manifest.Cases[0].Weight != 1 {
		t.Fatalf("expected weight 1, got %d", parsed.Manifest.Cases[0].Weight)
	}
}

func TestBuildProblemPackageAllowsEmptyCaseFiles(t *testing.T) {
	body, parsed, err := BuildProblemPackage(ProblemPackageDraft{
		Slug:  "empty-io",
		Title: "Empty IO",
		Cases: []ProblemCaseDraft{
			{Name: "empty", Input: "", Output: "", Weight: 100},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(body) == 0 || len(parsed.Manifest.Cases) != 1 {
		t.Fatalf("unexpected package: size=%d cases=%d", len(body), len(parsed.Manifest.Cases))
	}
}

func TestParseProblemPackageRejectsUnsafeAssetPath(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	files := map[string]string{
		"problem.yaml":      "slug: bad\ntitle: Bad\nassets:\n  - path: ../bad.png\ncases:\n  - input: tests/1.in\n    output: tests/1.out\n    weight: 100\n",
		"tests/1.in":        "1 2\n",
		"tests/1.out":       "3\n",
		"assets/../bad.png": "not an image",
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
	if _, err := ParseProblemPackage(buf.Bytes()); err == nil {
		t.Fatal("expected unsafe asset path to be rejected")
	}
}

func TestBuildProblemPackageRejectsUnsupportedAssetType(t *testing.T) {
	_, _, err := BuildProblemPackage(ProblemPackageDraft{
		Slug:  "bad-svg",
		Title: "Bad SVG",
		Assets: []ProblemAssetDraft{
			{Name: "bad.svg", Path: "assets/bad.svg", ContentType: "image/svg+xml", Data: "PHN2Zz48L3N2Zz4="},
		},
		Cases: []ProblemCaseDraft{
			{Name: "sample", Input: "1 2\n", Output: "3\n", Weight: 100},
		},
	})
	if err == nil {
		t.Fatal("expected unsupported asset type to be rejected")
	}
}

const tinyPNGDataURL = "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+/p9sAAAAASUVORK5CYII="
