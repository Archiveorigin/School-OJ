package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"school-oj/apps/api/internal/config"
	appdb "school-oj/apps/api/internal/db"
	"school-oj/apps/api/internal/middleware"
	"school-oj/apps/api/internal/models"
)

type examTestFixture struct {
	s                             Server
	router                        *gin.Engine
	student, peer, teacher, admin models.User
	exam, other                   models.Exam
	problem                       models.Problem
	version                       models.ProblemVersion
}

func newExamFixture(t *testing.T) *examTestFixture {
	t.Helper()
	dsn := os.Getenv("OJ_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set OJ_TEST_DATABASE_URL to an isolated local PostgreSQL test database")
	}
	connection, configErr := pgx.ParseConfig(dsn)
	if configErr != nil || connection.Host != "127.0.0.1" || connection.Database != "oj_exam_security_test" {
		t.Fatal("integration tests require local oj_exam_security_test database")
	}
	adminDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("exam_security_%d", time.Now().UnixNano())
	if err := adminDB.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatal(err)
	}
	db, err := gorm.Open(postgres.Open(dsn+" search_path="+schema), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		sql, _ := db.DB()
		_ = sql.Close()
		_ = adminDB.Exec("DROP SCHEMA " + schema + " CASCADE").Error
		sql, _ = adminDB.DB()
		_ = sql.Close()
	})
	// Mirror the existing production ORM baseline, then exercise forward SQL
	// migrations (including the new activity table) on that schema.
	var baseline []any
	for _, model := range models.AllModels() {
		if _, activity := model.(*models.ExamActivity); !activity {
			baseline = append(baseline, model)
		}
	}
	if err := db.AutoMigrate(baseline...); err != nil {
		t.Fatal(err)
	}
	if err := appdb.Migrate(db); err != nil {
		t.Fatal(err)
	}
	f := &examTestFixture{s: Server{DB: db, Cfg: config.Config{JWTSecret: "isolated-exam-fixture-secret", CORSOrigins: []string{"http://127.0.0.1:4176"}}}}
	if addr := os.Getenv("OJ_TEST_REDIS_ADDR"); addr != "" {
		if addr != "127.0.0.1:56379" {
			t.Fatal("integration tests require dedicated local Redis port 56379")
		}
		f.s.Redis = redis.NewClient(&redis.Options{Addr: addr})
		t.Cleanup(func() { _ = f.s.Redis.Close() })
	}
	password, _ := bcrypt.GenerateFromPassword([]byte("ExamFixture-2026!"), bcrypt.MinCost)
	for index, user := range []*models.User{&f.student, &f.peer, &f.teacher, &f.admin} {
		role := []models.Role{models.RoleStudent, models.RoleStudent, models.RoleTeacher, models.RoleAdmin}[index]
		*user = models.User{Email: fmt.Sprintf("exam-%s-%d@example.test", role, index), Name: []string{"考生甲", "考生乙", "课程教师", "系统管理员"}[index], Role: role, PasswordHash: string(password), EmailVerified: true}
		if err := db.Create(user).Error; err != nil {
			t.Fatal(err)
		}
	}
	course := models.Course{Code: "EXAM-QA", Name: "考试安全验收课程", TeacherID: f.teacher.ID}
	if err := db.Create(&course).Error; err != nil {
		t.Fatal(err)
	}
	for _, user := range []models.User{f.student, f.peer} {
		if err := db.Create(&models.CourseMembership{CourseID: course.ID, UserID: user.ID, Role: models.RoleStudent}).Error; err != nil {
			t.Fatal(err)
		}
	}
	f.problem = models.Problem{OwnerID: f.teacher.ID, Slug: "exam-qa", Title: "特殊扫雷", DisplayCode: "T001", Statement: "## 题目描述\n输出两个整数之和。\n\n```text\n1 2\n```", Difficulty: "入门", Manifest: datatypes.JSONMap{"cases": []any{map[string]any{"name": "one", "input": "tests/1.in", "output": "tests/1.out"}, map[string]any{"name": "two", "input": "tests/2.in", "output": "tests/2.out"}}}}
	if err := db.Create(&f.problem).Error; err != nil {
		t.Fatal(err)
	}
	f.version = models.ProblemVersion{ProblemID: f.problem.ID, Version: 1, Title: f.problem.Title, Statement: f.problem.Statement, Manifest: f.problem.Manifest, CreatedBy: f.teacher.ID}
	if err := db.Create(&f.version).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&f.problem).Update("current_version_id", f.version.ID).Error; err != nil {
		t.Fatal(err)
	}
	start, end := time.Now().Add(-time.Hour), time.Now().Add(time.Hour)
	f.exam = models.Exam{CourseID: course.ID, Title: "课程考试安全验收", Description: "请独立完成本场考试。", StartsAt: &start, EndsAt: &end, ScoringRule: "ioi", Settings: datatypes.JSONMap{"ranking_visible": true}}
	if err := db.Create(&f.exam).Error; err != nil {
		t.Fatal(err)
	}
	f.other = f.exam
	f.other.ID = 0
	f.other.Title = "另一场考试"
	if err := db.Create(&f.other).Error; err != nil {
		t.Fatal(err)
	}
	for _, exam := range []models.Exam{f.exam, f.other} {
		if err := db.Create(&models.ExamProblem{ExamID: exam.ID, ProblemID: f.problem.ID, ProblemVersionID: f.version.ID, Label: "B", Score: 35}).Error; err != nil {
			t.Fatal(err)
		}
	}
	f.router = f.s.Router()
	return f
}

