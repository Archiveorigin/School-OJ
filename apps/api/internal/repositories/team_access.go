package repositories

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"school-oj/apps/api/internal/domain/team"
	"school-oj/apps/api/internal/models"
	"time"
)

type TeamAccess struct{ DB *gorm.DB }
type teamTransaction struct {
	db    *gorm.DB
	value models.Team
}

func (r TeamAccess) Invitation(ctx context.Context, code string) (models.Team, error) {
	var matches []models.Team
	err := r.DB.WithContext(ctx).Where("join_mode = 'invitation' AND upper(btrim(join_code)) = ?", code).Limit(2).Find(&matches).Error
	if err != nil {
		return models.Team{}, err
	}
	if len(matches) != 1 {
		return models.Team{}, team.Fail(team.InvalidInvitation, "邀请码无效或已失效，请向团队管理员确认")
	}
	return matches[0], nil
}
func (r TeamAccess) Preview(ctx context.Context, value models.Team, userID uint) (team.Preview, error) {
	view := team.Preview{ID: value.ID, Name: value.Name, Slug: value.Slug, IconURL: value.IconURL, Visibility: value.Visibility, JoinMode: value.JoinMode}
	var membership models.TeamMembership
	err := r.DB.WithContext(ctx).Where("team_id = ? AND user_id = ?", value.ID, userID).First(&membership).Error
	if err == nil {
		view.Joined = true
		return view, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return view, err
	}
	var application models.TeamJoinApplication
	err = r.DB.WithContext(ctx).Where("team_id = ? AND user_id = ?", value.ID, userID).Order("id desc").First(&application).Error
	if err == nil {
		view.ApplicationStatus = application.Status
		return view, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return view, nil
	}
	return view, err
}
func (r TeamAccess) WithTeam(ctx context.Context, id uint, fn func(team.Transaction) error) error {
	return r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var value models.Team
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&value, id).Error; err != nil {
			return err
		}
		return fn(teamTransaction{db: tx, value: value})
	})
}
func (t teamTransaction) Team() models.Team { return t.value }
func (t teamTransaction) ActiveUser(id uint) (models.User, error) {
	var user models.User
	err := t.db.Clauses(clause.Locking{Strength: "SHARE"}).Where("account_deleted = false").First(&user, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = team.Fail(team.Conflict, "用户账号已删除或不可用")
	}
	return user, err
}
func (t teamTransaction) Membership(id uint) (models.TeamMembership, bool, error) {
	var membership models.TeamMembership
	err := t.db.Where("team_id = ? AND user_id = ?", t.value.ID, id).First(&membership).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return membership, false, nil
	}
	return membership, err == nil, err
}
func (t teamTransaction) PendingApplication(id uint) (models.TeamJoinApplication, bool, error) {
	var application models.TeamJoinApplication
	err := t.db.Where("team_id = ? AND user_id = ? AND status = 'pending'", t.value.ID, id).First(&application).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return application, false, nil
	}
	return application, err == nil, err
}
func (t teamTransaction) Application(id uint) (models.TeamJoinApplication, error) {
	var application models.TeamJoinApplication
	err := t.db.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND team_id = ?", id, t.value.ID).First(&application).Error
	return application, err
}
func (t teamTransaction) CreateApplication(id uint, message string) error {
	return t.db.Create(&models.TeamJoinApplication{TeamID: t.value.ID, UserID: id, Status: "pending", Message: message}).Error
}
func (t teamTransaction) AddMember(id uint) error {
	return t.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&models.TeamMembership{TeamID: t.value.ID, UserID: id, Role: models.TeamRoleMember}).Error
}
func (t teamTransaction) CompletePending(id uint) error {
	return t.db.Model(&models.TeamJoinApplication{}).Where("team_id = ? AND user_id = ? AND status = 'pending'", t.value.ID, id).Updates(map[string]any{"status": "approved", "reviewed_at": time.Now()}).Error
}
func (t teamTransaction) Review(value models.TeamJoinApplication, status string, actor uint) error {
	return t.db.Model(&value).Updates(map[string]any{"status": status, "reviewed_by": actor, "reviewed_at": time.Now()}).Error
}
func (t teamTransaction) SetRole(value models.TeamMembership, role models.TeamRole) error {
	return t.db.Model(&value).Update("role", role).Error
}
func (t teamTransaction) Remove(value models.TeamMembership) error { return t.db.Delete(&value).Error }

var _ team.Repository = TeamAccess{}
var _ team.Transaction = teamTransaction{}
