package handlers

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"gorm.io/datatypes"
	"school-oj/apps/api/internal/middleware"
	"school-oj/apps/api/internal/models"
	"school-oj/apps/api/internal/streams"
)

func TestExamActivityAndSubmissionPostgres(t *testing.T) {
	f := newExamFixture(t)
	if f.s.Redis == nil {
		t.Skip("requires OJ_TEST_REDIS_ADDR")
	}
	path := fmt.Sprintf("/api/exams/%d", f.exam.ID)
	assertStatus(t, f.request("POST", "/api/auth/login", nil, gin.H{"email": f.student.Email, "password": "ExamFixture-2026!"}), 200)
	assertStatus(t, f.request("POST", path+"/enter", &f.student, nil), 200)
	assertStatus(t, f.request("POST", "/api/auth/login", nil, gin.H{"email": f.student.Email, "password": "ExamFixture-2026!"}), 200)
	event := gin.H{"id": "0123456789abcdef0123456789abcdef", "event": "window_leave", "client_at": time.Now(), "ip": "forged", "user_id": f.peer.ID}
	for i := 0; i < 2; i++ {
		assertStatus(t, f.request("POST", path+"/activity", &f.student, gin.H{"events": []any{event}}), 200)
	}
	assertStatus(t, f.request("POST", path+"/activity", &f.peer, gin.H{"events": []any{event}}), 403)
	event["event"] = "login"
	assertStatus(t, f.request("POST", path+"/activity", &f.student, gin.H{"events": []any{event}}), 400)
	assertStatus(t, f.request("GET", path+"/activity", &f.student, nil), 403)
	assertStatus(t, f.request("GET", path+"/activity?from=bad", &f.teacher, nil), 400)
	assertStatus(t, f.request("GET", path+"/activity?page=1&page_size=1&from="+time.Now().Add(-time.Hour).UTC().Format(time.RFC3339), &f.teacher, nil), 200)
	var events []models.ExamActivity
	f.s.DB.Where("exam_id = ? AND user_id = ?", f.exam.ID, f.student.ID).Find(&events)
	counts := map[string]int{}
	for _, row := range events {
		counts[row.Event]++
		if row.IP == "198.51.100.99" || row.IP == "forged" {
			t.Fatal("forged source IP")
		}
		if row.Event == "enter" && row.Meta["login_at"] == nil {
			t.Fatal("missing actual login time")
		}
	}
	if counts["window_leave"] != 1 || counts["login"] != 1 || counts["enter"] != 1 {
		t.Fatalf("unexpected events: %+v", counts)
	}
	// An unrelated teacher cannot read logs even with a teacher base role.
	otherTeacher := models.User{Email: "unrelated@example.test", Name: "其他教师", Role: models.RoleTeacher, EmailVerified: true}
	f.s.DB.Create(&otherTeacher)
	assertStatus(t, f.request("GET", path+"/activity", &otherTeacher, nil), 403)
	assertStatus(t, f.request("POST", path+"/enter", &f.teacher, nil), 403)
	// Actual HTTP submission creation pins the event version and creates outbox.
	v2 := f.version
	v2.ID = 0
	v2.Version = 2
	f.s.DB.Create(&v2)
	f.s.DB.Model(&f.problem).Update("current_version_id", v2.ID)
	response := f.request("POST", "/api/submissions", &f.student, gin.H{"problem_id": f.problem.ID, "exam_id": f.exam.ID, "language": "python", "source_code": "print(3)"})
	assertStatus(t, response, 201)
	var sub models.Submission
	f.s.DB.Where("user_id = ?", f.student.ID).Last(&sub)
	if sub.ProblemVersionID != f.version.ID || sub.Status != models.StatusQueued {
		t.Fatalf("incorrect new submission: %+v", sub)
	}
	f.s.DB.Model(&f.exam).Update("settings", datatypes.JSONMap{"manual_review": true, "ranking_visible": true})
	assertStatus(t, f.request("POST", "/api/submissions", &f.student, gin.H{"problem_id": f.problem.ID, "exam_id": f.exam.ID, "language": "python", "source_code": "print(4)"}), 201)
	sub = models.Submission{}
	f.s.DB.Last(&sub)
	if sub.Status != models.StatusPendingReview {
		t.Fatal("manual review was lost")
	}
	assertStatus(t, f.request("POST", path+"/finish", &f.student, nil), 200)
	assertStatus(t, f.request("POST", "/api/submissions", &f.student, gin.H{"problem_id": f.problem.ID, "exam_id": f.exam.ID, "language": "python", "source_code": "print(5)"}), 403)
}

