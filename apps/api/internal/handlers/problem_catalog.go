package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"school-oj/apps/api/internal/middleware"
	"school-oj/apps/api/internal/models"
	"school-oj/apps/api/internal/visibility"

	"github.com/gin-gonic/gin"
)

type problemCatalogItem struct {
	models.Problem
	ProgressStatus string  `json:"progress_status"`
	PassRate       float64 `json:"pass_rate"`
	AcceptedCount  int64   `json:"accepted_count"`
	EvaluatedCount int64   `json:"evaluated_count"`
}

func (s Server) listProblemCatalog(c *gin.Context) {
	user, authed := middleware.CurrentUser(c)
	var teamID uint64
	if raw := c.Query("team_id"); raw != "" {
		var err error
		teamID, err = strconv.ParseUint(raw, 10, 64)
		if err != nil || teamID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "团队编号无效"})
			return
		}
		if !authed {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "请先登录"})
			return
		}
		var team models.Team
		if err := s.DB.First(&team, teamID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "团队不存在"})
			return
		}
		if !s.requireTeamContentPermission(c, user, team) {
			return
		}
	}
	page := catalogInt(c.Query("page"), 1)
	if page > 1000000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "页码超出范围"})
		return
	}
	pageSize := catalogInt(c.Query("page_size"), 20)
	if pageSize > 100 {
		pageSize = 100
	}
	if pageSize < 1 {
		pageSize = 20
	}

	db := s.DB.WithContext(c.Request.Context())
	q := db.Model(&models.Problem{}).
		Where("problems.deleted_at IS NULL").
		Where("problems.archived_at IS NULL")
	if teamID > 0 {
		q = q.Where("("+publicProblemSQL()+") OR problems.team_id = ?", teamID)
	} else {
		q = q.Where(publicProblemSQL())
	}
	if keyword := strings.ToLower(strings.TrimSpace(c.Query("q"))); keyword != "" {
		if len(keyword) > 1024 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "搜索内容过长"})
			return
		}
		like := "%" + keyword + "%"
		q = q.Where("lower(problems.title) LIKE ? OR lower(problems.display_code) LIKE ? OR lower(problems.slug) LIKE ?", like, like, like)
	}
	if difficulty := strings.TrimSpace(c.Query("difficulty")); difficulty != "" {
		q = q.Where("problems.difficulty = ?", difficulty)
	}
	tags := parseTagFields(c.QueryArray("tag"), c.Query("tags"))
	if len(tags) > 64 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "最多筛选 64 个标签"})
		return
	}
	if len(tags) > 0 {
		conditions := make([]string, 0, len(tags))
		args := make([]any, 0, len(tags))
		for _, tag := range tags {
			conditions = append(conditions, fmt.Sprintf("problems.tags @> ?::jsonb"))
			args = append(args, fmt.Sprintf(`{"labels":[%s]}`, strconv.Quote(tag)))
		}
		q = q.Where("("+strings.Join(conditions, " OR ")+")", args...)
	}
	status := strings.TrimSpace(c.Query("status"))
	if status != "" && authed {
		switch status {
		case string(models.ProgressAccepted), string(models.ProgressAttempted):
			q = q.Joins("JOIN problem_progresses catalog_progress ON catalog_progress.problem_id = problems.id AND catalog_progress.user_id = ?", user.ID).
				Where("catalog_progress.status = ?", status)
		case string(models.ProgressUnattempted):
			q = q.Joins("LEFT JOIN problem_progresses catalog_progress ON catalog_progress.problem_id = problems.id AND catalog_progress.user_id = ?", user.ID).
				Where("catalog_progress.id IS NULL OR catalog_progress.status = ?", models.ProgressUnattempted)
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "status must be unattempted, attempted or accepted"})
			return
		}
	}

	var total int64
	if err := q.Distinct("problems.id").Count(&total).Error; err != nil {
		catalogQueryError(c, err)
		return
	}
	var problems []models.Problem
	if err := q.Select("problems.*").Distinct().Order("problems.id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&problems).Error; err != nil {
		catalogQueryError(c, err)
		return
	}

	progress := map[uint]models.ProblemProgress{}
	ids := make([]uint, 0, len(problems))
	for _, problem := range problems {
		ids = append(ids, problem.ID)
	}
	if authed && len(ids) > 0 {
		var rows []models.ProblemProgress
		if err := db.Where("user_id = ? AND problem_id IN ?", user.ID, ids).Find(&rows).Error; err != nil {
			catalogQueryError(c, err)
			return
		}
		for _, row := range rows {
			progress[row.ProblemID] = row
		}
	}
	// One grouped query for the current page, instead of two COUNTs per problem.
	type counts struct {
		ProblemID      uint
		AcceptedCount  int64
		EvaluatedCount int64
	}
	statistics := map[uint]counts{}
	if len(ids) > 0 {
		var rows []counts
		err := db.Model(&models.Submission{}).Scopes(visibility.PublicPractice).
			Select("problem_id, COUNT(*) FILTER (WHERE status = ?) AS accepted_count, COUNT(*) FILTER (WHERE status NOT IN ?) AS evaluated_count",
				models.StatusAccepted, []models.SubmissionStatus{models.StatusQueued, models.StatusRunning, models.StatusPendingReview, models.StatusSystemError}).
			Where("problem_id IN ?", ids).Group("problem_id").Scan(&rows).Error
		if err != nil {
			catalogQueryError(c, err)
			return
		}
		for _, row := range rows {
			statistics[row.ProblemID] = row
		}
	}
	items := make([]problemCatalogItem, 0, len(problems))
	for _, problem := range problems {
		item := problemCatalogItem{Problem: problem, ProgressStatus: string(models.ProgressUnattempted)}
		if row, ok := progress[problem.ID]; ok {
			item.ProgressStatus = string(row.Status)
		}
		item.AcceptedCount = statistics[problem.ID].AcceptedCount
		item.EvaluatedCount = statistics[problem.ID].EvaluatedCount
		if item.EvaluatedCount > 0 {
			item.PassRate = float64(item.AcceptedCount) * 100 / float64(item.EvaluatedCount)
		}
		items = append(items, item)
	}

	var all []models.Problem
	if err := db.Model(&models.Problem{}).Select("tags").Where("deleted_at IS NULL AND archived_at IS NULL").Where(publicProblemSQL()).Find(&all).Error; err != nil {
		catalogQueryError(c, err)
		return
	}
	availableTags := collectProblemTags(all)
	c.JSON(http.StatusOK, gin.H{
		"items": items, "total": total, "page": page, "page_size": pageSize,
		"available_tags": availableTags,
	})
}

func catalogQueryError(c *gin.Context, err error) {
	_ = c.Error(err)
	c.JSON(http.StatusServiceUnavailable, gin.H{"error": "题库暂不可用，请稍后重试"})
}

func collectProblemTags(problems []models.Problem) []string {
	seen := map[string]bool{}
	var tags []string
	for _, problem := range problems {
		labels, ok := problem.Tags["labels"].([]any)
		if !ok {
			if stringsList, ok := problem.Tags["labels"].([]string); ok {
				for _, tag := range stringsList {
					if !seen[tag] {
						seen[tag] = true
						tags = append(tags, tag)
					}
				}
			}
			continue
		}
		for _, raw := range labels {
			tag := strings.TrimSpace(fmt.Sprint(raw))
			if tag != "" && !seen[tag] {
				seen[tag] = true
				tags = append(tags, tag)
			}
		}
	}
	return tags
}

func catalogInt(raw string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value < 1 {
		return fallback
	}
	return value
}
