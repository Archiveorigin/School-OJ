package handlers

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	teamdomain "school-oj/apps/api/internal/domain/team"
	"strconv"
	"strings"
	"time"

	"school-oj/apps/api/internal/middleware"
	"school-oj/apps/api/internal/models"
	"school-oj/apps/api/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Query every unfinished attempt; neither token claims nor a limited recent list
// is a source of authority for the temporary exam permission.
func activeExamInDB(db *gorm.DB, userID uint, now time.Time) (models.Exam, error) {
	var exam models.Exam
	err := db.Model(&models.Exam{}).Select("exams.*").
		Joins("JOIN exam_attempts ON exam_attempts.exam_id = exams.id").
		Where("exam_attempts.user_id = ? AND exam_attempts.finished_at IS NULL AND exams.deleted_at IS NULL", userID).
		Where("exams.review_status = 'approved'").
		Where("(exams.starts_at IS NULL OR exams.starts_at <= ?) AND (exams.ends_at IS NULL OR exams.ends_at > ?)", now, now).
		Order("exam_attempts.created_at, exam_attempts.id").First(&exam).Error
	return exam, err
}

func examAPIPath(path string) string {
	if strings.HasPrefix(path, "/api/v1/") {
		return strings.TrimPrefix(path, "/api/v1")
	}
	return strings.TrimPrefix(path, "/api")
}

func (s Server) enforceExamScope(c *gin.Context) {
	user, authenticated := middleware.CurrentUser(c)
	if !authenticated || user.Role != models.RoleStudent {
		c.Next()
		return
	}
	exam, err := activeExamInDB(s.DB, user.ID, time.Now())
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.Next()
		return
	}
	if err != nil {
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "考试权限暂不可用，请重试", "code": "EXAM_STATE_UNAVAILABLE"})
		return
	}
	allowed, err := s.examRequestAllowed(c, user.ID, exam.ID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "考试权限暂不可用，请重试", "code": "EXAM_STATE_UNAVAILABLE"})
		return
	}
	if !allowed {
		// At most one record per user/exam/route/minute. Never retain request
		// bodies or arbitrary query strings in the access-denial log.
		routeHash := sha256.Sum256([]byte(c.Request.Method + ":" + c.FullPath()))
		eventID := fmt.Sprintf("blocked:%x:%d", routeHash[:16], time.Now().Unix()/60)
		_ = appendExamActivity(s.DB, exam.ID, user.ID, eventID, "access_blocked", c.ClientIP(), nil, datatypes.JSONMap{"route": c.FullPath(), "method": c.Request.Method})
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "考试中仅可访问当前考试", "code": "EXAM_SCOPE_REQUIRED", "exam_id": exam.ID})
		return
	}
	c.Set("active_exam_id", exam.ID)
	c.Next()
}

