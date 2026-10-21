package authz

import (
	"school-oj/apps/api/internal/models"
	"testing"
)

func TestCapabilityMatrix(t *testing.T) {
	// Expected grants are intentionally independent of the production registry.
	expected := map[string][]string{
		"admin":   {UserManage, PermissionManage, AuditView, EventReview, ProblemReview, ProblemAuthor, CourseTeach, CourseCreate, CourseAny, TeamAny, ProblemAny, SubmissionAny, SubmissionResultsAny, TicketAny, EventAny},
		"teacher": {CourseTeach, CourseCreate}, "student": {ExamParticipate},
	}
	for role, allowed := range expected {
		for _, policy := range Policies() {
			if policy.Scope != "global" {
				continue
			}
			want := false
			for _, code := range allowed {
				want = want || code == policy.Code
			}
			ctx := Context{UserID: 7, Roles: []string{role}}
			if got := Check(ctx, policy.Code, Resource{}); got.Allow != want {
				t.Errorf("%s %s: %+v want=%v", role, policy.Code, got, want)
			}
			for _, invalid := range []Context{{}, {UserID: 7}, {UserID: 7, Roles: []string{"unknown"}}, {UserID: 7, Roles: []string{"admin", "unknown"}}, {UserID: 7, Roles: []string{"problem_author"}}} {
				if Check(invalid, policy.Code, Resource{}).Allow {
					t.Errorf("invalid identity allowed %s", policy.Code)
				}
			}
		}
		if Check(Context{UserID: 7, Roles: []string{role}}, "undefined:operation", Resource{ID: 1}).Allow {
			t.Fatal("undefined policy allowed")
		}
	}
	for _, role := range []models.Role{models.RoleStudent, models.RoleTeacher} {
		user := models.User{ID: 7, Role: role, CanAuthor: true}
		if !Has(user, ProblemAuthor) || Has(user, PermissionManage) {
			t.Fatal("orthogonal author grant leaked privileges")
		}
		user.AccountDeleted = true
		if Has(user, ProblemAuthor) {
			t.Fatal("deleted identity allowed")
		}
	}
}

func TestResourceMatrix(t *testing.T) {
	for _, base := range []string{"student", "teacher", "admin"} {
		ctx := Context{UserID: 7, Roles: []string{base}}
		for _, membership := range []string{"", "owner", "admin", "member", "invalid"} {
			for _, joined := range []bool{false, true} {
				for _, setting := range []string{"owner", "admin", "all", "invalid"} {
					member := joined && (membership == "owner" || membership == "admin" || membership == "member")
					want := map[string]bool{
						TeamRead:        member,
						TeamManage:      member && (membership == "owner" || membership == "admin"),
						TeamPermissions: member && membership == "owner",
						TeamContent:     member && (membership == "owner" || setting == "all" || setting == "admin" && membership == "admin"),
					}
					for code, allowed := range want {
						resource := Resource{ID: 3, Member: joined, Membership: membership, ContentPermission: setting}
						if got := Check(ctx, code, resource); got.Allow != (allowed || base == "admin") {
							t.Errorf("%s %s joined=%v %s %s: %+v", base, membership, joined, setting, code, got)
						}
					}
				}
			}
		}
		for _, membership := range []string{"course_admin", "course_assistant", "student", "invalid"} {
			for _, code := range []string{CourseManage, CourseAdmin} {
				want := base == "admin" || base == "teacher" && (membership == "course_admin" || membership == "course_assistant" && code == CourseManage)
				if Check(ctx, code, Resource{ID: 2, Member: true, Membership: membership}).Allow != want {
					t.Errorf("%s %s %s", base, membership, code)
				}
				if Check(ctx, code, Resource{ID: 2}).Allow != (base == "admin") {
					t.Fatal("cross-course membership leaked")
				}
				if Check(ctx, code, Resource{}).Allow {
					t.Fatal("missing resource allowed")
				}
			}
		}
	}
}

func BenchmarkResourceDecision(b *testing.B) {
	ctx := Context{UserID: 1, Roles: []string{"teacher"}}
	for i := 0; i < b.N; i++ {
		Check(ctx, CourseManage, Resource{ID: 3, Member: true, Membership: "course_assistant"})
	}
}
