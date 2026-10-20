package services

import (
	"strings"
	"testing"
)

func TestProblemStatementFormatAndOptionalNotes(t *testing.T) {
	s := &ProblemStatementSections{Description: "Task", InputFormat: "One integer", OutputFormat: "One integer", Samples: []ProblemStatementSample{{Input: "```\n1", Output: "2"}}}
	text, e := RenderProblemStatement(s)
	if e != nil {
		t.Fatal(e)
	}
	if strings.Contains(text, "## 说明") || !strings.Contains(text, "````text\n```\n1\n````") {
		t.Fatal(text)
	}
	s.Notes = "hint"
	text, e = RenderProblemStatement(s)
	if e != nil || !strings.Contains(text, "## 说明\n\nhint") {
		t.Fatal(text, e)
	}
	s.OutputFormat = " "
	if _, e = RenderProblemStatement(s); e == nil {
		t.Fatal("missing output format accepted")
	}
}

func TestStructuredStatementSurvivesPackageAndLegacyTicketEdit(t *testing.T) {
	sections := &ProblemStatementSections{Description: "Task", InputFormat: "Integer", OutputFormat: "Integer", Samples: []ProblemStatementSample{{Input: "1", Output: "2"}}}
	body, pkg, err := BuildProblemPackage(ProblemPackageDraft{Title: "Structured", StatementSections: sections, Cases: []ProblemCaseDraft{{Name: "one", Input: "1", Output: "2"}}})
	if err != nil {
		t.Fatal(err)
	}
	if pkg.Manifest.StatementSections == nil {
		t.Fatal("package lost structured fields")
	}
	manifest := pkg.Manifest
	manifest.Statement = "Reviewed raw Markdown change"
	_, edited, err := RebuildProblemPackage(body, manifest, nil)
	if err != nil {
		t.Fatal(err)
	}
	if edited.Manifest.StatementSections != nil || edited.Manifest.Statement != manifest.Statement {
		t.Fatal("stale fields can overwrite reviewed text")
	}
}
