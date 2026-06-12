package handlers

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/mail"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"school-oj/apps/api/internal/config"
	"school-oj/apps/api/internal/jplag"
	"school-oj/apps/api/internal/middleware"
	"school-oj/apps/api/internal/models"
	"school-oj/apps/api/internal/services"
	"school-oj/apps/api/internal/streams"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Server struct {
	DB    *gorm.DB
	Redis *redis.Client
	MinIO *minio.Client
	Cfg   config.Config
}

type leaderboardRow struct {
	UserID         uint       `json:"user_id"`
	Name           string     `json:"name"`
	Solved         int        `json:"solved"`
	Score          int        `json:"score"`
	LastSubmission *time.Time `json:"last_submission"`
	Rank           int        `json:"rank"`
}

type courseClassFields struct {
	CourseCode string `json:"course_code"`
	CourseName string `json:"course_name"`
	ClassName  string `json:"class_name"`
	JoinCode   string `json:"join_code,omitempty"`
}

type classListView struct {
	models.Class
	CourseCode string `json:"course_code"`
	CourseName string `json:"course_name"`
	Term       string `json:"term"`
}

type courseMemberView struct {
	ID         uint        `json:"id"`
	CourseID   uint        `json:"course_id"`
	UserID     uint        `json:"user_id"`
	Role       models.Role `json:"role"`
	UserRole   models.Role `json:"user_role"`
	Email      string      `json:"email"`
	Name       string      `json:"name"`
	StudentNo  string      `json:"student_no"`
	ClassCount int64       `json:"class_count"`
	CreatedAt  time.Time   `json:"created_at"`
}

type classStudentView struct {
	MembershipID uint      `json:"membership_id"`
	UserID       uint      `json:"user_id"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	StudentNo    string    `json:"student_no"`
	JoinedAt     time.Time `json:"joined_at"`
}

type studentImportInput struct {
	StudentNo string `json:"student_no"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Password  string `json:"password"`
}

type studentImportResult struct {
	Email     string `json:"email"`
	StudentNo string `json:"student_no"`
	Name      string `json:"name"`
	UserID    uint   `json:"user_id,omitempty"`
	Created   bool   `json:"created"`
	Joined    bool   `json:"joined"`
	Error     string `json:"error,omitempty"`
}

type assignmentListView struct {
	models.Assignment
	CourseCode string `json:"course_code"`
	CourseName string `json:"course_name"`
	ClassName  string `json:"class_name"`
	WorkStatus string `json:"work_status,omitempty"`
	TotalScore int    `json:"total_score,omitempty"`
	MaxScore   int    `json:"max_score,omitempty"`
	ScoreReady bool   `json:"score_ready,omitempty"`
}

