package handlers

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"school-oj/apps/api/internal/authz"
	"school-oj/apps/api/internal/middleware"
	"school-oj/apps/api/internal/models"
	"school-oj/apps/api/internal/services"
	"school-oj/apps/api/internal/visibility"
)

func TestAuthorizationRouteInventory(t *testing.T) {
	router := (Server{}).Router()
	counts := map[string]int{}
	for _, route := range router.Routes() {
		if !strings.HasPrefix(route.Path, "/api/") {
			continue
		}
		policy, ok := authz.Route(route.Method, route.Path)
		if !ok {
			t.Fatalf("unregistered %s %s", route.Method, route.Path)
		}
		counts[policy.Method+" "+policy.Path]++
		for _, role := range []string{"admin", "teacher", "student"} {
			decision := authz.CheckRoute(authz.Context{UserID: 1, Roles: []string{role}}, policy)
			want := policy.Access == "public" || policy.Access == "authenticated" || authz.HasContext(authz.Context{UserID: 1, Roles: []string{role}}, policy.Access)
			if decision.Allow != want {
				t.Fatal(route, role, decision)
			}
		}
		if authz.CheckRoute(authz.Context{}, policy).Allow != (policy.Access == "public") {
			t.Fatal("anonymous grant", route)
		}
	}
	for _, policy := range authz.Routes {
		if counts[policy.Method+" "+policy.Path] != 2 {
			t.Fatal("missing/duplicate route", policy)
		}
	}
	if authz.CheckRoute(authz.Context{UserID: 1, Roles: []string{"admin"}}, authz.RoutePolicy{}).Allow {
		t.Fatal("unknown route allowed")
	}
	// OpenAPI and route policy metadata are independently maintained artifacts.
	data, err := os.ReadFile("../../openapi/openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		Paths map[string]map[string]struct {
			Code   string `json:"x-policy"`
			Access string `json:"x-access"`
		}
	}
	if err = json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	for _, policy := range authz.Routes {
		path := policy.Path
		for _, segment := range strings.Split(path, "/") {
			if strings.HasPrefix(segment, ":") || strings.HasPrefix(segment, "*") {
				path = strings.Replace(path, segment, "{"+segment[1:]+"}", 1)
			}
		}
		op := document.Paths[path][strings.ToLower(policy.Method)]
		if op.Code != policy.Code || op.Access != policy.Access {
			t.Fatal("OpenAPI authorization drift", policy)
		}
	}
}

