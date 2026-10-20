package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"school-oj/apps/api/internal/middleware"
	"school-oj/apps/api/internal/models"
	"school-oj/apps/api/internal/services"
)

func eventTestPaper(t *testing.T, f *examTestFixture, kind string) (string, uint) {
	t.Helper()
	start, end := time.Now().Add(time.Hour), time.Now().Add(2*time.Hour)
	if kind == "exam" {
		if err := f.s.DB.Model(&f.exam).Updates(map[string]any{"starts_at": start, "ends_at": end, "review_status": "draft", "review_revision": 1}).Error; err != nil {
			t.Fatal(err)
		}
		return fmt.Sprintf("/api/event-papers/exam/%d", f.exam.ID), f.exam.ID
	}
	team := models.Team{Name: "review QA", Slug: "review-qa", OwnerID: f.teacher.ID, ContestPermission: "all"}
	if err := f.s.DB.Create(&team).Error; err != nil {
		t.Fatal(err)
	}
	for _, u := range []models.User{f.teacher, f.student} {
		role := models.TeamRoleMember
		if u.ID == f.teacher.ID {
			role = models.TeamRoleOwner
		}
		if err := f.s.DB.Create(&models.TeamMembership{TeamID: team.ID, UserID: u.ID, Role: role}).Error; err != nil {
			t.Fatal(err)
		}
	}
	contest := models.TeamContest{TeamID: team.ID, Title: "contest review QA", StartsAt: &start, DurationMinutes: 60, ScoringRule: "ioi", State: models.TeamContestDraft, CreatedBy: f.teacher.ID, EventReview: models.EventReview{ReviewStatus: "draft", ReviewRevision: 1}}
	if err := f.s.DB.Create(&contest).Error; err != nil {
		t.Fatal(err)
	}
	if err := f.s.DB.Create(&models.TeamContestProblem{ContestID: contest.ID, ProblemID: f.problem.ID, ProblemVersionID: f.version.ID, Label: "A"}).Error; err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf("/api/event-papers/contest/%d", contest.ID), contest.ID
}
func eventRevision(t *testing.T, f *examTestFixture, kind string, id uint) int {
	t.Helper()
	p, e := loadEventPaper(f.s.DB, kind, id, false)
	if e != nil {
		t.Fatal(e)
	}
	return p.review().ReviewRevision
}
func eventReviewRequest(t *testing.T, f *examTestFixture, base, kind string, id uint, actor *models.User, action string) *httptest.ResponseRecorder {
	t.Helper()
	return f.request("POST", base+"/review", actor, gin.H{"revision": eventRevision(t, f, kind, id), "action": action, "note": "QA review"})
}

