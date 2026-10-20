package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"school-oj/apps/api/internal/models"
)

type eventStandings struct {
	Kind        string                `json:"kind"`
	EventID     uint                  `json:"event_id"`
	Title       string                `json:"title"`
	ScoringRule string                `json:"scoring_rule"`
	Lifecycle   string                `json:"lifecycle"`
	StartsAt    *time.Time            `json:"starts_at"`
	EndsAt      *time.Time            `json:"ends_at"`
	GeneratedAt time.Time             `json:"generated_at"`
	Checks      eventCompletionChecks `json:"completion_checks"`
	Problems    []examRankingProblem  `json:"problems"`
	Rows        []examRankingRow      `json:"rows"`
}

func (s Server) loadExamRankingStudents(item models.Exam) ([]models.User, error) {
	users := []models.User{}
	q := s.DB.Model(&models.User{}).Where("users.account_deleted = false")
	if item.ParticipantsFrozen {
		q = q.Where("id IN (?)", s.DB.Model(&models.ExamParticipant{}).Select("user_id").Where("exam_id=?", item.ID))
	} else if item.ClassID != nil {
		q = q.Where("users.role = ? AND id IN (?)", models.RoleStudent, s.DB.Model(&models.ClassMembership{}).Select("user_id").Where("class_id=?", *item.ClassID))
	} else {
		q = q.Where("users.role = ? AND id IN (?)", models.RoleStudent, s.DB.Model(&models.CourseMembership{}).Select("user_id").Where("course_id=? AND role=?", item.CourseID, models.RoleStudent))
	}
	err := q.Order("id").Find(&users).Error
	return users, err
}

func (s Server) loadContestRankingParticipants(contest models.TeamContest) ([]models.TeamContestProblem, []models.User, error) {
	links := []models.TeamContestProblem{}
	if err := s.DB.Preload("Problem").Preload("ProblemVersion").Where("contest_id = ?", contest.ID).Order("sort_order,id").Find(&links).Error; err != nil {
		return nil, nil, err
	}
	hydrateTeamContestProblemVersions(links)
	links = filterActiveTeamContestProblemLinks(links)
	normalizeTeamContestProblemLabels(links)
	users := []models.User{}
	q := s.DB.Model(&models.User{}).Where("account_deleted = false")
	if contest.State == models.TeamContestDraft {
		q = q.Where("id IN (?)", s.DB.Model(&models.TeamMembership{}).Select("user_id").Where("team_id=?", contest.TeamID))
	} else {
		q = q.Where("id IN (?)", s.DB.Model(&models.TeamContestParticipant{}).Select("user_id").Where("contest_id=?", contest.ID))
	}
	err := q.Order("id").Find(&users).Error
	return links, users, err
}

func (s Server) rankingExportReady(p eventPaper) error {
	state := p.lifecycle(time.Now())
	if state != "closed" && state != "completed" {
		return eventFail(http.StatusConflict, "仅已截止或已完结的场次可以导出实际排名")
	}
	checks, err := completionChecks(s.DB, p)
	if err != nil {
		return err
	}
	if checks.Judging > 0 || checks.Ungraded > 0 {
		return eventFail(http.StatusConflict, "仍有评测或人工评分未完成，请完成后再导出实际排名")
	}
	if checks.Failed > 0 {
		return eventFail(http.StatusConflict, "仍有判题系统异常，请处理并重新评测后再导出实际排名")
	}
	return nil
}

// Read one consistent database snapshot. A worker or manual grader finishing
// during export cannot mix old readiness flags with newer score rows.
func (s Server) eventStandingsSnapshot(p eventPaper, exporting bool) (eventStandings, error) {
	var result eventStandings
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		scoped := s
		scoped.DB = tx
		current, err := loadEventPaper(tx, p.kind, p.id, false)
		if err != nil {
			return err
		}
		if exporting {
			if err := scoped.rankingExportReady(current); err != nil {
				return err
			}
		}
		result, err = scoped.buildEventStandings(current, time.Now())
		return err
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	return result, err
}

func (s Server) checkedExamMarkdownReport(p eventPaper) (string, error) {
	var body string
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		scoped := s
		scoped.DB = tx
		current, err := loadEventPaper(tx, "exam", p.id, false)
		if err != nil {
			return err
		}
		if err := scoped.rankingExportReady(current); err != nil {
			return err
		}
		item := current.exam
		if err := tx.Preload("Problem").Preload("ProblemVersion").Where("exam_id=?", item.ID).Order("sort_order,id").Find(&item.Problems).Error; err != nil {
			return err
		}
		hydrateExamProblemVersions(&item)
		students, err := scoped.loadExamRankingStudents(item)
		if err != nil {
			return err
		}
		body, err = scoped.buildExamMarkdownReport(item, students)
		return err
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	return body, err
}

