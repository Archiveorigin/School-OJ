package team

import (
	"school-oj/apps/api/internal/models"
	"testing"
)

func TestTeamPermissionMatrix(t *testing.T) {
	for _, base := range []models.Role{models.RoleStudent, models.RoleTeacher, models.RoleAdmin} {
		for _, role := range []models.TeamRole{"", "unknown", models.TeamRoleMember, models.TeamRoleAdmin, models.TeamRoleOwner} {
			for _, joined := range []bool{false, true} {
				wantManage := base == models.RoleAdmin || joined && (role == models.TeamRoleOwner || role == models.TeamRoleAdmin)
				wantEdit := base == models.RoleAdmin || joined && role == models.TeamRoleOwner
				if CanManage(base, role, joined) != wantManage || CanEditPermissions(base, role, joined) != wantEdit {
					t.Fatalf("base=%s team=%s joined=%v", base, role, joined)
				}
			}
		}
	}
	for _, role := range []models.TeamRole{"owner", "admin", "member"} {
		if !ValidRole(role) {
			t.Fatal(role)
		}
	}
	for _, role := range []models.TeamRole{"teacher", "student", "creator", ""} {
		if ValidRole(role) {
			t.Fatal(role)
		}
	}
}

func TestApplicationTransitions(t *testing.T) {
	for _, action := range []string{"approve", "reject"} {
		next, replay, err := ReviewTransition("pending", action)
		if err != nil || replay {
			t.Fatalf("pending %s: %v", action, err)
		}
		if _, replay, err = ReviewTransition(next, action); err != nil || !replay {
			t.Fatal("same decision is not idempotent")
		}
		other := "approve"
		if action == other {
			other = "reject"
		}
		if _, _, err = ReviewTransition(next, other); err == nil {
			t.Fatal("conflicting decision accepted")
		}
	}
	for _, action := range []string{"", "cancel", "APPROVE"} {
		if _, _, err := ReviewTransition("pending", action); err == nil {
			t.Fatal(action)
		}
	}
	if NormalizeCode("  abCd01234567  ") != "ABCD01234567" {
		t.Fatal("normalization")
	}
}
