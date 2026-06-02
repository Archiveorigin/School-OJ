package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

type workProblemInput struct {
	ProblemID uint `json:"problem_id"`
	Score     int  `json:"score"`
}

type problemScoreView struct {
	Problem          models.Problem          `json:"problem"`
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
	auth.GET("/profile", s.getProfile)
	auth.PUT("/profile", s.updateProfile)
	auth.POST("/profile/email-code", s.sendProfileEmailCode)
	auth.POST("/profile/email", s.rebindEmail)
	auth.DELETE("/profile", s.deleteProfile)
	auth.POST("/feedback", s.createFeedback)
	auth.GET("/courses", s.listCourses)
	auth.POST("/courses", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.createCourse)
	auth.POST("/courses/:id/classes", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.createClass)
	auth.POST("/courses/:id/members", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.addCourseMember)
	auth.GET("/classes", s.listClasses)
	auth.GET("/me/classes", s.myClasses)
	auth.POST("/classes/:id/join", middleware.RequireRoles(models.RoleStudent), s.joinClass)
	auth.POST("/classes/:id/leave", middleware.RequireRoles(models.RoleStudent), s.leaveClass)
	auth.GET("/problems", s.listProblems)
	auth.POST("/problems", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.createProblem)
	auth.POST("/problems/upload", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.uploadProblem)
	auth.GET("/problems/:id/assets/*asset_path", s.getProblemAsset)
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
	auth.POST("/exams", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.createExam)
	auth.GET("/exams/:id/report", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.examReport)
	auth.DELETE("/exams/:id", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.deleteExam)
	auth.POST("/exams/:id/submissions/:submission_id/judge", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.judgeManualExamSubmission)
	auth.PUT("/exams/:id/submissions/:submission_id/grade", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.gradeManualExamSubmission)
	auth.POST("/submissions", s.createSubmission)
	auth.GET("/submissions", s.listSubmissions)
	auth.GET("/submissions/:id", s.getSubmission)
	auth.GET("/submissions/:id/events", s.submissionEvents)
	auth.GET("/leaderboard", s.leaderboard)
	auth.GET("/plagiarism/jobs", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.listPlagiarismJobs)
	auth.POST("/plagiarism/jobs", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.createPlagiarismJob)
	auth.GET("/audit-logs", middleware.RequireRoles(models.RoleAdmin), s.listAuditLogs)
	auth.GET("/users", middleware.RequireRoles(models.RoleAdmin), s.listUsers)
	auth.POST("/users", middleware.RequireRoles(models.RoleAdmin), s.createUser)
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

func (s Server) listUsers(c *gin.Context) {
	var users []models.User
	s.DB.Where("account_deleted = false").Order("id asc").Find(&users)
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
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Name = strings.TrimSpace(req.Name)
	req.StudentNo = strings.TrimSpace(req.StudentNo)
	hash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	user := models.User{Email: req.Email, Name: req.Name, Role: req.Role, StudentNo: req.StudentNo, PasswordHash: string(hash), EmailVerified: true}
	if err := s.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	services.Audit(c, s.DB, "user.create", "user", user.ID, nil)
	c.JSON(http.StatusCreated, user)
}

func (s Server) listCourses(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	var courses []models.Course
	q := s.DB.Order("id desc")
	switch user.Role {
	case models.RoleAdmin:
	case models.RoleTeacher:
		q = q.Where("teacher_id = ? OR id IN (?)", user.ID, s.DB.Model(&models.CourseMembership{}).Select("course_id").Where("user_id = ? AND role IN ?", user.ID, []models.Role{models.RoleTeacher, models.RoleAdmin}))
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
	if req.TeacherID == 0 {
		req.TeacherID = user.ID
	}
	if user.Role == models.RoleTeacher {
		req.TeacherID = user.ID
	}
	if err := s.DB.Create(&req).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_ = s.DB.Create(&models.CourseMembership{CourseID: req.ID, UserID: req.TeacherID, Role: models.RoleTeacher}).Error
	services.Audit(c, s.DB, "course.create", "course", req.ID, nil)
	c.JSON(http.StatusCreated, req)
}

func (s Server) createClass(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	courseID, ok := idParam(c, "id")
	if !ok {
		return
	}
	if !s.canManageCourse(user, courseID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if !bind(c, &req) {
		return
	}
	class := models.Class{CourseID: courseID, Name: req.Name}
	if err := s.DB.Create(&class).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_ = s.DB.Where("class_id = ? AND user_id = ?", class.ID, user.ID).FirstOrCreate(&models.ClassMembership{ClassID: class.ID, UserID: user.ID}).Error
	services.Audit(c, s.DB, "class.create", "class", class.ID, datatypes.JSONMap{"course_id": courseID})
	c.JSON(http.StatusCreated, class)
}

func (s Server) addCourseMember(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	courseID, ok := idParam(c, "id")
	if !ok {
		return
	}
	if !s.canManageCourse(user, courseID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	var req struct {
		UserID  uint        `json:"user_id" binding:"required"`
		Role    models.Role `json:"role" binding:"required"`
		ClassID *uint       `json:"class_id"`
	}
	if !bind(c, &req) {
		return
	}
	member := models.CourseMembership{CourseID: courseID, UserID: req.UserID, Role: req.Role}
	if err := s.DB.Where("course_id = ? AND user_id = ?", courseID, req.UserID).FirstOrCreate(&member).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.ClassID != nil {
		var class models.Class
		if err := s.DB.First(&class, *req.ClassID).Error; err != nil || class.CourseID != courseID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "class not found in course"})
			return
		}
		_ = s.DB.Where("class_id = ? AND user_id = ?", *req.ClassID, req.UserID).FirstOrCreate(&models.ClassMembership{ClassID: *req.ClassID, UserID: req.UserID}).Error
	}
	services.Audit(c, s.DB, "course.member.add", "course", courseID, datatypes.JSONMap{"user_id": req.UserID})
	c.JSON(http.StatusCreated, member)
}

func (s Server) listClasses(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	classes := []models.Class{}
	q := s.DB.Order("id desc")
	if courseID := c.Query("course_id"); courseID != "" {
		q = q.Where("course_id = ?", courseID)
	}
	switch user.Role {
	case models.RoleAdmin:
	case models.RoleTeacher:
		q = q.Where("course_id IN (?) OR course_id IN (?)",
			s.DB.Model(&models.Course{}).Select("id").Where("teacher_id = ?", user.ID),
			s.DB.Model(&models.CourseMembership{}).Select("course_id").Where("user_id = ? AND role IN ?", user.ID, []models.Role{models.RoleTeacher, models.RoleAdmin}),
		)
	default:
		q = q.Where("id IN (?)", s.DB.Model(&models.ClassMembership{}).Select("class_id").Where("user_id = ?", user.ID))
	}
	q.Find(&classes)
	c.JSON(http.StatusOK, classes)
}

func (s Server) myClasses(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	type classView struct {
		ID         uint   `json:"id"`
		ClassID    uint   `json:"class_id"`
		ClassName  string `json:"class_name"`
		CourseID   uint   `json:"course_id"`
		CourseCode string `json:"course_code"`
		CourseName string `json:"course_name"`
		Term       string `json:"term"`
	}
	rows := []classView{}
	q := s.DB.Table("classes").
		Select("classes.id as id, classes.id as class_id, classes.name as class_name, courses.id as course_id, courses.code as course_code, courses.name as course_name, courses.term as term").
		Joins("join courses on courses.id = classes.course_id").
		Order("courses.id desc, classes.id desc")
	switch user.Role {
	case models.RoleAdmin:
	case models.RoleTeacher:
		q = q.Where("courses.teacher_id = ? OR courses.id IN (?)", user.ID, s.DB.Model(&models.CourseMembership{}).Select("course_id").Where("user_id = ? AND role IN ?", user.ID, []models.Role{models.RoleTeacher, models.RoleAdmin}))
	default:
		q = q.Joins("join class_memberships on class_memberships.class_id = classes.id").Where("class_memberships.user_id = ?", user.ID)
	}
	q.Scan(&rows)
	c.JSON(http.StatusOK, rows)
}

func (s Server) joinClass(c *gin.Context) {
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
	if err := s.DB.Where("class_id = ? AND user_id = ?", classID, user.ID).FirstOrCreate(&models.ClassMembership{ClassID: classID, UserID: user.ID}).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_ = s.DB.Where("course_id = ? AND user_id = ?", class.CourseID, user.ID).FirstOrCreate(&models.CourseMembership{CourseID: class.CourseID, UserID: user.ID, Role: models.RoleStudent}).Error
	services.Audit(c, s.DB, "class.join", "class", classID, nil)
	c.JSON(http.StatusCreated, gin.H{"joined": true, "class_id": classID})
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
	if s.blockedByActiveLockedExam(c, user, nil) {
		return
	}
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
	body, err := io.ReadAll(io.LimitReader(src, 128<<20))
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

func (s Server) saveProblemPackage(c *gin.Context, user models.User, body []byte, pkg services.ParsedProblemPackage, classIDs []uint, releaseAt *time.Time, tags datatypes.JSONMap, action string) (models.Problem, bool) {
	baseObject := fmt.Sprintf("problems/%s/%d", pkg.Manifest.Slug, time.Now().UnixNano())
	object := baseObject + ".zip"
	if _, err := s.MinIO.PutObject(c.Request.Context(), s.Cfg.MinIOBucket, object, bytes.NewReader(body), int64(len(body)), minio.PutObjectOptions{ContentType: "application/zip"}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return models.Problem{}, false
	}
	for _, asset := range pkg.Assets {
		assetObject := fmt.Sprintf("%s/%s", baseObject, asset.Path)
		if _, err := s.MinIO.PutObject(c.Request.Context(), s.Cfg.MinIOBucket, assetObject, bytes.NewReader(asset.Body), asset.Size, minio.PutObjectOptions{ContentType: asset.ContentType}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return models.Problem{}, false
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
	problem := models.Problem{
		OwnerID:         user.ID,
		Slug:            pkg.Manifest.Slug,
		Title:           pkg.Manifest.Title,
		Statement:       pkg.Manifest.Statement,
		Tags:            tags,
		TimeLimitMS:     pkg.Manifest.TimeLimitMS,
		MemoryLimitMB:   pkg.Manifest.MemoryLimitMB,
		OutputLimitKB:   pkg.Manifest.OutputLimitKB,
		PackageObject:   object,
		PackageChecksum: pkg.SHA256,
		Manifest:        manifest,
	}
	if err := s.DB.Create(&problem).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return models.Problem{}, false
	}
	for _, classID := range classIDs {
		if err := s.linkProblemToClass(s.DB, classID, problem.ID, releaseAt); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return models.Problem{}, false
		}
	}
	services.Audit(c, s.DB, action, "problem", problem.ID, datatypes.JSONMap{"slug": problem.Slug})
	return problem, true
}

func (s Server) getProblem(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	if s.blockedByActiveLockedExam(c, user, nil) {
		return
	}
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
	body, err := io.ReadAll(io.LimitReader(src, 128<<20))
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
	if s.blockedByActiveLockedExam(c, user, nil) {
		return
	}
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
				s.DB.Model(&models.CourseMembership{}).Select("course_id").Where("user_id = ? AND role IN ?", user.ID, []models.Role{models.RoleTeacher, models.RoleAdmin}),
			)
		default:
			q = q.Where("class_id IN (?)", s.DB.Model(&models.ClassMembership{}).Select("class_id").Where("user_id = ?", user.ID))
		}
	}
	q.Find(&items)
	if user.Role != models.RoleStudent {
		c.JSON(http.StatusOK, items)
		return
	}
	type assignmentListView struct {
		models.Assignment
		WorkStatus string `json:"work_status"`
		TotalScore int    `json:"total_score"`
		MaxScore   int    `json:"max_score"`
		ScoreReady bool   `json:"score_ready"`
	}
	views := make([]assignmentListView, 0, len(items))
	for _, item := range items {
		summary := s.assignmentSummary(item.ID, user.ID, false)
		views = append(views, assignmentListView{
			Assignment: item,
			WorkStatus: summary.WorkStatus,
			TotalScore: summary.TotalScore,
			MaxScore:   summary.MaxScore,
			ScoreReady: summary.ScoreReady,
		})
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
	if s.blockedByActiveLockedExam(c, user, nil) {
		return
	}
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
				s.DB.Model(&models.CourseMembership{}).Select("course_id").Where("user_id = ? AND role IN ?", user.ID, []models.Role{models.RoleTeacher, models.RoleAdmin}),
			)
		default:
			q = q.Where("class_id IN (?)", s.DB.Model(&models.ClassMembership{}).Select("class_id").Where("user_id = ?", user.ID))
		}
	}
	q.Find(&items)
	if user.Role != models.RoleStudent {
		c.JSON(http.StatusOK, items)
		return
	}
	type examListView struct {
		models.Exam
		WorkStatus string `json:"work_status"`
		TotalScore int    `json:"total_score"`
		MaxScore   int    `json:"max_score"`
		ScoreReady bool   `json:"score_ready"`
	}
	views := make([]examListView, 0, len(items))
	for _, item := range items {
		summary := s.examSummary(item.ID, user.ID, false)
		views = append(views, examListView{
			Exam:       item,
			WorkStatus: summary.WorkStatus,
			TotalScore: summary.TotalScore,
			MaxScore:   summary.MaxScore,
			ScoreReady: summary.ScoreReady,
		})
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
			if err := tx.Create(&models.ExamProblem{ExamID: item.ID, ProblemID: problemItem.ProblemID, Score: problemItem.Score, SortOrder: i}).Error; err != nil {
				return err
			}
			if _, isPrepared := prepared[problemItem.ProblemID]; isPrepared && req.ClassID != nil {
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
	if user.Role == models.RoleStudent {
		_ = s.recordExamAttempt(item.ID, user.ID)
	}
	summary := s.examSummary(item.ID, user.ID, true)
	allSubmitted := user.Role == models.RoleStudent && s.examAllSubmitted(item.ID, user.ID)
	c.JSON(http.StatusOK, gin.H{
		"exam":           item,
		"problems":       examProblemViews(item),
		"now":            now,
		"closed":         closed,
		"not_started":    item.StartsAt != nil && now.Before(*item.StartsAt),
		"can_submit":     user.Role != models.RoleStudent || !closed,
		"manual_review":  examManualReview(item),
		"lock_exit":      examLockExit(item),
		"all_submitted":  allSubmitted,
		"work_status":    summary.WorkStatus,
		"total_score":    summary.TotalScore,
		"max_score":      summary.MaxScore,
		"score_ready":    summary.ScoreReady,
		"problem_scores": summary.Problems,
	})
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
	if req.AssignmentID != nil && req.ExamID != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "assignment_id and exam_id cannot both be set"})
		return
	}
	if s.blockedByActiveLockedExam(c, user, req.ExamID) {
		return
	}
	if user.Role == models.RoleStudent {
		if req.AssignmentID != nil {
			if ok, msg := s.canStudentSubmitAssignment(user.ID, *req.AssignmentID, req.ProblemID); !ok {
				c.JSON(http.StatusForbidden, gin.H{"error": msg})
				return
			}
			_ = s.recordAssignmentAttempt(*req.AssignmentID, user.ID)
		} else if req.ExamID != nil {
			if ok, msg := s.canStudentSubmitExam(user.ID, *req.ExamID, req.ProblemID); !ok {
				c.JSON(http.StatusForbidden, gin.H{"error": msg})
				return
			}
			_ = s.recordExamAttempt(*req.ExamID, user.ID)
		} else if !s.canStudentAccessProblem(user.ID, req.ProblemID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "problem is not available in your classes"})
			return
		}
	}
	status := models.StatusQueued
	manualReview := false
	if req.ExamID != nil {
		var exam models.Exam
		if err := s.DB.Where("deleted_at IS NULL").First(&exam, *req.ExamID).Error; err == nil && examManualReview(exam) {
			manualReview = true
			status = models.StatusPendingReview
		}
	}
	sub := models.Submission{
		UserID:       user.ID,
		ProblemID:    req.ProblemID,
		AssignmentID: req.AssignmentID,
		ExamID:       req.ExamID,
		Language:     req.Language,
		SourceCode:   req.SourceCode,
		Status:       status,
	}
	if err := s.DB.Create(&sub).Error; err != nil {
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
	}
	if problemID := c.Query("problem_id"); problemID != "" {
		q = q.Where("problem_id = ?", problemID)
	}
	if assignmentID := c.Query("assignment_id"); assignmentID != "" {
		q = q.Where("assignment_id = ?", assignmentID)
	}
	if examID := c.Query("exam_id"); examID != "" {
		q = q.Where("exam_id = ?", examID)
	}
	q.Find(&items)
	c.JSON(http.StatusOK, items)
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
	s.DB.Where("submission_id = ?", sub.ID).Order("id asc").Find(&results)
	c.JSON(http.StatusOK, gin.H{"submission": sub, "results": results})
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
		rows := s.classLeaderboardRows(classID)
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
			Rows:       s.classLeaderboardRows(classID),
		})
	}
	c.JSON(http.StatusOK, groups)
}

