package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"mime/multipart"
	"net/http/httptest"
	"school-oj/apps/api/internal/middleware"
	"school-oj/apps/api/internal/models"
	"testing"
	"time"
)

func TestAnnouncementFilesAndReplayExport(t *testing.T) {
	f := newExamFixture(t)
	setupTicketObjects(t, f)
	base, id := eventTestPaper(t, f, "contest")
	p, _ := loadEventPaper(f.s.DB, "contest", id, false)
	start := time.Now().Add(-time.Minute)
	if err := f.s.DB.Model(p.model()).Updates(map[string]any{"review_status": "approved", "starts_at": start}).Error; err != nil {
		t.Fatal(err)
	}
	r := f.request("POST", base+"/announcements", &f.teacher, gin.H{"body": "更正"})
	assertStatus(t, r, 200)
	var announcement eventAnnouncement
	_ = json.Unmarshal(r.Body.Bytes(), &announcement)
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("announcement_id", fmt.Sprint(announcement.ID))
	file, _ := writer.CreateFormFile("file", "解释.txt")
	_, _ = file.Write([]byte("说明"))
	_ = writer.Close()
	req := httptest.NewRequest("POST", base+"/attachments", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	token, _ := middleware.SignToken(f.s.Cfg.JWTSecret, f.teacher)
	req.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	f.router.ServeHTTP(response, req)
	assertStatus(t, response, 201)
	if eventRevision(t, f, "contest", id) != 1 {
		t.Fatal("announcement file changed paper revision")
	}
	response = f.request("GET", base+"/attachments", &f.teacher, nil)
	assertStatus(t, response, 200)
	var files []models.EventAttachment
	_ = json.Unmarshal(response.Body.Bytes(), &files)
	if len(files) != 0 {
		t.Fatal("live announcement file leaked into paper attachments")
	}
	assertStatus(t, f.request("GET", base+"/ranking/export?format=json", &f.admin, nil), 409)
	assertStatus(t, f.request("GET", base+"/ranking/export?format=json", &f.student, nil), 403)
	if err := f.s.DB.Model(p.model()).Update("starts_at", time.Now().Add(-2*time.Hour)).Error; err != nil {
		t.Fatal(err)
	}
	response = f.request("GET", base+"/attachments", &f.teacher, nil)
	assertStatus(t, response, 200)
	_ = json.Unmarshal(response.Body.Bytes(), &files)
	if len(files) != 1 {
		t.Fatal("file not archived after contest end")
	}
	start = time.Now().Add(-2 * time.Hour)
	f.s.DB.Model(p.model()).Updates(map[string]any{"starts_at": start, "freeze_enabled": true, "freeze_duration_minutes": 30})
	sub := models.Submission{UserID: f.student.ID, ProblemID: f.problem.ID, ProblemVersionID: f.version.ID, TeamContestID: &id, Language: "cpp", SourceCode: "secret-must-not-export", Status: models.StatusAccepted, Score: 100, CreatedAt: start.Add(45 * time.Minute)}
	if err := f.s.DB.Create(&sub).Error; err != nil {
		t.Fatal(err)
	}
	response = f.request("GET", base+"/ranking/replay", &f.admin, nil)
	assertStatus(t, response, 200)
	if bytes.Contains(response.Body.Bytes(), []byte("secret-must-not-export")) {
		t.Fatal("source code leaked")
	}
	var replay struct {
		Initial []teamContestRankingRow `json:"initial_rows"`
		Final   []teamContestRankingRow `json:"final_rows"`
		Events  []any                   `json:"events"`
	}
	_ = json.Unmarshal(response.Body.Bytes(), &replay)
	if len(replay.Events) != 1 {
		t.Fatal("missing post-freeze event")
	}
	for _, row := range replay.Initial {
		if row.Solved != 0 {
			t.Fatal("initial snapshot leaked hidden acceptance")
		}
	}
	found := false
	for _, row := range replay.Final {
		if row.UserID == f.student.ID && row.Solved == 1 {
			found = true
		}
	}
	if !found {
		t.Fatal("final snapshot missing acceptance")
	}
	var packet map[string]any
	_ = json.Unmarshal(response.Body.Bytes(), &packet)
	if packet["kind"] != "contest-replay" {
		t.Fatal("missing replay schema")
	}
}