type examListView struct {
	models.Exam
	CourseCode string     `json:"course_code"`
	CourseName string     `json:"course_name"`
	ClassName  string     `json:"class_name"`
	WorkStatus string     `json:"work_status,omitempty"`
	TotalScore int        `json:"total_score,omitempty"`
	MaxScore   int        `json:"max_score,omitempty"`
	ScoreReady bool       `json:"score_ready,omitempty"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
}

type preparedProblemInput struct {
	Slug          string                       `json:"slug"`
	Title         string                       `json:"title"`
	Statement     string                       `json:"statement"`
	Tags          []string                     `json:"tags"`
	TimeLimitMS   int                          `json:"time_limit_ms"`
	MemoryLimitMB int                          `json:"memory_limit_mb"`
	OutputLimitKB int                          `json:"output_limit_kb"`
	Assets        []services.ProblemAssetDraft `json:"assets"`
	Cases         []services.ProblemCaseDraft  `json:"cases"`
	Folder        string                       `json:"folder"`
	Difficulty    string                       `json:"difficulty"`
	Source        string                       `json:"source"`
	Notes         string                       `json:"notes"`
}

type problemUpdateInput struct {
	Title         string   `json:"title"`
	Statement     string   `json:"statement"`
	Tags          []string `json:"tags"`
	TimeLimitMS   int      `json:"time_limit_ms"`
	MemoryLimitMB int      `json:"memory_limit_mb"`
	OutputLimitKB int      `json:"output_limit_kb"`
}

type workProblemInput struct {
	ProblemID        uint   `json:"problem_id"`
	Score            int    `json:"score"`
	Label            string `json:"label"`
	ReleaseAfterExam bool   `json:"release_after_exam"`
}

type problemScoreView struct {
	Problem          models.Problem          `json:"problem"`
	Label            string                  `json:"label,omitempty"`
	Score            int                     `json:"score"`
	BestScore        int                     `json:"best_score"`
	RawScore         int                     `json:"raw_score"`
	SubmissionID     *uint                   `json:"submission_id"`
	SubmissionStatus models.SubmissionStatus `json:"submission_status"`
	ScoreReady       bool                    `json:"score_ready"`
	PendingReview    bool                    `json:"pending_review"`
	SubmittedAt      *time.Time              `json:"submitted_at"`
}

type workSummary struct {
	WorkStatus string             `json:"work_status"`
	TotalScore int                `json:"total_score"`
	MaxScore   int                `json:"max_score"`
	ScoreReady bool               `json:"score_ready"`
	Problems   []problemScoreView `json:"problem_scores,omitempty"`
}

type submissionListView struct {
	models.Submission
	UserName        string `json:"user_name,omitempty"`
	StudentNo       string `json:"student_no,omitempty"`
	ProblemCode     string `json:"problem_code,omitempty"`
	ProblemTitle    string `json:"problem_title,omitempty"`
	AssignmentTitle string `json:"assignment_title,omitempty"`
	ExamTitle       string `json:"exam_title,omitempty"`
	ErrorPoint      string `json:"error_point,omitempty"`
}

type plagiarismJobView struct {
	models.PlagiarismJob
	CourseCode      string `json:"course_code,omitempty"`
	CourseName      string `json:"course_name,omitempty"`
	AssignmentTitle string `json:"assignment_title,omitempty"`
	ExamTitle       string `json:"exam_title,omitempty"`
	CreatedByName   string `json:"created_by_name,omitempty"`
}

type auditLogView struct {
	models.AuditLog
	ActorName     string `json:"actor_name,omitempty"`
	ResourceLabel string `json:"resource_label,omitempty"`
}

type examRankingProblem struct {
	ProblemID   uint   `json:"problem_id"`
	Label       string `json:"label"`
	DisplayCode string `json:"display_code"`
	Title       string `json:"title"`
	Score       int    `json:"score"`
}

type examRankingCell struct {
	ProblemID   uint                    `json:"problem_id"`
	Label       string                  `json:"label"`
	BestScore   int                     `json:"best_score"`
	MaxScore    int                     `json:"max_score"`
	Status      models.SubmissionStatus `json:"status,omitempty"`
	ScoreReady  bool                    `json:"score_ready"`
	Pending     bool                    `json:"pending"`
	SubmittedAt *time.Time              `json:"submitted_at,omitempty"`
}

type examRankingRow struct {
	Rank            int               `json:"rank"`
	UserID          uint              `json:"user_id"`
	Name            string            `json:"name"`
	StudentNo       string            `json:"student_no"`
	TotalScore      int               `json:"total_score"`
	MaxScore        int               `json:"max_score"`
	Solved          int               `json:"solved"`
	Attempted       int               `json:"attempted"`
	SubmissionCount int               `json:"submission_count"`
	PendingCount    int               `json:"pending_count"`
	ScoreReady      bool              `json:"score_ready"`
	WorkStatus      string            `json:"work_status"`
	LastSubmission  *time.Time        `json:"last_submission"`
	FinishedAt      *time.Time        `json:"finished_at"`
	Problems        []examRankingCell `json:"problems"`
}

func (req preparedProblemInput) draft() services.ProblemPackageDraft {
	return services.ProblemPackageDraft{
		Slug:          req.Slug,
		Title:         req.Title,
		Statement:     req.Statement,
		Tags:          req.Tags,
		TimeLimitMS:   req.TimeLimitMS,
		MemoryLimitMB: req.MemoryLimitMB,
		OutputLimitKB: req.OutputLimitKB,
		Assets:        req.Assets,
		Cases:         req.Cases,
	}
}

func (s Server) Router() *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	r.GET("/healthz", s.health)
	api := r.Group("/api")
	api.POST("/auth/login", s.login)
	api.POST("/auth/send-code", s.sendEmailCode)
	api.POST("/auth/register", s.register)
	api.POST("/auth/password-reset", s.resetPassword)
	auth := api.Group("")
	auth.Use(middleware.Auth(s.DB, s.Cfg.JWTSecret))
	auth.GET("/me", s.me)
	auth.GET("/me/active-exam", s.activeExam)
	auth.GET("/profile", s.getProfile)
	auth.PUT("/profile", s.updateProfile)
	auth.POST("/profile/email-code", s.sendProfileEmailCode)
	auth.POST("/profile/email", s.rebindEmail)
	auth.DELETE("/profile", s.deleteProfile)
	auth.POST("/feedback", s.createFeedback)
	auth.GET("/courses", s.listCourses)
	auth.POST("/courses", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.createCourse)
	auth.PUT("/courses/:id", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.updateCourse)
	auth.DELETE("/courses/:id", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.archiveCourse)
	auth.POST("/courses/:id/archive", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.archiveCourse)
	auth.POST("/courses/:id/classes", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.createClass)
	auth.GET("/courses/:id/members", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.listCourseMembers)
	auth.POST("/courses/:id/members", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.addCourseMember)
	auth.DELETE("/courses/:id/members/:user_id", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.removeCourseMember)
	auth.GET("/courses/preview", s.previewCourseByCode)
	auth.POST("/courses/join", s.joinCourseByCode)
	auth.POST("/courses/:id/leave", s.leaveCourse)
	auth.GET("/courses/:id/students", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.listCourseStudents)
	auth.GET("/classes", s.listClasses)
	auth.GET("/classes/join-preview", middleware.RequireRoles(models.RoleStudent), s.previewClassJoin)
	auth.GET("/me/classes", s.myClasses)
	auth.POST("/classes/join", middleware.RequireRoles(models.RoleStudent), s.joinClassByCode)
	auth.POST("/classes/:id/join", middleware.RequireRoles(models.RoleStudent), s.joinClass)
	auth.POST("/classes/:id/leave", middleware.RequireRoles(models.RoleStudent), s.leaveClass)
	auth.PUT("/classes/:id", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.updateClass)
	auth.DELETE("/classes/:id", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.archiveClass)
	auth.POST("/classes/:id/archive", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.archiveClass)
	auth.GET("/classes/:id/students", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.listClassStudents)
	auth.DELETE("/classes/:id/students/:user_id", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.removeClassStudent)
	auth.POST("/classes/:id/students/import", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.importClassStudents)
	auth.GET("/problems", s.listProblems)
	auth.POST("/problems", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.createProblem)
	auth.POST("/problems/parse-markdown", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.parseMarkdownBatch)
	auth.POST("/problems/upload", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.uploadProblem)
	auth.PUT("/problems/:id", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.updateProblem)
	auth.GET("/problems/:id/assets/*asset_path", s.getProblemAsset)
	auth.GET("/problems/:id/tests", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.listProblemTests)
	auth.GET("/problems/:id/tests/download", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.downloadProblemTests)
	auth.GET("/problems/:id/tests/file/*file_path", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.downloadProblemTestFile)
	auth.GET("/problems/:id", s.getProblem)
	auth.DELETE("/problems/:id", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.deleteProblem)
	auth.GET("/prepared-problems", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.listPreparedProblems)
	auth.POST("/prepared-problems", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.createPreparedProblem)
	auth.POST("/prepared-problems/upload", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.uploadPreparedProblem)
	auth.GET("/prepared-problems/:id", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.getPreparedProblem)
	auth.PUT("/prepared-problems/:id", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.updatePreparedProblem)
	auth.POST("/prepared-problems/:id/publish", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.publishPreparedProblem)
	auth.GET("/assignments", s.listAssignments)
	auth.GET("/assignments/:id", s.getAssignment)
	auth.POST("/assignments", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.createAssignment)
	auth.GET("/assignments/:id/report", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.assignmentReport)
	auth.DELETE("/assignments/:id", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.deleteAssignment)
	auth.GET("/exams", s.listExams)
	auth.GET("/exams/:id", s.getExam)
	auth.POST("/exams/:id/finish", middleware.RequireRoles(models.RoleStudent), s.finishExam)
	auth.POST("/exams", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.createExam)
	auth.GET("/exams/:id/report/export", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.exportExamReport)
	auth.GET("/exams/:id/report", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.examReport)
	auth.GET("/exams/:id/ranking", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.examRanking)
	auth.DELETE("/exams/:id", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.deleteExam)
	auth.POST("/exams/:id/submissions/:submission_id/judge", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.judgeManualExamSubmission)
	auth.PUT("/exams/:id/submissions/:submission_id/grade", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.gradeManualExamSubmission)
	auth.POST("/submissions", s.createSubmission)
	auth.GET("/submissions", s.listSubmissions)
	auth.GET("/submissions/:id", s.getSubmission)
	auth.GET("/submissions/:id/events", s.submissionEvents)
	// leaderboard disabled: auth.GET("/leaderboard", s.leaderboard)
	auth.GET("/plagiarism/jobs", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.listPlagiarismJobs)
	auth.POST("/plagiarism/jobs", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.createPlagiarismJob)
	auth.GET("/audit-logs", middleware.RequireRoles(models.RoleAdmin), s.listAuditLogs)
	auth.GET("/users", middleware.RequireRoles(models.RoleAdmin), s.listUsers)
	auth.POST("/users", middleware.RequireRoles(models.RoleAdmin), s.createUser)
	auth.GET("/users/:id", middleware.RequireRoles(models.RoleAdmin), s.getUser)
	auth.PUT("/users/:id", middleware.RequireRoles(models.RoleAdmin), s.updateUser)
	auth.DELETE("/users/:id", middleware.RequireRoles(models.RoleAdmin), s.deleteUser)
	auth.POST("/users/:id/reset-password", middleware.RequireRoles(models.RoleAdmin), s.resetUserPassword)
	return r
}

func (s Server) health(c *gin.Context) {
	sqlDB, err := s.DB.DB()
	if err != nil || sqlDB.PingContext(c.Request.Context()) != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "database_unhealthy"})
		return
	}
	if err := s.Redis.Ping(c.Request.Context()).Err(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "redis_unhealthy"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s Server) login(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if !bind(c, &req) {
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	var user models.User
	if err := s.DB.Where("email = ? AND account_deleted = false", req.Email).First(&user).Error; err != nil {
		failed := s.recordFailedLogin(req.Email)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials", "failed_attempts": failed, "password_reset_available": failed >= 3})
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		failed := s.recordFailedLogin(req.Email)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials", "failed_attempts": failed, "password_reset_available": failed >= 3})
		return
	}
	s.clearFailedLogin(req.Email)
	token, err := middleware.SignToken(s.Cfg.JWTSecret, user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	services.Audit(c, s.DB, "auth.login", "user", user.ID, nil)
	c.JSON(http.StatusOK, gin.H{"token": token, "user": user})
}

func (s Server) recordFailedLogin(email string) int {
	email = strings.ToLower(strings.TrimSpace(email))
	now := time.Now()
	var item models.LoginAttempt
	if err := s.DB.Where("email = ?", email).First(&item).Error; err != nil {
		item = models.LoginAttempt{Email: email, FailedCount: 1, LastFailedAt: &now}
		_ = s.DB.Create(&item).Error
		return 1
	}
	item.FailedCount++
	item.LastFailedAt = &now
	_ = s.DB.Save(&item).Error
	return item.FailedCount
}

func (s Server) clearFailedLogin(email string) {
	_ = s.DB.Where("email = ?", strings.ToLower(strings.TrimSpace(email))).Delete(&models.LoginAttempt{}).Error
}

func (s Server) me(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	c.JSON(http.StatusOK, user)
}

func (s Server) activeExam(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	if user.Role != models.RoleStudent {
		c.JSON(http.StatusOK, gin.H{"active": false})
		return
	}
	exam, ok := s.activeStartedExamAttemptForStudent(user.ID)
	if !ok {
		c.JSON(http.StatusOK, gin.H{"active": false})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"active": true,
		"exam": gin.H{
			"id":          exam.ID,
			"course_id":   exam.CourseID,
			"class_id":    exam.ClassID,
			"title":       exam.Title,
			"description": exam.Description,
			"starts_at":   exam.StartsAt,
			"ends_at":     exam.EndsAt,
		},
	})
}

func (s Server) listUsers(c *gin.Context) {
	var users []models.User
	s.DB.Select(userPublicColumns()).Where("account_deleted = false").Order("id asc").Find(&users)
	c.JSON(http.StatusOK, users)
}

func (s Server) createUser(c *gin.Context) {
	var req struct {
		Email     string      `json:"email" binding:"required"`
		Name      string      `json:"name" binding:"required"`
		Role      models.Role `json:"role" binding:"required"`
		Password  string      `json:"password" binding:"required"`
		StudentNo string      `json:"student_no"`
	}
	if !bind(c, &req) {
		return
	}
	if !validRole(req.Role) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "role must be one of student, teacher, admin"})
		return
	}
	req.Email = normalizeEmail(req.Email)
	req.Name = strings.TrimSpace(req.Name)
	req.StudentNo = strings.TrimSpace(req.StudentNo)
	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}
	if !validEmail(req.Email) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid email"})
		return
	}
	if len(req.Password) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "password must be at least 6 characters"})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	user := models.User{Email: req.Email, Name: req.Name, Role: req.Role, StudentNo: req.StudentNo, PasswordHash: string(hash), EmailVerified: true}
	if err := s.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	services.Audit(c, s.DB, "user.create", "user", user.ID, nil)
	c.JSON(http.StatusCreated, user)
}

func (s Server) getUser(c *gin.Context) {
	userID, ok := idParam(c, "id")
	if !ok {
		return
	}
	var user models.User
	if err := s.DB.Select(userPublicColumns()).Where("id = ? AND account_deleted = false", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, user)
}

func (s Server) updateUser(c *gin.Context) {
	userID, ok := idParam(c, "id")
	if !ok {
		return
	}
	var user models.User
	if err := s.DB.Where("id = ? AND account_deleted = false", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	var req struct {
		Email     string      `json:"email" binding:"required"`
		Name      string      `json:"name" binding:"required"`
		Role      models.Role `json:"role" binding:"required"`
		StudentNo string      `json:"student_no"`
	}
	if !bind(c, &req) {
		return
	}
	email := normalizeEmail(req.Email)
	name := strings.TrimSpace(req.Name)
	studentNo := strings.TrimSpace(req.StudentNo)
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}
	if !validEmail(email) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid email"})
		return
	}
	if !validRole(req.Role) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "role must be one of student, teacher, admin"})
		return
	}
	if email != user.Email {
		var count int64
		if err := s.DB.Model(&models.User{}).Where("email = ? AND id <> ? AND account_deleted = false", email, userID).Count(&count).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if count > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "email already registered"})
			return
		}
	}
	updates := map[string]any{
		"email":      email,
		"name":       name,
		"role":       req.Role,
		"student_no": studentNo,
	}
	if err := s.DB.Model(&user).Updates(updates).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	services.Audit(c, s.DB, "user.update", "user", userID, datatypes.JSONMap{"role": req.Role})
	_ = s.DB.Select(userPublicColumns()).First(&user, userID).Error
	c.JSON(http.StatusOK, user)
}

func (s Server) deleteUser(c *gin.Context) {
	current, _ := middleware.CurrentUser(c)
	userID, ok := idParam(c, "id")
	if !ok {
		return
	}
	if current.ID == userID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot delete yourself"})
		return
	}
	var user models.User
	if err := s.DB.Where("id = ? AND account_deleted = false", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	deletedEmail := fmt.Sprintf("deleted-%d@local.invalid", user.ID)
	if err := s.DB.Model(&user).Updates(map[string]any{
		"email":           deletedEmail,
		"name":            "deleted user",
		"password_hash":   "deleted",
		"student_no":      "",
		"avatar_url":      "",
		"email_verified":  false,
		"account_deleted": true,
	}).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	services.Audit(c, s.DB, "user.delete", "user", user.ID, datatypes.JSONMap{"email": user.Email})
	c.JSON(http.StatusOK, gin.H{"deleted": true, "user_id": user.ID})
}

func (s Server) resetUserPassword(c *gin.Context) {
	userID, ok := idParam(c, "id")
	if !ok {
		return
	}
	var user models.User
	if err := s.DB.Where("id = ? AND account_deleted = false", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	var req struct {
		Password string `json:"password"`
	}
	if !bind(c, &req) {
		return
	}
	password := req.Password
	generated := false
	if password == "" {
		var err error
		password, err = randomPassword()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		generated = true
	}
	if len(password) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "password must be at least 6 characters"})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := s.DB.Model(&user).Update("password_hash", string(hash)).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	services.Audit(c, s.DB, "user.password_reset", "user", user.ID, datatypes.JSONMap{"generated": generated})
	resp := gin.H{"reset": true, "user_id": user.ID}
	if generated {
		resp["password"] = password
	}
	c.JSON(http.StatusOK, resp)
}

func (s Server) listCourses(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	var courses []models.Course
	q := s.DB.Order("id desc")
	if !includeArchived(c) {
		q = q.Where("archived = false")
	}
	if term := strings.TrimSpace(c.Query("term")); term != "" {
		q = q.Where("term = ?", term)
	}
	switch user.Role {
	case models.RoleAdmin:
	case models.RoleTeacher:
		q = q.Where("teacher_id = ? OR id IN (?)", user.ID, s.DB.Model(&models.CourseMembership{}).Select("course_id").Where("user_id = ? AND role IN ?", user.ID, courseTeachingRoles()))
	default:
		q = q.Where("id IN (?)", s.DB.Model(&models.CourseMembership{}).Select("course_id").Where("user_id = ?", user.ID))
	}
	q.Find(&courses)
	c.JSON(http.StatusOK, courses)
}

func (s Server) createCourse(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	var req models.Course
	if !bind(c, &req) {
		return
	}
	req.ID = 0
	req.Code = strings.TrimSpace(req.Code)
	req.Name = strings.TrimSpace(req.Name)
	req.Term = strings.TrimSpace(req.Term)
	req.Description = strings.TrimSpace(req.Description)
	if req.TeacherID == 0 {
		req.TeacherID = user.ID
	}
	if user.Role == models.RoleTeacher {
		req.TeacherID = user.ID
	}
	if req.Code == "" || req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "course code and name are required"})
		return
	}
	if err := s.DB.Create(&req).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.JoinCode = models.FormatCourseJoinCode(req.ID)
	_ = s.DB.Model(&models.Course{}).Where("id = ?", req.ID).Update("join_code", req.JoinCode).Error
	_ = s.DB.Create(&models.CourseMembership{CourseID: req.ID, UserID: req.TeacherID, Role: models.RoleCourseAdmin}).Error
	services.Audit(c, s.DB, "course.create", "course", req.ID, nil)
	c.JSON(http.StatusCreated, req)
}

func (s Server) updateCourse(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	courseID, ok := idParam(c, "id")
	if !ok {
		return
	}
	if !s.canAdminCourse(user, courseID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	var course models.Course
	if err := s.DB.First(&course, courseID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "course not found"})
		return
	}
	var req struct {
		Code        string `json:"code"`
		Name        string `json:"name"`
		Term        string `json:"term"`
		Description string `json:"description"`
		Archived    *bool  `json:"archived"`
	}
	if !bind(c, &req) {
		return
	}
	code := strings.TrimSpace(req.Code)
	name := strings.TrimSpace(req.Name)
	if code == "" || name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "course code and name are required"})
		return
	}
	updates := map[string]any{
		"code":        code,
		"name":        name,
		"term":        strings.TrimSpace(req.Term),
		"description": strings.TrimSpace(req.Description),
	}
	if req.Archived != nil {
		updates["archived"] = *req.Archived
	}
	if err := s.DB.Model(&course).Updates(updates).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	services.Audit(c, s.DB, "course.update", "course", courseID, nil)
	_ = s.DB.First(&course, courseID).Error
	c.JSON(http.StatusOK, course)
}

func (s Server) archiveCourse(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	courseID, ok := idParam(c, "id")
	if !ok {
		return
	}
	if !s.canAdminCourse(user, courseID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	archived := true
	if c.Request.Method == http.MethodPost {
		var req struct {
			Archived *bool `json:"archived"`
		}
		if !bind(c, &req) {
			return
		}
		if req.Archived != nil {
			archived = *req.Archived
		}
	}
	if err := s.DB.Model(&models.Course{}).Where("id = ?", courseID).Update("archived", archived).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	services.Audit(c, s.DB, "course.archive", "course", courseID, datatypes.JSONMap{"archived": archived})
	c.JSON(http.StatusOK, gin.H{"id": courseID, "archived": archived})
}

func (s Server) createClass(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	courseID, ok := idParam(c, "id")
	if !ok {
		return
	}
	if !s.canAdminCourse(user, courseID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	var course models.Course
	if err := s.DB.First(&course, courseID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "course not found"})
		return
	}
	if course.Archived {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot add members to archived course"})
		return
	}
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if !bind(c, &req) {
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "class name is required"})
		return
	}
	class := models.Class{CourseID: courseID, Name: req.Name}
	if err := s.DB.Create(&class).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	class.JoinCode = models.FormatClassJoinCode(class.ID)
	if err := s.DB.Model(&models.Class{}).Where("id = ?", class.ID).Update("join_code", class.JoinCode).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_ = s.DB.Where("class_id = ? AND user_id = ?", class.ID, user.ID).FirstOrCreate(&models.ClassMembership{ClassID: class.ID, UserID: user.ID}).Error
	services.Audit(c, s.DB, "class.create", "class", class.ID, datatypes.JSONMap{"course_id": courseID})
	c.JSON(http.StatusCreated, class)
}

func (s Server) listCourseMembers(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	courseID, ok := idParam(c, "id")
	if !ok {
		return
	}
	if !s.canManageCourse(user, courseID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	var memberships []models.CourseMembership
	if err := s.DB.Where("course_id = ?", courseID).Order("id asc").Find(&memberships).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userIDs := make([]uint, 0, len(memberships))
	for _, membership := range memberships {
		userIDs = append(userIDs, membership.UserID)
	}
	users := map[uint]models.User{}
	if ids := dedupeUint(userIDs); len(ids) > 0 {
		var rows []models.User
		s.DB.Where("id IN ?", ids).Find(&rows)
		for _, row := range rows {
			users[row.ID] = row
		}
	}
	classCounts := map[uint]int64{}
	var countRows []struct {
		UserID uint
		Count  int64
	}
	s.DB.Table("class_memberships").
		Select("class_memberships.user_id as user_id, count(*) as count").
		Joins("join classes on classes.id = class_memberships.class_id").
		Where("classes.course_id = ? AND classes.archived = false", courseID).
		Group("class_memberships.user_id").
		Scan(&countRows)
	for _, row := range countRows {
		classCounts[row.UserID] = row.Count
	}
	out := make([]courseMemberView, 0, len(memberships))
	for _, membership := range memberships {
		view := courseMemberView{
			ID:         membership.ID,
			CourseID:   membership.CourseID,
			UserID:     membership.UserID,
			Role:       membership.Role,
			ClassCount: classCounts[membership.UserID],
			CreatedAt:  membership.CreatedAt,
		}
		if memberUser, ok := users[membership.UserID]; ok {
			view.UserRole = memberUser.Role
			view.Email = memberUser.Email
			view.Name = memberUser.Name
			view.StudentNo = memberUser.StudentNo
		}
		out = append(out, view)
	}
	c.JSON(http.StatusOK, out)
}

func (s Server) addCourseMember(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	courseID, ok := idParam(c, "id")
	if !ok {
		return
	}
	if !s.canAdminCourse(user, courseID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	var course models.Course
	if err := s.DB.First(&course, courseID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "course not found"})
		return
	}
	if course.Archived {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot create class in archived course"})
		return
	}
	var req struct {
		UserID  uint        `json:"user_id"`
		Email   string      `json:"email"`
		Role    models.Role `json:"role" binding:"required"`
		ClassID *uint       `json:"class_id"`
	}
	if !bind(c, &req) {
		return
	}
	if !validCourseMemberRole(req.Role) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "role must be one of student, teacher, admin, course_admin, course_assistant"})
		return
	}
	if req.UserID == 0 {
		req.Email = strings.ToLower(strings.TrimSpace(req.Email))
		if req.Email == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "user_id or email is required"})
			return
		}
		var memberUser models.User
		if err := s.DB.Where("email = ? AND account_deleted = false", req.Email).First(&memberUser).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		req.UserID = memberUser.ID
	}
	var memberUser models.User
	if err := s.DB.Where("id = ? AND account_deleted = false", req.UserID).First(&memberUser).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	if req.Role == models.RoleStudent && memberUser.Role != models.RoleStudent {
		c.JSON(http.StatusBadRequest, gin.H{"error": "student member must use a student account"})
		return
	}
	if (req.Role == models.RoleTeacher || req.Role == models.RoleAdmin || req.Role == models.RoleCourseAdmin || req.Role == models.RoleCourseAssistant) && memberUser.Role == models.RoleStudent {
		c.JSON(http.StatusBadRequest, gin.H{"error": "teacher member must use a teacher or admin account"})
		return
	}
	member := models.CourseMembership{CourseID: courseID, UserID: req.UserID}
	result := s.DB.
		Where("course_id = ? AND user_id = ?", courseID, req.UserID).
		Attrs(models.CourseMembership{CourseID: courseID, UserID: req.UserID, Role: req.Role}).
		FirstOrCreate(&member)
	if result.Error != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": result.Error.Error()})
		return
	}
	if result.RowsAffected == 0 && member.Role != req.Role {
		member.Role = req.Role
		if err := s.DB.Save(&member).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	if req.ClassID != nil {
		var class models.Class
		if err := s.DB.First(&class, *req.ClassID).Error; err != nil || class.CourseID != courseID || class.Archived {
			c.JSON(http.StatusBadRequest, gin.H{"error": "class not found in course"})
			return
		}
		_ = s.DB.Where("class_id = ? AND user_id = ?", *req.ClassID, req.UserID).FirstOrCreate(&models.ClassMembership{ClassID: *req.ClassID, UserID: req.UserID}).Error
	}
	services.Audit(c, s.DB, "course.member.add", "course", courseID, datatypes.JSONMap{"user_id": req.UserID})
	c.JSON(http.StatusCreated, member)
}

func (s Server) removeCourseMember(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	courseID, ok := idParam(c, "id")
	if !ok {
		return
	}
	memberUserID, ok := idParam(c, "user_id")
	if !ok {
		return
	}
	if !s.canAdminCourse(user, courseID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	var course models.Course
	if err := s.DB.First(&course, courseID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "course not found"})
		return
	}
	if course.TeacherID == memberUserID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot remove primary teacher"})
		return
	}
	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("course_id = ? AND user_id = ?", courseID, memberUserID).Delete(&models.CourseMembership{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ? AND class_id IN (?)", memberUserID, tx.Model(&models.Class{}).Select("id").Where("course_id = ?", courseID)).Delete(&models.ClassMembership{}).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	services.Audit(c, s.DB, "course.member.remove", "course", courseID, datatypes.JSONMap{"user_id": memberUserID})
	c.JSON(http.StatusOK, gin.H{"removed": true, "course_id": courseID, "user_id": memberUserID})
}

func (s Server) listClasses(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	classes := []models.Class{}
	q := s.DB.Order("id desc")
	if !includeArchived(c) {
		q = q.Where("archived = false").Where("course_id IN (?)", s.DB.Model(&models.Course{}).Select("id").Where("archived = false"))
	}
	if courseID := c.Query("course_id"); courseID != "" {
		q = q.Where("course_id = ?", courseID)
	}
	switch user.Role {
	case models.RoleAdmin:
	case models.RoleTeacher:
		q = q.Where("course_id IN (?) OR course_id IN (?)",
			s.DB.Model(&models.Course{}).Select("id").Where("teacher_id = ?", user.ID),
			s.DB.Model(&models.CourseMembership{}).Select("course_id").Where("user_id = ? AND role IN ?", user.ID, courseTeachingRoles()),
		)
	default:
		q = q.Where("id IN (?)", s.DB.Model(&models.ClassMembership{}).Select("class_id").Where("user_id = ?", user.ID))
	}
	q.Find(&classes)
	c.JSON(http.StatusOK, s.classListViews(classes))
}

func (s Server) updateClass(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	classID, ok := idParam(c, "id")
	if !ok {
		return
	}
	var class models.Class
	if err := s.DB.First(&class, classID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "class not found"})
		return
	}
	if !s.canAdminCourse(user, class.CourseID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	var req struct {
		Name     string `json:"name"`
		Archived *bool  `json:"archived"`
	}
	if !bind(c, &req) {
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "class name is required"})
		return
	}
	updates := map[string]any{"name": req.Name}
	if req.Archived != nil {
		updates["archived"] = *req.Archived
	}
	if err := s.DB.Model(&class).Updates(updates).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	services.Audit(c, s.DB, "class.update", "class", classID, datatypes.JSONMap{"course_id": class.CourseID})
	_ = s.DB.First(&class, classID).Error
	c.JSON(http.StatusOK, class)
}

func (s Server) archiveClass(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	classID, ok := idParam(c, "id")
	if !ok {
		return
	}
	var class models.Class
	if err := s.DB.First(&class, classID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "class not found"})
		return
	}
	if !s.canAdminCourse(user, class.CourseID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	archived := true
	if c.Request.Method == http.MethodPost {
		var req struct {
			Archived *bool `json:"archived"`
		}
		if !bind(c, &req) {
			return
		}
		if req.Archived != nil {
			archived = *req.Archived
		}
	}
	if err := s.DB.Model(&class).Update("archived", archived).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	services.Audit(c, s.DB, "class.archive", "class", classID, datatypes.JSONMap{"course_id": class.CourseID, "archived": archived})
	c.JSON(http.StatusOK, gin.H{"id": classID, "archived": archived})
}

func (s Server) listClassStudents(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	classID, ok := idParam(c, "id")
	if !ok {
		return
	}
	if !s.canManageClass(user, classID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	rows := []classStudentView{}
	if err := s.DB.Table("class_memberships").
		Select("class_memberships.id as membership_id, users.id as user_id, users.email as email, users.name as name, users.student_no as student_no, class_memberships.created_at as joined_at").
		Joins("join users on users.id = class_memberships.user_id").
		Where("class_memberships.class_id = ? AND users.role = ? AND users.account_deleted = false", classID, models.RoleStudent).
		Order("class_memberships.created_at asc, users.id asc").
		Scan(&rows).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, rows)
}

func (s Server) removeClassStudent(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	classID, ok := idParam(c, "id")
	if !ok {
		return
	}
	studentID, ok := idParam(c, "user_id")
	if !ok {
		return
	}
	var class models.Class
	if err := s.DB.First(&class, classID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "class not found"})
		return
	}
	if !s.canAdminCourse(user, class.CourseID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	if err := s.removeStudentFromClass(class, studentID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	services.Audit(c, s.DB, "class.student.remove", "class", classID, datatypes.JSONMap{"user_id": studentID})
	c.JSON(http.StatusOK, gin.H{"removed": true, "class_id": classID, "user_id": studentID})
}

func (s Server) importClassStudents(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	classID, ok := idParam(c, "id")
	if !ok {
		return
	}
	var class models.Class
	if err := s.DB.First(&class, classID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "class not found"})
		return
	}
	if class.Archived {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot import into archived class"})
		return
	}
	if !s.canAdminCourse(user, class.CourseID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	students, defaultPassword, ok := parseStudentImportRequest(c)
	if !ok {
		return
	}
	results := s.importStudentsIntoClass(class, students, defaultPassword)
	created := 0
	joined := 0
	failed := 0
	for _, result := range results {
		if result.Created {
			created++
		}
		if result.Joined {
			joined++
		}
		if result.Error != "" {
			failed++
		}
	}
	services.Audit(c, s.DB, "class.student.import", "class", classID, datatypes.JSONMap{"created": created, "joined": joined, "failed": failed})
	c.JSON(http.StatusOK, gin.H{"created": created, "joined": joined, "failed": failed, "results": results})
}

func (s Server) myClasses(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	type classView struct {
		ID         uint   `json:"id"`
		ClassID    uint   `json:"class_id"`
		ClassName  string `json:"class_name"`
		JoinCode   string `json:"join_code"`
		CourseID   uint   `json:"course_id"`
		CourseCode string `json:"course_code"`
		CourseName string `json:"course_name"`
		Term       string `json:"term"`
	}
	rows := []classView{}
	q := s.DB.Table("classes").
		Select("classes.id as id, classes.id as class_id, classes.name as class_name, classes.join_code as join_code, courses.id as course_id, courses.code as course_code, courses.name as course_name, courses.term as term").
		Joins("join courses on courses.id = classes.course_id").
		Where("classes.archived = false AND courses.archived = false").
		Order("courses.id desc, classes.id desc")
	switch user.Role {
	case models.RoleAdmin:
	case models.RoleTeacher:
		q = q.Where("courses.teacher_id = ? OR courses.id IN (?)", user.ID, s.DB.Model(&models.CourseMembership{}).Select("course_id").Where("user_id = ? AND role IN ?", user.ID, courseTeachingRoles()))
	default:
		q = q.Joins("join class_memberships on class_memberships.class_id = classes.id").Where("class_memberships.user_id = ?", user.ID)
	}
	q.Scan(&rows)
	c.JSON(http.StatusOK, rows)
}

func (s Server) classListViews(classes []models.Class) []classListView {
	courseIDs := make([]uint, 0, len(classes))
	for _, class := range classes {
		courseIDs = append(courseIDs, class.CourseID)
	}
	courses := s.courseMap(courseIDs)
	views := make([]classListView, 0, len(classes))
	for _, class := range classes {
		view := classListView{Class: class}
		if course, ok := courses[class.CourseID]; ok {
			view.CourseCode = course.Code
			view.CourseName = course.Name
			view.Term = course.Term
		}
		views = append(views, view)
	}
	return views
}

func (s Server) assignmentListViews(items []models.Assignment) []assignmentListView {
	courseIDs := make([]uint, 0, len(items))
	classIDs := make([]uint, 0, len(items))
	for _, item := range items {
		courseIDs = append(courseIDs, item.CourseID)
		if item.ClassID != nil {
			classIDs = append(classIDs, *item.ClassID)
		}
	}
	courses := s.courseMap(courseIDs)
	classes := s.classMap(classIDs)
	views := make([]assignmentListView, 0, len(items))
	for _, item := range items {
		view := assignmentListView{Assignment: item}
		if course, ok := courses[item.CourseID]; ok {
			view.CourseCode = course.Code
			view.CourseName = course.Name
		}
		if item.ClassID != nil {
			if class, ok := classes[*item.ClassID]; ok {
				view.ClassName = class.Name
			}
		}
		views = append(views, view)
	}
	return views
}

func (s Server) examListViews(items []models.Exam) []examListView {
	courseIDs := make([]uint, 0, len(items))
	classIDs := make([]uint, 0, len(items))
	for _, item := range items {
		courseIDs = append(courseIDs, item.CourseID)
		if item.ClassID != nil {
			classIDs = append(classIDs, *item.ClassID)
		}
	}
	courses := s.courseMap(courseIDs)
	classes := s.classMap(classIDs)
	views := make([]examListView, 0, len(items))
	for _, item := range items {
		view := examListView{Exam: item}
		if course, ok := courses[item.CourseID]; ok {
			view.CourseCode = course.Code
			view.CourseName = course.Name
		}
		if item.ClassID != nil {
			if class, ok := classes[*item.ClassID]; ok {
				view.ClassName = class.Name
			}
		}
		views = append(views, view)
	}
	return views
}

func (s Server) courseMap(ids []uint) map[uint]models.Course {
	out := map[uint]models.Course{}
	ids = dedupeUint(ids)
	if len(ids) == 0 {
		return out
	}
	var courses []models.Course
	s.DB.Where("id IN ?", ids).Find(&courses)
	for _, course := range courses {
		out[course.ID] = course
	}
	return out
}

func (s Server) classMap(ids []uint) map[uint]models.Class {
	out := map[uint]models.Class{}
	ids = dedupeUint(ids)
	if len(ids) == 0 {
		return out
	}
	var classes []models.Class
	s.DB.Where("id IN ?", ids).Find(&classes)
	for _, class := range classes {
		out[class.ID] = class
	}
	return out
}

func (s Server) previewClassJoin(c *gin.Context) {
	code := strings.ToUpper(strings.TrimSpace(c.Query("join_code")))
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "join_code is required"})
		return
	}
	var class models.Class
	if err := s.DB.Where("upper(join_code) = ? AND archived = false", code).First(&class).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "class not found"})
		return
	}
	var course models.Course
	if err := s.DB.Where("id = ? AND archived = false", class.CourseID).First(&course).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "course not found"})
		return
	}
	var teacher models.User
	_ = s.DB.Select("id", "email", "name", "role").Where("id = ? AND account_deleted = false", course.TeacherID).First(&teacher).Error
	c.JSON(http.StatusOK, gin.H{
		"class_id":           class.ID,
		"class_name":         class.Name,
		"course_id":          course.ID,
		"course_code":        course.Code,
		"course_name":        course.Name,
		"course_description": course.Description,
		"term":               course.Term,
		"teacher_id":         teacher.ID,
		"teacher_name":       teacher.Name,
	})
}

func (s Server) joinClassByCode(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	var req struct {
		JoinCode string `json:"join_code" binding:"required"`
	}
	if !bind(c, &req) {
		return
	}
	code := strings.ToUpper(strings.TrimSpace(req.JoinCode))
	var class models.Class
	if err := s.DB.Where("upper(join_code) = ? AND archived = false", code).First(&class).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "class not found"})
		return
	}
	s.joinStudentToClass(c, user, class)
}

func (s Server) joinClass(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	classID, ok := idParam(c, "id")
	if !ok {
		return
	}
	var class models.Class
	if err := s.DB.Where("id = ? AND archived = false", classID).First(&class).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "class not found"})
		return
	}
	s.joinStudentToClass(c, user, class)
}

func (s Server) joinStudentToClass(c *gin.Context, user models.User, class models.Class) {
	var course models.Course
	if err := s.DB.Where("id = ? AND archived = false", class.CourseID).First(&course).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "course not found"})
		return
	}
	if err := s.DB.Where("class_id = ? AND user_id = ?", class.ID, user.ID).FirstOrCreate(&models.ClassMembership{ClassID: class.ID, UserID: user.ID}).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_ = s.DB.Where("course_id = ? AND user_id = ?", class.CourseID, user.ID).FirstOrCreate(&models.CourseMembership{CourseID: class.CourseID, UserID: user.ID, Role: models.RoleStudent}).Error
	services.Audit(c, s.DB, "class.join", "class", class.ID, nil)
	c.JSON(http.StatusCreated, gin.H{"joined": true, "class_id": class.ID, "join_code": class.JoinCode})
}

func (s Server) leaveClass(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	classID, ok := idParam(c, "id")
	if !ok {
		return
	}
	var class models.Class
	if err := s.DB.First(&class, classID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "class not found"})
		return
	}
	var count int64
	if err := s.DB.Model(&models.ClassMembership{}).Where("class_id = ? AND user_id = ?", classID, user.ID).Count(&count).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if count == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "class membership not found"})
		return
	}
	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("class_id = ? AND user_id = ?", classID, user.ID).Delete(&models.ClassMembership{}).Error; err != nil {
			return err
		}
		var remaining int64
		if err := tx.Table("class_memberships").
			Joins("join classes on classes.id = class_memberships.class_id").
			Where("class_memberships.user_id = ? AND classes.course_id = ?", user.ID, class.CourseID).
			Count(&remaining).Error; err != nil {
			return err
		}
		if remaining == 0 {
			if err := tx.Where("course_id = ? AND user_id = ? AND role = ?", class.CourseID, user.ID, models.RoleStudent).Delete(&models.CourseMembership{}).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	services.Audit(c, s.DB, "class.leave", "class", classID, nil)
	c.JSON(http.StatusOK, gin.H{"left": true, "class_id": classID})
}

func (s Server) listProblems(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	var problems []models.Problem
	q := s.DB.Model(&models.Problem{}).Where("problems.deleted_at IS NULL").Order("problems.id desc")
	if classID, ok := queryUint(c, "class_id"); ok {
		if !s.canAccessClass(user, classID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		q = q.Joins("join class_problems on class_problems.problem_id = problems.id").
			Where("class_problems.class_id = ?", classID).
			Where(releasedClassProblemSQL(), time.Now())
	} else if user.Role == models.RoleStudent {
		classIDs := s.visibleClassIDs(user)
		if len(classIDs) == 0 {
			c.JSON(http.StatusOK, []models.Problem{})
			return
		}
		q = q.Joins("join class_problems on class_problems.problem_id = problems.id").
			Where("class_problems.class_id IN ?", classIDs).
			Where(releasedClassProblemSQL(), time.Now()).
			Group("problems.id")
	} else if user.Role == models.RoleAdmin {
		released := s.DB.Model(&models.ClassProblem{}).Select("problem_id").Where(releasedClassProblemSQL(), time.Now())
		prepared := s.DB.Model(&models.PreparedProblem{}).Select("problem_id")
		q = q.Where("problems.id NOT IN (?) OR problems.id IN (?)", prepared, released)
	} else if user.Role == models.RoleTeacher {
		classIDs := s.visibleClassIDs(user)
		prepared := s.DB.Model(&models.PreparedProblem{}).Select("problem_id")
		if len(classIDs) > 0 {
			released := s.DB.Model(&models.ClassProblem{}).Select("problem_id").Where("class_id IN ?", classIDs).Where(releasedClassProblemSQL(), time.Now())
			q = q.Where("(owner_id = ? AND problems.id NOT IN (?)) OR problems.id IN (?)", user.ID, prepared, released)
		} else {
			q = q.Where("owner_id = ? AND problems.id NOT IN (?)", user.ID, prepared)
		}
	}
	q.Find(&problems)
	if user.Role != models.RoleStudent {
		c.JSON(http.StatusOK, problems)
		return
	}
	progress := map[uint]models.ProblemProgress{}
	var rows []models.ProblemProgress
	if len(problems) > 0 {
		ids := make([]uint, 0, len(problems))
		for _, item := range problems {
			ids = append(ids, item.ID)
		}
		s.DB.Where("user_id = ? AND problem_id IN ?", user.ID, ids).Find(&rows)
		for _, item := range rows {
			progress[item.ProblemID] = item
		}
	}
	type problemView struct {
		models.Problem
		ProgressStatus string `json:"progress_status"`
		Points         int    `json:"points"`
		PointsAwarded  bool   `json:"points_awarded"`
	}
	views := make([]problemView, 0, len(problems))
	for _, item := range problems {
		status := string(models.ProgressUnattempted)
		points := 0
		awarded := false
		if row, ok := progress[item.ID]; ok {
			status = string(row.Status)
			points = row.Points
			awarded = row.PointsAwarded
		}
		views = append(views, problemView{Problem: item, ProgressStatus: status, Points: points, PointsAwarded: awarded})
	}
	c.JSON(http.StatusOK, views)
}

func (s Server) uploadProblem(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	classIDs, ok := s.parseProblemClassIDs(c, user)
	if !ok {
		return
	}
	file, err := c.FormFile("package")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "package file is required"})
		return
	}
	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	defer src.Close()
	body, err := services.ReadLimited(src, services.MaxProblemPackageSize, "problem package")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	pkg, err := services.ParseProblemPackage(body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	problem, ok := s.saveProblemPackage(c, user, body, pkg, classIDs, nil, tagsJSONMap(parseTagFields(c.PostFormArray("tags"), c.PostForm("tags"))), "problem.upload")
	if !ok {
		return
	}
	c.JSON(http.StatusCreated, problem)
}

func (s Server) createProblem(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	if strings.HasPrefix(strings.ToLower(c.GetHeader("Content-Type")), "multipart/form-data") {
		s.createProblemMultipart(c, user)
		return
	}
	var req services.ProblemPackageDraft
	if !bind(c, &req) {
		return
	}
	if !s.validateClassIDs(c, user, req.ClassIDs) {
		return
	}
	body, pkg, err := services.BuildProblemPackage(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	problem, ok := s.saveProblemPackage(c, user, body, pkg, req.ClassIDs, nil, tagsJSONMap(req.Tags), "problem.create")
	if !ok {
		return
	}
	c.JSON(http.StatusCreated, problem)
}

func (s Server) createProblemMultipart(c *gin.Context, user models.User) {
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	rawDraft := strings.TrimSpace(c.PostForm("draft"))
	if rawDraft == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "draft is required"})
		return
	}
	var req services.ProblemPackageDraft
	if err := json.Unmarshal([]byte(rawDraft), &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "parse draft: " + err.Error()})
		return
	}
	if !s.validateClassIDs(c, user, req.ClassIDs) {
		return
	}
	uploads, err := testPointUploadsFromMultipart(c.Request.MultipartForm)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	cases, err := services.BuildProblemCasesFromTestPointFiles(uploads)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Cases = cases
	body, pkg, err := services.BuildProblemPackage(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	problem, ok := s.saveProblemPackage(c, user, body, pkg, req.ClassIDs, nil, tagsJSONMap(req.Tags), "problem.create")
	if !ok {
		return
	}
	c.JSON(http.StatusCreated, problem)
}

func (s Server) parseMarkdownBatch(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	defer file.Close()

	if !strings.HasSuffix(strings.ToLower(header.Filename), ".md") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only .md markdown files are supported"})
		return
	}

	raw, err := services.ReadLimited(file, 32<<20, "markdown file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "read file: " + err.Error()})
		return
	}

	result, err := services.ParseBatchMarkdown(string(raw))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func testPointUploadsFromMultipart(form *multipart.Form) ([]services.TestPointUploadFile, error) {
	if form == nil || len(form.File) == 0 {
		return nil, fmt.Errorf("test files are required")
	}
	var uploads []services.TestPointUploadFile
	totalSize := 0
	for _, headers := range form.File {
		for _, header := range headers {
			src, err := header.Open()
			if err != nil {
				return nil, err
			}
			body, err := services.ReadLimited(src, services.MaxProblemTestFilesSize, "test files")
			_ = src.Close()
			if err != nil {
				return nil, err
			}
			totalSize += len(body)
			if len(body) > services.MaxProblemTestFilesSize || totalSize > services.MaxProblemTestFilesSize {
				return nil, fmt.Errorf("test files are too large")
			}
			uploads = append(uploads, services.TestPointUploadFile{Name: header.Filename, Body: body})
		}
	}
	if len(uploads) == 0 {
		return nil, fmt.Errorf("test files are required")
	}
	return uploads, nil
}

type problemPackageArtifacts struct {
	Object   string
	Checksum string
	Manifest datatypes.JSONMap
}

func (s Server) uploadProblemPackageArtifacts(c *gin.Context, body []byte, pkg *services.ParsedProblemPackage) (problemPackageArtifacts, bool) {
	baseObject := fmt.Sprintf("problems/%s/%d", pkg.Manifest.Slug, time.Now().UnixNano())
	object := baseObject + ".zip"
	if _, err := s.MinIO.PutObject(c.Request.Context(), s.Cfg.MinIOBucket, object, bytes.NewReader(body), int64(len(body)), minio.PutObjectOptions{ContentType: "application/zip"}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return problemPackageArtifacts{}, false
	}
	for _, asset := range pkg.Assets {
		assetObject := fmt.Sprintf("%s/%s", baseObject, asset.Path)
		if _, err := s.MinIO.PutObject(c.Request.Context(), s.Cfg.MinIOBucket, assetObject, bytes.NewReader(asset.Body), asset.Size, minio.PutObjectOptions{ContentType: asset.ContentType}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return problemPackageArtifacts{}, false
		}
		for i := range pkg.Manifest.Assets {
			if pkg.Manifest.Assets[i].Path == asset.Path {
				pkg.Manifest.Assets[i].Object = assetObject
				break
			}
		}
	}
	manifestJSON, _ := json.Marshal(pkg.Manifest)
	var manifest datatypes.JSONMap
	_ = json.Unmarshal(manifestJSON, &manifest)
	return problemPackageArtifacts{Object: object, Checksum: pkg.SHA256, Manifest: manifest}, true
}

func (s Server) saveProblemPackage(c *gin.Context, user models.User, body []byte, pkg services.ParsedProblemPackage, classIDs []uint, releaseAt *time.Time, tags datatypes.JSONMap, action string) (models.Problem, bool) {
	artifacts, ok := s.uploadProblemPackageArtifacts(c, body, &pkg)
	if !ok {
		return models.Problem{}, false
	}
	problem := models.Problem{
		OwnerID:         user.ID,
		DisplayCode:     "",
		Slug:            pkg.Manifest.Slug,
		Title:           pkg.Manifest.Title,
		Statement:       pkg.Manifest.Statement,
		Tags:            tags,
		TimeLimitMS:     pkg.Manifest.TimeLimitMS,
		MemoryLimitMB:   pkg.Manifest.MemoryLimitMB,
		OutputLimitKB:   pkg.Manifest.OutputLimitKB,
		PackageObject:   artifacts.Object,
		PackageChecksum: artifacts.Checksum,
		Manifest:        artifacts.Manifest,
	}
	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		displayCode, err := nextProblemDisplayCode(tx)
		if err != nil {
			return err
		}
		problem.DisplayCode = displayCode
		if err := tx.Create(&problem).Error; err != nil {
			return err
		}
		for _, classID := range classIDs {
			if err := s.linkProblemToClass(tx, classID, problem.ID, releaseAt); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return models.Problem{}, false
	}
	services.Audit(c, s.DB, action, "problem", problem.ID, datatypes.JSONMap{"slug": problem.Slug})
	return problem, true
}

func nextProblemDisplayCode(tx *gorm.DB) (string, error) {
	var codes []string
	if err := tx.Model(&models.Problem{}).Pluck("display_code", &codes).Error; err != nil {
		return "", err
	}
	maxIndex := 0
	for _, code := range codes {
		if index := parseProblemDisplayCode(code); index > maxIndex {
			maxIndex = index
		}
	}
	return models.FormatProblemDisplayCode(maxIndex + 1), nil
}

func parseProblemDisplayCode(value string) int {
	value = strings.TrimSpace(strings.ToUpper(value))
	if !strings.HasPrefix(value, "T") {
		return 0
	}
	index, err := strconv.Atoi(strings.TrimPrefix(value, "T"))
	if err != nil || index < 0 {
		return 0
	}
	return index
}

func (s Server) getProblem(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	id, ok := idParam(c, "id")
	if !ok {
		return
	}
	var problem models.Problem
	if err := s.DB.Where("deleted_at IS NULL").First(&problem, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "problem not found"})
		return
	}
	if user.Role == models.RoleStudent && !s.canStudentAccessProblem(user.ID, problem.ID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	c.JSON(http.StatusOK, problem)
}

func (s Server) updateProblem(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	id, ok := idParam(c, "id")
	if !ok {
		return
	}
	var problem models.Problem
	if err := s.DB.Where("deleted_at IS NULL").First(&problem, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "problem not found"})
		return
	}
	if !s.canManageProblemData(user, problem) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	req, replacementCases, testsUpdated, ok := s.parseProblemUpdateRequest(c)
	if !ok {
		return
	}
	currentBody, ok := s.problemPackageBody(c, problem)
	if !ok {
		return
	}
	currentPkg, err := services.ParseProblemPackage(currentBody)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	manifest := currentPkg.Manifest
	if title := strings.TrimSpace(req.Title); title != "" {
		manifest.Title = title
	} else {
		manifest.Title = problem.Title
	}
	manifest.Statement = req.Statement
	if req.TimeLimitMS > 0 {
		manifest.TimeLimitMS = req.TimeLimitMS
	} else {
		manifest.TimeLimitMS = problem.TimeLimitMS
	}
	if req.MemoryLimitMB > 0 {
		manifest.MemoryLimitMB = req.MemoryLimitMB
	} else {
		manifest.MemoryLimitMB = problem.MemoryLimitMB
	}
	if req.OutputLimitKB > 0 {
		manifest.OutputLimitKB = req.OutputLimitKB
	} else {
		manifest.OutputLimitKB = problem.OutputLimitKB
	}
	rebuiltBody, rebuiltPkg, err := services.RebuildProblemPackage(currentBody, manifest, replacementCases)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	artifacts, ok := s.uploadProblemPackageArtifacts(c, rebuiltBody, &rebuiltPkg)
	if !ok {
		return
	}
	updates := map[string]any{
		"title":            rebuiltPkg.Manifest.Title,
		"statement":        rebuiltPkg.Manifest.Statement,
		"time_limit_ms":    rebuiltPkg.Manifest.TimeLimitMS,
		"memory_limit_mb":  rebuiltPkg.Manifest.MemoryLimitMB,
		"output_limit_kb":  rebuiltPkg.Manifest.OutputLimitKB,
		"package_object":   artifacts.Object,
		"package_checksum": artifacts.Checksum,
		"manifest":         artifacts.Manifest,
	}
	if req.Tags != nil {
		updates["tags"] = tagsJSONMap(req.Tags)
	}
	if err := s.DB.Model(&models.Problem{}).Where("id = ?", problem.ID).Updates(updates).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	services.Audit(c, s.DB, "problem.update", "problem", problem.ID, datatypes.JSONMap{"slug": problem.Slug, "tests_updated": testsUpdated})
	var fresh models.Problem
	if err := s.DB.First(&fresh, problem.ID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, fresh)
}

func (s Server) parseProblemUpdateRequest(c *gin.Context) (problemUpdateInput, []services.ProblemCaseDraft, bool, bool) {
	var req problemUpdateInput
	var replacementCases []services.ProblemCaseDraft
	testsUpdated := false
	if strings.HasPrefix(strings.ToLower(c.GetHeader("Content-Type")), "multipart/form-data") {
		if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return req, nil, false, false
		}
		rawDraft := strings.TrimSpace(c.PostForm("draft"))
		if rawDraft == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "draft is required"})
			return req, nil, false, false
		}
		if err := json.Unmarshal([]byte(rawDraft), &req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "parse draft: " + err.Error()})
			return req, nil, false, false
		}
		uploads, hasFiles, err := optionalTestPointUploadsFromMultipart(c.Request.MultipartForm)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return req, nil, false, false
		}
		if hasFiles {
			cases, err := services.BuildProblemCasesFromTestPointFiles(uploads)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return req, nil, false, false
			}
			replacementCases = cases
			testsUpdated = true
		}
		return req, replacementCases, testsUpdated, true
	}
	if !bind(c, &req) {
		return req, nil, false, false
	}
	return req, nil, false, true
}

func optionalTestPointUploadsFromMultipart(form *multipart.Form) ([]services.TestPointUploadFile, bool, error) {
	if form == nil || len(form.File) == 0 {
		return nil, false, nil
	}
	uploads, err := testPointUploadsFromMultipart(form)
	if err != nil {
		return nil, false, err
	}
	return uploads, true, nil
}

func (s Server) getProblemAsset(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	id, ok := idParam(c, "id")
	if !ok {
		return
	}
	assetPath, err := services.NormalizeAssetPath(strings.TrimPrefix(c.Param("asset_path"), "/"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var problem models.Problem
	if err := s.DB.First(&problem, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "problem not found"})
		return
	}
	if !s.canReadProblemAsset(user, problem, assetPath) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	object, contentType := problemAssetObject(problem.Manifest, assetPath)
	if object == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "asset not found"})
		return
	}
	src, err := s.MinIO.GetObject(c.Request.Context(), s.Cfg.MinIOBucket, object, minio.GetObjectOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer src.Close()
	stat, err := src.Stat()
	if err != nil {
		resp := minio.ToErrorResponse(err)
		if resp.Code == "NoSuchKey" || resp.Code == "NoSuchBucket" {
			c.JSON(http.StatusNotFound, gin.H{"error": "asset not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if contentType == "" {
		contentType = stat.ContentType
	}
	c.Header("Cache-Control", "private, max-age=3600")
	c.DataFromReader(http.StatusOK, stat.Size, contentType, src, nil)
}

type problemTestView struct {
	Name   string `json:"name"`
	Input  string `json:"input"`
	Output string `json:"output"`
	Weight int    `json:"weight"`
}

func (s Server) listProblemTests(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	problem, ok := s.problemForTestDownload(c, user)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, gin.H{"tests": problemTestCases(problem.Manifest)})
}

func (s Server) downloadProblemTests(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	problem, ok := s.problemForTestDownload(c, user)
	if !ok {
		return
	}
	cases := problemTestCases(problem.Manifest)
	if len(cases) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "tests not found"})
		return
	}
	body, ok := s.problemPackageBody(c, problem)
	if !ok {
		return
	}
	var out bytes.Buffer
	zw := zip.NewWriter(&out)
	for i, tc := range cases {
		for _, item := range []struct {
			path string
			ext  string
		}{
			{path: tc.Input, ext: ".in"},
			{path: tc.Output, ext: ".out"},
		} {
			data, found, err := problemZipFile(body, item.path)
			if err != nil {
				_ = zw.Close()
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			if !found {
				_ = zw.Close()
				c.JSON(http.StatusNotFound, gin.H{"error": "test file not found"})
				return
			}
			name := fmt.Sprintf("tests/%02d_%s%s", i+1, safeDownloadStem(tc.Name, i+1), item.ext)
			w, err := zw.Create(name)
			if err != nil {
				_ = zw.Close()
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			if _, err := w.Write(data); err != nil {
				_ = zw.Close()
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}
	}
	if err := zw.Close(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	filename := fmt.Sprintf("%s-tests.zip", problemDownloadStem(problem))
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Data(http.StatusOK, "application/zip", out.Bytes())
}

func (s Server) downloadProblemTestFile(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	problem, ok := s.problemForTestDownload(c, user)
	if !ok {
		return
	}
	testPath, err := normalizeTestPath(strings.TrimPrefix(c.Param("file_path"), "/"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !problemManifestHasTestPath(problem.Manifest, testPath) {
		c.JSON(http.StatusNotFound, gin.H{"error": "test file not found"})
		return
	}
	body, ok := s.problemPackageBody(c, problem)
	if !ok {
		return
	}
	data, found, err := problemZipFile(body, testPath)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "test file not found"})
		return
	}
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filepath.Base(testPath)))
	c.Data(http.StatusOK, "text/plain; charset=utf-8", data)
}

func (s Server) problemForTestDownload(c *gin.Context, user models.User) (models.Problem, bool) {
	id, ok := idParam(c, "id")
	if !ok {
		return models.Problem{}, false
	}
	var problem models.Problem
	if err := s.DB.Where("deleted_at IS NULL").First(&problem, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "problem not found"})
		return models.Problem{}, false
	}
	if !s.canManageProblemData(user, problem) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return models.Problem{}, false
	}
	return problem, true
}

func (s Server) canManageProblemData(user models.User, problem models.Problem) bool {
	if user.Role == models.RoleAdmin {
		return true
	}
	if user.Role != models.RoleTeacher {
		return false
	}
	return problem.OwnerID == user.ID || s.problemInVisibleClass(user, problem.ID)
}

func (s Server) problemPackageBody(c *gin.Context, problem models.Problem) ([]byte, bool) {
	src, err := s.MinIO.GetObject(c.Request.Context(), s.Cfg.MinIOBucket, problem.PackageObject, minio.GetObjectOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return nil, false
	}
	defer src.Close()
	body, err := services.ReadLimited(src, services.MaxProblemPackageSize, "problem package")
	if err != nil {
		resp := minio.ToErrorResponse(err)
		if resp.Code == "NoSuchKey" || resp.Code == "NoSuchBucket" {
			c.JSON(http.StatusNotFound, gin.H{"error": "problem package not found"})
			return nil, false
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return nil, false
	}
	return body, true
}

func problemTestCases(manifest datatypes.JSONMap) []problemTestView {
	raw, ok := manifest["cases"].([]any)
	if !ok {
		return nil
	}
	out := make([]problemTestView, 0, len(raw))
	for i, item := range raw {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		name, _ := entry["name"].(string)
		if strings.TrimSpace(name) == "" {
			name = fmt.Sprintf("case-%02d", i+1)
		}
		input, _ := entry["input"].(string)
		output, _ := entry["output"].(string)
		cleanInput, err := normalizeTestPath(input)
		if err != nil {
			continue
		}
		cleanOutput, err := normalizeTestPath(output)
		if err != nil {
			continue
		}
		out = append(out, problemTestView{
			Name:   name,
			Input:  cleanInput,
			Output: cleanOutput,
			Weight: intFromJSON(entry["weight"]),
		})
	}
	return out
}

func problemManifestHasTestPath(manifest datatypes.JSONMap, testPath string) bool {
	for _, tc := range problemTestCases(manifest) {
		if tc.Input == testPath || tc.Output == testPath {
			return true
		}
	}
	return false
}

func problemZipFile(body []byte, name string) ([]byte, bool, error) {
	clean, err := normalizeTestPath(name)
	if err != nil {
		return nil, false, err
	}
	reader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return nil, false, err
	}
	for _, file := range reader.File {
		filePath := filepath.ToSlash(filepath.Clean(file.Name))
		if filePath != clean {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			return nil, false, err
		}
		data, err := services.ReadLimited(rc, services.MaxProblemTestFilesSize, "test file")
		_ = rc.Close()
		if err != nil {
			return nil, false, err
		}
		return data, true, nil
	}
	return nil, false, nil
}

func normalizeTestPath(value string) (string, error) {
	clean := filepath.ToSlash(filepath.Clean(strings.TrimSpace(value)))
	if clean == "." || clean == "" || strings.HasPrefix(clean, "../") || strings.HasPrefix(clean, "/") {
		return "", fmt.Errorf("unsafe test path: %s", value)
	}
	if !strings.HasPrefix(clean, "tests/") || strings.TrimPrefix(clean, "tests/") == "" {
		return "", fmt.Errorf("test path must be under tests/: %s", value)
	}
	switch strings.ToLower(filepath.Ext(clean)) {
	case ".in", ".out":
		return clean, nil
	default:
		return "", fmt.Errorf("test file type is not supported: %s", value)
	}
}

func intFromJSON(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case json.Number:
		n, _ := v.Int64()
		return int(n)
	default:
		return 0
	}
}

func problemDownloadStem(problem models.Problem) string {
	if strings.TrimSpace(problem.DisplayCode) != "" {
		return safeDownloadStem(problem.DisplayCode, int(problem.ID))
	}
	return fmt.Sprintf("problem-%d", problem.ID)
}

func safeDownloadStem(value string, fallback int) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Sprintf("case-%02d", fallback)
	}
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_", "*", "_", "?", "_", "\"", "_", "<", "_", ">", "_", "|", "_")
	value = replacer.Replace(value)
	value = strings.Trim(value, " .")
	if value == "" {
		return fmt.Sprintf("case-%02d", fallback)
	}
	return value
}

func (s Server) deleteProblem(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	id, ok := idParam(c, "id")
	if !ok {
		return
	}
	var problem models.Problem
	if err := s.DB.Where("deleted_at IS NULL").First(&problem, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "problem not found"})
		return
	}
	if user.Role != models.RoleAdmin && problem.OwnerID != user.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	now := time.Now()
	if err := s.DB.Model(&models.Problem{}).Where("id = ?", problem.ID).Update("deleted_at", now).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	services.Audit(c, s.DB, "problem.delete", "problem", problem.ID, datatypes.JSONMap{"slug": problem.Slug})
	c.JSON(http.StatusOK, gin.H{"deleted": true, "problem_id": problem.ID})
}

func (s Server) listPreparedProblems(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	items := []models.PreparedProblem{}
	q := s.DB.Model(&models.PreparedProblem{}).Preload("Problem").Joins("join problems on problems.id = prepared_problems.problem_id").Where("problems.deleted_at IS NULL").Order("prepared_problems.updated_at desc, prepared_problems.id desc")
	if user.Role != models.RoleAdmin {
		q = q.Where("prepared_problems.owner_id = ?", user.ID)
	}
	if raw := strings.TrimSpace(c.Query("q")); raw != "" {
		like := "%" + strings.ToLower(raw) + "%"
		q = q.Where(
			"lower(problems.slug) LIKE ? OR lower(problems.title) LIKE ? OR lower(prepared_problems.folder) LIKE ? OR lower(prepared_problems.source) LIKE ? OR lower(prepared_problems.notes) LIKE ?",
			like, like, like, like, like,
		)
	}
	if folder := strings.TrimSpace(c.Query("folder")); folder != "" {
		q = q.Where("prepared_problems.folder = ?", folder)
	}
	if difficulty := strings.TrimSpace(c.Query("difficulty")); difficulty != "" {
		q = q.Where("prepared_problems.difficulty = ?", difficulty)
	}
	if tag := strings.TrimSpace(c.Query("tag")); tag != "" {
		q = q.Where("lower(cast(problems.tags as text)) LIKE ?", "%"+strings.ToLower(tag)+"%")
	}
	switch strings.TrimSpace(c.Query("archived")) {
	case "true", "1":
		q = q.Where("prepared_problems.archived = true")
	case "all":
	default:
		q = q.Where("prepared_problems.archived = false")
	}
	if err := q.Find(&items).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (s Server) getPreparedProblem(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	id, ok := idParam(c, "id")
	if !ok {
		return
	}
	item, ok := s.preparedProblemForUser(c, user, id)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, item)
}

func (s Server) createPreparedProblem(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	var req preparedProblemInput
	if !bind(c, &req) {
		return
	}
	body, pkg, err := services.BuildProblemPackage(req.draft())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	problem, ok := s.saveProblemPackage(c, user, body, pkg, nil, nil, tagsJSONMap(req.Tags), "prepared_problem.create")
	if !ok {
		return
	}
	prepared := models.PreparedProblem{
		ProblemID:  problem.ID,
		OwnerID:    user.ID,
		Folder:     strings.TrimSpace(req.Folder),
		Difficulty: strings.TrimSpace(req.Difficulty),
		Source:     strings.TrimSpace(req.Source),
		Notes:      strings.TrimSpace(req.Notes),
		Problem:    problem,
	}
	if err := s.DB.Create(&prepared).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	services.Audit(c, s.DB, "prepared_problem.create", "prepared_problem", prepared.ID, datatypes.JSONMap{"problem_id": problem.ID})
	c.JSON(http.StatusCreated, prepared)
}

func (s Server) uploadPreparedProblem(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	file, err := c.FormFile("package")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "package file is required"})
		return
	}
	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	defer src.Close()
	body, err := services.ReadLimited(src, services.MaxProblemPackageSize, "problem package")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	pkg, err := services.ParseProblemPackage(body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	tags := parseTagFields(c.PostFormArray("tags"), c.PostForm("tags"))
	problem, ok := s.saveProblemPackage(c, user, body, pkg, nil, nil, tagsJSONMap(tags), "prepared_problem.upload")
	if !ok {
		return
	}
	prepared := models.PreparedProblem{
		ProblemID:  problem.ID,
		OwnerID:    user.ID,
		Folder:     strings.TrimSpace(c.PostForm("folder")),
		Difficulty: strings.TrimSpace(c.PostForm("difficulty")),
		Source:     strings.TrimSpace(c.PostForm("source")),
		Notes:      strings.TrimSpace(c.PostForm("notes")),
		Problem:    problem,
	}
	if err := s.DB.Create(&prepared).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	services.Audit(c, s.DB, "prepared_problem.upload", "prepared_problem", prepared.ID, datatypes.JSONMap{"problem_id": problem.ID})
	c.JSON(http.StatusCreated, prepared)
}

func (s Server) updatePreparedProblem(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	id, ok := idParam(c, "id")
	if !ok {
		return
	}
	item, ok := s.preparedProblemForUser(c, user, id)
	if !ok {
		return
	}
	var req struct {
		Folder     string   `json:"folder"`
		Difficulty string   `json:"difficulty"`
		Source     string   `json:"source"`
		Notes      string   `json:"notes"`
		Tags       []string `json:"tags"`
		Archived   *bool    `json:"archived"`
	}
	if !bind(c, &req) {
		return
	}
	updates := map[string]any{
		"folder":     strings.TrimSpace(req.Folder),
		"difficulty": strings.TrimSpace(req.Difficulty),
		"source":     strings.TrimSpace(req.Source),
		"notes":      strings.TrimSpace(req.Notes),
	}
	if req.Archived != nil {
		updates["archived"] = *req.Archived
	}
	if err := s.DB.Model(&models.PreparedProblem{}).Where("id = ?", item.ID).Updates(updates).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Tags != nil {
		if err := s.DB.Model(&models.Problem{}).Where("id = ?", item.ProblemID).Update("tags", tagsJSONMap(req.Tags)).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	services.Audit(c, s.DB, "prepared_problem.update", "prepared_problem", item.ID, datatypes.JSONMap{"problem_id": item.ProblemID})
	var fresh models.PreparedProblem
	_ = s.DB.Preload("Problem").First(&fresh, item.ID).Error
	c.JSON(http.StatusOK, fresh)
}

func (s Server) publishPreparedProblem(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	id, ok := idParam(c, "id")
	if !ok {
		return
	}
	item, ok := s.preparedProblemForUser(c, user, id)
	if !ok {
		return
	}
	if item.Archived {
		c.JSON(http.StatusBadRequest, gin.H{"error": "archived prepared problem cannot be published"})
		return
	}
	var req struct {
		ClassIDs  []uint     `json:"class_ids"`
		ReleaseAt *time.Time `json:"release_at"`
	}
	if !bind(c, &req) {
		return
	}
	if len(req.ClassIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "class_ids is required"})
		return
	}
	if !s.validateClassIDs(c, user, req.ClassIDs) {
		return
	}
	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		for _, classID := range dedupeUint(req.ClassIDs) {
			if err := s.linkProblemToClass(tx, classID, item.ProblemID, req.ReleaseAt); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	services.Audit(c, s.DB, "prepared_problem.publish", "prepared_problem", item.ID, datatypes.JSONMap{"problem_id": item.ProblemID, "class_ids": req.ClassIDs, "release_at": req.ReleaseAt})
	c.JSON(http.StatusOK, gin.H{"published": true, "problem_id": item.ProblemID})
}

func (s Server) listAssignments(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	var items []models.Assignment
	q := s.DB.Preload("Problems").Where("assignments.deleted_at IS NULL").Order("id desc")
	if classID, ok := queryUint(c, "class_id"); ok {
		if !s.canAccessClass(user, classID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		q = q.Where("class_id = ?", classID)
	}
	if courseID := c.Query("course_id"); courseID != "" {
		q = q.Where("course_id = ?", courseID)
	}
	if c.Query("class_id") == "" && c.Query("course_id") == "" {
		switch user.Role {
		case models.RoleAdmin:
		case models.RoleTeacher:
			q = q.Where("course_id IN (?) OR course_id IN (?)",
				s.DB.Model(&models.Course{}).Select("id").Where("teacher_id = ?", user.ID),
				s.DB.Model(&models.CourseMembership{}).Select("course_id").Where("user_id = ? AND role IN ?", user.ID, courseTeachingRoles()),
			)
		default:
			q = q.Where("class_id IN (?) OR (class_id IS NULL AND course_id IN (?))",
				s.DB.Model(&models.ClassMembership{}).Select("class_id").Where("user_id = ?", user.ID),
				s.DB.Model(&models.CourseMembership{}).Select("course_id").Where("user_id = ? AND role = ?", user.ID, models.RoleStudent),
			)
		}
	}
	q.Find(&items)
	views := s.assignmentListViews(items)
	if user.Role != models.RoleStudent {
		c.JSON(http.StatusOK, views)
		return
	}
	for _, item := range items {
		summary := s.assignmentSummary(item.ID, user.ID, false)
		for i := range views {
			if views[i].ID != item.ID {
				continue
			}
			views[i].WorkStatus = summary.WorkStatus
			views[i].TotalScore = summary.TotalScore
			views[i].MaxScore = summary.MaxScore
			views[i].ScoreReady = summary.ScoreReady
			break
		}
	}
	c.JSON(http.StatusOK, views)
}

func (s Server) createAssignment(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	var req struct {
		CourseID    uint               `json:"course_id"`
		ClassID     *uint              `json:"class_id"`
		Title       string             `json:"title" binding:"required"`
		Description string             `json:"description"`
		StartsAt    *time.Time         `json:"starts_at"`
		DueAt       *time.Time         `json:"due_at"`
		ProblemIDs  []uint             `json:"problem_ids"`
		Problems    []workProblemInput `json:"problems"`
	}
	if !bind(c, &req) {
		return
	}
	problemItems := normalizeWorkProblemInputs(req.Problems, req.ProblemIDs)
	problemIDs := workProblemIDs(problemItems)
	if req.ClassID != nil {
		var class models.Class
		if err := s.DB.First(&class, *req.ClassID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "class not found"})
			return
		}
		req.CourseID = class.CourseID
		if !s.canManageClass(user, *req.ClassID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
	}
	if req.CourseID == 0 || !s.canManageCourse(user, req.CourseID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	prepared, ok := s.validateProblemSelection(c, user, problemIDs, req.ClassID, req.DueAt, "assignment")
	if !ok {
		return
	}
	item := models.Assignment{CourseID: req.CourseID, ClassID: req.ClassID, Title: req.Title, Description: req.Description, StartsAt: req.StartsAt, DueAt: req.DueAt}
	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&item).Error; err != nil {
			return err
		}
		for i, problemItem := range problemItems {
			if err := tx.Create(&models.AssignmentProblem{AssignmentID: item.ID, ProblemID: problemItem.ProblemID, Score: problemItem.Score, SortOrder: i}).Error; err != nil {
				return err
			}
			if _, isPrepared := prepared[problemItem.ProblemID]; isPrepared && req.ClassID != nil {
				if err := s.linkProblemToClass(tx, *req.ClassID, problemItem.ProblemID, req.DueAt); err != nil {
					return err
				}
			}
		}
		return nil
	}); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	services.Audit(c, s.DB, "assignment.create", "assignment", item.ID, nil)
	c.JSON(http.StatusCreated, item)
}

func (s Server) getAssignment(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	id, ok := idParam(c, "id")
	if !ok {
		return
	}
	var item models.Assignment
	if err := s.DB.
		Preload("Problems", func(db *gorm.DB) *gorm.DB { return db.Order("assignment_problems.sort_order asc") }).
		Preload("Problems.Problem").
		Where("assignments.deleted_at IS NULL").
		First(&item, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "assignment not found"})
		return
	}
	if !s.canAccessAssignment(user, item) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	now := time.Now()
	closed := item.DueAt != nil && now.After(*item.DueAt)
	notStarted := item.StartsAt != nil && now.Before(*item.StartsAt)
	if user.Role == models.RoleStudent {
		_ = s.recordAssignmentAttempt(item.ID, user.ID)
	}
	summary := s.assignmentSummary(item.ID, user.ID, true)
	c.JSON(http.StatusOK, gin.H{
		"assignment":     item,
		"problems":       assignmentProblemViews(item),
		"now":            now,
		"closed":         closed,
		"not_started":    notStarted,
		"can_submit":     user.Role != models.RoleStudent || (!closed && !notStarted),
		"work_status":    summary.WorkStatus,
		"total_score":    summary.TotalScore,
		"max_score":      summary.MaxScore,
		"score_ready":    summary.ScoreReady,
		"problem_scores": summary.Problems,
	})
}

func (s Server) assignmentReport(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	id, ok := idParam(c, "id")
	if !ok {
		return
	}
	var item models.Assignment
	if err := s.DB.Preload("Problems", func(db *gorm.DB) *gorm.DB { return db.Order("assignment_problems.sort_order asc") }).
		Preload("Problems.Problem").
		Where("assignments.deleted_at IS NULL").
		First(&item, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "assignment not found"})
		return
	}
	if !s.canManageCourse(user, item.CourseID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	students := s.classStudents(item.ClassID)
	rows := make([]gin.H, 0, len(students))
	for _, student := range students {
		summary := s.assignmentSummary(item.ID, student.ID, true)
		rows = append(rows, gin.H{"user": student, "work_status": summary.WorkStatus, "total_score": summary.TotalScore, "max_score": summary.MaxScore, "score_ready": summary.ScoreReady, "problem_scores": summary.Problems})
	}
	c.JSON(http.StatusOK, gin.H{"assignment": item, "problems": assignmentProblemViews(item), "rows": rows})
}

func (s Server) deleteAssignment(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	id, ok := idParam(c, "id")
	if !ok {
		return
	}
	var item models.Assignment
	if err := s.DB.Preload("Problems").Where("deleted_at IS NULL").First(&item, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "assignment not found"})
		return
	}
	if !s.canManageCourse(user, item.CourseID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	now := time.Now()
	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Assignment{}).Where("id = ?", item.ID).Update("deleted_at", now).Error; err != nil {
			return err
		}
		return s.cleanupFutureReleaseRows(tx, "assignment", item.ID, item.ClassID, item.DueAt)
	}); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	services.Audit(c, s.DB, "assignment.delete", "assignment", item.ID, nil)
	c.JSON(http.StatusOK, gin.H{"deleted": true})
}

func (s Server) listExams(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	var items []models.Exam
	q := s.DB.Preload("Problems").Where("exams.deleted_at IS NULL").Order("id desc")
	if classID, ok := queryUint(c, "class_id"); ok {
		if !s.canAccessClass(user, classID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		q = q.Where("class_id = ?", classID)
	}
	if courseID := c.Query("course_id"); courseID != "" {
		q = q.Where("course_id = ?", courseID)
	}
	if c.Query("class_id") == "" && c.Query("course_id") == "" {
		switch user.Role {
		case models.RoleAdmin:
		case models.RoleTeacher:
			q = q.Where("course_id IN (?) OR course_id IN (?)",
				s.DB.Model(&models.Course{}).Select("id").Where("teacher_id = ?", user.ID),
				s.DB.Model(&models.CourseMembership{}).Select("course_id").Where("user_id = ? AND role IN ?", user.ID, courseTeachingRoles()),
			)
		default:
			q = q.Where("class_id IN (?) OR (class_id IS NULL AND course_id IN (?))",
				s.DB.Model(&models.ClassMembership{}).Select("class_id").Where("user_id = ?", user.ID),
				s.DB.Model(&models.CourseMembership{}).Select("course_id").Where("user_id = ? AND role = ?", user.ID, models.RoleStudent),
			)
		}
	}
	q.Find(&items)
	views := s.examListViews(items)
	if user.Role != models.RoleStudent {
		c.JSON(http.StatusOK, views)
		return
	}
	finished := s.examFinishedAtMap(user.ID, modelIDs(items))
	for _, item := range items {
		summary := s.examSummary(item.ID, user.ID, false)
		for i := range views {
			if views[i].ID != item.ID {
				continue
			}
			views[i].WorkStatus = summary.WorkStatus
			views[i].TotalScore = summary.TotalScore
			views[i].MaxScore = summary.MaxScore
			views[i].ScoreReady = summary.ScoreReady
			views[i].FinishedAt = finished[item.ID]
			break
		}
	}
	c.JSON(http.StatusOK, views)
}

func (s Server) createExam(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	var req struct {
		CourseID     uint               `json:"course_id"`
		ClassID      *uint              `json:"class_id"`
		Title        string             `json:"title" binding:"required"`
		Description  string             `json:"description"`
		StartsAt     *time.Time         `json:"starts_at"`
		EndsAt       *time.Time         `json:"ends_at"`
		ProblemIDs   []uint             `json:"problem_ids"`
		Problems     []workProblemInput `json:"problems"`
		ManualReview bool               `json:"manual_review"`
		LockExit     bool               `json:"lock_exit"`
	}
	if !bind(c, &req) {
		return
	}
	problemItems := normalizeWorkProblemInputs(req.Problems, req.ProblemIDs)
	problemIDs := workProblemIDs(problemItems)
	if req.ClassID != nil {
		var class models.Class
		if err := s.DB.First(&class, *req.ClassID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "class not found"})
			return
		}
		req.CourseID = class.CourseID
		if !s.canManageClass(user, *req.ClassID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
	}
	if req.CourseID == 0 || !s.canManageCourse(user, req.CourseID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	if !prepareExamProblemLabels(c, problemItems, req.ClassID, req.EndsAt) {
		return
	}
	prepared, ok := s.validateProblemSelection(c, user, problemIDs, req.ClassID, req.EndsAt, "exam")
	if !ok {
		return
	}
	settings := datatypes.JSONMap{}
	if req.ManualReview {
		settings["manual_review"] = true
	}
	if req.LockExit {
		settings["lock_exit"] = true
	}
	item := models.Exam{CourseID: req.CourseID, ClassID: req.ClassID, Title: req.Title, Description: req.Description, StartsAt: req.StartsAt, EndsAt: req.EndsAt, Settings: settings}
	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&item).Error; err != nil {
			return err
		}
		for i, problemItem := range problemItems {
			if err := tx.Create(&models.ExamProblem{ExamID: item.ID, ProblemID: problemItem.ProblemID, Label: problemItem.Label, Score: problemItem.Score, SortOrder: i}).Error; err != nil {
				return err
			}
			if _, isPrepared := prepared[problemItem.ProblemID]; isPrepared && req.ClassID != nil {
				if err := s.linkProblemToClass(tx, *req.ClassID, problemItem.ProblemID, req.EndsAt); err != nil {
					return err
				}
			}
			if problemItem.ReleaseAfterExam && req.ClassID != nil {
				if err := s.linkProblemToClass(tx, *req.ClassID, problemItem.ProblemID, req.EndsAt); err != nil {
					return err
				}
			}
		}
		return nil
	}); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	services.Audit(c, s.DB, "exam.create", "exam", item.ID, nil)
	c.JSON(http.StatusCreated, item)
}

func (s Server) getExam(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	id, ok := idParam(c, "id")
	if !ok {
		return
	}
	var item models.Exam
	if err := s.DB.
		Preload("Problems", func(db *gorm.DB) *gorm.DB { return db.Order("exam_problems.sort_order asc") }).
		Preload("Problems.Problem").
		Where("exams.deleted_at IS NULL").
		First(&item, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "exam not found"})
		return
	}
	if !s.canAccessExam(user, item) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	now := time.Now()
	closed := item.EndsAt != nil && now.After(*item.EndsAt)
	notStarted := item.StartsAt != nil && now.Before(*item.StartsAt)
	var finishedAt *time.Time
	if user.Role == models.RoleStudent {
		finishedAt = s.examFinishedAt(item.ID, user.ID)
		if finishedAt != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "exam has been finished", "finished_at": finishedAt})
			return
		}
		if !closed && !notStarted {
			_ = s.recordExamAttempt(item.ID, user.ID)
		}
	}
	summary := s.examSummary(item.ID, user.ID, true)
	allSubmitted := user.Role == models.RoleStudent && s.examAllSubmitted(item.ID, user.ID)
	c.JSON(http.StatusOK, gin.H{
		"exam":           item,
		"problems":       examProblemViews(item),
		"now":            now,
		"closed":         closed,
		"not_started":    notStarted,
		"can_submit":     user.Role != models.RoleStudent || (!closed && !notStarted && finishedAt == nil),
		"manual_review":  examManualReview(item),
		"lock_exit":      examLockExit(item),
		"all_submitted":  allSubmitted,
		"finished_at":    finishedAt,
		"work_status":    summary.WorkStatus,
		"total_score":    summary.TotalScore,
		"max_score":      summary.MaxScore,
		"score_ready":    summary.ScoreReady,
		"problem_scores": summary.Problems,
	})
}

func (s Server) finishExam(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	id, ok := idParam(c, "id")
	if !ok {
		return
	}
	var item models.Exam
	if err := s.DB.Where("exams.deleted_at IS NULL").First(&item, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "exam not found"})
		return
	}
	if item.ClassID != nil && !s.studentInClass(user.ID, *item.ClassID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "exam is not available in your class"})
		return
	}
	if item.ClassID == nil && !s.studentInCourse(user.ID, item.CourseID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "exam is not available in your course"})
		return
	}
	if item.StartsAt != nil && time.Now().Before(*item.StartsAt) {
		c.JSON(http.StatusForbidden, gin.H{"error": "exam has not started"})
		return
	}
	attempt, err := s.finishExamAttempt(item.ID, user.ID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	services.Audit(c, s.DB, "exam.finish", "exam", item.ID, datatypes.JSONMap{"user_id": user.ID, "finished_at": attempt.FinishedAt})
	c.JSON(http.StatusOK, gin.H{"finished": true, "exam_id": item.ID, "finished_at": attempt.FinishedAt})
}

func (s Server) examReport(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	id, ok := idParam(c, "id")
	if !ok {
		return
	}
	var item models.Exam
	if err := s.DB.Preload("Problems", func(db *gorm.DB) *gorm.DB { return db.Order("exam_problems.sort_order asc") }).
		Preload("Problems.Problem").
		Where("exams.deleted_at IS NULL").
		First(&item, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "exam not found"})
		return
	}
	if !s.canManageCourse(user, item.CourseID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	students := s.classStudents(item.ClassID)
	rows := make([]gin.H, 0, len(students))
	for _, student := range students {
		summary := s.examSummary(item.ID, student.ID, true)
		rows = append(rows, gin.H{"user": student, "work_status": summary.WorkStatus, "total_score": summary.TotalScore, "max_score": summary.MaxScore, "score_ready": summary.ScoreReady, "problem_scores": summary.Problems})
	}
	c.JSON(http.StatusOK, gin.H{"exam": item, "manual_review": examManualReview(item), "problems": examProblemViews(item), "rows": rows})
}

func (s Server) examRanking(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	id, ok := idParam(c, "id")
	if !ok {
		return
	}
	var item models.Exam
	if err := s.DB.Preload("Problems", func(db *gorm.DB) *gorm.DB { return db.Order("exam_problems.sort_order asc") }).
		Preload("Problems.Problem").
		Where("exams.deleted_at IS NULL").
		First(&item, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "exam not found"})
		return
	}
	if !s.canManageCourse(user, item.CourseID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	examViews := s.examListViews([]models.Exam{item})
	examView := examListView{Exam: item}
	if len(examViews) > 0 {
		examView = examViews[0]
	}
	problems := make([]examRankingProblem, 0, len(item.Problems))
	maxScore := 0
	for index, problem := range item.Problems {
		label := strings.TrimSpace(problem.Label)
		if label == "" {
			label = defaultProblemLabel(index)
		}
		problems = append(problems, examRankingProblem{
			ProblemID:   problem.ProblemID,
			Label:       label,
			DisplayCode: problem.Problem.DisplayCode,
			Title:       problem.Problem.Title,
			Score:       problem.Score,
		})
		maxScore += problem.Score
	}
	now := time.Now()
	status := "进行中"
	if item.StartsAt != nil && now.Before(*item.StartsAt) {
		status = "未开始"
	}
	if item.EndsAt != nil && now.After(*item.EndsAt) {
		status = "已结束"
	}
	if item.ClassID == nil {
		c.JSON(http.StatusOK, gin.H{
			"exam": gin.H{
				"id":          item.ID,
				"title":       item.Title,
				"description": item.Description,
				"starts_at":   item.StartsAt,
				"ends_at":     item.EndsAt,
				"course_code": examView.CourseCode,
				"course_name": examView.CourseName,
				"class_name":  examView.ClassName,
				"status":      status,
			},
			"has_class":     false,
			"manual_review": examManualReview(item),
			"problems":      problems,
			"rows":          []examRankingRow{},
			"stats":         gin.H{"total_students": 0, "max_score": maxScore, "updated_at": now},
			"now":           now,
		})
		return
	}
	rows := make([]examRankingRow, 0)
	pendingRows := 0
	finishedRows := 0
	for _, student := range s.classStudents(item.ClassID) {
		summary := s.examSummary(item.ID, student.ID, true)
		row := examRankingRow{
			UserID:     student.ID,
			Name:       student.Name,
			StudentNo:  student.StudentNo,
			TotalScore: summary.TotalScore,
			MaxScore:   summary.MaxScore,
			ScoreReady: summary.ScoreReady,
			WorkStatus: summary.WorkStatus,
			FinishedAt: s.examFinishedAt(item.ID, student.ID),
			Problems:   make([]examRankingCell, 0, len(summary.Problems)),
		}
		row.SubmissionCount, row.LastSubmission = s.examSubmissionStats(item.ID, student.ID)
		if row.FinishedAt != nil {
			finishedRows++
		}
		for _, problemScore := range summary.Problems {
			if problemScore.SubmissionID != nil {
				row.Attempted++
			}
			if problemScore.ScoreReady && problemScore.Score > 0 && problemScore.BestScore >= problemScore.Score {
				row.Solved++
			}
			pending := problemScore.PendingReview || (problemScore.SubmissionID != nil && !problemScore.ScoreReady)
			if pending {
				row.PendingCount++
			}
			row.Problems = append(row.Problems, examRankingCell{
				ProblemID:   problemScore.Problem.ID,
				Label:       problemScore.Label,
				BestScore:   problemScore.BestScore,
				MaxScore:    problemScore.Score,
				Status:      problemScore.SubmissionStatus,
				ScoreReady:  problemScore.ScoreReady,
				Pending:     pending,
				SubmittedAt: problemScore.SubmittedAt,
			})
		}
		if row.PendingCount > 0 {
			pendingRows++
		}
		rows = append(rows, row)
	}
	sortExamRankingRows(rows)
	for i := range rows {
		rows[i].Rank = i + 1
	}
	c.JSON(http.StatusOK, gin.H{
		"exam": gin.H{
			"id":          item.ID,
			"title":       item.Title,
			"description": item.Description,
			"starts_at":   item.StartsAt,
			"ends_at":     item.EndsAt,
			"course_code": examView.CourseCode,
			"course_name": examView.CourseName,
			"class_name":  examView.ClassName,
			"status":      status,
		},
		"has_class":     true,
		"manual_review": examManualReview(item),
		"problems":      problems,
		"rows":          rows,
		"stats": gin.H{
			"total_students": len(rows),
			"finished":       finishedRows,
			"pending":        pendingRows,
			"max_score":      maxScore,
			"updated_at":     now,
		},
		"now": now,
	})
}

func (s Server) examSubmissionStats(examID uint, userID uint) (int, *time.Time) {
	var count int64
	s.DB.Model(&models.Submission{}).Where("exam_id = ? AND user_id = ?", examID, userID).Count(&count)
	var row struct {
		LastSubmission *time.Time
	}
	s.DB.Model(&models.Submission{}).
		Select("max(created_at) as last_submission").
		Where("exam_id = ? AND user_id = ?", examID, userID).
		Scan(&row)
	return int(count), row.LastSubmission
}

func sortExamRankingRows(rows []examRankingRow) {
	sort.SliceStable(rows, func(i, j int) bool {
		left := rows[i]
		right := rows[j]
		if left.TotalScore != right.TotalScore {
			return left.TotalScore > right.TotalScore
		}
		if left.Solved != right.Solved {
			return left.Solved > right.Solved
		}
		if cmp := compareTimePtr(left.LastSubmission, right.LastSubmission); cmp != 0 {
			return cmp < 0
		}
		if cmp := compareTimePtr(left.FinishedAt, right.FinishedAt); cmp != 0 {
			return cmp < 0
		}
		if left.StudentNo != right.StudentNo {
			return left.StudentNo < right.StudentNo
		}
		if left.Name != right.Name {
			return left.Name < right.Name
		}
		return left.UserID < right.UserID
	})
}

func compareTimePtr(left *time.Time, right *time.Time) int {
	if left == nil && right == nil {
		return 0
	}
	if left == nil {
		return 1
	}
	if right == nil {
		return -1
	}
	if left.Before(*right) {
		return -1
	}
	if left.After(*right) {
		return 1
	}
	return 0
}

func (s Server) exportExamReport(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	id, ok := idParam(c, "id")
	if !ok {
		return
	}
	var item models.Exam
	if err := s.DB.Where("exams.deleted_at IS NULL").First(&item, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "exam not found"})
		return
	}
	if !s.canManageCourse(user, item.CourseID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	if item.EndsAt == nil || time.Now().Before(*item.EndsAt) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "exam has not ended"})
		return
	}
	rows := [][]xlsxCell{
		{
			xlsxString("学生姓名"),
			xlsxString("学号"),
			xlsxString("通过题目数"),
			xlsxString("所得分数"),
		},
	}
	for _, student := range s.classStudents(item.ClassID) {
		summary := s.examSummary(item.ID, student.ID, true)
		solved := 0
		for _, problem := range summary.Problems {
			if problem.ScoreReady && problem.Score > 0 && problem.BestScore >= problem.Score {
				solved++
			}
		}
		rows = append(rows, []xlsxCell{
			xlsxString(student.Name),
			xlsxString(student.StudentNo),
			xlsxNumber(solved),
			xlsxNumber(summary.TotalScore),
		})
	}
	body, err := buildXLSX(rows)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="exam-%d-report.xlsx"`, item.ID))
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", body)
}