func TestEventReviewLifecycleAndArchive(t *testing.T) {
	for _, kind := range []string{"exam", "contest"} {
		t.Run(kind, func(t *testing.T) {
			f := newExamFixture(t)
			base, id := eventTestPaper(t, f, kind)
			assertStatus(t, f.request("GET", base, &f.student, nil), 403)
			assertStatus(t, eventReviewRequest(t, f, base, kind, id, &f.teacher, "submit"), 200)
			stale := eventRevision(t, f, kind, id)
			assertStatus(t, f.request("PUT", base+"/problems", &f.teacher, gin.H{"revision": stale, "problems": []any{}}), 409)
			assertStatus(t, eventReviewRequest(t, f, base, kind, id, &f.teacher, "approve"), 403)
			assertStatus(t, eventReviewRequest(t, f, base, kind, id, &f.admin, "approve"), 200)
			assertStatus(t, f.request("GET", base+"/attachments", &f.student, nil), 403)
			assertStatus(t, f.request("POST", base+"/review", &f.admin, gin.H{"revision": stale, "action": "approve"}), 409)
			assertStatus(t, eventReviewRequest(t, f, base, kind, id, &f.teacher, "withdraw"), 200)
			assertStatus(t, eventReviewRequest(t, f, base, kind, id, &f.teacher, "submit"), 200)
			assertStatus(t, eventReviewRequest(t, f, base, kind, id, &f.admin, "reject"), 200)
			assertStatus(t, f.request("GET", base+"/archive", &f.teacher, nil), 404)
			assertStatus(t, eventReviewRequest(t, f, base, kind, id, &f.teacher, "submit"), 200)
			assertStatus(t, eventReviewRequest(t, f, base, kind, id, &f.admin, "approve"), 200)
			p, _ := loadEventPaper(f.s.DB, kind, id, false)
			if kind == "exam" {
				if !p.exam.ParticipantsFrozen {
					t.Fatal("approved exam roster not saved")
				}
				if err := f.s.DB.Where("course_id=? AND user_id=?", p.exam.CourseID, f.peer.ID).Delete(&models.CourseMembership{}).Error; err != nil {
					t.Fatal(err)
				}
				if len(f.s.examStudents(p.exam)) != 2 {
					t.Fatal("historical exam roster changed after course member left")
				}
			}
			start, end := time.Now().Add(-2*time.Hour), time.Now().Add(-time.Hour)
			updates := map[string]any{"starts_at": start}
			if kind == "exam" {
				updates["ends_at"] = end
			}
			if err := f.s.DB.Model(p.model()).Updates(updates).Error; err != nil {
				t.Fatal(err)
			}
			assertStatus(t, eventReviewRequest(t, f, base, kind, id, &f.teacher, "withdraw"), 409)
			assertStatus(t, f.request("GET", base+"/archive", &f.student, nil), 403)
			assertStatus(t, eventReviewRequest(t, f, base, kind, id, &f.teacher, "complete"), 403)
			assertStatus(t, eventReviewRequest(t, f, base, kind, id, &f.admin, "complete"), 200)
			assertStatus(t, f.request("GET", base+"/archive", &f.student, nil), 200)
			assertStatus(t, f.request("GET", "/api/event-archives", &f.student, nil), 200)
			if kind == "exam" {
				assertStatus(t, f.request("POST", fmt.Sprintf("/api/exams/%d/enter", id), &f.student, nil), 403)
			} else {
				assertStatus(t, f.request("GET", base+"/archive", &f.peer, nil), 403)
			}
			deletePath := fmt.Sprintf("/api/exams/%d", id)
			if kind == "contest" {
				deletePath = fmt.Sprintf("/api/contests/%d", id)
			}
			assertStatus(t, f.request("DELETE", deletePath, &f.teacher, nil), 200)
			assertStatus(t, f.request("GET", base+"/archive", &f.student, nil), 404)
			response := f.request("GET", "/api/event-archives", &f.student, nil)
			assertStatus(t, response, 200)
			var list struct{ Items []eventListRow }
			_ = json.Unmarshal(response.Body.Bytes(), &list)
			for _, row := range list.Items {
				if row.Kind == kind && row.ID == id {
					t.Fatal("deleted event became archived")
				}
			}
		})
	}
}

func TestEventPendingDeadlineAndConcurrentReview(t *testing.T) {
	f := newExamFixture(t)
	base, id := eventTestPaper(t, f, "exam")
	assertStatus(t, eventReviewRequest(t, f, base, "exam", id, &f.teacher, "submit"), 200)
	revision := eventRevision(t, f, "exam", id)
	var wg sync.WaitGroup
	codes := make(chan int, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			codes <- f.request("POST", base+"/review", &f.admin, gin.H{"revision": revision, "action": "approve"}).Code
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
		t.Fatalf("double review: success=%d conflicts=%d", success, conflict)
	}
	assertStatus(t, eventReviewRequest(t, f, base, "exam", id, &f.teacher, "withdraw"), 200)
	assertStatus(t, eventReviewRequest(t, f, base, "exam", id, &f.teacher, "submit"), 200)
	if err := f.s.DB.Model(&f.exam).Updates(map[string]any{"starts_at": time.Now().Add(-2 * time.Hour), "ends_at": time.Now().Add(-time.Hour)}).Error; err != nil {
		t.Fatal(err)
	}
	assertStatus(t, eventReviewRequest(t, f, base, "exam", id, &f.admin, "approve"), 409)
	assertStatus(t, f.request("GET", base+"/archive", &f.teacher, nil), 404)
	assertStatus(t, f.request("GET", base+"/attachments", &f.student, nil), 403)
	assertStatus(t, f.request("POST", fmt.Sprintf("/api/exams/%d/enter", id), &f.student, nil), 403)
	assertStatus(t, eventReviewRequest(t, f, base, "exam", id, &f.teacher, "withdraw"), 200)
	assertStatus(t, f.request("PUT", base+"/settings", &f.teacher, gin.H{"revision": eventRevision(t, f, "exam", id), "title": "rescheduled", "description": "retry", "starts_at": time.Now().Add(time.Hour), "ends_at": time.Now().Add(2 * time.Hour)}), 200)
	assertStatus(t, eventReviewRequest(t, f, base, "exam", id, &f.teacher, "submit"), 200)
}

