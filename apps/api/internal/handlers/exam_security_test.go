package handlers

import (
	"gorm.io/datatypes"
	"school-oj/apps/api/internal/models"
	"testing"
	"time"
)

func TestSubmissionCaseSummaryUsesVersionTotal(t *testing.T) {
	manifest := datatypes.JSONMap{"cases": []any{
		map[string]any{"name": "one", "input": "tests/1.in", "output": "tests/1.out"},
		map[string]any{"name": "two", "input": "tests/2.in", "output": "tests/2.out"},
		map[string]any{"name": "three", "input": "tests/3.in", "output": "tests/3.out"},
	}}
	results := []models.SubmissionResult{{CaseName: "one", Status: models.StatusAccepted}, {CaseName: "two", Status: models.StatusWrongAnswer}}
	summary := summarizeSubmissionCases(models.StatusWrongAnswer, manifest, results)
	if summary == nil || summary.Passed != 1 || summary.Total != 3 {
		t.Fatalf("summary = %+v", summary)
	}
	for _, status := range []models.SubmissionStatus{models.StatusQueued, models.StatusRunning, models.StatusPendingReview, models.StatusManualGraded, "recorded", models.StatusSystemError} {
		if summarizeSubmissionCases(status, manifest, results) != nil {
			t.Fatalf("%s must hide summary", status)
		}
	}
}

func TestExamWindowExclusiveEnd(t *testing.T) {
	now := time.Now()
	if reason, entered := studentExamEntryDecision(models.Exam{EndsAt: &now}, now, nil); reason != "exam is closed" || entered {
		t.Fatalf("end boundary admitted: %q %v", reason, entered)
	}
	if reason, entered := studentExamEntryDecision(models.Exam{StartsAt: &now}, now, nil); reason != "" || !entered {
		t.Fatalf("start boundary rejected: %q %v", reason, entered)
	}
}

func TestClientExamActivityValidation(t *testing.T) {
	now := time.Now()
	id := "1234567890abcdef1234567890abcdef"
	if !validClientExamEvent(id, "window_leave", now, now, now) {
		t.Fatal("valid event rejected")
	}
	for _, event := range []string{"enter", "login", "finish", "access_blocked", "key_content"} {
		if validClientExamEvent(id, event, now, now, now) {
			t.Fatalf("forged server event admitted: %s", event)
		}
	}
	if validClientExamEvent("enter", "window_leave", now, now, now) {
		t.Fatal("invalid id admitted")
	}
	if validClientExamEvent(id, "window_leave", now.Add(25*time.Hour), now, now) {
		t.Fatal("unbounded future timestamp admitted")
	}
}
