package handlers

import (
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
)

func (s Server) sendEmailCode(c *gin.Context) {
	var req struct {
		Email   string `json:"email" binding:"required"`
		Purpose string `json:"purpose" binding:"required"`
	}
	if !bind(c, &req) {
		return
	}
	email := normalizeEmail(req.Email)
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
	if err := services.ConsumeVerification(s.DB, email, services.VerificationRegister, req.Code); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	user := models.User{
		Email:         email,
		Name:          strings.TrimSpace(req.Name),
		Role:          models.RoleStudent,
		PasswordHash:  string(hash),
		StudentNo:     strings.TrimSpace(req.StudentNo),
		EmailVerified: true,
	}
	if err := s.DB.Create(&user).Error; err != nil {
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
	if err := services.ConsumeVerification(s.DB, email, services.VerificationPasswordReset, req.Code); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	result := s.DB.Model(&models.User{}).Where("email = ? AND account_deleted = false", email).Updates(map[string]any{"password_hash": string(hash)})
	if result.Error != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": result.Error.Error()})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "email not found"})
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
	if err := services.ConsumeVerification(s.DB, email, services.VerificationRebindEmail, req.Code); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := s.DB.Model(&models.User{}).Where("id = ?", user.ID).Updates(map[string]any{"email": email, "email_verified": true}).Error; err != nil {
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
	type activityRow struct {
		Date  string `json:"date"`
		Count int    `json:"count"`
	}
	var byStatus []statusRow
	s.DB.Table("submissions").Select("status, count(*) as count").Where("user_id = ?", user.ID).Group("status").Scan(&byStatus)
	var activityRows []activityRow
	s.DB.Raw("select to_char(created_at::date, 'YYYY-MM-DD') as date, count(*) as count from submissions where user_id = ? and created_at >= ? group by created_at::date order by date asc", user.ID, time.Now().AddDate(0, 0, -364)).Scan(&activityRows)
	counts := map[string]int{}
	for _, item := range activityRows {
		counts[item.Date] = item.Count
	}
	activity := make([]activityRow, 0, 365)
	today := time.Now()
	for i := 364; i >= 0; i-- {
		key := today.AddDate(0, 0, -i).Format("2006-01-02")
		activity = append(activity, activityRow{Date: key, Count: counts[key]})
	}
	var recent []models.Submission
	s.DB.Where("user_id = ?", user.ID).Order("id desc").Limit(10).Find(&recent)
	var solved int64
	s.DB.Model(&models.Submission{}).Where("user_id = ? AND status = ?", user.ID, models.StatusAccepted).Distinct("problem_id").Count(&solved)
	var submissions int64
	s.DB.Model(&models.Submission{}).Where("user_id = ?", user.ID).Count(&submissions)
	c.JSON(http.StatusOK, gin.H{
		"user":        user,
		"solved":      solved,
		"submissions": submissions,
		"by_status":   byStatus,
		"activity":    activity,
		"recent":      recent,
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
