package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"school-oj/apps/api/internal/middleware"
	"school-oj/apps/api/internal/models"
	"school-oj/apps/api/internal/services"
	"strings"
	"testing"
	"time"
)

func TestEventCompletionRequiresDeadlineAndAdmin(t *testing.T) {
	for _, kind := range []string{"exam", "contest"} {
		t.Run(kind, func(t *testing.T) {
			f := newExamFixture(t)
			base, id := eventTestPaper(t, f, kind)
			assertStatus(t, eventReviewRequest(t, f, base, kind, id, &f.admin, "complete"), 409)
			assertStatus(t, eventReviewRequest(t, f, base, kind, id, &f.teacher, "submit"), 200)
			listPath := "/api/exams"
			if kind == "contest" {
				p, _ := loadEventPaper(f.s.DB, kind, id, false)
				listPath = fmt.Sprintf("/api/teams/%d/contests", p.contest.TeamID)
			}
			pendingList := f.request("GET", listPath, &f.student, nil)
			assertStatus(t, pendingList, 200)
			var pendingRows []struct {
				ID uint `json:"id"`
			}
			if err := json.Unmarshal(pendingList.Body.Bytes(), &pendingRows); err != nil {
				t.Fatal(err)
			}
			for _, row := range pendingRows {
				if row.ID == id {
					t.Fatal("participant can see pending event")
				}
			}
			assertStatus(t, eventReviewRequest(t, f, base, kind, id, &f.admin, "approve"), 200)
			assertStatus(t, eventReviewRequest(t, f, base, kind, id, &f.admin, "complete"), 409)
			p, _ := loadEventPaper(f.s.DB, kind, id, false)
			updates := map[string]any{"starts_at": time.Now().Add(-2 * time.Hour)}
			if kind == "exam" {
				updates["ends_at"] = time.Now().Add(-time.Minute)
			}
			if e := f.s.DB.Model(p.model()).Updates(updates).Error; e != nil {
				t.Fatal(e)
			}
			sub := models.Submission{UserID: f.student.ID, ProblemID: f.problem.ID, ProblemVersionID: f.version.ID, Language: "python", SourceCode: "print(2)", Status: models.StatusQueued}
			if kind == "exam" {
				sub.ExamID = &id
			} else {
				sub.TeamContestID = &id
			}
			if e := f.s.DB.Create(&sub).Error; e != nil {
				t.Fatal(e)
			}
			assertStatus(t, eventReviewRequest(t, f, base, kind, id, &f.admin, "complete"), 409)
			f.s.DB.Model(&sub).Update("status", models.StatusAccepted)
			path := fmt.Sprintf("/api/exams/%d", id)
			if kind == "contest" {
				path = fmt.Sprintf("/api/contests/%d", id)
			}
			detail := f.request("GET", path, &f.student, nil)
			assertStatus(t, detail, 200)
			var view map[string]json.RawMessage
			_ = json.Unmarshal(detail.Body.Bytes(), &view)
			if string(view["can_read_problems"]) != "true" || string(view["lifecycle"]) != "\"closed\"" {
				t.Fatal(detail.Body.String())
			}
			if string(view["can_submit"]) != "false" {
				t.Fatal("closed event still accepts submissions")
			}
			assertStatus(t, f.request("GET", base+"/attachments", &f.student, nil), 200)
			submitPath := "/api/submissions"
			body := gin.H{"problem_id": f.problem.ID, "language": "python", "source_code": "print(2)"}
			if kind == "exam" {
				body["exam_id"] = id
			} else {
				submitPath = fmt.Sprintf("/api/contests/%d/submissions", id)
			}
			for _, actor := range []*models.User{&f.student, &f.teacher, &f.admin} {
				assertStatus(t, f.request("POST", submitPath, actor, body), 403)
			}
			list := f.request("GET", "/api/event-archives", &f.student, nil)
			assertStatus(t, list, 200)
			var rows struct{ Items []eventListRow }
			_ = json.Unmarshal(list.Body.Bytes(), &rows)
			for _, r := range rows.Items {
				if r.Kind == kind && r.ID == id {
					t.Fatal("deadline alone archived event")
				}
			}
			assertStatus(t, eventReviewRequest(t, f, base, kind, id, &f.teacher, "complete"), 403)
			assertStatus(t, eventReviewRequest(t, f, base, kind, id, &f.admin, "complete"), 200)
			assertStatus(t, f.request("GET", base+"/archive", &f.student, nil), 200)
			assertStatus(t, eventReviewRequest(t, f, base, kind, id, &f.admin, "complete"), 409)
			assertStatus(t, eventReviewRequest(t, f, base, kind, id, &f.admin, "reopen"), 409)
			assertStatus(t, f.request("GET", base+"/archive", &f.student, nil), 200)
			assertStatus(t, f.request("GET", path, &f.student, nil), 410)
			assertStatus(t, f.request("GET", path, &f.admin, nil), 410)
			f.s.DB.Model(p.model()).Update("deleted_at", time.Now())
			assertStatus(t, f.request("POST", base+"/review", &f.admin, gin.H{"action": "complete", "revision": 0}), 404)
		})
	}
}

