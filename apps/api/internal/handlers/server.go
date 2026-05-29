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
	auth.GET("/problems", s.listProblems)
	auth.POST("/problems", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.createProblem)
	auth.POST("/problems/upload", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.uploadProblem)
	auth.GET("/problems/:id", s.getProblem)
	auth.GET("/assignments", s.listAssignments)
	auth.POST("/assignments", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.createAssignment)
	auth.GET("/exams", s.listExams)
	auth.POST("/exams", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.createExam)
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
	var courses []models.Course
	s.DB.Order("id desc").Find(&courses)
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
	if err := s.DB.Create(&req).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_ = s.DB.Create(&models.CourseMembership{CourseID: req.ID, UserID: req.TeacherID, Role: models.RoleTeacher}).Error
	services.Audit(c, s.DB, "course.create", "course", req.ID, nil)
	c.JSON(http.StatusCreated, req)
}

func (s Server) createClass(c *gin.Context) {
	courseID, ok := idParam(c, "id")
	if !ok {
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
	services.Audit(c, s.DB, "class.create", "class", class.ID, datatypes.JSONMap{"course_id": courseID})
	c.JSON(http.StatusCreated, class)
}

func (s Server) addCourseMember(c *gin.Context) {
	courseID, ok := idParam(c, "id")
	if !ok {
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
		_ = s.DB.Where("class_id = ? AND user_id = ?", *req.ClassID, req.UserID).FirstOrCreate(&models.ClassMembership{ClassID: *req.ClassID, UserID: req.UserID}).Error
	}
	services.Audit(c, s.DB, "course.member.add", "course", courseID, datatypes.JSONMap{"user_id": req.UserID})
	c.JSON(http.StatusCreated, member)
}

func (s Server) listClasses(c *gin.Context) {
	var classes []models.Class
	q := s.DB.Order("id desc")
	if courseID := c.Query("course_id"); courseID != "" {
		q = q.Where("course_id = ?", courseID)
	}
	q.Find(&classes)
	c.JSON(http.StatusOK, classes)
}

func (s Server) listProblems(c *gin.Context) {
	var problems []models.Problem
	s.DB.Order("id desc").Find(&problems)
	c.JSON(http.StatusOK, problems)
}

func (s Server) uploadProblem(c *gin.Context) {
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
	problem, ok := s.saveProblemPackage(c, user, body, pkg, "problem.upload")
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
	body, pkg, err := services.BuildProblemPackage(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	problem, ok := s.saveProblemPackage(c, user, body, pkg, "problem.create")
	if !ok {
		return
	}
	c.JSON(http.StatusCreated, problem)
}

func (s Server) saveProblemPackage(c *gin.Context, user models.User, body []byte, pkg services.ParsedProblemPackage, action string) (models.Problem, bool) {
	object := fmt.Sprintf("problems/%s/%d.zip", pkg.Manifest.Slug, time.Now().UnixNano())
	if _, err := s.MinIO.PutObject(c.Request.Context(), s.Cfg.MinIOBucket, object, bytes.NewReader(body), int64(len(body)), minio.PutObjectOptions{ContentType: "application/zip"}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return models.Problem{}, false
	}
	manifestJSON, _ := json.Marshal(pkg.Manifest)
	var manifest datatypes.JSONMap
	_ = json.Unmarshal(manifestJSON, &manifest)
	problem := models.Problem{
		OwnerID:         user.ID,
		Slug:            pkg.Manifest.Slug,
		Title:           pkg.Manifest.Title,
		Statement:       pkg.Manifest.Statement,
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
	services.Audit(c, s.DB, action, "problem", problem.ID, datatypes.JSONMap{"slug": problem.Slug})
	return problem, true
}

func (s Server) getProblem(c *gin.Context) {
	id, ok := idParam(c, "id")
	if !ok {
		return
	}
	var problem models.Problem
	if err := s.DB.First(&problem, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "problem not found"})
		return
	}
	c.JSON(http.StatusOK, problem)
}

func (s Server) listAssignments(c *gin.Context) {
	var items []models.Assignment
	q := s.DB.Preload("Problems").Order("id desc")
	if courseID := c.Query("course_id"); courseID != "" {
		q = q.Where("course_id = ?", courseID)
	}
	q.Find(&items)
	c.JSON(http.StatusOK, items)
}

func (s Server) createAssignment(c *gin.Context) {
	var req struct {
		CourseID    uint       `json:"course_id" binding:"required"`
		Title       string     `json:"title" binding:"required"`
		Description string     `json:"description"`
		StartsAt    *time.Time `json:"starts_at"`
		DueAt       *time.Time `json:"due_at"`
		ProblemIDs  []uint     `json:"problem_ids"`
	}
	if !bind(c, &req) {
		return
	}
	item := models.Assignment{CourseID: req.CourseID, Title: req.Title, Description: req.Description, StartsAt: req.StartsAt, DueAt: req.DueAt}
	if err := s.DB.Create(&item).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	for i, pid := range req.ProblemIDs {
		_ = s.DB.Create(&models.AssignmentProblem{AssignmentID: item.ID, ProblemID: pid, Score: 100, SortOrder: i}).Error
	}
	services.Audit(c, s.DB, "assignment.create", "assignment", item.ID, nil)
	c.JSON(http.StatusCreated, item)
}

func (s Server) listExams(c *gin.Context) {
	var items []models.Exam
	q := s.DB.Preload("Problems").Order("id desc")
	if courseID := c.Query("course_id"); courseID != "" {
		q = q.Where("course_id = ?", courseID)
	}
	q.Find(&items)
	c.JSON(http.StatusOK, items)
}

func (s Server) createExam(c *gin.Context) {
	var req struct {
		CourseID    uint       `json:"course_id" binding:"required"`
		Title       string     `json:"title" binding:"required"`
		Description string     `json:"description"`
		StartsAt    *time.Time `json:"starts_at"`
		EndsAt      *time.Time `json:"ends_at"`
		ProblemIDs  []uint     `json:"problem_ids"`
	}
	if !bind(c, &req) {
		return
	}
	item := models.Exam{CourseID: req.CourseID, Title: req.Title, Description: req.Description, StartsAt: req.StartsAt, EndsAt: req.EndsAt}
	if err := s.DB.Create(&item).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	for i, pid := range req.ProblemIDs {
		_ = s.DB.Create(&models.ExamProblem{ExamID: item.ID, ProblemID: pid, Score: 100, SortOrder: i}).Error
	}
	services.Audit(c, s.DB, "exam.create", "exam", item.ID, nil)
	c.JSON(http.StatusCreated, item)
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
	sub := models.Submission{
		UserID:       user.ID,
		ProblemID:    req.ProblemID,
		AssignmentID: req.AssignmentID,
		ExamID:       req.ExamID,
		Language:     req.Language,
		SourceCode:   req.SourceCode,
		Status:       models.StatusQueued,
	}
	if err := s.DB.Create(&sub).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
	if user.Role == models.RoleStudent && sub.UserID != user.ID {
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
		if user.Role == models.RoleStudent && sub.UserID != user.ID {
			writeSSE(c, "error", gin.H{"error": "forbidden"})
			return
		}
		payload := gin.H{"id": sub.ID, "status": sub.Status, "score": sub.Score, "time_ms": sub.TimeMS, "memory_kb": sub.MemoryKB, "message": sub.Message, "updated_at": sub.UpdatedAt}
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
	type row struct {
		UserID         uint      `json:"user_id"`
		Name           string    `json:"name"`
		Solved         int       `json:"solved"`
		Score          int       `json:"score"`
		LastSubmission time.Time `json:"last_submission"`
	}
	var rows []row
	q := s.DB.Table("submissions").
		Select("users.id as user_id, users.name as name, count(distinct submissions.problem_id) as solved, coalesce(sum(submissions.score),0) as score, max(submissions.updated_at) as last_submission").
		Joins("join users on users.id = submissions.user_id").
		Where("submissions.status = ?", models.StatusAccepted).
		Group("users.id, users.name").
		Order("solved desc, score desc, last_submission asc")
	if assignmentID := c.Query("assignment_id"); assignmentID != "" {
		q = q.Where("submissions.assignment_id = ?", assignmentID)
	}
	if examID := c.Query("exam_id"); examID != "" {
		q = q.Where("submissions.exam_id = ?", examID)
	}
	q.Scan(&rows)
	c.JSON(http.StatusOK, rows)
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

func idParam(c *gin.Context, name string) (uint, bool) {
	raw := c.Param(name)
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return 0, false
	}
	return uint(id), true
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