func (f *examTestFixture) request(method, path string, user *models.User, body any) *httptest.ResponseRecorder {
	data, _ := json.Marshal(body)
	request := httptest.NewRequest(method, path, bytes.NewReader(data))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Forwarded-For", "198.51.100.99")
	if user != nil {
		token, _ := middleware.SignToken(f.s.Cfg.JWTSecret, *user)
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response := httptest.NewRecorder()
	f.router.ServeHTTP(response, request)
	return response
}

func assertStatus(t *testing.T, response *httptest.ResponseRecorder, status int) {
	t.Helper()
	if response.Code != status {
		t.Fatalf("status=%d want=%d body=%s", response.Code, status, response.Body.String())
	}
}

func TestExamSecurityPostgres(t *testing.T) {
	f := newExamFixture(t)
	student := &f.student
	get := fmt.Sprintf("/api/exams/%d", f.exam.ID)
	assertExamDescriptionOnly(t, f.request("GET", get, student, nil))
	var count int64
	f.s.DB.Model(&models.ExamAttempt{}).Count(&count)
	if count != 0 {
		t.Fatal("GET silently entered exam")
	}
	assertStatus(t, f.request("POST", get+"/enter", student, nil), 200)
	assertStatus(t, f.request("POST", get+"/enter", student, nil), 200)
	assertStatus(t, f.request("GET", get, student, nil), 200)
	staleClaims := *student
	staleClaims.Role = models.RoleAdmin
	assertStatus(t, f.request("GET", "/api/problems", &staleClaims, nil), 403)
	for _, base := range []string{"/api", "/api/v1"} {
		for _, path := range []string{"/problems", "/problems/catalog", "/tags", "/profile", "/teams", "/courses", "/submissions", "/submissions/latest", fmt.Sprintf("/exams/%d", f.other.ID)} {
			assertStatus(t, f.request("GET", base+path, student, nil), 403)
		}
		assertStatus(t, f.request("GET", base+"/me", student, nil), 200)
		assertStatus(t, f.request("GET", base+"/problems", nil, nil), 200)
		assertStatus(t, f.request("GET", base+"/problems", &f.teacher, nil), 200)
		assertStatus(t, f.request("GET", base+fmt.Sprintf("/submissions?exam_id=%d", f.exam.ID), student, nil), 200)
		assertStatus(t, f.request("GET", base+fmt.Sprintf("/submissions?exam_id=%d&visibility=problem", f.exam.ID), student, nil), 403)
		assertStatus(t, f.request("POST", base+"/submissions", student, gin.H{"problem_id": f.problem.ID, "exam_id": f.other.ID}), 403)
	}
	var activity models.ExamActivity
	f.s.DB.Where("event = 'enter'").First(&activity)
	if activity.IP == "198.51.100.99" || activity.UserID != student.ID {
		t.Fatalf("forged identity/IP admitted: %+v", activity)
	}
	sub := models.Submission{UserID: student.ID, ProblemID: f.problem.ID, ProblemVersionID: f.version.ID, ExamID: &f.exam.ID, Language: "python", SourceCode: "print(3)", Status: models.StatusWrongAnswer, TimeMS: 302, MemoryKB: 14656}
	if err := f.s.DB.Create(&sub).Error; err != nil {
		t.Fatal(err)
	}
	f.s.DB.Create(&models.SubmissionResult{SubmissionID: sub.ID, CaseName: "one", Status: models.StatusWrongAnswer})
	response := f.request("GET", fmt.Sprintf("/api/submissions/%d", sub.ID), student, nil)
	assertStatus(t, response, 200)
	if !strings.Contains(response.Body.String(), `"total":2`) {
		t.Fatal(response.Body.String())
	}
	v2 := f.version
	v2.ID = 0
	v2.Version = 2
	v2.Manifest = datatypes.JSONMap{"cases": []any{map[string]any{"input": "tests/1.in", "output": "tests/1.out"}}}
	f.s.DB.Create(&v2)
	f.s.DB.Model(&f.problem).Update("current_version_id", v2.ID)
	response = f.request("GET", fmt.Sprintf("/api/submissions/%d", sub.ID), student, nil)
	if !strings.Contains(response.Body.String(), `"total":2`) {
		t.Fatal("historical version changed")
	}
	assertStatus(t, f.request("GET", fmt.Sprintf("/api/submissions/%d", sub.ID), &f.peer, nil), 403)
	f.s.DB.Model(&f.exam).Update("scoring_rule", "oi")
	response = f.request("GET", fmt.Sprintf("/api/submissions/%d", sub.ID), student, nil)
	if !strings.Contains(response.Body.String(), `"case_summary":null`) || !strings.Contains(response.Body.String(), `"status":"recorded"`) || strings.Contains(response.Body.String(), `"time_ms":302`) {
		t.Fatal(response.Body.String())
	}
	assertStatus(t, f.request("GET", get+"/ranking", student, nil), 403)
	assertStatus(t, f.request("POST", get+"/finish", student, nil), 200)
	assertStatus(t, f.request("POST", get+"/finish", student, nil), 200)
	assertStatus(t, f.request("GET", "/api/problems", student, nil), 200)
	assertExamDescriptionOnly(t, f.request("GET", get, student, nil))
	assertStatus(t, f.request("POST", get+"/enter", student, nil), 403)
	f.s.DB.Model(&models.ExamActivity{}).Where("event = 'finish'").Count(&count)
	if count != 1 {
		t.Fatalf("finish log count=%d", count)
	}
}

func assertExamDescriptionOnly(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()
	assertStatus(t, response, 200)
	var data struct {
		CanRead   bool              `json:"can_read_problems"`
		CanSubmit bool              `json:"can_submit"`
		Problems  []json.RawMessage `json:"problems"`
		Exam      struct {
			Problems []json.RawMessage `json:"problems"`
		} `json:"exam"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &data); err != nil {
		t.Fatal(err)
	}
	if data.CanRead || data.CanSubmit || len(data.Problems) != 0 || len(data.Exam.Problems) != 0 {
		t.Fatal("description exposed exam paper")
	}
}

func TestExamEntryConcurrencyPostgres(t *testing.T) {
	f := newExamFixture(t)
	var wg sync.WaitGroup
	for _, id := range []uint{f.exam.ID, f.other.ID} {
		wg.Add(1)
		go func(id uint) {
			defer wg.Done()
			f.request("POST", fmt.Sprintf("/api/exams/%d/enter", id), &f.student, nil)
		}(id)
	}
	wg.Wait()
	var count int64
	f.s.DB.Model(&models.ExamAttempt{}).Where("user_id = ?", f.student.ID).Count(&count)
	if count != 1 {
		t.Fatalf("concurrent entry created %d attempts", count)
	}
	active, err := activeExamInDB(f.s.DB, f.student.ID, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if f.s.streamExamAllowed(&f.student, models.Submission{UserID: f.student.ID, ProblemID: f.problem.ID}) {
		t.Fatal("old practice stream not revoked")
	}
	now := time.Now()
	f.s.DB.Model(&active).Update("ends_at", now)
	assertStatus(t, f.request("GET", "/api/problems", &f.student, nil), 200)
}

func TestTeamJoinConcurrencyPostgres(t *testing.T) {
	f := newExamFixture(t)
	team := models.Team{Name: "私有邀请团队", Slug: "private-qa", OwnerID: f.teacher.ID, Visibility: "private", JoinMode: "invitation", JoinCode: "AbCd1234"}
	f.s.DB.Create(&team)
	f.s.DB.Create(&models.TeamMembership{TeamID: team.ID, UserID: f.teacher.ID, Role: models.TeamRoleOwner})
	assertStatus(t, f.request("GET", "/api/teams/private-qa", &f.student, nil), 404)
	preview := f.request("GET", "/api/teams/private-qa/join-preview", &f.student, nil)
	assertStatus(t, preview, 200)
	if strings.Contains(preview.Body.String(), "AbCd1234") || strings.Contains(preview.Body.String(), "announcement") {
		t.Fatal("preview leaks private fields")
	}
	path := fmt.Sprintf("/api/teams/%d/join", team.ID)
	assertStatus(t, f.request("POST", path, &f.student, gin.H{"join_code": "wrong"}), 403)
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			response := f.request("POST", path, &f.student, gin.H{"join_code": " abcd1234 "})
			if response.Code != 200 {
				t.Errorf("join: %s", response.Body.String())
			}
		}()
	}
	wg.Wait()
	var count int64
	f.s.DB.Model(&models.TeamMembership{}).Where("team_id = ? AND user_id = ?", team.ID, f.student.ID).Count(&count)
	if count != 1 {
		t.Fatalf("memberships=%d", count)
	}
	member := f.request("GET", "/api/teams/private-qa", &f.student, nil)
	if strings.Contains(member.Body.String(), "join_code") {
		t.Fatal("ordinary member sees invite code")
	}
	manager := f.request("GET", "/api/teams/private-qa", &f.admin, nil)
	if !strings.Contains(manager.Body.String(), "AbCd1234") {
		t.Fatal("admin cannot manage team")
	}
	f.s.DB.Model(&team).Update("join_mode", "application")
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			response := f.request("POST", path, &f.peer, gin.H{"message": "申请加入"})
			if response.Code != 202 {
				t.Errorf("apply: %s", response.Body.String())
			}
		}()
	}
	wg.Wait()
	var application models.TeamJoinApplication
	f.s.DB.Where("team_id = ? AND user_id = ?", team.ID, f.peer.ID).First(&application)
	f.s.DB.Model(&models.TeamJoinApplication{}).Where("team_id = ? AND user_id = ? AND status = 'pending'", team.ID, f.peer.ID).Count(&count)
	if count != 1 {
		t.Fatalf("applications=%d", count)
	}
	for _, action := range []string{"approve", "reject"} {
		wg.Add(1)
		go func(action string) {
			defer wg.Done()
			_ = f.s.teamAccess().Review(context.Background(), f.teacher.ID, team.ID, application.ID, action)
		}(action)
	}
	wg.Wait()
	f.s.DB.First(&application, application.ID)
	f.s.DB.Model(&models.TeamMembership{}).Where("team_id = ? AND user_id = ?", team.ID, f.peer.ID).Count(&count)
	if (application.Status == "approved") != (count == 1) {
		t.Fatal("review result and membership disagree")
	}
	if application.Status == "approved" {
		f.s.DB.Where("team_id = ? AND user_id = ?", team.ID, f.peer.ID).Delete(&models.TeamMembership{})
		if err := f.s.teamAccess().Review(context.Background(), f.teacher.ID, team.ID, application.ID, "approve"); err != nil {
			t.Fatal(err)
		}
		f.s.DB.Model(&models.TeamMembership{}).Where("team_id = ? AND user_id = ?", team.ID, f.peer.ID).Count(&count)
		if count != 0 {
			t.Fatal("review replay reinstated removed member")
		}
	}
}

// Opt-in fixture: the real router and authentication, never shipped in API builds.
func TestExamBrowserFixture(t *testing.T) {
	addr := os.Getenv("OJ_BROWSER_FIXTURE_ADDR")
	if addr == "" {
		t.Skip("browser fixture is opt-in")
	}
	if addr != "127.0.0.1:18086" {
		t.Fatal("fixture must bind the approved loopback port")
	}
	f := newExamFixture(t)
	if f.s.Redis == nil {
		t.Fatal("browser fixture requires a real test Redis")
	}
	exams := []models.Exam{f.exam, f.other}
	f.s.DB.Model(&exams[1]).Updates(map[string]any{"scoring_rule": "oi", "title": "OI 课程考试"})
	for _, rule := range []string{"acm", "ioi", "acm"} {
		exam := f.exam
		exam.ID = 0
		exam.ScoringRule = rule
		exam.Title = strings.ToUpper(rule) + " 课程考试"
		if len(exams) >= 3 {
			exam.FreezeEnabled = true
			end := time.Now().Add(30 * time.Minute)
			exam.EndsAt = &end
			exam.FreezeDurationMinutes = 60
			exam.Title += " · 封榜"
		}
		if err := f.s.DB.Create(&exam).Error; err != nil {
			t.Fatal(err)
		}
		if err := f.s.DB.Create(&models.ExamProblem{ExamID: exam.ID, ProblemID: f.problem.ID, ProblemVersionID: f.version.ID, Label: "B", Score: 35}).Error; err != nil {
			t.Fatal(err)
		}
		exams = append(exams, exam)
	}
	for _, exam := range exams {
		for _, user := range []models.User{f.student, f.peer} {
			sub := models.Submission{UserID: user.ID, ProblemID: f.problem.ID, ProblemVersionID: f.version.ID, ExamID: &exam.ID, Language: "python", SourceCode: "# 两数之和\na, b = map(int, input().split())\n\nprint(a + b)\n", Status: models.StatusAccepted, Score: 100, TimeMS: 302, MemoryKB: 14656}
			if err := f.s.DB.Create(&sub).Error; err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"one", "two"} {
				f.s.DB.Create(&models.SubmissionResult{SubmissionID: sub.ID, CaseName: name, Status: models.StatusAccepted, TimeMS: 302, MemoryKB: 14656})
			}
		}
	}
	for _, mode := range []string{"invitation", "application", "open"} {
		team := models.Team{Name: "验收团队 " + mode, Slug: "qa-" + mode, OwnerID: f.teacher.ID, Visibility: "private", JoinMode: mode, JoinCode: "ExamJoin2026"}
		if err := f.s.DB.Create(&team).Error; err != nil {
			t.Fatal(err)
		}
		f.s.DB.Create(&models.TeamMembership{TeamID: team.ID, UserID: f.teacher.ID, Role: models.TeamRoleOwner})
		if mode == "open" {
			start := time.Now().Add(-time.Hour)
			contest := models.TeamContest{TeamID: team.ID, CreatedBy: f.teacher.ID, Title: "团队比赛共享界面", Description: "团队比赛说明", StartsAt: &start, DurationMinutes: 180, ScoringRule: "ioi", State: "published", PublishedAt: &start}
			if err := f.s.DB.Create(&contest).Error; err != nil {
				t.Fatal(err)
			}
			f.s.DB.Create(&models.TeamContestProblem{ContestID: contest.ID, ProblemID: f.problem.ID, ProblemVersionID: f.version.ID, Label: "B"})
		}
	}
	assignment := models.Assignment{CourseID: f.exam.CourseID, Title: "共享提交详情作业"}
	if err := f.s.DB.Create(&assignment).Error; err != nil {
		t.Fatal(err)
	}
	f.s.DB.Create(&models.AssignmentProblem{AssignmentID: assignment.ID, ProblemID: f.problem.ID})
	set := models.TeamProblemSet{TeamID: 3, CreatedBy: f.teacher.ID, Title: "共享提交详情题单"}
	if err := f.s.DB.Create(&set).Error; err != nil {
		t.Fatal(err)
	}
	f.s.DB.Create(&models.TeamProblemSetProblem{ProblemSetID: set.ID, ProblemID: f.problem.ID, Label: "B"})
	manual := models.Exam{CourseID: f.exam.CourseID, Title: "共享人工阅卷", StartsAt: f.exam.StartsAt, EndsAt: f.exam.EndsAt, ScoringRule: "ioi", Settings: datatypes.JSONMap{"manual_review": true}}
	if err := f.s.DB.Create(&manual).Error; err != nil {
		t.Fatal(err)
	}
	f.s.DB.Create(&models.ExamProblem{ExamID: manual.ID, ProblemID: f.problem.ID, ProblemVersionID: f.version.ID, Label: "B", Score: 100})
	contestID := uint(1)
	for _, user := range []models.User{f.student, f.peer} {
		for _, scope := range []string{"practice", "assignment", "contest", "set", "manual"} {
			sub := models.Submission{UserID: user.ID, ProblemID: f.problem.ID, ProblemVersionID: f.version.ID, Language: "python", SourceCode: "# 共享提交详情\nprint(3)\n", Status: models.StatusAccepted, Score: 100, TimeMS: 302, MemoryKB: 14656, IsPublic: true}
			switch scope {
			case "assignment":
				sub.AssignmentID = &assignment.ID
			case "contest":
				sub.TeamContestID = &contestID
			case "set":
				sub.ProblemSetID = &set.ID
			case "manual":
				sub.ExamID = &manual.ID
				sub.Status = models.StatusPendingReview
			}
			if err := f.s.DB.Create(&sub).Error; err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"one", "two"} {
				f.s.DB.Create(&models.SubmissionResult{SubmissionID: sub.ID, CaseName: name, Status: models.StatusAccepted})
			}
		}
	}
	t.Logf("FIXTURE_READY exam=%d other=%d problem=%d student=%s teacher=%s admin=%s", f.exam.ID, f.other.ID, f.problem.ID, f.student.Email, f.teacher.Email, f.admin.Email)
	if err := http.ListenAndServe(addr, f.router); err != nil {
		t.Fatal(err)
	}
}