func (s Server) deleteExam(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	id, ok := idParam(c, "id")
	if !ok {
		return
	}
	var item models.Exam
	if err := s.DB.Preload("Problems").Where("deleted_at IS NULL").First(&item, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "exam not found"})
		return
	}
	if !s.canManageCourse(user, item.CourseID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	now := time.Now()
	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Exam{}).Where("id = ?", item.ID).Update("deleted_at", now).Error; err != nil {
			return err
		}
		return s.cleanupFutureReleaseRows(tx, "exam", item.ID, item.ClassID, item.EndsAt)
	}); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	services.Audit(c, s.DB, "exam.delete", "exam", item.ID, nil)
	c.JSON(http.StatusOK, gin.H{"deleted": true})
}

func (s Server) judgeManualExamSubmission(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	exam, sub, ok := s.manualExamSubmission(c, user)
	if !ok {
		return
	}
	if sub.Status == models.StatusQueued || sub.Status == models.StatusRunning {
		c.JSON(http.StatusBadRequest, gin.H{"error": "submission is already judging"})
		return
	}
	if err := s.DB.Model(&sub).Updates(map[string]any{"status": models.StatusQueued, "message": "queued for reference judging"}).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	streamID, err := streams.EnqueueSubmission(c.Request.Context(), s.Redis, sub.ID)
	if err != nil {
		s.DB.Model(&sub).Updates(map[string]any{"status": models.StatusSystemError, "message": err.Error()})
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	services.Audit(c, s.DB, "exam.manual_review.judge", "submission", sub.ID, datatypes.JSONMap{"exam_id": exam.ID, "stream_id": streamID})
	c.JSON(http.StatusOK, gin.H{"queued": true, "submission_id": sub.ID})
}

func (s Server) gradeManualExamSubmission(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	exam, sub, ok := s.manualExamSubmission(c, user)
	if !ok {
		return
	}
	maxScore, ok := s.examProblemScore(exam.ID, sub.ProblemID)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "problem is not part of this exam"})
		return
	}
	var req struct {
		Score int `json:"score"`
	}
	if !bind(c, &req) {
		return
	}
	if req.Score < 0 || req.Score > maxScore {
		c.JSON(http.StatusBadRequest, gin.H{"error": "score out of range"})
		return
	}
	now := time.Now()
	grader := user.ID
	if err := s.DB.Model(&sub).Updates(map[string]any{
		"status":           models.StatusManualGraded,
		"manual_score":     req.Score,
		"manual_graded_by": grader,
		"manual_graded_at": now,
		"message":          "manual graded",
	}).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	services.Audit(c, s.DB, "exam.manual_review.grade", "submission", sub.ID, datatypes.JSONMap{"exam_id": exam.ID, "score": req.Score})
	c.JSON(http.StatusOK, gin.H{"graded": true, "submission_id": sub.ID, "manual_score": req.Score})
}

