// Package authz owns operation decisions. It has no HTTP, database or cache state.
// Resource attributes must come from trusted repositories, never request bodies.
package authz

import (
	_ "embed"
	"encoding/json"
	"sort"

	"school-oj/apps/api/internal/models"
)

const (
	UserManage           = "user:manage"
	PermissionManage     = "permission:manage"
	AuditView            = "audit:view"
	EventReview          = "event:review"
	ProblemReview        = "problem:review"
	ProblemAuthor        = "problem:author"
	CourseTeach          = "course:teach"
	CourseCreate         = "course:create"
	CourseAny            = "course:manage:any"
	TeamAny              = "team:manage:any"
	ProblemAny           = "problem:manage:any"
	SubmissionAny        = "submission:read:any"
	SubmissionResultsAny = "submission:results:any"
	TicketAny            = "ticket:manage:any"
	EventAny             = "event:read:any"
	ExamParticipate      = "exam:participate"
	CourseManage         = "course:manage"
	CourseAdmin          = "course:admin"
	TeamRead             = "team:read"
	TeamManage           = "team:manage"
	TeamPermissions      = "team:permissions"
	TeamContent          = "team:content"
)

type Context struct {
	UserID      uint     `json:"user_id"`
	Roles       []string `json:"roles"`
	PermVersion int64    `json:"perm_version"`
}

type Resource struct {
	ID                uint
	OwnerID           uint
	Member            bool
	Membership        string
	ContentPermission string
}

type Decision struct {
	Allow  bool   `json:"allow"`
	Reason string `json:"reason"`
}

// Engine is the stable PDP contract. Additional bounded ABAC/ReBAC engines can
// implement it without changing authentication, SQL scopes or audit consumers.
type Engine interface {
	Check(Context, string, Resource) Decision
}
type BaselineEngine struct{}

type Policy struct {
	Code  string   `json:"code"`
	Model string   `json:"model"`
	Scope string   `json:"scope"`
	Roles []string `json:"roles"`
}

// No wildcard grants. Reviewed configuration is embedded in the release so a
// new role mapping never requires edits to handlers or database queries.
//
//go:embed grants.json
var grantDocument []byte
var grants = func() map[string][]string {
	var result map[string][]string
	if err := json.Unmarshal(grantDocument, &result); err != nil {
		panic(err)
	}
	return result
}()

func ValidBaseRole(role models.Role) bool {
	return role == models.RoleStudent || role == models.RoleTeacher || role == models.RoleAdmin
}

func FromUser(user models.User) Context {
	ctx := Context{UserID: user.ID, PermVersion: user.PermVersion, Roles: []string{}}
	if user.ID == 0 || user.AccountDeleted || !ValidBaseRole(user.Role) {
		return ctx
	}
	ctx.Roles = append(ctx.Roles, string(user.Role))
	if user.CanAuthor {
		ctx.Roles = append(ctx.Roles, "problem_author")
	}
	return ctx
}

func hasRole(ctx Context, role string) bool {
	for _, value := range ctx.Roles {
		if value == role {
			return true
		}
	}
	return false
}

func knownContext(ctx Context) bool {
	if ctx.UserID == 0 || len(ctx.Roles) == 0 {
		return false
	}
	base := 0
	for _, role := range ctx.Roles {
		switch role {
		case "admin", "teacher", "student":
			base++
		case "problem_author":
		default:
			return false
		}
	}
	return base == 1
}

func (BaselineEngine) Check(ctx Context, code string, resource Resource) Decision {
	deny := Decision{Reason: "POLICY_DENIED"}
	if !knownContext(ctx) {
		return Decision{Reason: "IDENTITY_UNDEFINED"}
	}
	if roles, exists := grants[code]; exists {
		for _, role := range roles {
			if hasRole(ctx, role) {
				return Decision{true, "EXPLICIT_GRANT"}
			}
		}
		return deny
	}
	if resource.ID == 0 {
		return Decision{Reason: "RESOURCE_UNDEFINED"}
	}
	allow := false
	switch code {
	case CourseManage, CourseAdmin:
		allow = HasContext(ctx, CourseAny)
		if !allow && hasRole(ctx, "teacher") {
			allow = resource.OwnerID == ctx.UserID || resource.Member && (resource.Membership == "course_admin" || resource.Membership == "teacher" || resource.Membership == "admin" || code == CourseManage && resource.Membership == "course_assistant")
		}
	case TeamRead, TeamManage, TeamPermissions, TeamContent:
		allow = HasContext(ctx, TeamAny)
		validMember := resource.Member && (resource.Membership == "owner" || resource.Membership == "admin" || resource.Membership == "member")
		if !allow && validMember {
			switch code {
			case TeamRead:
				allow = true
			case TeamManage:
				allow = resource.Membership == "owner" || resource.Membership == "admin"
			case TeamPermissions:
				allow = resource.Membership == "owner"
			case TeamContent:
				allow = resource.Membership == "owner" || resource.ContentPermission == "all" || resource.ContentPermission == "admin" && resource.Membership == "admin"
			}
		}
	default:
		return Decision{Reason: "POLICY_UNREGISTERED"}
	}
	if allow {
		return Decision{true, "RESOURCE_GRANT"}
	}
	return deny
}

func Check(ctx Context, code string, resource Resource) Decision {
	return (BaselineEngine{}).Check(ctx, code, resource)
}
func HasContext(ctx Context, code string) bool { return Check(ctx, code, Resource{}).Allow }
func Has(user models.User, code string) bool   { return HasContext(FromUser(user), code) }

func Capabilities(user models.User) []string {
	result := []string{}
	for code := range grants {
		if Has(user, code) {
			result = append(result, code)
		}
	}
	sort.Strings(result)
	return result
}

func Policies() []Policy {
	result := []Policy{}
	for code, roles := range grants {
		result = append(result, Policy{code, "RBAC", "global", append([]string{}, roles...)})
	}
	for _, code := range []string{CourseManage, CourseAdmin, TeamRead, TeamManage, TeamPermissions, TeamContent} {
		result = append(result, Policy{code, "ABAC", "resource", []string{}})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Code < result[j].Code })
	return result
}
