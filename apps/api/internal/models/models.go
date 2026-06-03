package models

import (
	"fmt"
	"time"

	"gorm.io/datatypes"
)

type Role string

const (
	RoleStudent Role = "student"
	RoleTeacher Role = "teacher"
	RoleAdmin   Role = "admin"
)

type User struct {
	ID             uint      `json:"id" gorm:"primaryKey"`
	Email          string    `json:"email" gorm:"uniqueIndex;size:255;not null"`
	Name           string    `json:"name" gorm:"size:120;not null"`
	Role           Role      `json:"role" gorm:"type:varchar(32);index;not null"`
	PasswordHash   string    `json:"-" gorm:"not null"`
	StudentNo      string    `json:"student_no" gorm:"size:64;index"`
	AvatarURL      string    `json:"avatar_url" gorm:"type:text"`
	EmailVerified  bool      `json:"email_verified" gorm:"not null;default:false"`
	AccountDeleted bool      `json:"account_deleted" gorm:"not null;default:false;index"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type Course struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Code        string    `json:"code" gorm:"uniqueIndex;size:64;not null"`
	Name        string    `json:"name" gorm:"size:160;not null"`
	Term        string    `json:"term" gorm:"size:64;index"`
	TeacherID   uint      `json:"teacher_id" gorm:"index;not null"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Class struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	CourseID  uint      `json:"course_id" gorm:"index;not null"`
	Name      string    `json:"name" gorm:"size:120;not null"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CourseMembership struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	CourseID  uint      `json:"course_id" gorm:"uniqueIndex:idx_course_user"`
	UserID    uint      `json:"user_id" gorm:"uniqueIndex:idx_course_user"`
	Role      Role      `json:"role" gorm:"type:varchar(32);not null"`
	CreatedAt time.Time `json:"created_at"`
}

type ClassMembership struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	ClassID   uint      `json:"class_id" gorm:"uniqueIndex:idx_class_user"`
	UserID    uint      `json:"user_id" gorm:"uniqueIndex:idx_class_user"`
	CreatedAt time.Time `json:"created_at"`
}

type ClassProblem struct {
	ID        uint       `json:"id" gorm:"primaryKey"`
	ClassID   uint       `json:"class_id" gorm:"uniqueIndex:idx_class_problem"`
	ProblemID uint       `json:"problem_id" gorm:"uniqueIndex:idx_class_problem"`
	ReleaseAt *time.Time `json:"release_at" gorm:"index"`
	CreatedAt time.Time  `json:"created_at"`
}

type Problem struct {
	ID              uint              `json:"id" gorm:"primaryKey"`
	OwnerID         uint              `json:"owner_id" gorm:"index;not null"`
	DisplayCode     string            `json:"display_code" gorm:"uniqueIndex;size:16"`
	Slug            string            `json:"slug" gorm:"uniqueIndex;size:120;not null"`
	Title           string            `json:"title" gorm:"size:200;not null"`
	Statement       string            `json:"statement" gorm:"type:text"`
	Tags            datatypes.JSONMap `json:"tags" gorm:"type:jsonb"`
	TimeLimitMS     int               `json:"time_limit_ms" gorm:"not null;default:1000"`
	MemoryLimitMB   int               `json:"memory_limit_mb" gorm:"not null;default:256"`
	OutputLimitKB   int               `json:"output_limit_kb" gorm:"not null;default:1024"`
	PackageObject   string            `json:"-" gorm:"size:512;not null"`
	PackageChecksum string            `json:"-" gorm:"size:128;not null"`
	Manifest        datatypes.JSONMap `json:"-" gorm:"type:jsonb"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
	DeletedAt       *time.Time        `json:"deleted_at,omitempty" gorm:"index"`
}