func TestExamFrozenResponsesPostgres(t *testing.T) {
	f := newExamFixture(t)
	if f.s.Redis == nil {
		t.Skip("requires OJ_TEST_REDIS_ADDR")
	}
	assertStatus(t, f.request("POST", fmt.Sprintf("/api/exams/%d/enter", f.exam.ID), &f.student, nil), 200)
	sub := models.Submission{UserID: f.student.ID, ProblemID: f.problem.ID, ProblemVersionID: f.version.ID, ExamID: &f.exam.ID, Language: "python", SourceCode: "print(3)", Status: models.StatusAccepted, Score: 100, TimeMS: 302, MemoryKB: 14656, Message: "private-judge-result"}
	f.s.DB.Create(&sub)
	f.s.DB.Create(&models.SubmissionResult{SubmissionID: sub.ID, CaseName: "one", Status: models.StatusAccepted})
	for _, rule := range []string{"oi", "ioi", "acm"} {
		end := time.Now().Add(30 * time.Minute)
		f.s.DB.Model(&f.exam).Updates(map[string]any{"scoring_rule": rule, "freeze_enabled": rule != "oi", "freeze_duration_minutes": 60, "ends_at": end})
		for _, base := range []string{"/api", "/api/v1"} {
			for _, endpoint := range []string{fmt.Sprintf("/submissions/%d", sub.ID), fmt.Sprintf("/submissions/%d/events", sub.ID), fmt.Sprintf("/submissions?exam_id=%d", f.exam.ID), fmt.Sprintf("/submissions/latest?exam_id=%d", f.exam.ID)} {
				response := f.request("GET", base+endpoint, &f.student, nil)
				assertStatus(t, response, 200)
				body := response.Body.String()
				if !strings.Contains(body, `"status":"recorded"`) || strings.Contains(body, "private-judge-result") || strings.Contains(body, `"time_ms":302`) || strings.Contains(body, `"memory_kb":14656`) || strings.Contains(body, `"score":100`) {
					t.Fatalf("%s leaked %s", endpoint, body)
				}
			}
		}
		ranking := f.request("GET", fmt.Sprintf("/api/exams/%d/ranking", f.exam.ID), &f.student, nil)
		if rule == "oi" {
			assertStatus(t, ranking, 403)
		} else {
			assertStatus(t, ranking, 200)
			if !strings.Contains(ranking.Body.String(), `"frozen":true`) || strings.Contains(ranking.Body.String(), `"total_score":35`) {
				t.Fatal(ranking.Body.String())
			}
		}
		teacher := f.request("GET", fmt.Sprintf("/api/submissions/%d", sub.ID), &f.teacher, nil)
		assertStatus(t, teacher, 200)
		if !strings.Contains(teacher.Body.String(), `"time_ms":302`) {
			t.Fatal("teacher lost result visibility")
		}
	}
}

func TestExamOldSSERevocationPostgres(t *testing.T) {
	f := newExamFixture(t)
	if f.s.Redis == nil {
		t.Skip("requires OJ_TEST_REDIS_ADDR")
	}
	sub := models.Submission{UserID: f.student.ID, ProblemID: f.problem.ID, ProblemVersionID: f.version.ID, Language: "python", SourceCode: "print(3)", Status: models.StatusQueued}
	f.s.DB.Create(&sub)
	server := httptest.NewServer(f.router)
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	token, _ := middleware.SignToken(f.s.Cfg.JWTSecret, f.student)
	req, _ := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/api/submissions/%d/events", server.URL, sub.ID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	scanner := bufio.NewScanner(response.Body)
	initial := false
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), `"status":"queued"`) {
			initial = true
		}
		if scanner.Text() == "" && initial {
			break
		}
	}
	if !initial {
		t.Fatal("initial stream missing")
	}
	assertStatus(t, f.request("POST", fmt.Sprintf("/api/exams/%d/enter", f.exam.ID), &f.student, nil), 200)
	if err := f.s.Redis.Publish(ctx, streams.SubmissionChannel(sub.ID), "updated").Err(); err != nil {
		t.Fatal(err)
	}
	remaining, err := io.ReadAll(response.Body)
	if err != nil || !strings.Contains(string(remaining), "EXAM_SCOPE_REQUIRED") {
		t.Fatalf("old SSE not closed safely: %s %v", remaining, err)
	}
}