func TestEventChunkedTestFilesStayScoped(t *testing.T) {
	f := newExamFixture(t)
	setupTicketObjects(t, f)
	base, id := eventTestPaper(t, f, "exam")
	upload := func(path string, user models.User, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("PUT", path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/octet-stream")
		token, _ := middleware.SignToken(f.s.Cfg.JWTSecret, user)
		req.Header.Set("Authorization", "Bearer "+token)
		r := httptest.NewRecorder()
		f.router.ServeHTTP(r, req)
		return r
	}
	inputID, outputID := "01234567-89ab-cdef-0123-456789abcdef", "fedcba98-7654-3210-fedc-ba9876543210"
	chunk := strings.Repeat("1", int(eventUploadChunkSize))
	tail := strings.Repeat("2", 100*1024)
	u := fmt.Sprintf("%s/test-uploads/%s/0?revision=1", base, inputID)
	assertStatus(t, upload(u, f.student, "bad"), 403)
	assertStatus(t, upload(u, f.teacher, chunk+"x"), 400)
	assertStatus(t, upload(u, f.teacher, chunk), 201)
	assertStatus(t, upload(u, f.teacher, chunk), 201)
	assertStatus(t, upload(u, f.teacher, "changed"), 409)
	assertStatus(t, upload(fmt.Sprintf("%s/test-uploads/%s/1?revision=1", base, inputID), f.teacher, tail), 201)
	assertStatus(t, upload(fmt.Sprintf("%s/test-uploads/%s/0?revision=1", base, outputID), f.teacher, "2\n"), 201)
	refs := []services.TestUploadReference{{ID: inputID, Name: "large.in", ChunkCount: 2, Size: int64(len(chunk) + len(tail))}, {ID: outputID, Name: "large.out", ChunkCount: 1, Size: 2}}
	// Another event or author cannot consume these objects.
	p, _ := loadEventPaper(f.s.DB, "exam", f.other.ID, false)
	if _, _, e := consumeEventTestUploads(f.teacher.ID, p, refs); e == nil {
		t.Fatal("cross event upload reuse")
	}
	p, _ = loadEventPaper(f.s.DB, "exam", id, false)
	if _, _, e := consumeEventTestUploads(f.student.ID, p, refs); e == nil {
		t.Fatal("cross user upload reuse")
	}
	req := eventProblemInput{Revision: 1, TestMode: "replace", Label: "C", Score: 100, ProblemPackageDraft: services.ProblemPackageDraft{Title: "chunked", TimeLimitMS: 1000, MemoryLimitMB: 128, OutputLimitKB: 64, TestUploads: refs, StatementSections: &services.ProblemStatementSections{Description: "Double the input", InputFormat: "An integer", OutputFormat: "An integer", Samples: []services.ProblemStatementSample{{Input: "1", Output: "2"}}}}}
	r := f.request("POST", base+"/problems", &f.teacher, req)
	assertStatus(t, r, 200)
	var problem models.Problem
	_ = json.Unmarshal(r.Body.Bytes(), &problem)
	if !strings.Contains(problem.Statement, "## 输入格式") || strings.Contains(problem.Statement, "## 说明") {
		t.Fatal("statement format mismatch")
	}
	var stored models.ProblemVersion
	if e := f.s.DB.First(&stored, *problem.CurrentVersionID).Error; e != nil {
		t.Fatal(e)
	}
	if stored.Manifest["statement_sections"] == nil {
		t.Fatal("structured source was not pinned")
	}
	paperResponse := f.request("GET", base, &f.teacher, nil)
	assertStatus(t, paperResponse, 200)
	if !strings.Contains(paperResponse.Body.String(), `"statement_sections"`) {
		t.Fatal("editor cannot reload structured source")
	}
	req.Revision = 2
	req.Label = "D"
	assertStatus(t, f.request("POST", base+"/problems", &f.teacher, req), 400)
	downloaded := f.request("GET", fmt.Sprintf("%s/problems/%d/package", base, problem.ID), &f.teacher, nil)
	assertStatus(t, downloaded, 200)
	if !bytes.HasPrefix(downloaded.Body.Bytes(), []byte("PK")) {
		t.Fatal("missing package")
	}
}

func TestEventManualGradesMustSettleBeforeCompletion(t *testing.T) {
	f := newExamFixture(t)
	base, id := eventTestPaper(t, f, "exam")
	assertStatus(t, eventReviewRequest(t, f, base, "exam", id, &f.teacher, "submit"), 200)
	assertStatus(t, eventReviewRequest(t, f, base, "exam", id, &f.admin, "approve"), 200)
	if e := f.s.DB.Exec(`UPDATE exams SET starts_at=now()-interval '2 hours',ends_at=now()-interval '1 hour',settings='{"manual_review":true}'::jsonb WHERE id=?`, id).Error; e != nil {
		t.Fatal(e)
	}
	sub := models.Submission{UserID: f.student.ID, ProblemID: f.problem.ID, ProblemVersionID: f.version.ID, ExamID: &id, Language: "python", SourceCode: "print(2)", Status: models.StatusAccepted}
	if e := f.s.DB.Create(&sub).Error; e != nil {
		t.Fatal(e)
	}
	assertStatus(t, eventReviewRequest(t, f, base, "exam", id, &f.admin, "complete"), 409)
	grade := fmt.Sprintf("/api/exams/%d/submissions/%d/grade", id, sub.ID)
	assertStatus(t, f.request("PUT", grade, &f.teacher, gin.H{"score": 20}), 200)
	assertStatus(t, eventReviewRequest(t, f, base, "exam", id, &f.admin, "complete"), 200)
	assertStatus(t, f.request("PUT", grade, &f.teacher, gin.H{"score": 25}), 409)
	assertStatus(t, f.request("POST", fmt.Sprintf("/api/exams/%d/submissions/%d/judge", id, sub.ID), &f.teacher, nil), 409)
	assertStatus(t, eventReviewRequest(t, f, base, "exam", id, &f.admin, "reopen"), 409)
	assertStatus(t, f.request("PUT", grade, &f.teacher, gin.H{"score": 25}), 409)
}