func TestAuthorizationIsolationPostgres(t *testing.T) {
	f := newExamFixture(t)
	for _, prefix := range []string{"/api", "/api/v1"} {
		assertStatus(t, f.request("GET", prefix+"/authorization/policies", &f.student, nil), 403)
		assertStatus(t, f.request("GET", prefix+"/authorization/policies", &f.admin, nil), 200)
		assertStatus(t, f.request("GET", prefix+"/users", nil, nil), 401)
	}
	t.Run("permission revocation and legacy token", func(t *testing.T) {
		if err := f.s.DB.Model(&f.student).Update("can_author", true).Error; err != nil {
			t.Fatal(err)
		}
		var before models.User
		f.s.DB.First(&before, f.student.ID)
		token, _ := middleware.SignToken(f.s.Cfg.JWTSecret, before)
		if err := f.s.DB.Model(&f.student).Update("can_author", false).Error; err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest("GET", "/api/v1/me", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		response := httptest.NewRecorder()
		f.router.ServeHTTP(response, req)
		assertStatus(t, response, 200)
		var after sessionUser
		if err := json.Unmarshal(response.Body.Bytes(), &after); err != nil {
			t.Fatal(err)
		}
		if after.PermVersion <= before.PermVersion || strings.Contains(response.Body.String(), authz.ProblemAuthor) {
			t.Fatal("stale permission snapshot", response.Body.String())
		}
		// No stale role claim can confer an administrator capability.
		forged := f.student
		forged.Role = models.RoleAdmin
		assertStatus(t, f.request("GET", "/api/users", &forged, nil), 403)
	})
	t.Run("teacher owned problem reused by unrelated team", func(t *testing.T) {
		team := models.Team{Name: "private", Slug: "authz-private", OwnerID: f.peer.ID, Visibility: "private", JoinMode: "open", ContestPermission: "owner"}
		if err := f.s.DB.Create(&team).Error; err != nil {
			t.Fatal(err)
		}
		f.s.DB.Create(&models.TeamMembership{TeamID: team.ID, UserID: f.peer.ID, Role: models.TeamRoleOwner})
		set := models.TeamProblemSet{TeamID: team.ID, CreatedBy: f.peer.ID, Title: "private set"}
		f.s.DB.Create(&set)
		sub := models.Submission{UserID: f.peer.ID, ProblemID: f.problem.ID, ProblemVersionID: f.version.ID, ProblemSetID: &set.ID, Language: "python", SourceCode: "private-team-secret", Status: models.StatusAccepted}
		if err := f.s.DB.Create(&sub).Error; err != nil {
			t.Fatal(err)
		}
		for _, prefix := range []string{"/api", "/api/v1"} {
			response := f.request("GET", prefix+fmt.Sprintf("/submissions?problem_id=%d", f.problem.ID), &f.teacher, nil)
			assertStatus(t, response, 200)
			if strings.Contains(response.Body.String(), "private-team-secret") {
				t.Fatal("cross-team submission leaked")
			}
			assertStatus(t, f.request("GET", prefix+fmt.Sprintf("/submissions/%d", sub.ID), &f.teacher, nil), 403)
		}
		var count int64
		f.s.DB.Model(&models.Submission{}).Scopes(visibility.Submissions(f.teacher)).Count(&count)
		if count != 0 {
			t.Fatal("scope count leaks hidden submissions", count)
		}
		f.s.DB.Create(&models.TeamMembership{TeamID: team.ID, UserID: f.student.ID, Role: models.TeamRoleAdmin})
		response := f.request("GET", "/api/submissions", &f.student, nil)
		assertStatus(t, response, 200)
		if !strings.Contains(response.Body.String(), "private-team-secret") {
			t.Fatal("scoped student team administrator denied")
		}
	})
	t.Run("course filter cannot cross class boundary", func(t *testing.T) {
		class := models.Class{CourseID: f.exam.CourseID, Name: "Other class", JoinCode: "authz-other"}
		f.s.DB.Create(&class)
		exam := models.Exam{CourseID: f.exam.CourseID, ClassID: &class.ID, Title: "hidden class exam", EventReview: models.EventReview{ReviewStatus: "approved"}}
		f.s.DB.Create(&exam)
		assignment := models.Assignment{CourseID: f.exam.CourseID, ClassID: &class.ID, Title: "hidden class assignment"}
		f.s.DB.Create(&assignment)
		for _, path := range []string{"exams", "assignments"} {
			response := f.request("GET", fmt.Sprintf("/api/%s?course_id=%d", path, f.exam.CourseID), &f.student, nil)
			assertStatus(t, response, 200)
			if strings.Contains(response.Body.String(), "hidden class") {
				t.Fatal("client course filter bypassed class visibility", response.Body.String())
			}
		}
	})
	t.Run("unrelated teacher cannot submit private problem", func(t *testing.T) {
		if err := f.s.DB.Create(&models.PreparedProblem{ProblemID: f.problem.ID, OwnerID: f.teacher.ID}).Error; err != nil {
			t.Fatal(err)
		}
		other := models.User{Email: "authz-other@example.test", Name: "other teacher", Role: models.RoleTeacher, PasswordHash: "unused"}
		f.s.DB.Create(&other)
		assertStatus(t, f.request("POST", "/api/submissions", &other, gin.H{"problem_id": f.problem.ID, "language": "python", "source_code": "print(3)"}), 403)
	})
	t.Run("audit deny and privileged allow without payload", func(t *testing.T) {
		assertStatus(t, f.request("GET", "/api/users?token=do-not-log-this", &f.student, nil), 403)
		assertStatus(t, f.request("GET", "/api/authorization/policies", &f.admin, nil), 200)
		for _, action := range []string{"authorization.deny", "authorization.allow"} {
			var logs []models.AuditLog
			f.s.DB.Where("action=?", action).Find(&logs)
			if len(logs) == 0 {
				t.Fatal("missing audit", action)
			}
			data, _ := json.Marshal(logs)
			if strings.Contains(string(data), "do-not-log-this") || strings.Contains(string(data), "private-team-secret") {
				t.Fatal("audit contains sensitive payload")
			}
		}
	})
	t.Run("password reset revokes existing sessions", func(t *testing.T) {
		oldToken, _ := middleware.SignToken(f.s.Cfg.JWTSecret, f.teacher)
		assertStatus(t, f.request("POST", fmt.Sprintf("/api/users/%d/reset-password", f.teacher.ID), &f.admin, gin.H{"password": "New-Authorization-2026!"}), 200)
		req := httptest.NewRequest("GET", "/api/me", nil)
		req.Header.Set("Authorization", "Bearer "+oldToken)
		response := httptest.NewRecorder()
		f.router.ServeHTTP(response, req)
		assertStatus(t, response, 401)
		if !strings.Contains(response.Body.String(), "SESSION_REVOKED") {
			t.Fatal(response.Body.String())
		}
		assertStatus(t, f.request("POST", "/api/auth/login", nil, gin.H{"email": f.teacher.Email, "password": "New-Authorization-2026!"}), 200)
	})
	t.Run("last administrator survives concurrent removals", func(t *testing.T) {
		second := models.User{Email: "authz-admin@example.test", Name: "second", Role: models.RoleAdmin, PasswordHash: "unused"}
		if err := f.s.DB.Create(&second).Error; err != nil {
			t.Fatal(err)
		}
		var wait sync.WaitGroup
		results := make(chan error, 2)
		for _, id := range []uint{f.admin.ID, second.ID} {
			wait.Add(1)
			go func(id uint) {
				defer wait.Done()
				results <- services.UpdateAccountAccess(f.s.DB, id, map[string]any{"role": models.RoleStudent})
			}(id)
		}
		wait.Wait()
		close(results)
		denied := 0
		for err := range results {
			if err == services.ErrLastAdministrator {
				denied++
			} else if err != nil {
				t.Fatal(err)
			}
		}
		if denied != 1 {
			t.Fatal("expected exactly one protected administrator", denied)
		}
		var last models.User
		f.s.DB.Where("role = ? AND account_deleted = false", models.RoleAdmin).First(&last)
		if last.ID == 0 {
			t.Fatal("no administrator remains")
		}
		assertStatus(t, f.request("DELETE", "/api/profile", &last, nil), 400)
	})
}
