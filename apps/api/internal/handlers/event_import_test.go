package handlers

import (
	"github.com/gin-gonic/gin"
	"school-oj/apps/api/internal/models"
	"sync"
	"testing"
)

func TestEventSubmissionVersionIndependentOfDraftRevision(t *testing.T) {
	for _, kind := range []string{"exam", "contest"} {
		t.Run(kind, func(t *testing.T) {
			f := newExamFixture(t)
			base, id := eventTestPaper(t, f, kind)
			p, _ := loadEventPaper(f.s.DB, kind, id, false)
			if err := bumpEventRevision(f.s.DB, p); err != nil {
				t.Fatal(err)
			}
			p, _ = loadEventPaper(f.s.DB, kind, id, false)
			if p.review().SubmissionVersion != 0 || p.review().ReviewRevision != 2 {
				t.Fatal("draft edit incremented submission version")
			}
			revision := p.review().ReviewRevision
			var wg sync.WaitGroup
			codes := make(chan int, 2)
			for i := 0; i < 2; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					codes <- f.request("POST", base+"/review", &f.teacher, gin.H{"revision": revision, "action": "submit"}).Code
				}()
			}
			wg.Wait()
			close(codes)
			success, conflict := 0, 0
			for code := range codes {
				if code == 200 {
					success++
				}
				if code == 409 {
					conflict++
				}
			}
			if success != 1 || conflict != 1 {
				t.Fatalf("concurrent submit %d/%d", success, conflict)
			}
			assertStatus(t, eventReviewRequest(t, f, base, kind, id, &f.admin, "reject"), 200)
			p, _ = loadEventPaper(f.s.DB, kind, id, false)
			if p.review().SubmissionVersion != 1 {
				t.Fatal("review incremented submission version")
			}
			assertStatus(t, eventReviewRequest(t, f, base, kind, id, &f.teacher, "submit"), 200)
			p, _ = loadEventPaper(f.s.DB, kind, id, false)
			if p.review().SubmissionVersion != 2 {
				t.Fatal("resubmission did not create new version")
			}
			var logs []models.EventReviewLog
			if err := f.s.DB.Where("event_kind=? AND event_id=?", kind, id).Order("id").Find(&logs).Error; err != nil {
				t.Fatal(err)
			}
			if len(logs) != 3 || logs[0].SubmissionVersion != 1 || logs[1].SubmissionVersion != 1 || logs[2].SubmissionVersion != 2 {
				t.Fatalf("version history %+v", logs)
			}
		})
	}
}
