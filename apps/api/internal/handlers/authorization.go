package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"school-oj/apps/api/internal/authz"
	"school-oj/apps/api/internal/middleware"
	"school-oj/apps/api/internal/models"
)

// authorizeRoute is the coarse operation gate. Resource PEPs below it use
// trusted SQL attributes; route parameters never become grants.
func (s Server) authorizeRoute(c *gin.Context) {
	route, exists := authz.Route(c.Request.Method, c.FullPath())
	user, _ := middleware.CurrentUser(c)
	decision := authz.CheckRoute(authz.FromUser(user), route)
	if !exists || !decision.Allow {
		c.Set("authorization.reason", decision.Reason)
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "没有执行此操作的权限", "code": decision.Reason})
		return
	}
	c.Next()
}

// L4 records every denied request and all privileged/sensitive successes. It
// never records credentials, raw URLs, query strings, request bodies or code.
func (s Server) authorizationAudit() gin.HandlerFunc {
	return func(c *gin.Context) {
		started := time.Now()
		if strings.HasPrefix(c.FullPath(), "/api/") {
			c.Header("Cache-Control", "private, no-store")
		}
		c.Next()
		status := c.Writer.Status()
		route, registered := authz.Route(c.Request.Method, c.FullPath())
		user, authenticated := middleware.CurrentUser(c)
		denied := status == 401 || status == 403 || status == 404 && registered && authenticated
		sensitive := authenticated && (authz.Has(user, authz.AuditView) || c.Request.Method != "GET" || strings.Contains(route.Path, "export") || strings.Contains(route.Path, "/tests") || strings.Contains(route.Path, "/package") || strings.Contains(route.Path, "/attachment") || strings.HasPrefix(route.Path, "/submissions/"))
		if !denied && !(registered && sensitive && status < 400) {
			return
		}
		middleware.RecordAuthorization(denied)
		action, reason := "authorization.allow", "OPERATION_ALLOWED"
		if denied {
			action, reason = "authorization.deny", "RESOURCE_DENIED"
			if status == 401 {
				reason = "AUTHENTICATION_REQUIRED"
			}
			if status == 404 {
				reason = "RESOURCE_UNAVAILABLE"
			}
		}
		if value, ok := c.Get("authorization.reason"); ok {
			if code, ok := value.(string); ok {
				reason = code
			}
		}
		if !registered {
			route.Code = "route:unregistered"
		}
		meta := datatypes.JSONMap{"policy": route.Code, "access": route.Access, "reason": reason, "method": c.Request.Method, "route": c.FullPath(), "status": status, "perm_version": user.PermVersion, "duration_ms": time.Since(started).Milliseconds()}
		resourceID := ""
		for _, name := range []string{"event_id", "contest_id", "set_id", "id"} {
			if value, err := strconv.ParseUint(c.Param(name), 10, 64); err == nil {
				resourceID = strconv.FormatUint(value, 10)
				break
			}
		}
		var actor *uint
		if authenticated {
			actor = &user.ID
		}
		if denied {
			slog.Warn("authorization denied", "policy", route.Code, "user_id", user.ID, "reason", reason, "status", status)
		}
		if s.DB == nil {
			return
		}
		auditContext, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := s.DB.WithContext(auditContext).Create(&models.AuditLog{ActorUserID: actor, Action: action, ResourceType: "authorization", ResourceID: resourceID, Meta: meta}).Error; err != nil {
			middleware.RecordAuthorizationAuditFailure()
			slog.Error("authorization audit persistence failed", "policy", route.Code, "status", status)
		}
	}
}

type sessionUser struct {
	models.User
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
}

func authenticatedUser(user models.User) sessionUser {
	return sessionUser{user, authz.FromUser(user).Roles, authz.Capabilities(user)}
}

func (s Server) authorizationPolicies(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"policies": authz.Policies(), "routes": authz.Routes, "default_deny": true})
}

func (s Server) courseDecision(user models.User, courseID uint, code string) bool {
	if !authz.ValidBaseRole(user.Role) || user.ID == 0 || courseID == 0 {
		return false
	}
	var course models.Course
	if err := s.DB.Select("id", "teacher_id").First(&course, courseID).Error; err != nil {
		return false
	}
	resource := authz.Resource{ID: course.ID, OwnerID: course.TeacherID}
	if course.TeacherID != user.ID && !authz.Has(user, authz.CourseAny) {
		var member models.CourseMembership
		if err := s.DB.Where("course_id = ? AND user_id = ?", courseID, user.ID).First(&member).Error; err == nil {
			resource.Member, resource.Membership = true, string(member.Role)
		}
	}
	return authz.Check(authz.FromUser(user), code, resource).Allow
}

func (s Server) teamDecision(user models.User, team models.Team, code string) bool {
	member, joined := s.teamMembership(team.ID, user.ID)
	return authz.Check(authz.FromUser(user), code, authz.Resource{ID: team.ID, OwnerID: team.OwnerID, Member: joined, Membership: string(member.Role), ContentPermission: team.ContestPermission}).Allow
}

// Batch PIP reads keep page controls aligned with resource decisions without
// turning every row into two additional database queries.
func (s Server) courseCapabilities(user models.User, courses []models.Course) []models.Course {
	ids := make([]uint, 0, len(courses))
	for _, course := range courses {
		ids = append(ids, course.ID)
	}
	var members []models.CourseMembership
	if len(ids) > 0 && s.DB.Where("course_id IN ? AND user_id = ?", ids, user.ID).Find(&members).Error != nil {
		return courses
	}
	byCourse := map[uint]models.CourseMembership{}
	for _, member := range members {
		byCourse[member.CourseID] = member
	}
	for index, course := range courses {
		member, joined := byCourse[course.ID]
		resource := authz.Resource{ID: course.ID, OwnerID: course.TeacherID, Member: joined, Membership: string(member.Role)}
		courses[index].CanManage = authz.Check(authz.FromUser(user), authz.CourseManage, resource).Allow
		courses[index].CanAdmin = authz.Check(authz.FromUser(user), authz.CourseAdmin, resource).Allow
	}
	return courses
}

func (s Server) classCapabilities(user models.User, classes []models.Class) []models.Class {
	ids := make([]uint, 0, len(classes))
	for _, class := range classes {
		ids = append(ids, class.CourseID)
	}
	var courses []models.Course
	if len(ids) > 0 && s.DB.Select("id", "teacher_id").Where("id IN ?", ids).Find(&courses).Error != nil {
		return classes
	}
	byCourse := map[uint]models.Course{}
	for _, course := range s.courseCapabilities(user, courses) {
		byCourse[course.ID] = course
	}
	for index, class := range classes {
		classes[index].CanManage = byCourse[class.CourseID].CanManage
		classes[index].CanAdmin = byCourse[class.CourseID].CanAdmin
	}
	return classes
}
