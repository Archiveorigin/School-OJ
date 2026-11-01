package handlers

import (
	"school-oj/apps/api/internal/authz"
	"time"

	"school-oj/apps/api/internal/models"

	"gorm.io/datatypes"
)

// shouldRedactSubmission is the single server-side policy gate used by list,
// detail and SSE responses. It deliberately returns less information when the
// event context cannot be loaded.
func (s Server) shouldRedactSubmission(user models.User, submission models.Submission, now time.Time) bool {
	return s.newSubmissionRedactor(user, now).hidden(submission)
}

// submissionRedactor applies the policy to many submissions of one request, so a
// 200-row list asks for each event and its permissions once instead of per row.
// It lives no longer than the request: revocations still apply to the next one.
type submissionRedactor struct {
	s             Server
	user          models.User
	now           time.Time
	exams         map[uint]*models.Exam
	contests      map[uint]*models.TeamContest
	teams         map[uint]*models.Team
	courseManager map[uint]bool
	teamManager   map[uint]bool
}

func (s Server) newSubmissionRedactor(user models.User, now time.Time) *submissionRedactor {
	return &submissionRedactor{
		s:             s,
		user:          user,
		now:           now,
		exams:         map[uint]*models.Exam{},
		contests:      map[uint]*models.TeamContest{},
		teams:         map[uint]*models.Team{},
		courseManager: map[uint]bool{},
		teamManager:   map[uint]bool{},
	}
}

func (r *submissionRedactor) hidden(submission models.Submission) bool {
	if authz.Has(r.user, authz.SubmissionResultsAny) {
		return false
	}
	if submission.ExamID != nil {
		exam := r.exam(*submission.ExamID)
		if exam == nil {
			return true
		}
		if r.managesCourse(exam.CourseID) {
			return false
		}
		return examSubmissionHidden(*exam, submission.CreatedAt, r.now)
	}
	if submission.TeamContestID != nil {
		contest := r.contest(*submission.TeamContestID)
		if contest == nil {
			return true
		}
		// Scoreboard freezing hides results from other viewers, not their author.
		// OI's separate pre-deadline result embargo still applies to the author.
		if r.user.ID == submission.UserID && contest.ScoringRule != "oi" {
			return false
		}
		if contestFreezeHeld(*contest, r.user, r.now) {
			return teamContestSubmissionHidden(*contest, submission.CreatedAt, r.now)
		}
		team := r.team(contest.TeamID)
		if team == nil {
			return true
		}
		if r.managesTeam(*team) {
			return false
		}
		return teamContestSubmissionHidden(*contest, submission.CreatedAt, r.now)
	}
	return false
}

func (r *submissionRedactor) exam(id uint) *models.Exam {
	item, known := r.exams[id]
	if !known {
		loaded := models.Exam{}
		if r.s.DB.First(&loaded, id).Error == nil {
			item = &loaded
		}
		r.exams[id] = item
	}
	return item
}

func (r *submissionRedactor) contest(id uint) *models.TeamContest {
	item, known := r.contests[id]
	if !known {
		loaded := models.TeamContest{}
		if r.s.DB.First(&loaded, id).Error == nil {
			item = &loaded
		}
		r.contests[id] = item
	}
	return item
}

func (r *submissionRedactor) team(id uint) *models.Team {
	item, known := r.teams[id]
	if !known {
		loaded := models.Team{}
		if r.s.DB.First(&loaded, id).Error == nil {
			item = &loaded
		}
		r.teams[id] = item
	}
	return item
}

func (r *submissionRedactor) managesCourse(id uint) bool {
	allowed, known := r.courseManager[id]
	if !known {
		allowed = r.s.canManageCourse(r.user, id)
		r.courseManager[id] = allowed
	}
	return allowed
}

func (r *submissionRedactor) managesTeam(team models.Team) bool {
	allowed, known := r.teamManager[team.ID]
	if !known {
		allowed = r.s.canManageTeam(r.user, team) || r.s.canTeamContentPermission(r.user, team)
		r.teamManager[team.ID] = allowed
	}
	return allowed
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
	if !contest.FreezeEnabled || endsAt == nil || contest.CompletedAt != nil {
		return false
	}
	freezeAt := endsAt.Add(-time.Duration(contest.FreezeDurationMinutes) * time.Minute)
	return !now.Before(freezeAt) && !submittedAt.Before(freezeAt)
}

func (s Server) redactSubmissionViews(user models.User, views []submissionListView) []submissionListView {
	redactor := s.newSubmissionRedactor(user, time.Now())
	for index := range views {
		if redactor.hidden(views[index].Submission) {
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
	if authz.Has(user, authz.SubmissionResultsAny) || s.canManageCourse(user, exam.CourseID) {
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
