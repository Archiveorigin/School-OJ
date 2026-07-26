package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"school-oj/apps/api/internal/middleware"
	"school-oj/apps/api/internal/models"
	"school-oj/apps/api/internal/services"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var errEmailNotFound = errors.New("email not found")

func (s Server) sendEmailCode(c *gin.Context) {
	var req struct {
		Email   string `json:"email" binding:"required"`
		Purpose string `json:"purpose" binding:"required"`
	}
	if !bind(c, &req) {
		return
	}
	email := normalizeEmail(req.Email)
	if !validEmail(email) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid email"})
		return
	}
	if !publicVerificationPurpose(req.Purpose) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid purpose"})
		return
	}
	if req.Purpose == services.VerificationRegister {
		var count int64
		s.DB.Model(&models.User{}).Where("email = ? AND account_deleted = false", email).Count(&count)
		if count > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "email already registered"})
			return
		}
	}
	if req.Purpose == services.VerificationPasswordReset {
		var count int64
		s.DB.Model(&models.User{}).Where("email = ? AND account_deleted = false", email).Count(&count)
		if count == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "email not found"})
			return
		}
	}
	if err := s.sendCode(c, email, req.Purpose); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"sent": true})
}

func (s Server) register(c *gin.Context) {
	var req struct {
		Email     string `json:"email" binding:"required"`
		Code      string `json:"code" binding:"required"`
		Name      string `json:"name" binding:"required"`
		Password  string `json:"password" binding:"required"`
		StudentNo string `json:"student_no"`
	}
	if !bind(c, &req) {
		return
	}
	email := normalizeEmail(req.Email)
	name := strings.TrimSpace(req.Name)
	studentNo := strings.TrimSpace(req.StudentNo)
	if !validEmail(email) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid email"})
		return
	}
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
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
	user := models.User{
		Email:         email,
		Name:          name,
		Role:          models.RoleStudent,
		PasswordHash:  string(hash),
		StudentNo:     studentNo,
		EmailVerified: true,
	}
	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := services.ConsumeVerification(tx, email, services.VerificationRegister, req.Code); err != nil {
			return err
		}
		return tx.Create(&user).Error
	}); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	services.Audit(c, s.DB, "auth.register", "user", user.ID, nil)
	token, err := middleware.SignToken(s.Cfg.JWTSecret, user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"token": token, "user": user})
}

func (s Server) resetPassword(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required"`
		Code     string `json:"code" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if !bind(c, &req) {
		return
	}
	email := normalizeEmail(req.Email)
	if !validEmail(email) {
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
	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := services.ConsumeVerification(tx, email, services.VerificationPasswordReset, req.Code); err != nil {
			return err
		}
		result := tx.Model(&models.User{}).Where("email = ? AND account_deleted = false", email).Updates(map[string]any{"password_hash": string(hash)})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return errEmailNotFound
		}
		return nil
	}); err != nil {
		if errors.Is(err, errEmailNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "email not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s.clearFailedLogin(email)
	services.Audit(c, s.DB, "auth.password_reset", "user", email, nil)
	c.JSON(http.StatusOK, gin.H{"reset": true})
}

func (s Server) sendProfileEmailCode(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required"`
	}
	if !bind(c, &req) {
		return
	}
	email := normalizeEmail(req.Email)
	if !validEmail(email) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid email"})
		return
	}
	var count int64
	s.DB.Model(&models.User{}).Where("email = ? AND account_deleted = false", email).Count(&count)
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email already registered"})
		return
	}
	if err := s.sendCode(c, email, services.VerificationRebindEmail); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"sent": true})
}

func (s Server) rebindEmail(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	var req struct {
		Email string `json:"email" binding:"required"`
		Code  string `json:"code" binding:"required"`
	}
	if !bind(c, &req) {
		return
	}
	email := normalizeEmail(req.Email)
	if !validEmail(email) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid email"})
		return
	}
	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := services.ConsumeVerification(tx, email, services.VerificationRebindEmail, req.Code); err != nil {
			return err
		}
		return tx.Model(&models.User{}).Where("id = ?", user.ID).Updates(map[string]any{"email": email, "email_verified": true}).Error
	}); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	services.Audit(c, s.DB, "profile.email_rebind", "user", user.ID, datatypes.JSONMap{"email": email})
	var updated models.User
	s.DB.First(&updated, user.ID)
	c.JSON(http.StatusOK, updated)
}

