package models

import (
	"testing"
	"time"
)

func TestCompletionIsTerminalAcrossClockChanges(t *testing.T) {
	now := time.Now()
	start, end := now.Add(time.Hour), now.Add(2*time.Hour)
	review := EventReview{ReviewStatus: "approved", CompletedAt: &now}
	for _, clock := range []time.Time{now, start, end, end.Add(time.Hour)} {
		if got := review.Lifecycle(&start, &end, clock); got != "completed" {
			t.Fatalf("completed event regressed to %s", got)
		}
	}
	review.ReviewStatus = "draft"
	if review.Editable() {
		t.Fatal("completed event became editable")
	}
}