func uploadEventTestFile(t *testing.T, f *examTestFixture, base string, user models.User, revision int, name, contents string) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("revision", fmt.Sprint(revision))
	file, err := writer.CreateFormFile("file", name)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = file.Write([]byte(contents))
	_ = writer.Close()
	req := httptest.NewRequest("POST", base+"/attachments", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	token, _ := middleware.SignToken(f.s.Cfg.JWTSecret, user)
	req.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	f.router.ServeHTTP(response, req)
	return response
}
func TestEventAttachmentsAccessAndRevision(t *testing.T) {
	for _, kind := range []string{"exam", "contest"} {
		t.Run(kind, func(t *testing.T) {
			f := newExamFixture(t)
			store := setupTicketObjects(t, f)
			base, id := eventTestPaper(t, f, kind)
			r := uploadEventTestFile(t, f, base, f.teacher, 1, "说明.txt", "event attachment")
			assertStatus(t, r, 201)
			var file models.EventAttachment
			_ = json.Unmarshal(r.Body.Bytes(), &file)
			if strings.Contains(r.Body.String(), "event-attachments/") {
				t.Fatal("storage key disclosed")
			}
			assertStatus(t, uploadEventTestFile(t, f, base, f.teacher, 1, "stale.txt", "stale"), 409)
			if len(store.objects) != 1 {
				t.Fatalf("failed upload left objects: %d", len(store.objects))
			}
			assertStatus(t, uploadEventTestFile(t, f, base, f.student, 2, "bad.txt", "bad"), 403)
			assertStatus(t, uploadEventTestFile(t, f, base, f.teacher, 2, "说明.txt", "duplicate"), 409)
			url := fmt.Sprintf("%s/attachments/%d", base, file.ID)
			assertStatus(t, f.request("GET", url, &f.student, nil), 403)
			assertStatus(t, eventReviewRequest(t, f, base, kind, id, &f.teacher, "submit"), 200)
			assertStatus(t, eventReviewRequest(t, f, base, kind, id, &f.admin, "approve"), 200)
			assertStatus(t, uploadEventTestFile(t, f, base, f.teacher, eventRevision(t, f, kind, id), "frozen.txt", "frozen"), 409)
			p, _ := loadEventPaper(f.s.DB, kind, id, false)
			if err := f.s.DB.Model(p.model()).Update("starts_at", time.Now().Add(-time.Minute)).Error; err != nil {
				t.Fatal(err)
			}
			if kind == "exam" {
				assertStatus(t, f.request("GET", url, &f.student, nil), 403)
				assertStatus(t, f.request("POST", fmt.Sprintf("/api/exams/%d/enter", id), &f.student, nil), 200)
			}
			r = f.request("GET", url, &f.student, nil)
			assertStatus(t, r, 200)
			if r.Body.String() != "event attachment" || r.Header().Get("Cache-Control") != "private, no-store" || !strings.Contains(r.Header().Get("Content-Disposition"), "attachment") {
				t.Fatalf("unsafe download: %s", r.Body.String())
			}
			assertStatus(t, f.request("GET", fmt.Sprintf("/api/event-papers/%s/%d/attachments/%d", kind, id+1, file.ID), &f.student, nil), map[bool]int{true: 403, false: 404}[kind == "exam"])
			if err := f.s.DB.Model(p.model()).Update("deleted_at", time.Now()).Error; err != nil {
				t.Fatal(err)
			}
			assertStatus(t, f.request("GET", url, &f.student, nil), 404)
		})
	}
}

