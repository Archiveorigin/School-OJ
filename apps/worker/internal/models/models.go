package models

import (
	"time"

	"gorm.io/datatypes"
)

type Problem struct {
	ID              uint              `json:"id" gorm:"primaryKey"`
	OwnerID         uint              `json:"owner_id"`
	Slug            string            `json:"slug"`
	Title           string            `json:"title"`
	Statement       string            `json:"statement"`
	Tags            datatypes.JSONMap `json:"tags"`
	TimeLimitMS     int               `json:"time_limit_ms"`
	MemoryLimitMB   int               `json:"memory_limit_mb"`
	OutputLimitKB   int               `json:"output_limit_kb"`
	PackageObject   string            `json:"package_object"`
	PackageChecksum string            `json:"package_checksum"`
	Manifest        datatypes.JSONMap `json:"manifest"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

type SubmissionStatus string

const (
	StatusQueued        SubmissionStatus = "queued"
	StatusRunning       SubmissionStatus = "running"
	StatusPendingReview SubmissionStatus = "pending_review"
	StatusManualGraded  SubmissionStatus = "manual_graded"
	StatusAccepted      SubmissionStatus = "accepted"
	StatusWrongAnswer   SubmissionStatus = "wrong_answer"
	StatusCompileError  SubmissionStatus = "compile_error"
	StatusRuntimeError  SubmissionStatus = "runtime_error"
	StatusTimeLimit     SubmissionStatus = "time_limit"
	StatusMemoryLimit   SubmissionStatus = "memory_limit"
	StatusOutputLimit   SubmissionStatus = "output_limit"
	StatusSystemError   SubmissionStatus = "system_error"
)

type Submission struct {
	ID           uint              `json:"id" gorm:"primaryKey"`
	UserID       uint              `json:"user_id"`
	ProblemID    uint              `json:"problem_id"`
	Problem      Problem           `json:"problem" gorm:"foreignKey:ProblemID"`
	AssignmentID *uint             `json:"assignment_id"`
	ExamID       *uint             `json:"exam_id"`
	Language     string            `json:"language"`
	SourceCode   string            `json:"source_code"`
	Status       SubmissionStatus  `json:"status"`
	Score        int               `json:"score"`
	ManualScore  *int              `json:"manual_score"`
	ManualGradedBy *uint           `json:"manual_graded_by"`
	ManualGradedAt *time.Time      `json:"manual_graded_at"`
	TimeMS       int               `json:"time_ms"`
	MemoryKB     int               `json:"memory_kb"`
	Message      string            `json:"message"`
	Trace        datatypes.JSONMap `json:"trace"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

type SubmissionResult struct {
	ID           uint             `json:"id" gorm:"primaryKey"`
	SubmissionID uint             `json:"submission_id"`
	CaseName     string           `json:"case_name"`
	Status       SubmissionStatus `json:"status"`
	TimeMS       int              `json:"time_ms"`
	MemoryKB     int              `json:"memory_kb"`
	Message      string           `json:"message"`
	CreatedAt    time.Time        `json:"created_at"`
}

type ProblemProgressStatus string

const (
	ProgressUnattempted ProblemProgressStatus = "unattempted"
	ProgressAttempted   ProblemProgressStatus = "attempted"
	ProgressAccepted    ProblemProgressStatus = "accepted"
)

type ProblemProgress struct {
	ID            uint                  `json:"id" gorm:"primaryKey"`
	UserID        uint                  `json:"user_id"`
	ProblemID     uint                  `json:"problem_id"`
	Status        ProblemProgressStatus `json:"status"`
	Points        int                   `json:"points"`
	PointsAwarded bool                  `json:"points_awarded"`
	FirstAccepted *time.Time            `json:"first_accepted_at"`
	LastSubmitted *time.Time            `json:"last_submitted_at"`
	CreatedAt     time.Time             `json:"created_at"`
	UpdatedAt     time.Time             `json:"updated_at"`
}
