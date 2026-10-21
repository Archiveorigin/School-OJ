package handlers

import (
	"context"
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
	"school-oj/apps/api/internal/models"
	"testing"
	"time"
)

func TestAuthorizationBrowserFixture(t *testing.T) {
	addr := os.Getenv("OJ_AUTHORIZATION_BROWSER_FIXTURE_ADDR")
	if addr == "" {
		t.Skip("opt-in authorization browser fixture")
	}
	if addr != "127.0.0.1:18090" {
		t.Fatal("fixture must bind isolated loopback")
	}
	f := newExamFixture(t)
	if err := f.s.DB.Model(&f.student).Update("can_author", true).Error; err != nil {
		t.Fatal(err)
	}
	if err := f.s.DB.Model(&f.peer).Update("role", models.RoleTeacher).Error; err != nil {
		t.Fatal(err)
	}
	f.s.DB.Where("user_id = ?", f.peer.ID).Delete(&models.CourseMembership{})
	f.s.DB.Create(&models.CourseMembership{CourseID: f.exam.CourseID, UserID: f.peer.ID, Role: models.RoleCourseAssistant})
	class := models.Class{CourseID: f.exam.CourseID, Name: "权限验收班", JoinCode: "AUTHZQA"}
	f.s.DB.Create(&class)
	team := models.Team{Name: "权限验收团队", Slug: "authorization-qa", OwnerID: f.student.ID, Visibility: "public", JoinMode: "open", ContestPermission: "owner"}
	f.s.DB.Create(&team)
	f.s.DB.Create(&models.TeamMembership{TeamID: team.ID, UserID: f.student.ID, Role: models.TeamRoleOwner})
	f.s.DB.Create(&models.TeamMembership{TeamID: team.ID, UserID: f.peer.ID, Role: models.TeamRoleMember})
	f.s.Cfg.CORSOrigins = []string{"http://127.0.0.1:4180"}
	f.router = f.s.Router()
	server := &http.Server{Addr: addr, Handler: f.router, ReadHeaderTimeout: 10 * time.Second}
	f.router.GET("/__qa/info", func(c *gin.Context) {
		c.JSON(200, gin.H{"student_id": f.student.ID, "assistant_id": f.peer.ID, "teacher_id": f.teacher.ID, "admin_id": f.admin.ID, "team_id": team.ID})
	})
	f.router.POST("/__qa/stop", func(c *gin.Context) {
		c.Status(200)
		go func() { time.Sleep(50 * time.Millisecond); _ = server.Shutdown(context.Background()) }()
	})
	t.Log("AUTHORIZATION_FIXTURE_READY", addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		t.Fatal(err)
	}
}
