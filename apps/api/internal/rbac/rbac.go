package rbac

import "school-oj/apps/api/internal/models"

func CanManageCourse(role models.Role) bool {
	return role == models.RoleAdmin || role == models.RoleTeacher
}

func CanManageUsers(role models.Role) bool {
	return role == models.RoleAdmin
}

func CanViewAudit(role models.Role) bool {
	return role == models.RoleAdmin
}

func IsAllowed(role models.Role, allowed ...models.Role) bool {
	for _, item := range allowed {
		if role == item {
			return true
		}
	}
	return false
}