func (s Server) examRequestAllowed(c *gin.Context, userID, examID uint) (bool, error) {
	path, method := examAPIPath(c.FullPath()), c.Request.Method
	if strings.HasPrefix(path, "/event-papers/:kind/:event_id/") {
		if c.Param("kind") != "exam" || c.Param("event_id") != strconv.FormatUint(uint64(examID), 10) || method != http.MethodGet {
			return false, nil
		}
		return path == "/event-papers/:kind/:event_id/attachments" || path == "/event-papers/:kind/:event_id/attachments/:attachment_id" || path == "/event-papers/:kind/:event_id/problems/:problem_id/assets/*asset_path", nil
	}
	if method == http.MethodGet && (path == "/me" || path == "/me/active-exam") {
		return true, nil
	}
	if strings.HasPrefix(path, "/exams/:id") {
		id, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil || uint(id) != examID {
			return false, nil
		}
		switch path {
		case "/exams/:id/change-tickets":
			return method == http.MethodGet || method == http.MethodPost, nil
		case "/exams/:id/change-tickets/:ticket_id":
			return method == http.MethodGet || method == http.MethodPatch || method == http.MethodDelete, nil
		case "/exams/:id/change-tickets/:ticket_id/attachments/:attachment_id":
			return method == http.MethodGet, nil
		case "/exams/:id/change-tickets/:ticket_id/cancel":
			return method == http.MethodPost, nil
		}
		if method == http.MethodGet {
			return path == "/exams/:id" || path == "/exams/:id/ranking" || path == "/exams/:id/problems/:problem_id/assets/*asset_path", nil
		}
		return method == http.MethodPost && (path == "/exams/:id/enter" || path == "/exams/:id/finish" || path == "/exams/:id/activity"), nil
	}
	if method == http.MethodGet && (path == "/submissions/:id" || path == "/submissions/:id/events") {
		var sub models.Submission
		id, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			return false, nil
		}
		err = s.DB.Select("id", "user_id", "exam_id").First(&sub, id).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return err == nil && sub.UserID == userID && sub.ExamID != nil && *sub.ExamID == examID, err
	}
	if method == http.MethodGet && (path == "/submissions" || path == "/submissions/latest") {
		query := c.Request.URL.Query()
		if len(query["exam_id"]) != 1 || c.Query("exam_id") != strconv.FormatUint(uint64(examID), 10) {
			return false, nil
		}
		for _, key := range []string{"assignment_id", "team_contest_id", "problem_set_id", "visibility", "user_id"} {
			if query.Has(key) {
				return false, nil
			}
		}
		return true, nil // Handlers additionally restrict to this user's rows.
	}
	if method == http.MethodPost && path == "/submissions" {
		body, err := io.ReadAll(io.LimitReader(c.Request.Body, 8*services.MaxSubmissionSourceSize+1))
		if err != nil || len(body) > 8*services.MaxSubmissionSourceSize {
			return false, nil
		}
		c.Request.Body = io.NopCloser(bytes.NewReader(body))
		var req struct {
			ExamID        uint  `json:"exam_id"`
			AssignmentID  *uint `json:"assignment_id"`
			TeamContestID *uint `json:"team_contest_id"`
			ProblemSetID  *uint `json:"problem_set_id"`
		}
		if json.Unmarshal(body, &req) != nil {
			return false, nil
		}
		return req.ExamID == examID && req.AssignmentID == nil && req.TeamContestID == nil && req.ProblemSetID == nil, nil
	}
	return false, nil
}

func (s Server) streamExamAllowed(user *models.User, sub models.Submission) bool {
	var current models.User
	if err := s.DB.Where("account_deleted = false").First(&current, user.ID).Error; err != nil {
		return false
	}
	*user = current
	if !s.canAccessSubmission(*user, sub) {
		return false
	}
	if user.Role != models.RoleStudent {
		return true
	}
	exam, err := activeExamInDB(s.DB, user.ID, time.Now())
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return true
	}
	return err == nil && sub.UserID == user.ID && sub.ExamID != nil && *sub.ExamID == exam.ID
}

func appendExamActivity(db *gorm.DB, examID, userID uint, eventID, event, ip string, clientAt *time.Time, meta datatypes.JSONMap) error {
	return db.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "exam_id"}, {Name: "user_id"}, {Name: "event_id"}}, DoNothing: true}).Create(&models.ExamActivity{
		ExamID: examID, UserID: userID, EventID: eventID, Event: event, IP: ip, ClientAt: clientAt, Meta: meta,
	}).Error
}

func (s Server) enterExam(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	id, ok := idParam(c, "id")
	if !ok {
		return
	}
	var attempt models.ExamAttempt
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		var lockedUser models.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("account_deleted = false").First(&lockedUser, user.ID).Error; err != nil {
			return err
		}
		var exam models.Exam
		if err := tx.Clauses(clause.Locking{Strength: "SHARE"}).Where("deleted_at IS NULL").First(&exam, id).Error; err != nil {
			return err
		}
		scoped := s
		scoped.DB = tx
		if !scoped.canAccessExam(user, exam) {
			return &teamContentMutationError{403, "考试不在你的课程或班级中"}
		}
		now := time.Now()
		if reason, _ := studentExamEntryDecision(exam, now, nil); reason != "" {
			return &teamContentMutationError{403, reason}
		}
		active, err := activeExamInDB(tx, user.ID, now)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err == nil && active.ID != id {
			return &teamContentMutationError{409, "请先完成当前考试"}
		}
		err = tx.Where("exam_id = ? AND user_id = ?", id, user.ID).First(&attempt).Error
		if err == nil {
			if attempt.FinishedAt != nil {
				return &teamContentMutationError{403, "exam has been finished"}
			}
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		attempt = models.ExamAttempt{ExamID: id, UserID: user.ID}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&models.ExamParticipant{ExamID: id, UserID: user.ID}).Error; err != nil {
			return err
		}
		if err := tx.Create(&attempt).Error; err != nil {
			return err
		}
		var login models.AuditLog
		result := tx.Where("action = ? AND resource_type = ? AND resource_id = ?", "auth.login", "user", strconv.FormatUint(uint64(user.ID), 10)).Order("created_at desc, id desc").Limit(1).Find(&login)
		if result.Error != nil {
			return result.Error
		}
		meta := datatypes.JSONMap{}
		if result.RowsAffected > 0 {
			meta["login_at"] = login.CreatedAt
			meta["login_ip"] = login.IP
		}
		return appendExamActivity(tx, id, user.ID, "enter", "enter", c.ClientIP(), nil, meta)
	})
	if err != nil {
		examMutationError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"exam_id": id, "entered_at": attempt.CreatedAt, "active": true})
}

