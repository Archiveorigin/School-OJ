package services

import (
	"context"
	"gorm.io/gorm"
	"log/slog"
	"time"
)

// DeleteExamRecords removes personal participation data; the deleted exam and
// its review/ticket history remain as provenance for moderation decisions.
func DeleteExamRecords(tx *gorm.DB, id uint) error {
	for _, table := range []string{"submissions", "exam_activities", "exam_attempts", "exam_participants", "plagiarism_jobs", "exam_problems"} {
		if err := tx.Table(table).Where("exam_id = ?", id).Delete(nil).Error; err != nil {
			return err
		}
	}
	return tx.Table("event_announcements").Where("event_kind = ? AND event_id = ?", "exam", id).Delete(nil).Error
}

// PostgreSQL calendar months avoid treating three months as a fixed 90 days.
func PurgeAuditLogs(ctx context.Context, db *gorm.DB) error {
	return db.WithContext(ctx).Exec("DELETE FROM audit_logs WHERE created_at < CURRENT_TIMESTAMP - INTERVAL '3 months'").Error
}

func RunAuditRetention(ctx context.Context, db *gorm.DB) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		cleanup, cancel := context.WithTimeout(ctx, 2*time.Minute)
		err := PurgeAuditLogs(cleanup, db)
		cancel()
		if err != nil {
			slog.Error("audit retention failed", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