func (s Server) createSubmission(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	var req struct {
		ProblemID    uint   `json:"problem_id" binding:"required"`
		AssignmentID *uint  `json:"assignment_id"`
		ExamID       *uint  `json:"exam_id"`
		Language     string `json:"language" binding:"required"`
		SourceCode   string `json:"source_code" binding:"required"`
	}
	if !bind(c, &req) {
		return
	}
	if !validLanguage(req.Language) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "language must be one of c, cpp, python, java"})
		return
	}
	if len([]byte(req.SourceCode)) > services.MaxSubmissionSourceSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "source code is too large"})
		return
	}
	if req.AssignmentID != nil && req.ExamID != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "assignment_id and exam_id cannot both be set"})
		return
	}
	var problem models.Problem
	if err := s.DB.Where("deleted_at IS NULL").First(&problem, req.ProblemID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "problem not found"})
		return
	}
	var assignmentAttempt *uint
	var examAttempt *uint
	var examForSubmission *models.Exam
	if user.Role == models.RoleStudent {
		if req.AssignmentID != nil {
			if ok, msg := s.canStudentSubmitAssignment(user.ID, *req.AssignmentID, req.ProblemID); !ok {
				c.JSON(http.StatusForbidden, gin.H{"error": msg})
				return
			}
			assignmentAttempt = req.AssignmentID
		} else if req.ExamID != nil {
			if ok, msg := s.canStudentSubmitExam(user.ID, *req.ExamID, req.ProblemID); !ok {
				c.JSON(http.StatusForbidden, gin.H{"error": msg})
				return
			}
			examAttempt = req.ExamID
		} else if !s.canStudentAccessProblem(user.ID, req.ProblemID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "problem is not available in your classes"})
			return
		}
	} else {
		if req.AssignmentID != nil {
			var assignment models.Assignment
			if err := s.DB.Where("deleted_at IS NULL").First(&assignment, *req.AssignmentID).Error; err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "assignment not found"})
				return
			}
			if !s.canManageCourse(user, assignment.CourseID) {
				c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
				return
			}
			var count int64
			s.DB.Model(&models.AssignmentProblem{}).Where("assignment_id = ? AND problem_id = ?", *req.AssignmentID, req.ProblemID).Count(&count)
			if count == 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "problem is not part of this assignment"})
				return
			}
		}
		if req.ExamID != nil {
			var exam models.Exam
			if err := s.DB.Where("deleted_at IS NULL").First(&exam, *req.ExamID).Error; err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "exam not found"})
				return
			}
			if !s.canManageCourse(user, exam.CourseID) {
				c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
				return
			}
			if !s.examContainsProblem(exam.ID, req.ProblemID) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "problem is not part of this exam"})
				return
			}
			examForSubmission = &exam
		}
	}
	status := models.StatusQueued
	manualReview := false
	if req.ExamID != nil {
		if examForSubmission == nil {
			var exam models.Exam
			if err := s.DB.Where("deleted_at IS NULL").First(&exam, *req.ExamID).Error; err == nil {
				examForSubmission = &exam
			}
		}
		if examForSubmission != nil && examManualReview(*examForSubmission) {
			manualReview = true
			status = models.StatusPendingReview
		}
	}
	sub := models.Submission{
		UserID:       user.ID,
		ProblemID:    problem.ID,
		AssignmentID: req.AssignmentID,
		ExamID:       req.ExamID,
		Language:     req.Language,
		SourceCode:   req.SourceCode,
		Status:       status,
	}
	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		if assignmentAttempt != nil {
			attempt := models.AssignmentAttempt{AssignmentID: *assignmentAttempt, UserID: user.ID}
			if err := tx.Where("assignment_id = ? AND user_id = ?", *assignmentAttempt, user.ID).FirstOrCreate(&attempt).Error; err != nil {
				return err
			}
		}
		if examAttempt != nil {
			attempt := models.ExamAttempt{ExamID: *examAttempt, UserID: user.ID}
			if err := tx.Where("exam_id = ? AND user_id = ?", *examAttempt, user.ID).FirstOrCreate(&attempt).Error; err != nil {
				return err
			}
		}
		return tx.Create(&sub).Error
	}); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if manualReview {
		services.Audit(c, s.DB, "submission.create.manual_review", "submission", sub.ID, nil)
		c.JSON(http.StatusCreated, sub)
		return
	}
	streamID, err := streams.EnqueueSubmission(c.Request.Context(), s.Redis, sub.ID)
	if err != nil {
		s.DB.Model(&sub).Updates(map[string]any{"status": models.StatusSystemError, "message": err.Error()})
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	services.Audit(c, s.DB, "submission.create", "submission", sub.ID, datatypes.JSONMap{"stream_id": streamID})
	c.JSON(http.StatusCreated, sub)
}