type PreparedProblem struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	ProblemID  uint      `json:"problem_id" gorm:"uniqueIndex;not null"`
	OwnerID    uint      `json:"owner_id" gorm:"index;not null"`
	Folder     string    `json:"folder" gorm:"size:160;index"`
	Difficulty string    `json:"difficulty" gorm:"size:32;index"`
	Source     string    `json:"source" gorm:"size:160"`
	Notes      string    `json:"notes" gorm:"type:text"`
	Archived   bool      `json:"archived" gorm:"not null;default:false;index"`
	Problem    Problem   `json:"problem" gorm:"foreignKey:ProblemID"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type ProblemProgressStatus string

const (
	ProgressUnattempted ProblemProgressStatus = "unattempted"
	ProgressAttempted   ProblemProgressStatus = "attempted"
	ProgressAccepted    ProblemProgressStatus = "accepted"
)

type ProblemProgress struct {
	ID            uint                  `json:"id" gorm:"primaryKey"`
	UserID        uint                  `json:"user_id" gorm:"uniqueIndex:idx_user_problem_progress"`
	ProblemID     uint                  `json:"problem_id" gorm:"uniqueIndex:idx_user_problem_progress"`
	Status        ProblemProgressStatus `json:"status" gorm:"type:varchar(32);index;not null;default:'unattempted'"`
	Points        int                   `json:"points" gorm:"not null;default:0"`
	PointsAwarded bool                  `json:"points_awarded" gorm:"not null;default:false"`
	FirstAccepted *time.Time            `json:"first_accepted_at"`
	LastSubmitted *time.Time            `json:"last_submitted_at"`
	CreatedAt     time.Time             `json:"created_at"`
	UpdatedAt     time.Time             `json:"updated_at"`
}

type Assignment struct {
	ID          uint                `json:"id" gorm:"primaryKey"`
	CourseID    uint                `json:"course_id" gorm:"index;not null"`
	ClassID     *uint               `json:"class_id" gorm:"index"`
	Title       string              `json:"title" gorm:"size:200;not null"`
	Description string              `json:"description"`
	StartsAt    *time.Time          `json:"starts_at"`
	DueAt       *time.Time          `json:"due_at"`
	Settings    datatypes.JSONMap   `json:"settings" gorm:"type:jsonb"`
	Problems    []AssignmentProblem `json:"problems,omitempty"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
	DeletedAt   *time.Time          `json:"deleted_at,omitempty" gorm:"index"`
}

type AssignmentProblem struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	AssignmentID uint      `json:"assignment_id" gorm:"index;not null"`
	ProblemID    uint      `json:"problem_id" gorm:"index;not null"`
	Score        int       `json:"score" gorm:"not null;default:100"`
	SortOrder    int       `json:"sort_order" gorm:"not null;default:0"`
	Problem      Problem   `json:"problem" gorm:"foreignKey:ProblemID"`
	CreatedAt    time.Time `json:"created_at"`
}

type AssignmentAttempt struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	AssignmentID uint      `json:"assignment_id" gorm:"uniqueIndex:idx_assignment_attempt"`
	UserID       uint      `json:"user_id" gorm:"uniqueIndex:idx_assignment_attempt"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Exam struct {
	ID          uint              `json:"id" gorm:"primaryKey"`
	CourseID    uint              `json:"course_id" gorm:"index;not null"`
	ClassID     *uint             `json:"class_id" gorm:"index"`
	Title       string            `json:"title" gorm:"size:200;not null"`
	Description string            `json:"description"`
	StartsAt    *time.Time        `json:"starts_at"`
	EndsAt      *time.Time        `json:"ends_at"`
	Settings    datatypes.JSONMap `json:"settings" gorm:"type:jsonb"`
	Problems    []ExamProblem     `json:"problems,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	DeletedAt   *time.Time        `json:"deleted_at,omitempty" gorm:"index"`
}

type ExamProblem struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	ExamID    uint      `json:"exam_id" gorm:"index;not null"`
	ProblemID uint      `json:"problem_id" gorm:"index;not null"`
	Label     string    `json:"label" gorm:"size:32;index"`
	Score     int       `json:"score" gorm:"not null;default:100"`
	SortOrder int       `json:"sort_order" gorm:"not null;default:0"`
	Problem   Problem   `json:"problem" gorm:"foreignKey:ProblemID"`
	CreatedAt time.Time `json:"created_at"`
}

func FormatProblemDisplayCode(index int) string {
	return fmt.Sprintf("T%03d", index)
}