func (s Server) buildEventStandings(p eventPaper, now time.Time) (eventStandings, error) {
	result := eventStandings{
		Kind: p.kind, EventID: p.id, Lifecycle: p.lifecycle(now), StartsAt: p.starts(), EndsAt: p.ends(), GeneratedAt: now,
		Problems: []examRankingProblem{}, Rows: []examRankingRow{},
	}
	var err error
	result.Checks, err = completionChecks(s.DB, p)
	if err != nil {
		return result, err
	}
	if p.kind == "exam" {
		item := p.exam
		if err := s.DB.Preload("Problem").Preload("ProblemVersion").Where("exam_id=?", item.ID).Order("sort_order,id").Find(&item.Problems).Error; err != nil {
			return result, err
		}
		hydrateExamProblemVersions(&item)
		students, err := s.loadExamRankingStudents(item)
		if err != nil {
			return result, err
		}
		result.Title, result.ScoringRule = item.Title, examScoringRule(item)
		for i, link := range item.Problems {
			label := link.Label
			if strings.TrimSpace(label) == "" {
				label = defaultProblemLabel(i)
			}
			result.Problems = append(result.Problems, examRankingProblem{link.ProblemID, label, link.Problem.DisplayCode, link.Problem.Title, link.Score})
		}
		result.Rows, _, _, err = s.examRankingRows(item, students, result.ScoringRule, nil)
		return result, err
	}
	links, users, err := s.loadContestRankingParticipants(p.contest)
	if err != nil {
		return result, err
	}
	aggregates, err := s.loadTeamContestCellAggregates(p.contest, now)
	if err != nil {
		return result, err
	}
	start := p.contest.CreatedAt
	if p.starts() != nil {
		start = *p.starts()
	}
	result.Title = p.contest.Title
	result.ScoringRule, _ = parseScoringRule(p.contest.ScoringRule)
	rows := buildTeamContestRanking(users, links, aggregates, start, result.ScoringRule)
	for _, link := range links {
		result.Problems = append(result.Problems, examRankingProblem{link.ProblemID, link.Label, link.Problem.DisplayCode, link.Problem.Title, 100})
	}
	for _, row := range rows {
		converted := examRankingRow{
			Rank: row.Rank, UserID: row.UserID, Name: row.Name, TotalScore: row.TotalScore, MaxScore: row.MaxScore,
			Solved: row.Solved, PenaltyMinutes: row.PenaltyMinutes, SubmissionCount: row.SubmissionCount,
			LastSubmission: row.LastSubmission, Problems: []examRankingCell{},
		}
		for i, cell := range row.Problems {
			converted.Problems = append(converted.Problems, examRankingCell{
				ProblemID: cell.ProblemID, Label: links[i].Label, BestScore: cell.BestScore, MaxScore: cell.MaxScore,
				Status: models.SubmissionStatus(cell.Status), Pending: cell.Pending, ScoreReady: !cell.Pending,
				Attempts: cell.Attempts, WrongAttempts: cell.WrongAttempts, ElapsedMinutes: cell.ElapsedMinutes,
				SubmittedAt: cell.SolvedAt, firstAccepted: cell.SolvedAt, Fastest: cell.Fastest,
			})
			if cell.Attempts > 0 {
				converted.Attempted++
			}
			if cell.Pending {
				converted.PendingCount++
			}
		}
		converted.ScoreReady = converted.PendingCount == 0
		result.Rows = append(result.Rows, converted)
	}
	return result, nil
}

func (s Server) getEventRanking(c *gin.Context) {
	p, ok := s.eventPaperByParams(c, true)
	if !ok {
		return
	}
	data, err := s.eventStandingsSnapshot(p, false)
	if err != nil {
		eventError(c, err)
		return
	}
	c.Header("Cache-Control", "private, no-store")
	c.JSON(http.StatusOK, data)
}

func (s Server) exportEventRanking(c *gin.Context) {
	p, ok := s.eventPaperByParams(c, true)
	if !ok {
		return
	}
	s.exportEventRankingForPaper(c, p, strings.ToLower(c.DefaultQuery("format", "xlsx")))
}

func (s Server) exportEventRankingForPaper(c *gin.Context, p eventPaper, format string) {
	if format != "xlsx" && format != "html" {
		eventError(c, eventFail(400, "仅支持 Excel 或 HTML 榜单"))
		return
	}
	data, err := s.eventStandingsSnapshot(p, true)
	if err != nil {
		eventError(c, err)
		return
	}
	var body []byte
	contentType := "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	if format == "html" {
		body, err = renderStandingsHTML(data)
		contentType = "text/html; charset=utf-8"
	} else {
		body, err = renderStandingsXLSX(data)
	}
	if err != nil {
		eventError(c, err)
		return
	}
	c.Header("Cache-Control", "private, no-store")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"standings-%d.%s\"", p.id, format))
	c.Data(http.StatusOK, contentType, body)
}