func (s Server) listSubmissions(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	var items []models.Submission
	q := s.DB.Order("id desc").Limit(200)
	if user.Role == models.RoleStudent {
		q = q.Where("user_id = ?", user.ID)
	} else if user.Role == models.RoleTeacher {
		primaryCourses := s.DB.Model(&models.Course{}).Select("id").Where("teacher_id = ?", user.ID)
		memberCourses := s.DB.Model(&models.CourseMembership{}).Select("course_id").Where("user_id = ? AND role IN ?", user.ID, courseTeachingRoles())
		visibleClassProblems := s.DB.Table("class_problems").
			Select("class_problems.problem_id").
			Joins("join classes on classes.id = class_problems.class_id").
			Where("classes.course_id IN (?) OR classes.course_id IN (?)", primaryCourses, memberCourses)
		q = q.Where(
			"user_id = ? OR problem_id IN (?) OR assignment_id IN (?) OR exam_id IN (?)",
			user.ID,
			s.DB.Model(&models.Problem{}).Select("id").Where("owner_id = ? OR id IN (?)", user.ID, visibleClassProblems),
			s.DB.Model(&models.Assignment{}).Select("id").Where("deleted_at IS NULL").Where("course_id IN (?) OR course_id IN (?)", primaryCourses, memberCourses),
			s.DB.Model(&models.Exam{}).Select("id").Where("deleted_at IS NULL").Where("course_id IN (?) OR course_id IN (?)", primaryCourses, memberCourses),
		)
	}
	if problemID := c.Query("problem_id"); problemID != "" {
		q = q.Where("problem_id = ?", problemID)
	}
	if assignmentID := c.Query("assignment_id"); assignmentID != "" {
		q = q.Where("assignment_id = ?", assignmentID)
	}
	if examID := c.Query("exam_id"); examID != "" {
		q = q.Where("exam_id = ?", examID)
	} else if user.Role == models.RoleStudent {
		if hiddenExamIDs := s.activeExamIDsForStudent(user.ID); len(hiddenExamIDs) > 0 {
			q = q.Where("(exam_id IS NULL OR exam_id NOT IN ?)", hiddenExamIDs)
		}
	}
	q.Find(&items)
	c.JSON(http.StatusOK, s.submissionListViews(items))
}