func TestEventNewProblemVersionIsolation(t *testing.T) {
	f := newExamFixture(t)
	setupTicketObjects(t, f)
	base, id := eventTestPaper(t, f, "exam")
	req := eventProblemInput{ProblemPackageDraft: services.ProblemPackageDraft{Title: "event only", Statement: "statement", TimeLimitMS: 1000, MemoryLimitMB: 256, OutputLimitKB: 1024, Cases: []services.ProblemCaseDraft{{Name: "first", Input: "1", Output: "2", Weight: 7}, {Name: "second", Input: "3", Output: "4", Weight: 13}}}, Revision: 1, TestMode: "replace", Label: "C", Score: 30}
	req.StatementSections = &services.ProblemStatementSections{Description: "statement", InputFormat: "integer", OutputFormat: "integer", Samples: []services.ProblemStatementSample{{Input: "1", Output: "2"}}}
	r := f.request("POST", base+"/problems", &f.teacher, req)
	assertStatus(t, r, 200)
	var problem models.Problem
	_ = json.Unmarshal(r.Body.Bytes(), &problem)
	old := *problem.CurrentVersionID
	if problem.EventKind != "exam" || problem.EventID == nil || *problem.EventID != id {
		t.Fatal("new question is not scoped")
	}
	if f.s.isProblemPublic(problem.ID) {
		t.Fatal("private event question became public")
	}
	assertStatus(t, f.request("GET", fmt.Sprintf("/api/problems/%d", problem.ID), &f.student, nil), 404)
	assertStatus(t, f.request("POST", "/api/submissions", &f.teacher, gin.H{"problem_id": problem.ID, "language": "cpp", "source_code": "int main(){}", "is_public": true}), 403)
	other := fmt.Sprintf("/api/event-papers/exam/%d", f.other.ID)
	if err := f.s.DB.Model(&f.other).Update("review_status", "draft").Error; err != nil {
		t.Fatal(err)
	}
	assertStatus(t, f.request("PUT", other+"/problems", &f.teacher, gin.H{"revision": 1, "problems": []gin.H{{"problem_id": problem.ID}}}), 403)
	req.Revision = eventRevision(t, f, "exam", id)
	req.ExpectedVersionID = old
	req.TestMode = "merge"
	req.Cases = req.Cases[:1]
	req.Cases[0].Input = "9"
	req.Cases[0].Output = "10"
	assertStatus(t, f.request("PUT", fmt.Sprintf("%s/problems/%d", base, problem.ID), &f.teacher, req), 200)
	var current models.Problem
	f.s.DB.First(&current, problem.ID)
	if *current.CurrentVersionID == old {
		t.Fatal("version overwritten")
	}
	if len(problemTestCases(current.Manifest)) != 2 {
		t.Fatal("partial upload lost test cases")
	}
	var prior models.ProblemVersion
	f.s.DB.First(&prior, old)
	if prior.PackageObject == current.PackageObject {
		t.Fatal("package object overwritten")
	}
	assertStatus(t, f.request("PUT", fmt.Sprintf("%s/problems/%d", base, problem.ID), &f.teacher, req), 409)
	assertStatus(t, eventReviewRequest(t, f, base, "exam", id, &f.teacher, "submit"), 200)
	req.Revision = eventRevision(t, f, "exam", id)
	req.ExpectedVersionID = *current.CurrentVersionID
	assertStatus(t, f.request("PUT", fmt.Sprintf("%s/problems/%d", base, problem.ID), &f.teacher, req), 409)
}
