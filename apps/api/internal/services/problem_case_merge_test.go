package services

import (
	"strings"
	"testing"
)

func TestNamedTestUploadDoesNotRenumberPartialReplacement(t *testing.T) {
	cases, err := BuildNamedProblemCasesFromTestPointFiles([]TestPointUploadFile{{Name: "09.in", Body: []byte("9")}, {Name: "09.out", Body: []byte("18")}})
	if err != nil || len(cases) != 1 || cases[0].Name != "09" {
		t.Fatalf("partial upload renumbered: %+v %v", cases, err)
	}
	base, _, err := BuildProblemPackage(ProblemPackageDraft{Title: "named", Statement: "test", Cases: []ProblemCaseDraft{{Name: "01", Input: "1", Output: "2", Weight: 3}, {Name: "09", Input: "8", Output: "16", Weight: 7}}})
	if err != nil {
		t.Fatal(err)
	}
	merged, err := MergeProblemCases(base, cases)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(merged[0].Input) != "1" || strings.TrimSpace(merged[1].Input) != "9" || merged[1].Weight != 7 {
		t.Fatalf("wrong pair overwritten: %+v", merged)
	}
	if _, err := BuildNamedProblemCasesFromTestPointFiles([]TestPointUploadFile{{Name: "1.in", Body: []byte("1")}, {Name: "2.out", Body: []byte("2")}}); err == nil {
		t.Fatal("unpaired files accepted")
	}
}

func TestMergeProblemCasesPreservesOmittedCasesAndWeights(t *testing.T) {
	base, _, err := BuildProblemPackage(ProblemPackageDraft{Title: "merge", Statement: "test", TimeLimitMS: 1000, MemoryLimitMB: 256, OutputLimitKB: 1024, Cases: []ProblemCaseDraft{{Name: "first", Input: "1", Output: "2", Weight: 7}, {Name: "second", Input: "3", Output: "4", Weight: 13}}})
	if err != nil {
		t.Fatal(err)
	}
	cases, err := MergeProblemCases(base, []ProblemCaseDraft{{Name: "first", Input: "9", Output: "10", Weight: 1}, {Name: "third", Input: "5", Output: "6", Weight: 1}})
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) != 3 || cases[0].Input != "9" || cases[0].Weight != 7 || cases[1].Name != "second" || strings.TrimSpace(cases[1].Input) != "3" || strings.TrimSpace(cases[1].Output) != "4" || cases[1].Weight != 13 || cases[2].Name != "third" {
		t.Fatalf("unsafe merge: %+v", cases)
	}
	if _, err := MergeProblemCases(base, []ProblemCaseDraft{{Name: "new"}, {Name: "NEW"}}); err == nil {
		t.Fatal("duplicate new names accepted")
	}
	if _, err := MergeProblemCases(base, []ProblemCaseDraft{{Name: "first"}, {Name: "FIRST"}}); err == nil {
		t.Fatal("duplicate replacement accepted")
	}
	_, pkg, err := RebuildProblemPackage(base, ProblemManifest{Slug: "merge", Title: "merge", Statement: "test", TimeLimitMS: 1000, MemoryLimitMB: 256, OutputLimitKB: 1024}, cases)
	if err != nil || len(pkg.Manifest.Cases) != 3 {
		t.Fatalf("rebuilt merge: %v", err)
	}
}
