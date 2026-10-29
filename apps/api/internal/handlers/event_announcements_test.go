package handlers

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"school-oj/apps/api/internal/models"
	"school-oj/apps/api/internal/services"
	"testing"
	"time"
)

func TestEventAnnouncementsPermissionsAndHistory(t *testing.T) {
	for _, kind := range []string{"exam", "contest"} {
		t.Run(kind, func(t *testing.T) {
			f := newExamFixture(t)
			base, id := eventTestPaper(t, f, kind)
			assertStatus(t, f.request("POST", base+"/announcements", &f.teacher, gin.H{"body": "draft"}), 409)
			p, err := loadEventPaper(f.s.DB, kind, id, false)
			if err != nil {
				t.Fatal(err)
			}
			start := time.Now().Add(-time.Minute)
			if err = f.s.DB.Model(p.model()).Updates(map[string]any{"review_status": "approved", "starts_at": start}).Error; err != nil {
				t.Fatal(err)
			}
			assertStatus(t, f.request("POST", base+"/announcements", &f.student, gin.H{"body": "forbidden"}), 403)
			assertStatus(t, f.request("POST", base+"/announcements", &f.teacher, gin.H{"body": " "}), 400)
			assertStatus(t, f.request("POST", base+"/announcements", &f.teacher, gin.H{"body": "invalid problem", "problem_id": 999999}), 400)
			revision := eventRevision(t, f, kind, id)
			assertStatus(t, f.request("POST", base+"/announcements", &f.teacher, gin.H{"body": "更正：输入范围为 100。", "problem_id": f.problem.ID}), 200)
			assertStatus(t, f.request("POST", base+"/announcements", &f.teacher, gin.H{"body": "全场提醒"}), 200)
			if eventRevision(t, f, kind, id) != revision {
				t.Fatal("announcement modified paper revision")
			}
			response := f.request("GET", base+"/announcements", &f.teacher, nil)
			assertStatus(t, response, 200)
			var count int64
			f.s.DB.Table("event_announcements").Where("event_kind=? AND event_id=?", kind, id).Count(&count)
			if count != 2 {
				t.Fatalf("history count %d", count)
			}
		})
	}
}
func TestDeleteExamRemovesParticipation(t *testing.T) {
	f := newExamFixture(t)
	sub := models.Submission{UserID: f.student.ID, ProblemID: f.problem.ID, ProblemVersionID: f.version.ID, ExamID: &f.exam.ID, Language: "cpp", SourceCode: "secret", Status: models.StatusAccepted}
	if err := f.s.DB.Create(&sub).Error; err != nil {
		t.Fatal(err)
	}
	assertStatus(t, f.request("DELETE", fmt.Sprintf("/api/exams/%d", f.exam.ID), &f.teacher, nil), 200)
	for _, table := range []string{"submissions", "exam_attempts", "exam_participants", "exam_activities", "exam_problems"} {
		var count int64
		if err := f.s.DB.Table(table).Where("exam_id=?", f.exam.ID).Count(&count).Error; err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("%s retained %d records", table, count)
		}
	}
}

func TestAuditRetentionCalendarMonths(t *testing.T) {
	f := newExamFixture(t)
	if err := f.s.DB.Exec("INSERT INTO audit_logs(action,created_at) VALUES ('retention.old',CURRENT_TIMESTAMP - INTERVAL '3 months' - INTERVAL '1 day'),('retention.new',CURRENT_TIMESTAMP - INTERVAL '3 months' + INTERVAL '1 day')").Error; err != nil {
		t.Fatal(err)
	}
	if err := services.PurgeAuditLogs(context.Background(), f.s.DB); err != nil {
		t.Fatal(err)
	}
	var count int64
	f.s.DB.Table("audit_logs").Where("action='retention.old'").Count(&count)
	if count != 0 {
		t.Fatal("expired audit retained")
	}
	f.s.DB.Table("audit_logs").Where("action='retention.new'").Count(&count)
	if count != 1 {
		t.Fatal("recent audit removed")
	}
}
