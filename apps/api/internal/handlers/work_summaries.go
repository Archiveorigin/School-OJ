package handlers

import (
	"context"
	"time"

	"school-oj/apps/api/internal/models"
)

// Scoring only reads these columns, so summaries never load submitted source.
var workSummarySubmissionColumns = []string{"id", "user_id", "problem_id", "assignment_id", "exam_id", "language", "status", "score", "manual_score", "manual_graded_by", "manual_graded_at", "time_ms", "memory_kb", "message", "created_at", "updated_at"}

type workSummaryLink struct {
	Problem   models.Problem
	ProblemID uint
	Label     string
	Score     int
}

// summarizeWork scores one user's submissions against one work's problem links.
// submissions must be ordered newest first within each problem.
func summarizeWork(links []workSummaryLink, submissions map[uint][]models.Submission, manualReview bool, attempted bool, includeProblems bool, scoringRule string) workSummary {
	summary := workSummary{WorkStatus: "unattempted", ScoreReady: false}
	if attempted {
		summary.WorkStatus = "unsubmitted"
	}
	hasSubmission := false
	hasPending := false
	for _, link := range links {
		summary.MaxScore += link.Score
		view, submitted, pending := problemScoreFromSubmissionsForRule(link.Problem, link.Score, manualReview, submissions[link.ProblemID], scoringRule)
		view.Label = link.Label
		if submitted {
			hasSubmission = true
		}
		if pending {
			hasPending = true
		}
		summary.TotalScore += view.BestScore
		if includeProblems {
			summary.Problems = append(summary.Problems, view)
		}
	}
	if hasSubmission {
		summary.WorkStatus = "submitted"
		summary.ScoreReady = !hasPending
	}
	return summary
}

