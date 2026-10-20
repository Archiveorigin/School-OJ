package models

import (
	"gorm.io/datatypes"
	"time"
)

// Existing events retain approved as a compatibility default. Every creation
// endpoint explicitly starts in draft; migration queues future published events.
type EventReview struct {
	ReviewStatus      string     `json:"review_status" gorm:"size:16;not null;default:'approved';index"`
	ReviewRevision    int        `json:"review_revision" gorm:"not null;default:1"`
	SubmissionVersion int        `json:"submission_version" gorm:"not null;default:0"`
	ReviewNote        string     `json:"review_note" gorm:"type:text;not null;default:''"`
	ReviewSubmittedAt *time.Time `json:"review_submitted_at"`
	ReviewedAt        *time.Time `json:"reviewed_at"`
	ReviewedBy        *uint      `json:"reviewed_by"`
	CompletedAt       *time.Time `json:"completed_at"`
	CompletedBy       *uint      `json:"completed_by"`
	CompletionNote    string     `json:"completion_note" gorm:"type:text;not null;default:''"`
}

// Lifecycle is derived from approval, the clock, and explicit completion.
// A deadline alone never publishes an archive.
func (r EventReview) Lifecycle(start, end *time.Time, now time.Time) string {
	if !r.Approved() {
		return r.ReviewStatus
	}
	if start != nil && now.Before(*start) {
		return "upcoming"
	}
	if end != nil && !now.Before(*end) {
		if r.CompletedAt != nil {
			return "completed"
		}
		return "closed"
	}
	return "running"
}

func (r EventReview) Approved() bool { return r.ReviewStatus == "approved" || r.ReviewStatus == "" }
func (r EventReview) Editable() bool {
	return r.ReviewStatus == "draft" || r.ReviewStatus == "rejected"
}

// Snapshots preserve exactly which versions, attachments and schedule were
// submitted/decided, even when a draft is subsequently revised or deleted.
type EventReviewLog struct {
	ID                uint           `json:"id" gorm:"primaryKey"`
	EventKind         string         `json:"event_kind" gorm:"size:16;not null;index:idx_event_review_log"`
	EventID           uint           `json:"event_id" gorm:"not null;index:idx_event_review_log"`
	Revision          int            `json:"revision" gorm:"not null"`
	SubmissionVersion int            `json:"submission_version" gorm:"not null;default:0"`
	Action            string         `json:"action" gorm:"size:16;not null"`
	ActorID           uint           `json:"actor_id" gorm:"not null"`
	Note              string         `json:"note" gorm:"type:text;not null"`
	Snapshot          datatypes.JSON `json:"snapshot" gorm:"type:jsonb;not null"`
	CreatedAt         time.Time      `json:"created_at"`
}

type EventAttachment struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	EventKind string    `json:"event_kind" gorm:"size:16;not null;index:idx_event_attachment"`
	EventID   uint      `json:"event_id" gorm:"not null;index:idx_event_attachment"`
	Name      string    `json:"name" gorm:"size:255;not null"`
	Object    string    `json:"-" gorm:"size:512;not null"`
	Size      int64     `json:"size" gorm:"not null"`
	Checksum  string    `json:"checksum" gorm:"size:64;not null"`
	CreatedBy uint      `json:"created_by" gorm:"not null"`
	CreatedAt time.Time `json:"created_at"`
}

type ExamParticipant struct {
	ID     uint `json:"id" gorm:"primaryKey"`
	ExamID uint `json:"exam_id" gorm:"uniqueIndex:idx_exam_participant;not null"`
	UserID uint `json:"user_id" gorm:"uniqueIndex:idx_exam_participant;not null"`
}