func examMutationError(c *gin.Context, err error) {
	var access *teamdomain.Error
	if errors.As(err, &access) {
		status := http.StatusBadRequest
		switch access.Code {
		case teamdomain.Forbidden, teamdomain.InvalidInvitation:
			status = http.StatusForbidden
		case teamdomain.Conflict:
			status = http.StatusConflict
		case teamdomain.NotFound:
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": access.Message, "code": access.Code})
		return
	}
	var mutation *teamContentMutationError
	if errors.As(err, &mutation) {
		c.JSON(mutation.status, gin.H{"error": mutation.message})
		return
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "record not found"})
		return
	}
	c.JSON(http.StatusServiceUnavailable, gin.H{"error": "操作暂不可用，请重试"})
}

func (s Server) recordExamActivity(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	id, ok := idParam(c, "id")
	if !ok {
		return
	}
	// A finished attempt may flush its bounded pending queue after hand-in.
	var attempt models.ExamAttempt
	if err := s.DB.Where("exam_id = ? AND user_id = ?", id, user.ID).First(&attempt).Error; err != nil {
		c.JSON(403, gin.H{"error": "exam has not been entered"})
		return
	}
	var exam models.Exam
	if err := s.DB.First(&exam, id).Error; err != nil {
		examMutationError(c, err)
		return
	}
	until := time.Now()
	if attempt.FinishedAt != nil && attempt.FinishedAt.Before(until) {
		until = *attempt.FinishedAt
	}
	if exam.EndsAt != nil && exam.EndsAt.Before(until) {
		until = *exam.EndsAt
	}
	var req struct {
		Events []struct {
			ID       string    `json:"id"`
			Event    string    `json:"event"`
			ClientAt time.Time `json:"client_at"`
		} `json:"events"`
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 16<<10)
	if !bind(c, &req) {
		return
	}
	if len(req.Events) == 0 || len(req.Events) > 30 {
		c.JSON(400, gin.H{"error": "events must contain 1-30 entries"})
		return
	}
	for _, event := range req.Events {
		if !validClientExamEvent(event.ID, event.Event, event.ClientAt, attempt.CreatedAt, until) {
			c.JSON(400, gin.H{"error": "invalid exam activity"})
			return
		}
	}
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		for _, event := range req.Events {
			if err := appendExamActivity(tx, id, user.ID, "client:"+event.ID, event.Event, c.ClientIP(), &event.ClientAt, nil); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		examMutationError(c, err)
		return
	}
	c.JSON(200, gin.H{"recorded": true})
}

func validClientExamEvent(id, event string, at, entered, now time.Time) bool {
	if len(id) < 16 || len(id) > 64 {
		return false
	}
	for _, r := range id {
		if !(r >= 'a' && r <= 'z') && !(r >= 'A' && r <= 'Z') && !(r >= '0' && r <= '9') && r != '-' {
			return false
		}
	}
	if event != "window_leave" && event != "window_return" && event != "devtools_shortcut" {
		return false
	}
	// Client clocks are evidence, not authority. Allow moderate skew/offline retry.
	return !at.IsZero() && !at.Before(entered.Add(-24*time.Hour)) && !at.After(now.Add(24*time.Hour))
}

