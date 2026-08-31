package handlers

import (
	"time"

	"school-oj/apps/api/internal/models"

	"gorm.io/datatypes"
)

// shouldRedactSubmission is the single server-side policy gate used by list,
// detail and SSE responses. It deliberately returns less information when the
// event context cannot be loaded.
func (s Server) shouldRedactSubmission(user models.User, submission models.Submission, now time.Time) bool {
	if user.Role == models.RoleAdmin {
		return false
	}
	if submission.ExamID != nil {
		var exam models.Exam
		if err := s.DB.First(&exam, *submission.ExamID).Error; err != nil {
			return true
		}
		if s.canManageCourse(user, exam.CourseID) {
			return false
		}
		return examSubmissionHidden(exam, submission.CreatedAt, now)
	}
	if submission.TeamContestID != nil {
		var contest models.TeamContest
		if err := s.DB.First(&contest, *submission.TeamContestID).Error; err != nil {
			return true
		}
		var team models.Team
		if err := s.DB.First(&team, contest.TeamID).Error; err != nil {
			return true
		}
		if s.canManageTeam(user, team) || s.canTeamContentPermission(user, team) {
			return false
		}
		return teamContestSubmissionHidden(contest, submission.CreatedAt, now)
	}
	return false
}

func examSubmissionHidden(exam models.Exam, submittedAt, now time.Time) bool {
	ended := exam.EndsAt != nil && !now.Before(*exam.EndsAt)
	if examScoringRule(exam) == "oi" && !ended {
		return true
	}
	if !exam.FreezeEnabled || exam.EndsAt == nil || ended {
		return false
	}
	freezeAt := exam.EndsAt.Add(-time.Duration(exam.FreezeDurationMinutes) * time.Minute)
	return !now.Before(freezeAt) && !submittedAt.Before(freezeAt)
}

func teamContestSubmissionHidden(contest models.TeamContest, submittedAt, now time.Time) bool {
	endsAt, status := teamContestWindow(contest, now)
	ended := status == models.TeamContestClosed
	if contest.ScoringRule == "oi" && !ended {
		return true
	}
	if !contest.FreezeEnabled || endsAt == nil || ended {
		return false
	}
	freezeAt := endsAt.Add(-time.Duration(contest.FreezeDurationMinutes) * time.Minute)
	return !now.Before(freezeAt) && !submittedAt.Before(freezeAt)
}

func (s Server) redactSubmissionViews(user models.User, views []submissionListView) []submissionListView {
	now := time.Now()
	for index := range views {
		if s.shouldRedactSubmission(user, views[index].Submission, now) {
			redactSubmission(&views[index].Submission)
			views[index].ErrorPoint = ""
		}
	}
	return views
}

func redactSubmission(submission *models.Submission) {
	submission.Status = models.SubmissionStatus("recorded")
	submission.Score = 0
	submission.ManualScore = nil
	submission.ManualGradedBy = nil
	submission.ManualGradedAt = nil
	submission.TimeMS = 0
	submission.MemoryKB = 0
	submission.Message = "已记录"
	submission.Trace = datatypes.JSONMap{}
}

func (s Server) examSummaryMustBeHidden(user models.User, exam models.Exam, now time.Time) bool {
	if user.Role == models.RoleAdmin || s.canManageCourse(user, exam.CourseID) {
		return false
	}
	ended := exam.EndsAt != nil && !now.Before(*exam.EndsAt)
	if examScoringRule(exam) == "oi" && !ended {
		return true
	}
	if exam.FreezeEnabled && exam.EndsAt != nil && !ended {
		freezeAt := exam.EndsAt.Add(-time.Duration(exam.FreezeDurationMinutes) * time.Minute)
		return !now.Before(freezeAt)
	}
	return false
}

func redactWorkSummary(summary workSummary) workSummary {
	summary.TotalScore = 0
	summary.ScoreReady = false
	for index := range summary.Problems {
		summary.Problems[index].BestScore = 0
		summary.Problems[index].RawScore = 0
		summary.Problems[index].ScoreReady = false
		if summary.Problems[index].SubmissionID != nil {
			summary.Problems[index].SubmissionStatus = models.SubmissionStatus("recorded")
		}
	}
	return summary
}
