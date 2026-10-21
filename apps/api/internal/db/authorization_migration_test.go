package db

import (
	"school-oj/apps/api/internal/models"
	"testing"
)

func TestFreshInstallationCourseRolesAndPermissionVersionsPostgres(t *testing.T) {
	database := migrationTestDB(t)
	if err := Migrate(database); err != nil {
		t.Fatal(err)
	}
	user := models.User{Email: "course-role@example.test", Name: "Role test", Role: models.RoleTeacher, PasswordHash: "old-hash"}
	if err := database.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	course := models.Course{Code: "ROLE-TEST", Name: "Role test", TeacherID: user.ID}
	if err := database.Create(&course).Error; err != nil {
		t.Fatal(err)
	}
	member := models.CourseMembership{CourseID: course.ID, UserID: user.ID, Role: models.RoleCourseAssistant}
	if err := database.Create(&member).Error; err != nil {
		t.Fatal("fresh SQL installation rejects assistant", err)
	}
	for _, role := range []models.Role{models.RoleCourseAdmin, models.RoleTeacher, models.RoleStudent, models.RoleAdmin} {
		if err := database.Model(&member).Update("role", role).Error; err != nil {
			t.Fatal(role, err)
		}
	}
	if err := database.Model(&member).Update("role", "unknown").Error; err == nil {
		t.Fatal("invalid course role accepted")
	}
	if err := database.Model(&user).Update("can_author", true).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.First(&user, user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if user.PermVersion != 2 || user.TokenVersion != 1 {
		t.Fatal("permission-only update", user.PermVersion, user.TokenVersion)
	}
	if err := database.Model(&user).Update("password_hash", "new-hash").Error; err != nil {
		t.Fatal(err)
	}
	database.First(&user, user.ID)
	if user.PermVersion != 2 || user.TokenVersion != 2 {
		t.Fatal("credential-only update", user.PermVersion, user.TokenVersion)
	}
	if err := Migrate(database); err != nil {
		t.Fatal("repeat migration", err)
	}
}