func (s Server) classLeaderboardRows(classID uint) []leaderboardRow {
	var rows []leaderboardRow
	s.DB.Table("class_memberships").
		Select("users.id as user_id, users.name as name, count(distinct case when problem_progresses.status = ? then problem_progresses.problem_id end) as solved, coalesce(sum(case when problem_progresses.status = ? then problem_progresses.points else 0 end), 0) as score, max(problem_progresses.last_submitted) as last_submission", models.ProgressAccepted, models.ProgressAccepted).
		Joins("join users on users.id = class_memberships.user_id and users.role = ?", models.RoleStudent).
		Joins("left join class_problems on class_problems.class_id = class_memberships.class_id and "+releasedClassProblemSQL(), time.Now()).
		Joins("left join problem_progresses on problem_progresses.user_id = users.id and problem_progresses.problem_id = class_problems.problem_id").
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
	c.JSON(http.StatusOK, jobs)
}

func (s Server) listAuditLogs(c *gin.Context) {
	var logs []models.AuditLog
	s.DB.Order("id desc").Limit(200).Find(&logs)
	c.JSON(http.StatusOK, logs)
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
	s.DB.Model(&models.CourseMembership{}).Where("course_id = ? AND user_id = ? AND role IN ?", courseID, user.ID, []models.Role{models.RoleTeacher, models.RoleAdmin}).Count(&count)
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
		s.DB.Model(&models.Class{}).Order("id desc").Pluck("id", &ids)
	case models.RoleTeacher:
		s.DB.Model(&models.Class{}).
			Where("course_id IN (?) OR course_id IN (?)",
				s.DB.Model(&models.Course{}).Select("id").Where("teacher_id = ?", user.ID),
				s.DB.Model(&models.CourseMembership{}).Select("course_id").Where("user_id = ? AND role IN ?", user.ID, []models.Role{models.RoleTeacher, models.RoleAdmin}),
			).
			Order("id desc").Pluck("id", &ids)
	default:
		s.DB.Model(&models.ClassMembership{}).Where("user_id = ?", user.ID).Order("class_id desc").Pluck("class_id", &ids)
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
	if exam, ok := s.activeLockedExamForStudent(user.ID); ok {
		return s.examContainsProblem(exam.ID, problem.ID)
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

func (s Server) activeLockedExamForStudent(userID uint) (models.Exam, bool) {
	var attempts []models.ExamAttempt
	if err := s.DB.Where("user_id = ?", userID).Order("updated_at desc, id desc").Limit(20).Find(&attempts).Error; err != nil {
		return models.Exam{}, false
	}
	now := time.Now()
	for _, attempt := range attempts {
		var exam models.Exam
		if err := s.DB.Where("deleted_at IS NULL").First(&exam, attempt.ExamID).Error; err != nil {
			continue
		}
		if !examLockExit(exam) {
			continue
		}
		if exam.EndsAt != nil && now.After(*exam.EndsAt) {
			continue
		}
		if s.examAllSubmitted(exam.ID, userID) {
			continue
		}
		return exam, true
	}
	return models.Exam{}, false
}

func (s Server) blockedByActiveLockedExam(c *gin.Context, user models.User, allowedExamID *uint) bool {
	if user.Role != models.RoleStudent {
		return false
	}
	exam, ok := s.activeLockedExamForStudent(user.ID)
	if !ok {
		return false
	}
	if allowedExamID != nil && *allowedExamID == exam.ID {
		return false
	}
	c.JSON(http.StatusLocked, gin.H{
		"error":   "locked exam is active; finish all exam problems before leaving",
		"exam_id": exam.ID,
		"title":   exam.Title,
	})
	return true
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
	if exam.EndsAt != nil && time.Now().After(*exam.EndsAt) {
		return false, "exam is closed"
	}
	return true, ""
}

func (s Server) studentInClass(userID uint, classID uint) bool {
	var count int64
	s.DB.Model(&models.ClassMembership{}).Where("class_id = ? AND user_id = ?", classID, userID).Count(&count)
	return count > 0
}

func (s Server) canAccessAssignment(user models.User, assignment models.Assignment) bool {
	if user.Role == models.RoleAdmin {
		return true
	}
	if user.Role == models.RoleTeacher {
		return s.canManageCourse(user, assignment.CourseID)
	}
	return assignment.ClassID != nil && s.studentInClass(user.ID, *assignment.ClassID)
}

func (s Server) canAccessExam(user models.User, exam models.Exam) bool {
	if user.Role == models.RoleAdmin {
		return true
	}
	if user.Role == models.RoleTeacher {
		return s.canManageCourse(user, exam.CourseID)
	}
	return exam.ClassID != nil && s.studentInClass(user.ID, *exam.ClassID)
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
		problems = append(problems, gin.H{"problem": link.Problem, "score": link.Score, "problem_id": link.ProblemID})
	}
	return problems
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
		Score     int
	}
	links := []linkInfo{}
	for _, link := range assignmentLinks {
		links = append(links, linkInfo{Problem: link.Problem, ProblemID: link.ProblemID, Score: link.Score})
	}
	for _, link := range examLinks {
		links = append(links, linkInfo{Problem: link.Problem, ProblemID: link.ProblemID, Score: link.Score})
	}
	hasSubmission := false
	hasPending := false
	for _, link := range links {
		summary.MaxScore += link.Score
		view, submitted, pending := s.problemScore(userID, link.ProblemID, link.Problem, link.Score, assignmentID, examID, manualReview)
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
			c.JSON(http.StatusBadRequest, gin.H{"error": contextName + " with prepared problems requires a class"})
			return nil, false
		}
		if releaseAt == nil {
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