func (s Server) listExamActivity(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	id, ok := idParam(c, "id")
	if !ok {
		return
	}
	var exam models.Exam
	if err := s.DB.First(&exam, id).Error; err != nil {
		examMutationError(c, err)
		return
	}
	if !s.canManageCourse(user, exam.CourseID) {
		c.JSON(403, gin.H{"error": "forbidden"})
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	size, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if size < 1 {
		size = 20
	}
	if size > 100 {
		size = 100
	}
	q := s.DB.Model(&models.ExamActivity{}).Where("exam_id = ?", id)
	if raw := c.Query("user_id"); raw != "" {
		uid, err := strconv.ParseUint(raw, 10, 64)
		if err != nil || uid == 0 {
			c.JSON(400, gin.H{"error": "invalid user_id"})
			return
		}
		q = q.Where("user_id = ?", uid)
	}
	if event := c.Query("event"); event != "" {
		q = q.Where("event = ?", event)
	}
	for _, filter := range []struct{ key, op string }{{"from", ">="}, {"to", "<="}} {
		if raw := c.Query(filter.key); raw != "" {
			at, err := time.Parse(time.RFC3339, raw)
			if err != nil {
				c.JSON(400, gin.H{"error": "invalid time filter"})
				return
			}
			q = q.Where("exam_activities.created_at "+filter.op+" ?", at)
		}
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		examMutationError(c, err)
		return
	}
	var rows []struct {
		models.ExamActivity
		UserName string `json:"user_name"`
	}
	err := q.Select("exam_activities.*, users.name AS user_name").Joins("JOIN users ON users.id = exam_activities.user_id").Order("exam_activities.id desc").Offset((page - 1) * size).Limit(size).Scan(&rows).Error
	if err != nil {
		examMutationError(c, err)
		return
	}
	if rows == nil {
		rows = make([]struct {
			models.ExamActivity
			UserName string `json:"user_name"`
		}, 0)
	}
	c.JSON(200, gin.H{"items": rows, "total": total, "page": page, "page_size": size, "exam_title": exam.Title})
}

func (s Server) getExamProblemAsset(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	id, ok := idParam(c, "id")
	if !ok {
		return
	}
	problemID, ok := idParam(c, "problem_id")
	if !ok {
		return
	}
	var exam models.Exam
	if err := s.DB.Where("deleted_at IS NULL").First(&exam, id).Error; err != nil {
		examMutationError(c, err)
		return
	}
	if !s.canAccessExam(user, exam) {
		c.JSON(403, gin.H{"error": "forbidden"})
		return
	}
	if !s.canReadEventPaper(user, eventPaper{kind: "exam", id: id, exam: exam}, time.Now()) {
		c.JSON(403, gin.H{"error": "本场题目资源尚未开放"})
		return
	}
	if user.Role == models.RoleStudent && exam.Lifecycle(exam.StartsAt, exam.EndsAt, time.Now()) == "running" {
		active, err := activeExamInDB(s.DB, user.ID, time.Now())
		if err != nil || active.ID != id {
			c.JSON(403, gin.H{"error": "exam is not active"})
			return
		}
	}
	var link models.ExamProblem
	if err := s.DB.Preload("ProblemVersion").Where("exam_id = ? AND problem_id = ?", id, problemID).First(&link).Error; err != nil {
		examMutationError(c, err)
		return
	}
	asset, err := services.NormalizeAssetPath(strings.TrimPrefix(c.Param("asset_path"), "/"))
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid asset path"})
		return
	}
	object, contentType := problemAssetObject(link.ProblemVersion.Manifest, asset)
	// Only images referenced by the statement are available to candidates.
	statement, decodeErr := url.PathUnescape(link.ProblemVersion.Statement)
	if decodeErr != nil {
		statement = link.ProblemVersion.Statement
	}
	if object == "" || !strings.Contains(statement, asset) {
		c.JSON(404, gin.H{"error": "asset not found"})
		return
	}
	if s.MinIO == nil {
		c.JSON(503, gin.H{"error": "asset unavailable"})
		return
	}
	src, err := s.MinIO.GetObject(c.Request.Context(), s.Cfg.MinIOBucket, object, minio.GetObjectOptions{})
	if err != nil {
		c.JSON(503, gin.H{"error": "asset unavailable"})
		return
	}
	defer src.Close()
	stat, err := src.Stat()
	if err != nil {
		c.JSON(404, gin.H{"error": "asset not found"})
		return
	}
	if contentType == "" {
		contentType = stat.ContentType
	}
	c.Header("Cache-Control", "private, no-store")
	c.DataFromReader(200, stat.Size, contentType, src, nil)
}
