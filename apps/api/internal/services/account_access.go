package services

import (
	"errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"school-oj/apps/api/internal/models"
)

var ErrLastAdministrator = errors.New("至少需要保留一位有效管理员，请先完成权限交接")

// Serialize administrator removals/demotions, including self-service account
// deletion. The advisory lock prevents two different admins removing the last
// administrators concurrently. Permission versions are maintained by SQL.
func UpdateAccountAccess(db *gorm.DB, userID uint, updates map[string]any) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(17349822)").Error; err != nil {
			return err
		}
		var user models.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("account_deleted = false").First(&user, userID).Error; err != nil {
			return err
		}
		demoting := false
		if role, ok := updates["role"]; ok {
			demoting = role != models.RoleAdmin && role != "admin"
		}
		deleted, _ := updates["account_deleted"].(bool)
		if user.Role == models.RoleAdmin && (demoting || deleted) {
			var remaining int64
			if err := tx.Model(&models.User{}).Where("role = ? AND account_deleted = false AND id <> ?", models.RoleAdmin, userID).Count(&remaining).Error; err != nil {
				return err
			}
			if remaining == 0 {
				return ErrLastAdministrator
			}
		}
		return tx.Model(&user).Updates(updates).Error
	})
}
