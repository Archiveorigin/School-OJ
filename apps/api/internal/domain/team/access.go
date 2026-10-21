// Package team defines permissions and transitions independently of HTTP and SQL.
package team

import (
	"context"
	"school-oj/apps/api/internal/authz"
	"school-oj/apps/api/internal/models"
	"strings"
)

type Error struct{ Code, Message string }

func (e *Error) Error() string        { return e.Message }
func Fail(code, message string) error { return &Error{Code: code, Message: message} }

const (
	Invalid           = "team_invalid_request"
	Forbidden         = "team_forbidden"
	Conflict          = "team_state_conflict"
	NotFound          = "team_not_found"
	InvalidInvitation = "team_invitation_invalid"
)

func NormalizeCode(code string) string { return strings.ToUpper(strings.TrimSpace(code)) }
func CanManage(base models.Role, membership models.TeamRole, joined bool) bool {
	return authz.Check(authz.Context{UserID: 1, Roles: []string{string(base)}}, authz.TeamManage, authz.Resource{ID: 1, Member: joined, Membership: string(membership)}).Allow
}
func CanEditPermissions(base models.Role, membership models.TeamRole, joined bool) bool {
	return authz.Check(authz.Context{UserID: 1, Roles: []string{string(base)}}, authz.TeamPermissions, authz.Resource{ID: 1, Member: joined, Membership: string(membership)}).Allow
}
func ValidRole(role models.TeamRole) bool {
	return role == models.TeamRoleOwner || role == models.TeamRoleAdmin || role == models.TeamRoleMember
}

// Replaying approval must never re-create a membership that has since been removed.
func ReviewTransition(current, action string) (next string, replay bool, err error) {
	if action != "approve" && action != "reject" {
		return "", false, Fail(Invalid, "审核操作无效")
	}
	next = "rejected"
	if action == "approve" {
		next = "approved"
	}
	if current == next {
		return next, true, nil
	}
	if current != "pending" {
		return "", false, Fail(Conflict, "申请已经处理，请刷新")
	}
	return next, false, nil
}

type Preview struct {
	ID                uint   `json:"id"`
	Name              string `json:"name"`
	Slug              string `json:"slug"`
	IconURL           string `json:"icon_url"`
	Visibility        string `json:"visibility"`
	JoinMode          string `json:"join_mode"`
	Joined            bool   `json:"joined"`
	ApplicationStatus string `json:"application_status"`
}

// Repository owns persistence and locks. Mutations serialize on one team.
type Repository interface {
	Invitation(context.Context, string) (models.Team, error)
	Preview(context.Context, models.Team, uint) (Preview, error)
	WithTeam(context.Context, uint, func(Transaction) error) error
}
type Transaction interface {
	Team() models.Team
	ActiveUser(uint) (models.User, error)
	Membership(uint) (models.TeamMembership, bool, error)
	PendingApplication(uint) (models.TeamJoinApplication, bool, error)
	Application(uint) (models.TeamJoinApplication, error)
	CreateApplication(uint, string) error
	AddMember(uint) error
	CompletePending(uint) error
	Review(models.TeamJoinApplication, string, uint) error
	SetRole(models.TeamMembership, models.TeamRole) error
	Remove(models.TeamMembership) error
}