func (s Server) submissionListViews(items []models.Submission) []submissionListView {
	views := make([]submissionListView, 0, len(items))
	if len(items) == 0 {
		return views
	}
	ids := make([]uint, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	var results []models.SubmissionResult
	s.DB.Where("submission_id IN ?", ids).Order("submission_id asc, id asc").Find(&results)
	firstFailure := map[uint]string{}
	caseIndex := map[uint]int{}
	for _, result := range results {
		caseIndex[result.SubmissionID]++
		if result.Status == models.StatusAccepted || firstFailure[result.SubmissionID] != "" {
			continue
		}
		firstFailure[result.SubmissionID] = verdictCode(result.Status) + strconv.Itoa(caseIndex[result.SubmissionID])
	}
	for _, item := range items {
		view := submissionListView{Submission: item, ErrorPoint: firstFailure[item.ID]}
		if view.ErrorPoint == "" && item.Status != models.StatusAccepted && item.Status != models.StatusQueued && item.Status != models.StatusRunning && item.Status != models.StatusPendingReview {
			view.ErrorPoint = verdictCode(item.Status)
		}
		views = append(views, view)
	}
	return s.enrichSubmissionViews(views)
}

func (s Server) enrichSubmissionViews(views []submissionListView) []submissionListView {
	if len(views) == 0 {
		return views
	}
	userIDs := make([]uint, 0, len(views))
	problemIDs := make([]uint, 0, len(views))
	assignmentIDs := []uint{}
	examIDs := []uint{}
	for _, view := range views {
		userIDs = append(userIDs, view.UserID)
		problemIDs = append(problemIDs, view.ProblemID)
		if view.AssignmentID != nil {
			assignmentIDs = append(assignmentIDs, *view.AssignmentID)
		}
		if view.ExamID != nil {
			examIDs = append(examIDs, *view.ExamID)
		}
	}
	users := map[uint]models.User{}
	var userRows []models.User
	s.DB.Where("id IN ?", dedupeUint(userIDs)).Find(&userRows)
	for _, user := range userRows {
		users[user.ID] = user
	}
	problems := map[uint]models.Problem{}
	var problemRows []models.Problem
	s.DB.Where("id IN ?", dedupeUint(problemIDs)).Find(&problemRows)
	for _, problem := range problemRows {
		problems[problem.ID] = problem
	}
	assignments := map[uint]models.Assignment{}
	if assignmentIDs = dedupeUint(assignmentIDs); len(assignmentIDs) > 0 {
		var rows []models.Assignment
		s.DB.Where("id IN ?", assignmentIDs).Find(&rows)
		for _, item := range rows {
			assignments[item.ID] = item
		}
	}
	exams := map[uint]models.Exam{}
	if examIDs = dedupeUint(examIDs); len(examIDs) > 0 {
		var rows []models.Exam
		s.DB.Where("id IN ?", examIDs).Find(&rows)
		for _, item := range rows {
			exams[item.ID] = item
		}
	}
	for i := range views {
		if user, ok := users[views[i].UserID]; ok {
			views[i].UserName = user.Name
			views[i].StudentNo = user.StudentNo
		}
		if problem, ok := problems[views[i].ProblemID]; ok {
			views[i].ProblemCode = problem.DisplayCode
			views[i].ProblemTitle = problem.Title
		}
		if views[i].AssignmentID != nil {
			if assignment, ok := assignments[*views[i].AssignmentID]; ok {
				views[i].AssignmentTitle = assignment.Title
			}
		}
		if views[i].ExamID != nil {
			if exam, ok := exams[*views[i].ExamID]; ok {
				views[i].ExamTitle = exam.Title
			}
		}
	}
	return views
}

func verdictCode(status models.SubmissionStatus) string {
	switch status {
	case models.StatusWrongAnswer:
		return "WA"
	case models.StatusTimeLimit:
		return "TLE"
	case models.StatusRuntimeError:
		return "RE"
	case models.StatusMemoryLimit:
		return "MLE"
	case models.StatusOutputLimit:
		return "OLE"
	case models.StatusCompileError:
		return "CE"
	case models.StatusSystemError:
		return "SE"
	case models.StatusManualGraded:
		return "MR"
	default:
		return strings.ToUpper(string(status))
	}
}

func (s Server) getSubmission(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	id, ok := idParam(c, "id")
	if !ok {
		return
	}
	var sub models.Submission
	if err := s.DB.First(&sub, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "submission not found"})
		return
	}
	if !s.canAccessSubmission(user, sub) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	var results []models.SubmissionResult
	if user.Role != models.RoleStudent {
		s.DB.Where("submission_id = ?", sub.ID).Order("id asc").Find(&results)
	}
	submission := s.enrichSubmissionViews([]submissionListView{{Submission: sub}})
	c.JSON(http.StatusOK, gin.H{"submission": submission[0], "results": results})
}

func (s Server) submissionEvents(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	id, ok := idParam(c, "id")
	if !ok {
		return
	}
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	var last string
	for {
		var sub models.Submission
		if err := s.DB.First(&sub, id).Error; err != nil {
			writeSSE(c, "error", gin.H{"error": "submission not found"})
			return
		}
		if !s.canAccessSubmission(user, sub) {
			writeSSE(c, "error", gin.H{"error": "forbidden"})
			return
		}
		payload := gin.H{"id": sub.ID, "status": sub.Status, "score": sub.Score, "manual_score": sub.ManualScore, "manual_graded_at": sub.ManualGradedAt, "time_ms": sub.TimeMS, "memory_kb": sub.MemoryKB, "message": sub.Message, "updated_at": sub.UpdatedAt}
		raw, _ := json.Marshal(payload)
		if string(raw) != last {
			last = string(raw)
			writeSSE(c, "status", payload)
		}
		if terminal(sub.Status) {
			return
		}
		select {
		case <-c.Request.Context().Done():
			return
		case <-ticker.C:
		}
	}
}

func (s Server) leaderboard(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	hiddenExamIDs := []uint{}
	if user.Role == models.RoleStudent {
		hiddenExamIDs = s.activeExamIDsForStudent(user.ID)
	}
	type group struct {
		ClassID    uint             `json:"class_id"`
		ClassName  string           `json:"class_name"`
		CourseID   uint             `json:"course_id"`
		CourseCode string           `json:"course_code"`
		CourseName string           `json:"course_name"`
		Rows       []leaderboardRow `json:"rows"`
	}
	if classID, ok := queryUint(c, "class_id"); ok {
		if !s.canAccessClass(user, classID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		rows := s.classLeaderboardRows(classID, hiddenExamIDs)
		c.JSON(http.StatusOK, rows)
		return
	}
	classIDs := s.visibleClassIDs(user)
	groups := make([]group, 0, len(classIDs))
	for _, classID := range classIDs {
		var info struct {
			ClassID    uint
			ClassName  string
			CourseID   uint
			CourseCode string
			CourseName string
		}
		if err := s.DB.Table("classes").
			Select("classes.id as class_id, classes.name as class_name, courses.id as course_id, courses.code as course_code, courses.name as course_name").
			Joins("join courses on courses.id = classes.course_id").
			Where("classes.id = ?", classID).
			Scan(&info).Error; err != nil || info.ClassID == 0 {
			continue
		}
		groups = append(groups, group{
			ClassID:    info.ClassID,
			ClassName:  info.ClassName,
			CourseID:   info.CourseID,
			CourseCode: info.CourseCode,
			CourseName: info.CourseName,
			Rows:       s.classLeaderboardRows(classID, hiddenExamIDs),
		})
	}
	c.JSON(http.StatusOK, groups)
}

func (s Server) classLeaderboardRows(classID uint, hiddenExamIDs []uint) []leaderboardRow {
	var rows []leaderboardRow
	submissionsJoin := "left join submissions on submissions.user_id = users.id and submissions.problem_id = class_problems.problem_id"
	submissionsArgs := []any{}
	if len(hiddenExamIDs) > 0 {
		submissionsJoin += " and (submissions.exam_id IS NULL OR submissions.exam_id NOT IN ?)"
		submissionsArgs = append(submissionsArgs, hiddenExamIDs)
	}
	s.DB.Table("class_memberships").
		Select("users.id as user_id, users.name as name, count(distinct case when submissions.status = ? then submissions.problem_id end) as solved, count(distinct case when submissions.status = ? then submissions.problem_id end) as score, max(submissions.created_at) as last_submission", models.StatusAccepted, models.StatusAccepted).
		Joins("join users on users.id = class_memberships.user_id and users.role = ?", models.RoleStudent).
		Joins("left join class_problems on class_problems.class_id = class_memberships.class_id and "+releasedClassProblemSQL(), time.Now()).
		Joins(submissionsJoin, submissionsArgs...).
		Where("class_memberships.class_id = ? AND users.account_deleted = false", classID).
		Group("users.id, users.name").
		Order("score desc, solved desc, last_submission asc nulls last, users.id asc").
		Scan(&rows)
	for i := range rows {
		rows[i].Rank = i + 1
	}
	return rows
}

func (s Server) createPlagiarismJob(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	var req struct {
		CourseID     uint   `json:"course_id" binding:"required"`
		AssignmentID *uint  `json:"assignment_id"`
		ExamID       *uint  `json:"exam_id"`
		Language     string `json:"language" binding:"required"`
	}
	if !bind(c, &req) {
		return
	}
	if !validLanguage(req.Language) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported language"})
		return
	}
	job := models.PlagiarismJob{CourseID: req.CourseID, AssignmentID: req.AssignmentID, ExamID: req.ExamID, Language: req.Language, Status: "queued", CreatedBy: user.ID}
	if err := s.DB.Create(&job).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	services.Audit(c, s.DB, "plagiarism.create", "plagiarism_job", job.ID, nil)
	go jplag.Service{DB: s.DB, MinIO: s.MinIO, Cfg: s.Cfg}.Run(context.Background(), job.ID)
	c.JSON(http.StatusCreated, job)
}

func (s Server) listPlagiarismJobs(c *gin.Context) {
	var jobs []models.PlagiarismJob
	s.DB.Order("id desc").Limit(100).Find(&jobs)
	c.JSON(http.StatusOK, s.plagiarismJobViews(jobs))
}

func (s Server) listAuditLogs(c *gin.Context) {
	var logs []models.AuditLog
	s.DB.Order("id desc").Limit(200).Find(&logs)
	c.JSON(http.StatusOK, s.auditLogViews(logs))
}

func (s Server) plagiarismJobViews(jobs []models.PlagiarismJob) []plagiarismJobView {
	courseIDs := []uint{}
	assignmentIDs := []uint{}
	examIDs := []uint{}
	userIDs := []uint{}
	for _, job := range jobs {
		courseIDs = append(courseIDs, job.CourseID)
		userIDs = append(userIDs, job.CreatedBy)
		if job.AssignmentID != nil {
			assignmentIDs = append(assignmentIDs, *job.AssignmentID)
		}
		if job.ExamID != nil {
			examIDs = append(examIDs, *job.ExamID)
		}
	}
	courses := s.courseMap(courseIDs)
	assignments := map[uint]models.Assignment{}
	if assignmentIDs = dedupeUint(assignmentIDs); len(assignmentIDs) > 0 {
		var rows []models.Assignment
		s.DB.Where("id IN ?", assignmentIDs).Find(&rows)
		for _, item := range rows {
			assignments[item.ID] = item
		}
	}
	exams := map[uint]models.Exam{}
	if examIDs = dedupeUint(examIDs); len(examIDs) > 0 {
		var rows []models.Exam
		s.DB.Where("id IN ?", examIDs).Find(&rows)
		for _, item := range rows {
			exams[item.ID] = item
		}
	}
	users := map[uint]models.User{}
	if userIDs = dedupeUint(userIDs); len(userIDs) > 0 {
		var rows []models.User
		s.DB.Where("id IN ?", userIDs).Find(&rows)
		for _, item := range rows {
			users[item.ID] = item
		}
	}
	views := make([]plagiarismJobView, 0, len(jobs))
	for _, job := range jobs {
		view := plagiarismJobView{PlagiarismJob: job}
		if course, ok := courses[job.CourseID]; ok {
			view.CourseCode = course.Code
			view.CourseName = course.Name
		}
		if job.AssignmentID != nil {
			if assignment, ok := assignments[*job.AssignmentID]; ok {
				view.AssignmentTitle = assignment.Title
			}
		}
		if job.ExamID != nil {
			if exam, ok := exams[*job.ExamID]; ok {
				view.ExamTitle = exam.Title
			}
		}
		if user, ok := users[job.CreatedBy]; ok {
			view.CreatedByName = user.Name
		}
		views = append(views, view)
	}
	return views
}

func (s Server) auditLogViews(logs []models.AuditLog) []auditLogView {
	userIDs := []uint{}
	for _, log := range logs {
		if log.ActorUserID != nil {
			userIDs = append(userIDs, *log.ActorUserID)
		}
	}
	users := map[uint]models.User{}
	if userIDs = dedupeUint(userIDs); len(userIDs) > 0 {
		var rows []models.User
		s.DB.Where("id IN ?", userIDs).Find(&rows)
		for _, item := range rows {
			users[item.ID] = item
		}
	}
	views := make([]auditLogView, 0, len(logs))
	for _, log := range logs {
		view := auditLogView{AuditLog: log, ResourceLabel: auditResourceLabel(log.ResourceType)}
		if log.ActorUserID != nil {
			if user, ok := users[*log.ActorUserID]; ok {
				view.ActorName = user.Name
			}
		}
		views = append(views, view)
	}
	return views
}

func auditResourceLabel(resourceType string) string {
	switch resourceType {
	case "user":
		return "用户"
	case "course":
		return "课程"
	case "class":
		return "班级"
	case "problem":
		return "题目"
	case "prepared_problem":
		return "预备题"
	case "assignment":
		return "作业"
	case "exam":
		return "考试"
	case "submission":
		return "提交记录"
	case "plagiarism_job":
		return "查重任务"
	default:
		if strings.TrimSpace(resourceType) == "" {
			return "-"
		}
		return resourceType
	}
}

func bind(c *gin.Context, dest any) bool {
	if err := c.ShouldBindJSON(dest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return false
	}
	return true
}

func queryUint(c *gin.Context, name string) (uint, bool) {
	raw := strings.TrimSpace(c.Query(name))
	if raw == "" {
		return 0, false
	}
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || id == 0 {
		return 0, false
	}
	return uint(id), true
}

func idParam(c *gin.Context, name string) (uint, bool) {
	raw := c.Param(name)
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return 0, false
	}
	return uint(id), true
}

func (s Server) canManageCourse(user models.User, courseID uint) bool {
	if user.Role == models.RoleAdmin {
		return true
	}
	if user.Role != models.RoleTeacher {
		return false
	}
	var count int64
	s.DB.Model(&models.Course{}).Where("id = ? AND teacher_id = ?", courseID, user.ID).Count(&count)
	if count > 0 {
		return true
	}
	s.DB.Model(&models.CourseMembership{}).Where("course_id = ? AND user_id = ? AND role IN ?", courseID, user.ID, courseTeachingRoles()).Count(&count)
	return count > 0
}

func (s Server) canAdminCourse(user models.User, courseID uint) bool {
	if user.Role == models.RoleAdmin {
		return true
	}
	if user.Role != models.RoleTeacher {
		return false
	}
	var count int64
	s.DB.Model(&models.Course{}).Where("id = ? AND teacher_id = ?", courseID, user.ID).Count(&count)
	if count > 0 {
		return true
	}
	s.DB.Model(&models.CourseMembership{}).Where("course_id = ? AND user_id = ? AND role IN ?", courseID, user.ID, courseAdminRoles()).Count(&count)
	return count > 0
}

func (s Server) canAccessClass(user models.User, classID uint) bool {
	if user.Role == models.RoleAdmin {
		return true
	}
	if user.Role == models.RoleTeacher {
		return s.canManageClass(user, classID)
	}
	var count int64
	s.DB.Model(&models.ClassMembership{}).Where("class_id = ? AND user_id = ?", classID, user.ID).Count(&count)
	return count > 0
}

func (s Server) canManageClass(user models.User, classID uint) bool {
	if user.Role == models.RoleAdmin {
		return true
	}
	var class models.Class
	if err := s.DB.First(&class, classID).Error; err != nil {
		return false
	}
	return s.canManageCourse(user, class.CourseID)
}

func (s Server) visibleClassIDs(user models.User) []uint {
	var ids []uint
	switch user.Role {
	case models.RoleAdmin:
		s.DB.Model(&models.Class{}).Where("archived = false").Order("id desc").Pluck("id", &ids)
	case models.RoleTeacher:
		s.DB.Model(&models.Class{}).
			Where("archived = false").
			Where("course_id IN (?) OR course_id IN (?)",
				s.DB.Model(&models.Course{}).Select("id").Where("teacher_id = ?", user.ID),
				s.DB.Model(&models.CourseMembership{}).Select("course_id").Where("user_id = ? AND role IN ?", user.ID, courseTeachingRoles()),
			).
			Order("id desc").Pluck("id", &ids)
	default:
		s.DB.Model(&models.ClassMembership{}).
			Joins("join classes on classes.id = class_memberships.class_id").
			Joins("join courses on courses.id = classes.course_id").
			Where("class_memberships.user_id = ? AND classes.archived = false AND courses.archived = false", user.ID).
			Order("class_id desc").Pluck("class_id", &ids)
	}
	return ids
}

func releasedClassProblemSQL() string {
	return "(class_problems.release_at IS NULL OR class_problems.release_at <= ?)"
}

func problemAssetObject(manifest datatypes.JSONMap, assetPath string) (string, string) {
	raw, ok := manifest["assets"]
	if !ok {
		return "", ""
	}
	assets, ok := raw.([]any)
	if !ok {
		return "", ""
	}
	for _, item := range assets {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		path, _ := entry["path"].(string)
		if path != assetPath {
			continue
		}
		object, _ := entry["object"].(string)
		contentType, _ := entry["content_type"].(string)
		return object, contentType
	}
	return "", ""
}

func (s Server) canReadProblemAsset(user models.User, problem models.Problem, assetPath string) bool {
	if object, _ := problemAssetObject(problem.Manifest, assetPath); object == "" {
		return false
	}
	if user.Role == models.RoleAdmin {
		return true
	}
	if user.Role == models.RoleTeacher {
		if problem.OwnerID == user.ID {
			return true
		}
		return s.problemInVisibleClass(user, problem.ID)
	}
	return s.canStudentAccessProblemRelation(user.ID, problem.ID) || s.studentHasProblemInAccessibleWork(user.ID, problem.ID)
}

func (s Server) canStudentAccessProblem(userID uint, problemID uint) bool {
	var count int64
	s.DB.Table("class_memberships").
		Joins("join class_problems on class_problems.class_id = class_memberships.class_id").
		Joins("join problems on problems.id = class_problems.problem_id").
		Where("class_memberships.user_id = ? AND class_problems.problem_id = ?", userID, problemID).
		Where("problems.deleted_at IS NULL").
		Where(releasedClassProblemSQL(), time.Now()).
		Count(&count)
	return count > 0
}

func (s Server) canStudentAccessProblemRelation(userID uint, problemID uint) bool {
	var count int64
	s.DB.Table("class_memberships").
		Joins("join class_problems on class_problems.class_id = class_memberships.class_id").
		Where("class_memberships.user_id = ? AND class_problems.problem_id = ?", userID, problemID).
		Where(releasedClassProblemSQL(), time.Now()).
		Count(&count)
	return count > 0
}

func (s Server) studentHasProblemInAccessibleWork(userID uint, problemID uint) bool {
	var count int64
	s.DB.Table("assignment_problems").
		Joins("join assignments on assignments.id = assignment_problems.assignment_id").
		Joins("join class_memberships on class_memberships.class_id = assignments.class_id").
		Where("class_memberships.user_id = ? AND assignment_problems.problem_id = ? AND assignments.deleted_at IS NULL", userID, problemID).
		Limit(1).
		Count(&count)
	if count > 0 {
		return true
	}
	s.DB.Table("exam_problems").
		Joins("join exams on exams.id = exam_problems.exam_id").
		Joins("join class_memberships on class_memberships.class_id = exams.class_id").
		Where("class_memberships.user_id = ? AND exam_problems.problem_id = ? AND exams.deleted_at IS NULL", userID, problemID).
		Limit(1).
		Count(&count)
	return count > 0
}

func (s Server) problemInVisibleClass(user models.User, problemID uint) bool {
	classIDs := s.visibleClassIDs(user)
	if len(classIDs) == 0 {
		return false
	}
	var count int64
	s.DB.Model(&models.ClassProblem{}).
		Where("class_id IN ? AND problem_id = ?", classIDs, problemID).
		Count(&count)
	return count > 0
}

func (s Server) examContainsProblem(examID uint, problemID uint) bool {
	var count int64
	s.DB.Model(&models.ExamProblem{}).Where("exam_id = ? AND problem_id = ?", examID, problemID).Count(&count)
	return count > 0
}

func (s Server) activeStartedExamAttemptForStudent(userID uint) (models.Exam, bool) {
	var attempts []models.ExamAttempt
	if err := s.DB.Where("user_id = ? AND finished_at IS NULL", userID).Order("updated_at desc, id desc").Limit(20).Find(&attempts).Error; err != nil {
		return models.Exam{}, false
	}
	now := time.Now()
	for _, attempt := range attempts {
		var exam models.Exam
		if err := s.DB.Where("deleted_at IS NULL").First(&exam, attempt.ExamID).Error; err != nil {
			continue
		}
		if exam.StartsAt != nil && now.Before(*exam.StartsAt) {
			continue
		}
		if exam.EndsAt != nil && now.After(*exam.EndsAt) {
			continue
		}
		return exam, true
	}
	return models.Exam{}, false
}

func (s Server) activeExamIDsForStudent(userID uint) []uint {
	now := time.Now()
	var ids []uint
	s.DB.Table("exams").
		Select("exams.id").
		Joins("join class_memberships on class_memberships.class_id = exams.class_id").
		Joins("left join exam_attempts on exam_attempts.exam_id = exams.id and exam_attempts.user_id = ?", userID).
		Where("class_memberships.user_id = ? AND exams.deleted_at IS NULL", userID).
		Where("(exams.starts_at IS NULL OR exams.starts_at <= ?) AND (exams.ends_at IS NULL OR exams.ends_at > ?)", now, now).
		Where("exam_attempts.finished_at IS NULL").
		Pluck("exams.id", &ids)
	return ids
}

func (s Server) canStudentSubmitAssignment(userID uint, assignmentID uint, problemID uint) (bool, string) {
	var assignment models.Assignment
	if err := s.DB.Where("deleted_at IS NULL").First(&assignment, assignmentID).Error; err != nil {
		return false, "assignment not found"
	}
	if assignment.ClassID == nil || !s.studentInClass(userID, *assignment.ClassID) {
		return false, "assignment is not available in your class"
	}
	var count int64
	s.DB.Model(&models.AssignmentProblem{}).Where("assignment_id = ? AND problem_id = ?", assignmentID, problemID).Count(&count)
	if count == 0 {
		return false, "problem is not part of this assignment"
	}
	now := time.Now()
	if assignment.StartsAt != nil && now.Before(*assignment.StartsAt) {
		return false, "assignment has not started"
	}
	if assignment.DueAt != nil && now.After(*assignment.DueAt) {
		return false, "assignment is closed"
	}
	return true, ""
}

func (s Server) canStudentSubmitExam(userID uint, examID uint, problemID uint) (bool, string) {
	var exam models.Exam
	if err := s.DB.Where("deleted_at IS NULL").First(&exam, examID).Error; err != nil {
		return false, "exam not found"
	}
	if exam.ClassID == nil || !s.studentInClass(userID, *exam.ClassID) {
		return false, "exam is not available in your class"
	}
	var count int64
	s.DB.Model(&models.ExamProblem{}).Where("exam_id = ? AND problem_id = ?", examID, problemID).Count(&count)
	if count == 0 {
		return false, "problem is not part of this exam"
	}
	now := time.Now()
	if exam.StartsAt != nil && now.Before(*exam.StartsAt) {
		return false, "exam has not started"
	}
	if exam.EndsAt != nil && now.After(*exam.EndsAt) {
		return false, "exam is closed"
	}
	if s.examFinishedAt(examID, userID) != nil {
		return false, "exam has been finished"
	}
	return true, ""
}