func assignmentIDs(items []models.Assignment) []uint {
	ids := make([]uint, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}

// assignmentSummaries covers both a student's assignment list and a teacher's
// report with the same fixed number of queries. Every requested work/user pair
// is present in the result, so an empty assignment still summarises as unattempted.
func (s Server) assignmentSummaries(ids []uint, users []uint, includeProblems bool) map[uint]map[uint]workSummary {
	result := make(map[uint]map[uint]workSummary, len(ids))
	links := map[uint][]workSummaryLink{}
	attempted := map[uint]map[uint]bool{}
	submissions := map[uint]map[uint]map[uint][]models.Submission{}
	if len(ids) > 0 && len(users) > 0 {
		var rows []models.AssignmentProblem
		q := s.DB.Where("assignment_id IN ?", ids).Order("assignment_id asc, sort_order asc")
		if includeProblems {
			q = q.Preload("Problem")
		}
		q.Find(&rows)
		for _, row := range rows {
			links[row.AssignmentID] = append(links[row.AssignmentID], workSummaryLink{Problem: row.Problem, ProblemID: row.ProblemID, Score: row.Score})
		}
		var attempts []models.AssignmentAttempt
		s.DB.Select("assignment_id", "user_id").Where("assignment_id IN ? AND user_id IN ?", ids, users).Find(&attempts)
		for _, attempt := range attempts {
			if attempted[attempt.AssignmentID] == nil {
				attempted[attempt.AssignmentID] = map[uint]bool{}
			}
			attempted[attempt.AssignmentID][attempt.UserID] = true
		}
		var subs []models.Submission
		s.DB.Select(workSummarySubmissionColumns).Where("assignment_id IN ? AND exam_id IS NULL AND user_id IN ?", ids, users).Order("id desc").Find(&subs)
		for _, sub := range subs {
			indexWorkSubmission(submissions, *sub.AssignmentID, sub)
		}
	}
	for _, id := range ids {
		result[id] = make(map[uint]workSummary, len(users))
		for _, user := range users {
			result[id][user] = summarizeWork(links[id], submissions[id][user], false, attempted[id][user], includeProblems, "ioi")
		}
	}
	return result
}

func (s Server) assignmentSummary(assignmentID uint, userID uint, includeProblems bool) workSummary {
	return s.assignmentSummaries([]uint{assignmentID}, []uint{userID}, includeProblems)[assignmentID][userID]
}

// examSummaries needs each exam's scoring settings, so it takes loaded exams.
func (s Server) examSummaries(exams []models.Exam, users []uint, includeProblems bool) map[uint]map[uint]workSummary {
	result := make(map[uint]map[uint]workSummary, len(exams))
	ids := modelIDs(exams)
	links := map[uint][]workSummaryLink{}
	attempted := map[uint]map[uint]bool{}
	submissions := map[uint]map[uint]map[uint][]models.Submission{}
	if len(ids) > 0 && len(users) > 0 {
		var rows []models.ExamProblem
		q := s.DB.Where("exam_id IN ?", ids).Order("exam_id asc, sort_order asc")
		if includeProblems {
			q = q.Preload("Problem")
		}
		q.Find(&rows)
		for _, row := range rows {
			links[row.ExamID] = append(links[row.ExamID], workSummaryLink{Problem: row.Problem, ProblemID: row.ProblemID, Label: row.Label, Score: row.Score})
		}
		var attempts []models.ExamAttempt
		s.DB.Select("exam_id", "user_id").Where("exam_id IN ? AND user_id IN ?", ids, users).Find(&attempts)
		for _, attempt := range attempts {
			if attempted[attempt.ExamID] == nil {
				attempted[attempt.ExamID] = map[uint]bool{}
			}
			attempted[attempt.ExamID][attempt.UserID] = true
		}
		var subs []models.Submission
		s.DB.Select(workSummarySubmissionColumns).Where("exam_id IN ? AND assignment_id IS NULL AND user_id IN ?", ids, users).Order("id desc").Find(&subs)
		for _, sub := range subs {
			indexWorkSubmission(submissions, *sub.ExamID, sub)
		}
	}
	for _, exam := range exams {
		result[exam.ID] = make(map[uint]workSummary, len(users))
		for _, user := range users {
			result[exam.ID][user] = summarizeWork(links[exam.ID], submissions[exam.ID][user], examManualReview(exam), attempted[exam.ID][user], includeProblems, examScoringRule(exam))
		}
	}
	return result
}

func (s Server) examSummary(examID uint, userID uint, includeProblems bool) workSummary {
	var exam models.Exam
	_ = s.DB.First(&exam, examID).Error
	exam.ID = examID
	return s.examSummaries([]models.Exam{exam}, []uint{userID}, includeProblems)[examID][userID]
}

func indexWorkSubmission(index map[uint]map[uint]map[uint][]models.Submission, workID uint, sub models.Submission) {
	if index[workID] == nil {
		index[workID] = map[uint]map[uint][]models.Submission{}
	}
	if index[workID][sub.UserID] == nil {
		index[workID][sub.UserID] = map[uint][]models.Submission{}
	}
	index[workID][sub.UserID][sub.ProblemID] = append(index[workID][sub.UserID][sub.ProblemID], sub)
}

// userProblemSubmissionState is one user's latest verdict per problem in a scope.
type userProblemSubmissionState struct {
	ProblemID uint
	Status    models.SubmissionStatus
	CreatedAt time.Time
	Accepted  bool
}

// userProblemSubmissionStates replaces per-problem lookups on polled pages.
// scopeColumn is a code constant, never request input.
func (s Server) userProblemSubmissionStates(scopeColumn string, scopeID uint, userID uint) (map[uint]userProblemSubmissionState, error) {
	var rows []userProblemSubmissionState
	err := s.DB.Raw(`
SELECT DISTINCT ON (problem_id) problem_id, status, created_at,
       BOOL_OR(status = ?) OVER (PARTITION BY problem_id) AS accepted
FROM submissions
WHERE user_id = ? AND `+scopeColumn+` = ?
ORDER BY problem_id, created_at DESC, id DESC`, models.StatusAccepted, userID, scopeID).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	states := make(map[uint]userProblemSubmissionState, len(rows))
	for _, row := range rows {
		states[row.ProblemID] = row
	}
	return states, nil
}

// loadUserAvatar fills the column the authentication middleware leaves out.
func (s Server) loadUserAvatar(ctx context.Context, user *models.User) error {
	var row models.User
	if err := s.DB.WithContext(ctx).Select("id", "avatar_url").First(&row, user.ID).Error; err != nil {
		return err
	}
	user.AvatarURL = row.AvatarURL
	return nil
}