func TestExamAssetAndFailureClosedPostgres(t *testing.T) {
	f := newExamFixture(t)
	assertStatus(t, f.request("POST", fmt.Sprintf("/api/exams/%d/enter", f.exam.ID), &f.student, nil), 200)
	asset := fmt.Sprintf("/api/exams/%d/problems/%d/assets/assets/%%E5%%9B%%BE%%E7%%A4%%BA.png", f.exam.ID, f.problem.ID)
	manifest := datatypes.JSONMap{"assets": []any{map[string]any{"path": "assets/图示.png", "object": "version-one.png", "content_type": "image/png"}}}
	f.s.DB.Model(&f.version).Updates(map[string]any{"manifest": manifest, "statement": "![示意图](assets/%E5%9B%BE%E7%A4%BA.png)"})
	var requested []string
	objectStore := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requested = append(requested, r.URL.Path)
		if r.URL.Path != "/exam-test/version-one.png" {
			t.Errorf("wrong version object: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Content-Length", "3")
		w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
		if r.Method != "HEAD" {
			_, _ = w.Write([]byte("PNG"))
		}
	}))
	defer objectStore.Close()
	var err error
	f.s.MinIO, err = minio.New(strings.TrimPrefix(objectStore.URL, "http://"), &minio.Options{Creds: credentials.NewStaticV4("test", "testsecret", ""), Secure: false, Region: "us-east-1"})
	if err != nil {
		t.Fatal(err)
	}
	f.s.Cfg.MinIOBucket = "exam-test"
	f.router = f.s.Router()
	response := f.request("GET", asset, &f.student, nil)
	assertStatus(t, response, 200)
	if response.Body.String() != "PNG" || len(requested) == 0 || response.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatal("asset response incorrect")
	}
	assertStatus(t, f.request("GET", fmt.Sprintf("/api/exams/%d/problems/%d/assets/tests/1.in", f.exam.ID, f.problem.ID), &f.student, nil), 400)
	assertStatus(t, f.request("GET", fmt.Sprintf("/api/exams/%d/problems/%d/assets/assets/missing.png", f.exam.ID, f.problem.ID), &f.student, nil), 404)
	assertStatus(t, f.request("GET", strings.Replace(asset, fmt.Sprintf("/exams/%d/", f.exam.ID), fmt.Sprintf("/exams/%d/", f.other.ID), 1), &f.student, nil), 403)
	// Simulate a failed permission lookup without changing authentication data.
	if err := f.s.DB.Exec("ALTER TABLE exam_attempts RENAME TO unavailable_attempts").Error; err != nil {
		t.Fatal(err)
	}
	assertStatus(t, f.request("GET", "/api/problems", &f.student, nil), 503)
	assertStatus(t, f.request("GET", "/api/me/active-exam", &f.student, nil), 503)
	if f.s.streamExamAllowed(&f.student, models.Submission{}) {
		t.Fatal("SSE fails open")
	}
}

func TestTeamRejoinLifecyclePostgres(t *testing.T) {
	f := newExamFixture(t)
	team := models.Team{Name: "加入状态验收", Slug: "lifecycle", OwnerID: f.teacher.ID, Visibility: "private", JoinMode: "application"}
	f.s.DB.Create(&team)
	f.s.DB.Create(&models.TeamMembership{TeamID: team.ID, UserID: f.teacher.ID, Role: models.TeamRoleOwner})
	join := func() {
		t.Helper()
		if _, err := f.s.joinTeamTransaction(team.ID, f.student.ID, "", ""); err != nil {
			t.Fatal(err)
		}
	}
	application := func() models.TeamJoinApplication {
		var row models.TeamJoinApplication
		f.s.DB.Where("team_id = ? AND user_id = ?", team.ID, f.student.ID).Last(&row)
		return row
	}
	join()
	first := application()
	if err := f.s.reviewTeamApplicationTransaction(f.teacher, team.ID, first.ID, "reject"); err != nil {
		t.Fatal(err)
	}
	join()
	second := application()
	if second.ID == first.ID {
		t.Fatal("reapplication reused rejected request")
	}
	if err := f.s.reviewTeamApplicationTransaction(f.teacher, team.ID, second.ID, "approve"); err != nil {
		t.Fatal(err)
	}
	if err := f.s.changeTeamMembership(f.teacher, team.ID, f.student.ID, "remove", ""); err != nil {
		t.Fatal(err)
	}
	if err := f.s.reviewTeamApplicationTransaction(f.teacher, team.ID, second.ID, "approve"); err != nil {
		t.Fatal(err)
	}
	if _, joined := f.s.teamMembership(team.ID, f.student.ID); joined {
		t.Fatal("replayed approval reinstated member")
	}
	join()
	third := application()
	f.s.DB.Model(&team).Update("join_mode", "open")
	join()
	f.s.DB.First(&third, third.ID)
	if third.Status != "approved" {
		t.Fatal("direct join leaves stale application")
	}
	if err := f.s.changeTeamMembership(f.student, team.ID, f.student.ID, "leave", ""); err != nil {
		t.Fatal(err)
	}
	join()
	if err := f.s.changeTeamMembership(f.teacher, team.ID, f.student.ID, "role", models.TeamRoleAdmin); err != nil {
		t.Fatal(err)
	}
	join()
	member, _ := f.s.teamMembership(team.ID, f.student.ID)
	if member.Role != models.TeamRoleAdmin {
		t.Fatal("duplicate join overwrites role")
	}
	f.s.DB.Model(&team).Update("join_mode", "application")
	if _, err := f.s.joinTeamTransaction(team.ID, f.peer.ID, "", ""); err != nil {
		t.Fatal(err)
	}
	var removed models.TeamJoinApplication
	f.s.DB.Where("team_id = ? AND user_id = ?", team.ID, f.peer.ID).First(&removed)
	f.s.DB.Model(&f.peer).Update("account_deleted", true)
	if err := f.s.reviewTeamApplicationTransaction(f.teacher, team.ID, removed.ID, "approve"); err == nil {
		t.Fatal("deleted user approved")
	}
}