func (s Server) getProfile(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	type statusRow struct {
		Status string `json:"status"`
		Count  int    `json:"count"`
	}
	type activityProblem struct {
		ID          uint   `json:"id"`
		DisplayCode string `json:"display_code"`
		Title       string `json:"title"`
	}
	type activityRow struct {
		Date     string            `json:"date"`
		Count    int               `json:"count"`
		Problems []activityProblem `json:"problems"`
	}
	type activityProblemRow struct {
		Date        string
		ID          uint
		DisplayCode string
		Title       string
	}
	var byStatus []statusRow
	s.DB.Table("submissions").Select("status, count(*) as count").Where("user_id = ?", user.ID).Group("status").Scan(&byStatus)
	var problemRows []activityProblemRow
	activityLabel := "解题活跃度"
	activityUnit := "道题"
	if user.CanAuthor || user.Role == models.RoleTeacher || user.Role == models.RoleProblemSetter || user.Role == models.RoleAdmin {
		activityLabel = "题目上传活跃度"
		activityUnit = "道题"
		s.DB.Raw(`
			select to_char(created_at::date, 'YYYY-MM-DD') as date, id, display_code, title
			from problems
			where owner_id = ? and created_at >= ? and deleted_at is null
			order by created_at asc
		`, user.ID, time.Now().AddDate(0, 0, -364)).Scan(&problemRows)
	} else {
		s.DB.Raw(`
			select to_char(min(s.created_at)::date, 'YYYY-MM-DD') as date,
			       p.id, p.display_code, p.title
			from submissions s
			join problems p on p.id = s.problem_id
			where s.user_id = ? and s.status = ? and s.created_at >= ? and p.deleted_at is null
			group by p.id, p.display_code, p.title
			order by date asc, p.display_code asc
		`, user.ID, models.StatusAccepted, time.Now().AddDate(0, 0, -364)).Scan(&problemRows)
	}
	problemsByDate := map[string][]activityProblem{}
	for _, item := range problemRows {
		problemsByDate[item.Date] = append(problemsByDate[item.Date], activityProblem{
			ID:          item.ID,
			DisplayCode: item.DisplayCode,
			Title:       item.Title,
		})
	}
	activity := make([]activityRow, 0, 365)
	today := time.Now()
	for i := 364; i >= 0; i-- {
		key := today.AddDate(0, 0, -i).Format("2006-01-02")
		problems := problemsByDate[key]
		if problems == nil {
			problems = []activityProblem{}
		}
		activity = append(activity, activityRow{Date: key, Count: len(problems), Problems: problems})
	}
	var recent []models.Submission
	s.DB.Where("user_id = ?", user.ID).Order("id desc").Limit(10).Find(&recent)
	var solved int64
	s.DB.Model(&models.Submission{}).Where("user_id = ? AND status = ?", user.ID, models.StatusAccepted).Distinct("problem_id").Count(&solved)
	var submissions int64
	if user.CanAuthor || user.Role == models.RoleTeacher || user.Role == models.RoleProblemSetter || user.Role == models.RoleAdmin {
		s.DB.Model(&models.Problem{}).Where("owner_id = ?", user.ID).Count(&submissions)
	} else {
		s.DB.Model(&models.Submission{}).Where("user_id = ?", user.ID).Count(&submissions)
	}
	c.JSON(http.StatusOK, gin.H{
		"user":           user,
		"solved":         solved,
		"submissions":    submissions,
		"by_status":      byStatus,
		"activity":       activity,
		"activity_label": activityLabel,
		"activity_unit":  activityUnit,
		"recent":         s.submissionListViews(recent),
	})
}

func (s Server) updateProfile(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	var req struct {
		Name      *string `json:"name"`
		AvatarURL *string `json:"avatar_url"`
	}
	if !bind(c, &req) {
		return
	}
	updates := map[string]any{}
	if req.Name != nil {
		updates["name"] = strings.TrimSpace(*req.Name)
	}
	if req.AvatarURL != nil {
		if len(*req.AvatarURL) > 2*1024*1024 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "avatar is too large"})
			return
		}
		updates["avatar_url"] = *req.AvatarURL
	}
	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "nothing to update"})
		return
	}
	if err := s.DB.Model(&models.User{}).Where("id = ?", user.ID).Updates(updates).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	services.Audit(c, s.DB, "profile.update", "user", user.ID, nil)
	var updated models.User
	s.DB.First(&updated, user.ID)
	c.JSON(http.StatusOK, updated)
}

func (s Server) updateProfilePassword(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	var req struct {
		CurrentPassword string `json:"current_password" binding:"required"`
		NewPassword     string `json:"new_password" binding:"required"`
	}
	if !bind(c, &req) {
		return
	}
	if len(req.NewPassword) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "password must be at least 6 characters"})
		return
	}
	var stored models.User
	if err := s.DB.First(&stored, user.ID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte(req.CurrentPassword)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "current password is incorrect"})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := s.DB.Model(&models.User{}).Where("id = ?", user.ID).Update("password_hash", string(hash)).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	services.Audit(c, s.DB, "profile.password_update", "user", user.ID, nil)
	c.JSON(http.StatusOK, gin.H{"updated": true})
}

func (s Server) deleteProfile(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	deletedEmail := fmt.Sprintf("deleted-%d@local.invalid", user.ID)
	if err := s.DB.Model(&models.User{}).Where("id = ?", user.ID).Updates(map[string]any{
		"email":           deletedEmail,
		"name":            "已注销用户",
		"password_hash":   "deleted",
		"avatar_url":      "",
		"email_verified":  false,
		"account_deleted": true,
	}).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	services.Audit(c, s.DB, "profile.delete", "user", user.ID, nil)
	c.JSON(http.StatusOK, gin.H{"deleted": true})
}

func (s Server) createFeedback(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	var req struct {
		Message string `json:"message" binding:"required"`
	}
	if !bind(c, &req) {
		return
	}
	feedback := models.Feedback{UserID: user.ID, Email: user.Email, Message: strings.TrimSpace(req.Message), Status: "open"}
	if err := s.DB.Create(&feedback).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	services.Audit(c, s.DB, "feedback.create", "feedback", feedback.ID, nil)
	c.JSON(http.StatusCreated, feedback)
}

func (s Server) sendCode(c *gin.Context, email, purpose string) error {
	code, err := services.GenerateSixDigitCode()
	if err != nil {
		return err
	}
	if err := services.CreateVerification(s.DB, email, purpose, code); err != nil {
		return err
	}
	if err := (services.Mailer{Cfg: s.Cfg}).SendVerificationCode(email, purpose, code); err != nil {
		return err
	}
	services.Audit(c, s.DB, "email_code.send", "email_verification", email, datatypes.JSONMap{"purpose": purpose})
	return nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func publicVerificationPurpose(purpose string) bool {
	return purpose == services.VerificationRegister || purpose == services.VerificationPasswordReset
}