type ExamAttempt struct {
	ID         uint       `json:"id" gorm:"primaryKey"`
	ExamID     uint       `json:"exam_id" gorm:"uniqueIndex:idx_exam_attempt"`
	UserID     uint       `json:"user_id" gorm:"uniqueIndex:idx_exam_attempt"`
	FinishedAt *time.Time `json:"finished_at"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
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
	ID             uint              `json:"id" gorm:"primaryKey"`
	UserID         uint              `json:"user_id" gorm:"index;not null"`
	ProblemID      uint              `json:"problem_id" gorm:"index;not null"`
	AssignmentID   *uint             `json:"assignment_id" gorm:"index"`
	ExamID         *uint             `json:"exam_id" gorm:"index"`
	Language       string            `json:"language" gorm:"size:32;index;not null"`
	SourceCode     string            `json:"source_code" gorm:"type:text;not null"`
	Status         SubmissionStatus  `json:"status" gorm:"type:varchar(32);index;not null"`
	Score          int               `json:"score" gorm:"not null;default:0"`
	ManualScore    *int              `json:"manual_score"`
	ManualGradedBy *uint             `json:"manual_graded_by" gorm:"index"`
	ManualGradedAt *time.Time        `json:"manual_graded_at"`
	TimeMS         int               `json:"time_ms" gorm:"not null;default:0"`
	MemoryKB       int               `json:"memory_kb" gorm:"not null;default:0"`
	Message        string            `json:"message" gorm:"type:text"`
	Trace          datatypes.JSONMap `json:"trace" gorm:"type:jsonb"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

type SubmissionResult struct {
	ID           uint             `json:"id" gorm:"primaryKey"`
	SubmissionID uint             `json:"submission_id" gorm:"index;not null"`
	CaseName     string           `json:"case_name" gorm:"size:200;not null"`
	Status       SubmissionStatus `json:"status" gorm:"type:varchar(32);index;not null"`
	TimeMS       int              `json:"time_ms" gorm:"not null;default:0"`
	MemoryKB     int              `json:"memory_kb" gorm:"not null;default:0"`
	Message      string           `json:"message" gorm:"type:text"`
	CreatedAt    time.Time        `json:"created_at"`
}

type PlagiarismJob struct {
	ID           uint              `json:"id" gorm:"primaryKey"`
	CourseID     uint              `json:"course_id" gorm:"index;not null"`
	AssignmentID *uint             `json:"assignment_id" gorm:"index"`
	ExamID       *uint             `json:"exam_id" gorm:"index"`
	Language     string            `json:"language" gorm:"size:32;index;not null"`
	Status       string            `json:"status" gorm:"type:varchar(32);index;not null"`
	ReportObject string            `json:"report_object" gorm:"size:512"`
	Summary      datatypes.JSONMap `json:"summary" gorm:"type:jsonb"`
	Message      string            `json:"message" gorm:"type:text"`
	CreatedBy    uint              `json:"created_by" gorm:"index;not null"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

type AuditLog struct {
	ID           uint              `json:"id" gorm:"primaryKey"`
	ActorUserID  *uint             `json:"actor_user_id" gorm:"index"`
	Action       string            `json:"action" gorm:"size:160;index;not null"`
	ResourceType string            `json:"resource_type" gorm:"size:80;index"`
	ResourceID   string            `json:"resource_id" gorm:"size:120;index"`
	IP           string            `json:"ip" gorm:"size:80"`
	UserAgent    string            `json:"user_agent" gorm:"size:512"`
	Meta         datatypes.JSONMap `json:"meta" gorm:"type:jsonb"`
	CreatedAt    time.Time         `json:"created_at"`
}

type EmailVerification struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Email     string    `json:"email" gorm:"size:255;index;not null"`
	Purpose   string    `json:"purpose" gorm:"size:32;index;not null"`
	CodeHash  string    `json:"-" gorm:"not null"`
	Attempts  int       `json:"attempts" gorm:"not null;default:0"`
	Consumed  bool      `json:"consumed" gorm:"not null;default:false;index"`
	ExpiresAt time.Time `json:"expires_at" gorm:"index;not null"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type LoginAttempt struct {
	ID           uint       `json:"id" gorm:"primaryKey"`
	Email        string     `json:"email" gorm:"uniqueIndex;size:255;not null"`
	FailedCount  int        `json:"failed_count" gorm:"not null;default:0"`
	LastFailedAt *time.Time `json:"last_failed_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type Feedback struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"user_id" gorm:"index;not null"`
	Email     string    `json:"email" gorm:"size:255"`
	Message   string    `json:"message" gorm:"type:text;not null"`
	Status    string    `json:"status" gorm:"size:32;index;not null;default:'open'"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func AllModels() []any {
	return []any{
		&User{},
		&Course{},
		&Class{},
		&CourseMembership{},
		&ClassMembership{},
		&ClassProblem{},
		&Problem{},
		&PreparedProblem{},
		&ProblemProgress{},
		&Assignment{},
		&AssignmentProblem{},
		&AssignmentAttempt{},
		&Exam{},
		&ExamProblem{},
		&ExamAttempt{},
		&Submission{},
		&SubmissionResult{},
		&PlagiarismJob{},
		&AuditLog{},
		&EmailVerification{},
		&LoginAttempt{},
		&Feedback{},
	}
}