func (s Server) studentInClass(userID uint, classID uint) bool {
	var count int64
	s.DB.Model(&models.ClassMembership{}).Where("class_id = ? AND user_id = ?", classID, userID).Count(&count)
	return count > 0
}

func (s Server) studentInCourse(userID uint, courseID uint) bool {
	var count int64
	s.DB.Model(&models.CourseMembership{}).Where("course_id = ? AND user_id = ? AND role = ?", courseID, userID, models.RoleStudent).Count(&count)
	return count > 0
}

func (s Server) canAccessAssignment(user models.User, assignment models.Assignment) bool {
	if user.Role == models.RoleAdmin {
		return true
	}
	if user.Role == models.RoleTeacher {
		return s.canManageCourse(user, assignment.CourseID)
	}
	if assignment.ClassID != nil {
		return s.studentInClass(user.ID, *assignment.ClassID)
	}
	return s.studentInCourse(user.ID, assignment.CourseID)
}

func (s Server) canAccessExam(user models.User, exam models.Exam) bool {
	if user.Role == models.RoleAdmin {
		return true
	}
	if user.Role == models.RoleTeacher {
		return s.canManageCourse(user, exam.CourseID)
	}
	if exam.ClassID != nil {
		return s.studentInClass(user.ID, *exam.ClassID)
	}
	return s.studentInCourse(user.ID, exam.CourseID)
}

func assignmentProblemViews(item models.Assignment) []gin.H {
	problems := make([]gin.H, 0, len(item.Problems))
	for _, link := range item.Problems {
		problems = append(problems, gin.H{"problem": link.Problem, "score": link.Score, "problem_id": link.ProblemID})
	}
	return problems
}

func examProblemViews(item models.Exam) []gin.H {
	problems := make([]gin.H, 0, len(item.Problems))
	for _, link := range item.Problems {
		problems = append(problems, gin.H{"problem": link.Problem, "score": link.Score, "label": link.Label, "problem_id": link.ProblemID})
	}
	return problems
}

func prepareExamProblemLabels(c *gin.Context, items []workProblemInput, classID *uint, endsAt *time.Time) bool {
	labels := map[string]bool{}
	for i := range items {
		items[i].Label = strings.TrimSpace(items[i].Label)
		if items[i].Label == "" {
			items[i].Label = defaultProblemLabel(i)
		}
		key := strings.ToLower(items[i].Label)
		if labels[key] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "problem labels must be unique"})
			return false
		}
		labels[key] = true
		if items[i].ReleaseAfterExam && endsAt == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "exam-only problems require an end time"})
			return false
		}
	}
	return true
}

func defaultProblemLabel(index int) string {
	index += 1
	label := ""
	for index > 0 {
		index--
		label = string(rune('A'+index%26)) + label
		index /= 26
	}
	return label
}

func normalizeWorkProblemInputs(items []workProblemInput, problemIDs []uint) []workProblemInput {
	if len(items) == 0 {
		items = make([]workProblemInput, 0, len(problemIDs))
		for _, id := range problemIDs {
			items = append(items, workProblemInput{ProblemID: id, Score: 100})
		}
	}
	seen := map[uint]bool{}
	out := make([]workProblemInput, 0, len(items))
	for _, item := range items {
		if item.ProblemID == 0 || seen[item.ProblemID] {
			continue
		}
		if item.Score <= 0 {
			item.Score = 100
		}
		item.Label = strings.TrimSpace(item.Label)
		seen[item.ProblemID] = true
		out = append(out, item)
	}
	return out
}

func workProblemIDs(items []workProblemInput) []uint {
	ids := make([]uint, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ProblemID)
	}
	return ids
}

func examManualReview(exam models.Exam) bool {
	value, ok := exam.Settings["manual_review"]
	if !ok {
		return false
	}
	if enabled, ok := value.(bool); ok {
		return enabled
	}
	if text, ok := value.(string); ok {
		return text == "true" || text == "1"
	}
	return false
}

func examLockExit(exam models.Exam) bool {
	value, ok := exam.Settings["lock_exit"]
	if !ok {
		return false
	}
	if enabled, ok := value.(bool); ok {
		return enabled
	}
	if text, ok := value.(string); ok {
		return text == "true" || text == "1"
	}
	return false
}

func (s Server) recordAssignmentAttempt(assignmentID uint, userID uint) error {
	attempt := models.AssignmentAttempt{AssignmentID: assignmentID, UserID: userID}
	return s.DB.Where("assignment_id = ? AND user_id = ?", assignmentID, userID).FirstOrCreate(&attempt).Error
}

func (s Server) recordExamAttempt(examID uint, userID uint) error {
	attempt := models.ExamAttempt{ExamID: examID, UserID: userID}
	return s.DB.Where("exam_id = ? AND user_id = ?", examID, userID).FirstOrCreate(&attempt).Error
}

func (s Server) finishExamAttempt(examID uint, userID uint) (models.ExamAttempt, error) {
	attempt := models.ExamAttempt{ExamID: examID, UserID: userID}
	if err := s.DB.Where("exam_id = ? AND user_id = ?", examID, userID).FirstOrCreate(&attempt).Error; err != nil {
		return models.ExamAttempt{}, err
	}
	if attempt.FinishedAt != nil {
		return attempt, nil
	}
	now := time.Now()
	if err := s.DB.Model(&attempt).Update("finished_at", now).Error; err != nil {
		return models.ExamAttempt{}, err
	}
	attempt.FinishedAt = &now
	return attempt, nil
}

func (s Server) examFinishedAt(examID uint, userID uint) *time.Time {
	var attempt models.ExamAttempt
	if err := s.DB.Where("exam_id = ? AND user_id = ? AND finished_at IS NOT NULL", examID, userID).First(&attempt).Error; err != nil {
		return nil
	}
	return attempt.FinishedAt
}

func (s Server) examFinishedAtMap(userID uint, examIDs []uint) map[uint]*time.Time {
	out := map[uint]*time.Time{}
	if len(examIDs) == 0 {
		return out
	}
	var attempts []models.ExamAttempt
	s.DB.Where("user_id = ? AND exam_id IN ? AND finished_at IS NOT NULL", userID, examIDs).Find(&attempts)
	for _, attempt := range attempts {
		out[attempt.ExamID] = attempt.FinishedAt
	}
	return out
}

func (s Server) assignmentSummary(assignmentID uint, userID uint, includeProblems bool) workSummary {
	var links []models.AssignmentProblem
	q := s.DB.Where("assignment_id = ?", assignmentID).Order("sort_order asc")
	if includeProblems {
		q = q.Preload("Problem")
	}
	q.Find(&links)
	var attempts int64
	s.DB.Model(&models.AssignmentAttempt{}).Where("assignment_id = ? AND user_id = ?", assignmentID, userID).Count(&attempts)
	return s.workSummaryForLinks(userID, links, nil, &assignmentID, nil, false, attempts > 0, includeProblems)
}

func (s Server) examSummary(examID uint, userID uint, includeProblems bool) workSummary {
	var exam models.Exam
	_ = s.DB.First(&exam, examID).Error
	var links []models.ExamProblem
	q := s.DB.Where("exam_id = ?", examID).Order("sort_order asc")
	if includeProblems {
		q = q.Preload("Problem")
	}
	q.Find(&links)
	var attempts int64
	s.DB.Model(&models.ExamAttempt{}).Where("exam_id = ? AND user_id = ?", examID, userID).Count(&attempts)
	return s.workSummaryForLinks(userID, nil, links, nil, &examID, examManualReview(exam), attempts > 0, includeProblems)
}

func (s Server) examAllSubmitted(examID uint, userID uint) bool {
	var problemIDs []uint
	if err := s.DB.Model(&models.ExamProblem{}).Where("exam_id = ?", examID).Pluck("problem_id", &problemIDs).Error; err != nil || len(problemIDs) == 0 {
		return false
	}
	var submitted int64
	s.DB.Model(&models.Submission{}).
		Where("exam_id = ? AND user_id = ? AND problem_id IN ?", examID, userID, problemIDs).
		Distinct("problem_id").
		Count(&submitted)
	return submitted >= int64(len(problemIDs))
}

func (s Server) workSummaryForLinks(userID uint, assignmentLinks []models.AssignmentProblem, examLinks []models.ExamProblem, assignmentID *uint, examID *uint, manualReview bool, attempted bool, includeProblems bool) workSummary {
	summary := workSummary{WorkStatus: "unattempted", ScoreReady: false}
	if attempted {
		summary.WorkStatus = "unsubmitted"
	}
	type linkInfo struct {
		Problem   models.Problem
		ProblemID uint
		Label     string
		Score     int
	}
	links := []linkInfo{}
	for _, link := range assignmentLinks {
		links = append(links, linkInfo{Problem: link.Problem, ProblemID: link.ProblemID, Score: link.Score})
	}
	for _, link := range examLinks {
		links = append(links, linkInfo{Problem: link.Problem, ProblemID: link.ProblemID, Label: link.Label, Score: link.Score})
	}
	hasSubmission := false
	hasPending := false
	for _, link := range links {
		summary.MaxScore += link.Score
		view, submitted, pending := s.problemScore(userID, link.ProblemID, link.Problem, link.Score, assignmentID, examID, manualReview)
		view.Label = link.Label
		if submitted {
			hasSubmission = true
		}
		if pending {
			hasPending = true
		}
		summary.TotalScore += view.BestScore
		if includeProblems {
			summary.Problems = append(summary.Problems, view)
		}
	}
	if hasSubmission {
		summary.WorkStatus = "submitted"
		summary.ScoreReady = !hasPending
	}
	return summary
}

func (s Server) problemScore(userID uint, problemID uint, problem models.Problem, maxScore int, assignmentID *uint, examID *uint, manualReview bool) (problemScoreView, bool, bool) {
	view := problemScoreView{Problem: problem, Score: maxScore}
	q := s.DB.Where("user_id = ? AND problem_id = ?", userID, problemID).Order("id desc")
	if assignmentID != nil {
		q = q.Where("assignment_id = ?", *assignmentID)
	} else {
		q = q.Where("assignment_id IS NULL")
	}
	if examID != nil {
		q = q.Where("exam_id = ?", *examID)
	} else {
		q = q.Where("exam_id IS NULL")
	}
	var subs []models.Submission
	q.Find(&subs)
	submitted := len(subs) > 0
	pending := false
	best := -1
	for _, sub := range subs {
		if view.SubmissionID == nil {
			id := sub.ID
			view.SubmissionID = &id
			view.SubmissionStatus = sub.Status
			created := sub.CreatedAt
			view.SubmittedAt = &created
		}
		if manualReview {
			if sub.ManualScore == nil {
				pending = true
				continue
			}
			score := clamp(*sub.ManualScore, 0, maxScore)
			if score > best {
				best = score
				id := sub.ID
				view.SubmissionID = &id
				view.SubmissionStatus = sub.Status
				created := sub.CreatedAt
				view.SubmittedAt = &created
				view.RawScore = sub.Score
			}
			continue
		}
		if sub.Status == models.StatusQueued || sub.Status == models.StatusRunning || sub.Status == models.StatusPendingReview {
			pending = true
			continue
		}
		score := clamp((sub.Score*maxScore+50)/100, 0, maxScore)
		if score > best {
			best = score
			id := sub.ID
			view.SubmissionID = &id
			view.SubmissionStatus = sub.Status
			created := sub.CreatedAt
			view.SubmittedAt = &created
			view.RawScore = sub.Score
		}
	}
	if best >= 0 {
		view.BestScore = best
		view.ScoreReady = true
	}
	view.PendingReview = pending
	return view, submitted, pending
}

