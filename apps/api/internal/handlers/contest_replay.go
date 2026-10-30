package handlers

import (
	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"school-oj/apps/api/internal/authz"
	"school-oj/apps/api/internal/middleware"
	"school-oj/apps/api/internal/models"
	"sort"
	"time"
)

func (s Server) getContestReplay(c *gin.Context) {
	p, ok := s.eventPaperByParams(c, true)
	if !ok {
		return
	}
	s.exportContestReplay(c, p)
}

func (s Server) exportContestReplay(c *gin.Context, p eventPaper) {
	user, _ := middleware.CurrentUser(c)
	if !authz.Has(user, authz.EventReview) {
		eventError(c, eventFail(403, "仅系统管理员可进行赛后滚榜"))
		return
	}
	if p.kind != "contest" {
		eventError(c, eventFail(400, "滚榜数据包仅支持比赛"))
		return
	}
	scoped, closeSnapshot, err := s.rankingReadSnapshot(c.Request.Context())
	if err != nil {
		eventError(c, err)
		return
	}
	defer closeSnapshot()
	p, err = loadEventPaper(scoped.DB, p.kind, p.id, false)
	if err != nil {
		eventError(c, err)
		return
	}
	if err = scoped.rankingExportReady(p); err != nil {
		eventError(c, err)
		return
	}
	contest := p.contest
	all := contest
	all.RankingUserIDs = datatypes.JSON("null")
	links, users, err := scoped.loadContestRankingParticipants(all)
	if err != nil {
		eventError(c, err)
		return
	}
	start := contest.CreatedAt
	if contest.StartsAt != nil {
		start = *contest.StartsAt
	}
	end := p.ends()
	if end == nil {
		eventError(c, eventFail(409, "缺少截止时间"))
		return
	}
	freeze := *end
	if contest.FreezeEnabled {
		freeze = end.Add(-time.Duration(contest.FreezeDurationMinutes) * time.Minute)
		if freeze.Before(start) {
			freeze = start
		}
	}
	before, err := scoped.loadTeamContestCellAggregates(contest, freeze)
	if err != nil {
		eventError(c, err)
		return
	}
	after, err := scoped.loadTeamContestCellAggregates(contest, *end)
	if err != nil {
		eventError(c, err)
		return
	}
	initial := buildTeamContestRanking(users, links, before, start, contest.ScoringRule)
	final := buildTeamContestRanking(users, links, after, start, contest.ScoringRule)
	if err := markContestRankingRows(initial, contest); err != nil {
		eventError(c, err)
		return
	}
	if err := markContestRankingRows(final, contest); err != nil {
		eventError(c, err)
		return
	}
	problems := []gin.H{}
	for _, link := range links {
		problems = append(problems, gin.H{"problem_id": link.ProblemID, "label": link.Label, "title": link.Problem.Title, "score": 100})
	}
	var events []struct {
		ID        uint                    `json:"id"`
		UserID    uint                    `json:"user_id"`
		ProblemID uint                    `json:"problem_id"`
		Status    models.SubmissionStatus `json:"status"`
		Score     int                     `json:"score"`
		CreatedAt time.Time               `json:"created_at"`
	}
	if err = scoped.DB.Model(&models.Submission{}).Select("id,user_id,problem_id,status,COALESCE(manual_score,score) AS score,created_at").Where("team_contest_id=? AND created_at >= ? AND created_at < ?", p.id, freeze, *end).Order("created_at,id").Find(&events).Error; err != nil {
		eventError(c, err)
		return
	}
	c.Header("Cache-Control", "private, no-store")

	c.JSON(200, gin.H{"schema_version": 1, "kind": "contest-replay", "contest": contest, "problems": problems, "initial_rows": initial, "final_rows": final, "events": events, "freeze_at": freeze, "generated_at": time.Now()})
}

func markContestRankingRows(rows []teamContestRankingRow, contest models.TeamContest) error {
	ids, selected, err := contestRankingIDs(contest.RankingUserIDs)
	if err != nil {
		return err
	}
	eligible := map[uint]bool{}
	for _, id := range ids {
		eligible[id] = true
	}
	count, priorRank, lastRank := 0, -1, 0
	for i := range rows {
		rows[i].Unranked = selected && !eligible[rows[i].UserID]
		if rows[i].Unranked {
			rows[i].Rank = 0
			continue
		}
		count++
		original := rows[i].Rank
		if original != priorRank {
			lastRank = count
			priorRank = original
		}
		rows[i].Rank = lastRank
	}
	sort.SliceStable(rows, func(i, j int) bool { return !rows[i].Unranked && rows[j].Unranked })
	return nil
}
