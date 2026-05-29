package rbac

import (
	"testing"

	"school-oj/apps/api/internal/models"
)

func TestRoleChecks(t *testing.T) {
	if !CanManageUsers(models.RoleAdmin) {
		t.Fatal("admin should manage users")
	}
	if CanManageUsers(models.RoleTeacher) {
		t.Fatal("teacher should not manage users")
	}
	if !CanManageCourse(models.RoleTeacher) {
		t.Fatal("teacher should manage courses")
	}
	if CanViewAudit(models.RoleStudent) {
		t.Fatal("student should not view audit logs")
	}
}