func clamp(value int, minValue int, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

type xlsxCell struct {
	Value  string
	Number bool
}

func xlsxString(value string) xlsxCell {
	return xlsxCell{Value: value}
}

func xlsxNumber(value int) xlsxCell {
	return xlsxCell{Value: strconv.Itoa(value), Number: true}
}

func buildXLSX(rows [][]xlsxCell) ([]byte, error) {
	var out bytes.Buffer
	zw := zip.NewWriter(&out)
	files := map[string]string{
		"[Content_Types].xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
			`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">` +
			`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>` +
			`<Default Extension="xml" ContentType="application/xml"/>` +
			`<Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>` +
			`<Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>` +
			`</Types>`,
		"_rels/.rels": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
			`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
			`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>` +
			`</Relationships>`,
		"xl/workbook.xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
			`<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">` +
			`<sheets><sheet name="考试成绩" sheetId="1" r:id="rId1"/></sheets>` +
			`</workbook>`,
		"xl/_rels/workbook.xml.rels": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
			`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
			`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/>` +
			`</Relationships>`,
		"xl/worksheets/sheet1.xml": buildSheetXML(rows),
	}
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			return nil, err
		}
		if _, err := w.Write([]byte(body)); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func buildSheetXML(rows [][]xlsxCell) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	b.WriteString(`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>`)
	for rowIndex, row := range rows {
		b.WriteString(fmt.Sprintf(`<row r="%d">`, rowIndex+1))
		for colIndex, cell := range row {
			ref := fmt.Sprintf("%s%d", xlsxColumnName(colIndex+1), rowIndex+1)
			if cell.Number {
				b.WriteString(fmt.Sprintf(`<c r="%s"><v>%s</v></c>`, ref, cell.Value))
				continue
			}
			b.WriteString(fmt.Sprintf(`<c r="%s" t="inlineStr"><is><t>%s</t></is></c>`, ref, xmlEscape(cell.Value)))
		}
		b.WriteString(`</row>`)
	}
	b.WriteString(`</sheetData></worksheet>`)
	return b.String()
}

func xlsxColumnName(index int) string {
	name := ""
	for index > 0 {
		index--
		name = string(rune('A'+index%26)) + name
		index /= 26
	}
	return name
}

func xmlEscape(value string) string {
	var b bytes.Buffer
	_ = xml.EscapeText(&b, []byte(value))
	return b.String()
}

func (s Server) classStudents(classID *uint) []models.User {
	if classID == nil {
		return []models.User{}
	}
	var students []models.User
	s.DB.Model(&models.User{}).
		Joins("join class_memberships on class_memberships.user_id = users.id").
		Where("class_memberships.class_id = ? AND users.role = ? AND users.account_deleted = false", *classID, models.RoleStudent).
		Order("users.id asc").
		Find(&students)
	return students
}

func (s Server) cleanupFutureReleaseRows(tx *gorm.DB, kind string, workID uint, classID *uint, releaseAt *time.Time) error {
	now := time.Now()
	if classID == nil || releaseAt == nil || !releaseAt.After(now) {
		return nil
	}
	var problemIDs []uint
	if kind == "assignment" {
		if err := tx.Model(&models.AssignmentProblem{}).Where("assignment_id = ?", workID).Pluck("problem_id", &problemIDs).Error; err != nil {
			return err
		}
	} else {
		if err := tx.Model(&models.ExamProblem{}).Where("exam_id = ?", workID).Pluck("problem_id", &problemIDs).Error; err != nil {
			return err
		}
	}
	for _, problemID := range problemIDs {
		var classProblem models.ClassProblem
		result := tx.Where("class_id = ? AND problem_id = ? AND release_at IS NOT NULL AND release_at > ?", *classID, problemID, now).Limit(1).Find(&classProblem)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			continue
		}
		if classProblem.ReleaseAt == nil || !classProblem.ReleaseAt.Equal(*releaseAt) {
			continue
		}
		nextRelease, err := s.nextFutureReleaseForProblem(tx, kind, workID, *classID, problemID, now)
		if err != nil {
			return err
		}
		if nextRelease == nil {
			if err := tx.Delete(&classProblem).Error; err != nil {
				return err
			}
			continue
		}
		if err := tx.Model(&classProblem).Update("release_at", *nextRelease).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s Server) nextFutureReleaseForProblem(tx *gorm.DB, kind string, workID uint, classID uint, problemID uint, now time.Time) (*time.Time, error) {
	var assignmentTimes []time.Time
	assignmentQuery := tx.Table("assignments").
		Joins("join assignment_problems on assignment_problems.assignment_id = assignments.id").
		Where("assignments.deleted_at IS NULL AND assignments.class_id = ? AND assignment_problems.problem_id = ? AND assignments.due_at IS NOT NULL AND assignments.due_at > ?", classID, problemID, now)
	if kind == "assignment" {
		assignmentQuery = assignmentQuery.Where("assignments.id <> ?", workID)
	}
	if err := assignmentQuery.Order("assignments.due_at asc").Limit(1).Pluck("assignments.due_at", &assignmentTimes).Error; err != nil {
		return nil, err
	}

	var examTimes []time.Time
	examQuery := tx.Table("exams").
		Joins("join exam_problems on exam_problems.exam_id = exams.id").
		Where("exams.deleted_at IS NULL AND exams.class_id = ? AND exam_problems.problem_id = ? AND exams.ends_at IS NOT NULL AND exams.ends_at > ?", classID, problemID, now)
	if kind == "exam" {
		examQuery = examQuery.Where("exams.id <> ?", workID)
	}
	if err := examQuery.Order("exams.ends_at asc").Limit(1).Pluck("exams.ends_at", &examTimes).Error; err != nil {
		return nil, err
	}
	var next *time.Time
	if len(assignmentTimes) > 0 {
		value := assignmentTimes[0]
		next = &value
	}
	if len(examTimes) > 0 && (next == nil || examTimes[0].Before(*next)) {
		value := examTimes[0]
		next = &value
	}
	return next, nil
}

func (s Server) manualExamSubmission(c *gin.Context, user models.User) (models.Exam, models.Submission, bool) {
	examID, ok := idParam(c, "id")
	if !ok {
		return models.Exam{}, models.Submission{}, false
	}
	submissionID, ok := idParam(c, "submission_id")
	if !ok {
		return models.Exam{}, models.Submission{}, false
	}
	var exam models.Exam
	if err := s.DB.Where("deleted_at IS NULL").First(&exam, examID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "exam not found"})
		return models.Exam{}, models.Submission{}, false
	}
	if !s.canManageCourse(user, exam.CourseID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return models.Exam{}, models.Submission{}, false
	}
	if !examManualReview(exam) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "exam is not in manual review mode"})
		return models.Exam{}, models.Submission{}, false
	}
	var sub models.Submission
	if err := s.DB.Where("id = ? AND exam_id = ?", submissionID, exam.ID).First(&sub).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "submission not found"})
		return models.Exam{}, models.Submission{}, false
	}
	return exam, sub, true
}

func (s Server) examProblemScore(examID uint, problemID uint) (int, bool) {
	var link models.ExamProblem
	if err := s.DB.Where("exam_id = ? AND problem_id = ?", examID, problemID).First(&link).Error; err != nil {
		return 0, false
	}
	return link.Score, true
}

func (s Server) canAccessSubmission(user models.User, sub models.Submission) bool {
	if user.Role == models.RoleAdmin {
		return true
	}
	if user.Role == models.RoleStudent {
		return sub.UserID == user.ID
	}
	if sub.AssignmentID != nil {
		var assignment models.Assignment
		if err := s.DB.First(&assignment, *sub.AssignmentID).Error; err != nil {
			return false
		}
		return s.canManageCourse(user, assignment.CourseID)
	}
	if sub.ExamID != nil {
		var exam models.Exam
		if err := s.DB.First(&exam, *sub.ExamID).Error; err != nil {
			return false
		}
		return s.canManageCourse(user, exam.CourseID)
	}
	var problem models.Problem
	if err := s.DB.First(&problem, sub.ProblemID).Error; err != nil {
		return false
	}
	return problem.OwnerID == user.ID
}

func (s Server) preparedProblemForUser(c *gin.Context, user models.User, id uint) (models.PreparedProblem, bool) {
	var item models.PreparedProblem
	if err := s.DB.Preload("Problem").Joins("join problems on problems.id = prepared_problems.problem_id").Where("prepared_problems.id = ? AND problems.deleted_at IS NULL", id).First(&item).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "prepared problem not found"})
		return models.PreparedProblem{}, false
	}
	if user.Role != models.RoleAdmin && item.OwnerID != user.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return models.PreparedProblem{}, false
	}
	return item, true
}

func (s Server) validateProblemSelection(c *gin.Context, user models.User, problemIDs []uint, classID *uint, releaseAt *time.Time, contextName string) (map[uint]models.PreparedProblem, bool) {
	ids := dedupeUint(problemIDs)
	if len(ids) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "problem_ids is required"})
		return nil, false
	}
	if classID != nil && !s.canManageClass(user, *classID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot publish to class"})
		return nil, false
	}
	var problems []models.Problem
	if err := s.DB.Where("id IN ? AND deleted_at IS NULL", ids).Find(&problems).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return nil, false
	}
	if len(problems) != len(ids) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "problem not found"})
		return nil, false
	}
	preparedRows := []models.PreparedProblem{}
	_ = s.DB.Where("problem_id IN ?", ids).Find(&preparedRows).Error
	prepared := map[uint]models.PreparedProblem{}
	for _, item := range preparedRows {
		if classID != nil && s.classHasReleasedProblem(*classID, item.ProblemID) {
			continue
		}
		if user.Role != models.RoleAdmin && item.OwnerID != user.ID {
			c.JSON(http.StatusForbidden, gin.H{"error": "prepared problem is not yours"})
			return nil, false
		}
		if item.Archived {
			c.JSON(http.StatusBadRequest, gin.H{"error": "archived prepared problem cannot be used"})
			return nil, false
		}
		prepared[item.ProblemID] = item
	}
	if len(prepared) > 0 {
		if classID == nil {
			// Course-wide exam: prepared problems are allowed but won't be auto-linked to a class
			// The linkProblemToClass in the transaction guards on req.ClassID != nil
		}
		if classID != nil && releaseAt == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": contextName + " with prepared problems requires a deadline"})
			return nil, false
		}
	}
	for _, problem := range problems {
		if _, isPrepared := prepared[problem.ID]; isPrepared || user.Role == models.RoleAdmin || problem.OwnerID == user.ID {
			continue
		}
		if classID != nil && s.classHasReleasedProblem(*classID, problem.ID) {
			continue
		}
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot use problem"})
		return nil, false
	}
	return prepared, true
}

func (s Server) classHasReleasedProblem(classID uint, problemID uint) bool {
	var count int64
	s.DB.Model(&models.ClassProblem{}).
		Joins("join problems on problems.id = class_problems.problem_id").
		Where("class_id = ? AND problem_id = ?", classID, problemID).
		Where("problems.deleted_at IS NULL").
		Where(releasedClassProblemSQL(), time.Now()).
		Count(&count)
	return count > 0
}

func (s Server) linkProblemToClass(tx *gorm.DB, classID uint, problemID uint, releaseAt *time.Time) error {
	item := models.ClassProblem{ClassID: classID, ProblemID: problemID}
	result := tx.Where("class_id = ? AND problem_id = ?", classID, problemID).
		Attrs(models.ClassProblem{ClassID: classID, ProblemID: problemID, ReleaseAt: releaseAt}).
		FirstOrCreate(&item)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		return nil
	}
	if releaseAt == nil {
		return tx.Model(&item).Update("release_at", nil).Error
	}
	if item.ReleaseAt != nil && releaseAt.Before(*item.ReleaseAt) {
		return tx.Model(&item).Update("release_at", releaseAt).Error
	}
	return nil
}

func includeArchived(c *gin.Context) bool {
	value := strings.ToLower(strings.TrimSpace(c.Query("include_archived")))
	if value == "true" || value == "1" || value == "yes" {
		return true
	}
	return strings.ToLower(strings.TrimSpace(c.Query("archived"))) == "all"
}

func courseTeachingRoles() []models.Role {
	return []models.Role{models.RoleTeacher, models.RoleAdmin, models.RoleCourseAdmin, models.RoleCourseAssistant}
}

func courseAdminRoles() []models.Role {
	return []models.Role{models.RoleTeacher, models.RoleAdmin, models.RoleCourseAdmin}
}

func validCourseMemberRole(role models.Role) bool {
	switch role {
	case models.RoleStudent, models.RoleTeacher, models.RoleAdmin, models.RoleCourseAdmin, models.RoleCourseAssistant:
		return true
	default:
		return false
	}
}

func parseStudentImportRequest(c *gin.Context) ([]studentImportInput, string, bool) {
	contentType := strings.ToLower(c.ContentType())
	if strings.HasPrefix(contentType, "multipart/") {
		defaultPassword := strings.TrimSpace(c.PostForm("default_password"))
		if text := strings.TrimSpace(c.PostForm("text")); text != "" {
			students, err := parseStudentRows([]byte(text), ".csv")
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return nil, "", false
			}
			return students, defaultPassword, true
		}
		file, err := c.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "file or text is required"})
			return nil, "", false
		}
		src, err := file.Open()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return nil, "", false
		}
		defer src.Close()
		raw, err := io.ReadAll(src)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return nil, "", false
		}
		students, err := parseStudentRows(raw, filepath.Ext(file.Filename))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return nil, "", false
		}
		return students, defaultPassword, true
	}
	var req struct {
		Students        []studentImportInput `json:"students"`
		Text            string               `json:"text"`
		DefaultPassword string               `json:"default_password"`
	}
	if !bind(c, &req) {
		return nil, "", false
	}
	if len(req.Students) > 0 {
		return req.Students, strings.TrimSpace(req.DefaultPassword), true
	}
	if strings.TrimSpace(req.Text) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "students or text is required"})
		return nil, "", false
	}
	students, err := parseStudentRows([]byte(req.Text), ".csv")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return nil, "", false
	}
	return students, strings.TrimSpace(req.DefaultPassword), true
}

func parseStudentRows(raw []byte, ext string) ([]studentImportInput, error) {
	ext = strings.ToLower(strings.TrimSpace(ext))
	var rows [][]string
	var err error
	if ext == ".xlsx" {
		rows, err = parseSimpleXLSXRows(raw)
	} else {
		rows, err = parseCSVRows(raw)
	}
	if err != nil {
		return nil, err
	}
	return studentInputsFromRows(rows), nil
}

func parseCSVRows(raw []byte) ([][]string, error) {
	reader := csv.NewReader(bytes.NewReader(raw))
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("invalid csv: %w", err)
	}
	return records, nil
}

func studentInputsFromRows(rows [][]string) []studentImportInput {
	cleanRows := make([][]string, 0, len(rows))
	for _, row := range rows {
		clean := make([]string, len(row))
		nonEmpty := false
		for i, cell := range row {
			clean[i] = cleanStudentCell(cell)
			if clean[i] != "" {
				nonEmpty = true
			}
		}
		if nonEmpty {
			cleanRows = append(cleanRows, clean)
		}
	}
	if len(cleanRows) == 0 {
		return []studentImportInput{}
	}
	indexes := map[string]int{"student_no": 0, "name": 1, "email": 2, "password": 3}
	start := 0
	if rowLooksLikeHeader(cleanRows[0]) {
		indexes = map[string]int{}
		for i, cell := range cleanRows[0] {
			if key := studentImportHeaderKey(cell); key != "" {
				indexes[key] = i
			}
		}
		start = 1
	}
	out := make([]studentImportInput, 0, len(cleanRows)-start)
	for _, row := range cleanRows[start:] {
		out = append(out, studentImportInput{
			StudentNo: studentRowValue(row, studentColumnIndex(indexes, "student_no")),
			Name:      studentRowValue(row, studentColumnIndex(indexes, "name")),
			Email:     studentRowValue(row, studentColumnIndex(indexes, "email")),
			Password:  studentRowValue(row, studentColumnIndex(indexes, "password")),
		})
	}
	return out
}

func studentColumnIndex(indexes map[string]int, key string) int {
	index, ok := indexes[key]
	if !ok {
		return -1
	}
	return index
}

func rowLooksLikeHeader(row []string) bool {
	for _, cell := range row {
		if studentImportHeaderKey(cell) != "" {
			return true
		}
	}
	return false
}

func studentImportHeaderKey(value string) string {
	key := strings.ToLower(strings.TrimSpace(value))
	key = strings.ReplaceAll(key, " ", "_")
	key = strings.ReplaceAll(key, "-", "_")
	switch key {
	case "student_no", "student_number", "student_id", "学号":
		return "student_no"
	case "name", "student_name", "姓名", "学生姓名":
		return "name"
	case "email", "mail", "邮箱", "电子邮箱":
		return "email"
	case "password", "密码", "初始密码":
		return "password"
	default:
		return ""
	}
}

func cleanStudentCell(value string) string {
	return strings.TrimSpace(strings.TrimPrefix(value, "\ufeff"))
}

func studentRowValue(row []string, index int) string {
	if index < 0 || index >= len(row) {
		return ""
	}
	return cleanStudentCell(row[index])
}

func parseSimpleXLSXRows(raw []byte) ([][]string, error) {
	reader, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return nil, fmt.Errorf("invalid xlsx: %w", err)
	}
	files := map[string]*zip.File{}
	for _, file := range reader.File {
		files[file.Name] = file
	}
	shared := []string{}
	if file := files["xl/sharedStrings.xml"]; file != nil {
		body, err := readZipFile(file)
		if err != nil {
			return nil, err
		}
		shared = parseSharedStrings(body)
	}
	sheet := files["xl/worksheets/sheet1.xml"]
	if sheet == nil {
		return nil, fmt.Errorf("xlsx sheet1.xml not found")
	}
	body, err := readZipFile(sheet)
	if err != nil {
		return nil, err
	}
	return parseWorksheetRows(body, shared)
}

func readZipFile(file *zip.File) ([]byte, error) {
	rc, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

func parseSharedStrings(raw []byte) []string {
	decoder := xml.NewDecoder(bytes.NewReader(raw))
	out := []string{}
	var current strings.Builder
	inItem := false
	inText := false
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return out
		}
		switch item := token.(type) {
		case xml.StartElement:
			if item.Name.Local == "si" {
				inItem = true
				current.Reset()
			}
			if inItem && item.Name.Local == "t" {
				inText = true
			}
		case xml.EndElement:
			if item.Name.Local == "t" {
				inText = false
			}
			if item.Name.Local == "si" {
				out = append(out, current.String())
				inItem = false
			}
		case xml.CharData:
			if inItem && inText {
				current.Write([]byte(item))
			}
		}
	}
	return out
}

type xlsxWorksheetXML struct {
	Rows []xlsxRowXML `xml:"sheetData>row"`
}

type xlsxRowXML struct {
	Cells []xlsxCellXML `xml:"c"`
}

type xlsxCellXML struct {
	Ref       string        `xml:"r,attr"`
	Type      string        `xml:"t,attr"`
	Value     string        `xml:"v"`
	InlineStr xlsxInlineXML `xml:"is"`
}

type xlsxInlineXML struct {
	Text string `xml:"t"`
}

func parseWorksheetRows(raw []byte, shared []string) ([][]string, error) {
	var sheet xlsxWorksheetXML
	if err := xml.Unmarshal(raw, &sheet); err != nil {
		return nil, fmt.Errorf("invalid xlsx sheet: %w", err)
	}
	rows := make([][]string, 0, len(sheet.Rows))
	for _, row := range sheet.Rows {
		values := []string{}
		for i, cell := range row.Cells {
			index := xlsxCellColumnIndex(cell.Ref)
			if index <= 0 {
				index = i + 1
			}
			for len(values) < index {
				values = append(values, "")
			}
			values[index-1] = xlsxCellText(cell, shared)
		}
		rows = append(rows, values)
	}
	return rows, nil
}

func xlsxCellText(cell xlsxCellXML, shared []string) string {
	if cell.Type == "s" {
		index, err := strconv.Atoi(strings.TrimSpace(cell.Value))
		if err == nil && index >= 0 && index < len(shared) {
			return shared[index]
		}
		return ""
	}
	if cell.Type == "inlineStr" {
		return cell.InlineStr.Text
	}
	return cell.Value
}

func xlsxCellColumnIndex(ref string) int {
	index := 0
	for _, r := range strings.ToUpper(ref) {
		if r < 'A' || r > 'Z' {
			break
		}
		index = index*26 + int(r-'A'+1)
	}
	return index
}

func (s Server) importStudentsIntoClass(class models.Class, students []studentImportInput, defaultPassword string) []studentImportResult {
	if defaultPassword == "" {
		defaultPassword = "Aa123456"
	}
	seen := map[string]bool{}
	results := make([]studentImportResult, 0, len(students))
	for _, input := range students {
		input.StudentNo = strings.TrimSpace(input.StudentNo)
		input.Name = strings.TrimSpace(input.Name)
		input.Email = strings.ToLower(strings.TrimSpace(input.Email))
		input.Password = strings.TrimSpace(input.Password)
		result := studentImportResult{Email: input.Email, StudentNo: input.StudentNo, Name: input.Name}
		if input.Email == "" || input.Name == "" {
			result.Error = "name and email are required"
			results = append(results, result)
			continue
		}
		if seen[input.Email] {
			result.Error = "duplicate email in import"
			results = append(results, result)
			continue
		}
		seen[input.Email] = true
		err := s.DB.Transaction(func(tx *gorm.DB) error {
			var student models.User
			if err := tx.Where("email = ? AND account_deleted = false", input.Email).First(&student).Error; err != nil {
				if err != gorm.ErrRecordNotFound {
					return err
				}
				password := input.Password
				if password == "" {
					password = defaultPassword
				}
				hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
				if err != nil {
					return err
				}
				student = models.User{Email: input.Email, Name: input.Name, Role: models.RoleStudent, StudentNo: input.StudentNo, PasswordHash: string(hash), EmailVerified: true}
				if err := tx.Create(&student).Error; err != nil {
					return err
				}
				result.Created = true
			} else if student.Role != models.RoleStudent {
				return fmt.Errorf("email belongs to a non-student account")
			} else {
				updates := map[string]any{}
				if student.Name == "" && input.Name != "" {
					updates["name"] = input.Name
				}
				if student.StudentNo == "" && input.StudentNo != "" {
					updates["student_no"] = input.StudentNo
				}
				if len(updates) > 0 {
					if err := tx.Model(&student).Updates(updates).Error; err != nil {
						return err
					}
				}
			}
			result.UserID = student.ID
			var existingClass int64
			if err := tx.Model(&models.ClassMembership{}).Where("class_id = ? AND user_id = ?", class.ID, student.ID).Count(&existingClass).Error; err != nil {
				return err
			}
			if err := tx.Where("class_id = ? AND user_id = ?", class.ID, student.ID).FirstOrCreate(&models.ClassMembership{ClassID: class.ID, UserID: student.ID}).Error; err != nil {
				return err
			}
			result.Joined = existingClass == 0
			return tx.Where("course_id = ? AND user_id = ?", class.CourseID, student.ID).FirstOrCreate(&models.CourseMembership{CourseID: class.CourseID, UserID: student.ID, Role: models.RoleStudent}).Error
		})
		if err != nil {
			result.Error = err.Error()
		}
		results = append(results, result)
	}
	return results
}

func (s Server) removeStudentFromClass(class models.Class, studentID uint) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		res := tx.Where("class_id = ? AND user_id = ?", class.ID, studentID).Delete(&models.ClassMembership{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return fmt.Errorf("class membership not found")
		}
		var remaining int64
		if err := tx.Table("class_memberships").
			Joins("join classes on classes.id = class_memberships.class_id").
			Where("class_memberships.user_id = ? AND classes.course_id = ?", studentID, class.CourseID).
			Count(&remaining).Error; err != nil {
			return err
		}
		if remaining == 0 {
			return tx.Where("course_id = ? AND user_id = ? AND role = ?", class.CourseID, studentID, models.RoleStudent).Delete(&models.CourseMembership{}).Error
		}
		return nil
	})
}

func dedupeUint(values []uint) []uint {
	seen := map[uint]bool{}
	out := make([]uint, 0, len(values))
	for _, value := range values {
		if value == 0 || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func modelIDs(items []models.Exam) []uint {
	ids := make([]uint, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}

func parseTagFields(values []string, single string) []string {
	if single != "" {
		values = append(values, single)
	}
	var tags []string
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				tags = append(tags, part)
			}
		}
	}
	return cleanTags(tags)
}

func cleanTags(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		key := strings.ToLower(value)
		if value == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, value)
	}
	return out
}

func tagsJSONMap(tags []string) datatypes.JSONMap {
	tags = cleanTags(tags)
	if len(tags) == 0 {
		return nil
	}
	return datatypes.JSONMap{"labels": tags}
}

func (s Server) validateClassIDs(c *gin.Context, user models.User, classIDs []uint) bool {
	seen := map[uint]bool{}
	for _, classID := range classIDs {
		if classID == 0 || seen[classID] {
			continue
		}
		seen[classID] = true
		if !s.canManageClass(user, classID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "cannot publish to class"})
			return false
		}
	}
	return true
}

func (s Server) parseProblemClassIDs(c *gin.Context, user models.User) ([]uint, bool) {
	raw := append([]string{}, c.PostFormArray("class_ids")...)
	if single := c.PostForm("class_ids"); single != "" {
		raw = append(raw, strings.Split(single, ",")...)
	}
	var ids []uint
	seen := map[uint]bool{}
	for _, item := range raw {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		id, err := strconv.ParseUint(item, 10, 64)
		if err != nil || id == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid class_ids"})
			return nil, false
		}
		classID := uint(id)
		if seen[classID] {
			continue
		}
		seen[classID] = true
		ids = append(ids, classID)
	}
	if !s.validateClassIDs(c, user, ids) {
		return nil, false
	}
	return ids, true
}

func validLanguage(language string) bool {
	switch language {
	case "c", "cpp", "python", "java":
		return true
	default:
		return false
	}
}

func validRole(role models.Role) bool {
	switch role {
	case models.RoleStudent, models.RoleTeacher, models.RoleAdmin:
		return true
	default:
		return false
	}
}

func userPublicColumns() []string {
	return []string{
		"id",
		"email",
		"name",
		"role",
		"student_no",
		"avatar_url",
		"email_verified",
		"account_deleted",
		"created_at",
		"updated_at",
	}
}

func validEmail(email string) bool {
	if email == "" {
		return false
	}
	address, err := mail.ParseAddress(email)
	return err == nil && strings.EqualFold(address.Address, email)
}

func randomPassword() (string, error) {
	raw := make([]byte, 18)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func terminal(status models.SubmissionStatus) bool {
	switch status {
	case models.StatusQueued, models.StatusRunning:
		return false
	default:
		return true
	}
}

func writeSSE(c *gin.Context, event string, payload any) {
	raw, _ := json.Marshal(payload)
	_, _ = fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, raw)
	c.Writer.Flush()
}
func (s Server) previewCourseByCode(c *gin.Context) {
	code := strings.ToUpper(strings.TrimSpace(c.Query("join_code")))
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "join_code is required"})
		return
	}
	var course models.Course
	if err := s.DB.Where("upper(join_code) = ? AND archived = false", code).First(&course).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "course not found"})
		return
	}
	var teacher models.User
	_ = s.DB.Select("id", "email", "name", "role").Where("id = ? AND account_deleted = false", course.TeacherID).First(&teacher).Error
	c.JSON(http.StatusOK, gin.H{
		"course_id":          course.ID,
		"course_code":        course.Code,
		"course_name":        course.Name,
		"course_description": course.Description,
		"term":               course.Term,
		"teacher_id":         teacher.ID,
		"teacher_name":       teacher.Name,
	})
}

func (s Server) joinCourseByCode(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	var req struct {
		JoinCode string `json:"join_code" binding:"required"`
	}
	if !bind(c, &req) {
		return
	}
	code := strings.ToUpper(strings.TrimSpace(req.JoinCode))
	var course models.Course
	if err := s.DB.Where("upper(join_code) = ? AND archived = false", code).First(&course).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "course not found"})
		return
	}
	if err := s.DB.Where("course_id = ? AND user_id = ?", course.ID, user.ID).FirstOrCreate(&models.CourseMembership{CourseID: course.ID, UserID: user.ID, Role: models.RoleStudent}).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	services.Audit(c, s.DB, "course.join", "course", course.ID, nil)
	c.JSON(http.StatusCreated, gin.H{"joined": true, "course_id": course.ID, "join_code": course.JoinCode})
}

func (s Server) leaveCourse(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	courseID, ok := idParam(c, "id")
	if !ok {
		return
	}
	var course models.Course
	if err := s.DB.First(&course, courseID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "course not found"})
		return
	}
	var count int64
	if err := s.DB.Model(&models.CourseMembership{}).Where("course_id = ? AND user_id = ?", courseID, user.ID).Count(&count).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if count == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "course membership not found"})
		return
	}
	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("course_id = ? AND user_id = ? AND role = ?", courseID, user.ID, models.RoleStudent).Delete(&models.CourseMembership{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ? AND class_id IN (?)", user.ID, tx.Model(&models.Class{}).Select("id").Where("course_id = ?", courseID)).Delete(&models.ClassMembership{}).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	services.Audit(c, s.DB, "course.leave", "course", courseID, nil)
	c.JSON(http.StatusOK, gin.H{"left": true, "course_id": courseID})
}

type courseStudentView struct {
	UserID    uint      `json:"user_id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	StudentNo string    `json:"student_no"`
	AvatarURL string    `json:"avatar_url"`
	ClassName string    `json:"class_name"`
	JoinedAt  time.Time `json:"joined_at"`
}

func (s Server) listCourseStudents(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	courseID, ok := idParam(c, "id")
	if !ok {
		return
	}
	if !s.canManageCourse(user, courseID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	type row struct {
		UserID    uint
		Name      string
		Email     string
		StudentNo string
		AvatarURL string
		JoinedAt  time.Time
		ClassID   *uint
		ClassName string
	}
	var rows []row
	s.DB.Table("course_memberships").
		Select("users.id as user_id, users.name, users.email, users.student_no, users.avatar_url, course_memberships.created_at as joined_at").
		Joins("join users on users.id = course_memberships.user_id").
		Where("course_memberships.course_id = ? AND course_memberships.role = ? AND users.account_deleted = false", courseID, models.RoleStudent).
		Order("course_memberships.created_at desc").
		Scan(&rows)

	// For each student, find their class membership in this course
	var classMemberships []struct {
		UserID    uint
		ClassID   uint
		ClassName string
	}
	s.DB.Table("class_memberships").
		Select("class_memberships.user_id, classes.id as class_id, classes.name as class_name").
		Joins("join classes on classes.id = class_memberships.class_id").
		Where("classes.course_id = ? AND classes.archived = false", courseID).
		Scan(&classMemberships)

	studentClasses := map[uint][]string{}
	for _, cm := range classMemberships {
		studentClasses[cm.UserID] = append(studentClasses[cm.UserID], cm.ClassName)
	}

	out := make([]courseStudentView, 0, len(rows))
	for _, r := range rows {
		names := studentClasses[r.UserID]
		className := ""
		if len(names) > 0 {
			className = strings.Join(names, ", ")
		}
		out = append(out, courseStudentView{
			UserID:    r.UserID,
			Name:      r.Name,
			Email:     r.Email,
			StudentNo: r.StudentNo,
			AvatarURL: r.AvatarURL,
			ClassName: className,
			JoinedAt:  r.JoinedAt,
		})
	}
	c.JSON(http.StatusOK, out)
}
