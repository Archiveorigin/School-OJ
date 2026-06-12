package services

import (
	"errors"
	"strings"
	"time"

	"school-oj/apps/api/internal/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	VerificationRegister      = "register"
	VerificationRebindEmail   = "rebind_email"
	VerificationPasswordReset = "password_reset"
)

var ErrVerificationTooFrequent = errors.New("验证码发送过于频繁，请稍后再试")

func CreateVerification(db *gorm.DB, email, purpose, code string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	var recent int64
	if err := db.Model(&models.EmailVerification{}).
		Where("email = ? AND purpose = ? AND created_at > ?", email, purpose, time.Now().Add(-time.Minute)).
		Count(&recent).Error; err != nil {
		return err
	}
	if recent > 0 {
		return ErrVerificationTooFrequent
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return db.Create(&models.EmailVerification{
		Email:     email,
		Purpose:   purpose,
		CodeHash:  string(hash),
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}).Error
}

func ConsumeVerification(db *gorm.DB, email, purpose, code string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	var item models.EmailVerification
	err := db.Where("email = ? AND purpose = ? AND consumed = false AND expires_at > ?", email, purpose, time.Now()).
		Order("id desc").
		First(&item).Error
	if err != nil {
		return errors.New("验证码无效或已过期")
	}
	if item.Attempts >= 5 {
		return errors.New("验证码尝试次数过多，请重新获取")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(item.CodeHash), []byte(strings.TrimSpace(code))); err != nil {
		db.Model(&item).Update("attempts", item.Attempts+1)
		return errors.New("验证码错误")
	}
	return db.Model(&item).Updates(map[string]any{"consumed": true}).Error
}
