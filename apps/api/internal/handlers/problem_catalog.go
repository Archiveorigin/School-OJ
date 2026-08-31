package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"school-oj/apps/api/internal/middleware"
	"school-oj/apps/api/internal/models"

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
	page := catalogInt(c.Query("page"), 1)
	pageSize := catalogInt(c.Query("page_size"), 20)
	if pageSize > 100 {
		pageSize = 100
	}
	if pageSize < 1 {
		pageSize = 20
	}

	q := s.DB.Model(&models.Problem{}).
		Where("problems.deleted_at IS NULL").
		Where("problems.archived_at IS NULL").
		Where(publicProblemSQL())
	if keyword := strings.ToLower(strings.TrimSpace(c.Query("q"))); keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("lower(problems.title) LIKE ? OR lower(problems.display_code) LIKE ? OR lower(problems.slug) LIKE ?", like, like, like)
	}
	if difficulty := strings.TrimSpace(c.Query("difficulty")); difficulty != "" {
		q = q.Where("problems.difficulty = ?", difficulty)
	}
	tags := parseTagFields(c.QueryArray("tag"), c.Query("tags"))
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var problems []models.Problem
	if err := q.Select("problems.*").Distinct().Order("problems.id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&problems).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	progress := map[uint]models.ProblemProgress{}
	if authed && len(problems) > 0 {
		ids := make([]uint, 0, len(problems))
		for _, problem := range problems {
			ids = append(ids, problem.ID)
		}
		var rows []models.ProblemProgress
		s.DB.Where("user_id = ? AND problem_id IN ?", user.ID, ids).Find(&rows)
		for _, row := range rows {
			progress[row.ProblemID] = row
		}
	}
	items := make([]problemCatalogItem, 0, len(problems))
	for _, problem := range problems {
		item := problemCatalogItem{Problem: problem, ProgressStatus: string(models.ProgressUnattempted)}
		if row, ok := progress[problem.ID]; ok {
			item.ProgressStatus = string(row.Status)
		}
		s.DB.Model(&models.Submission{}).Where("problem_id = ? AND status = ?", problem.ID, models.StatusAccepted).Count(&item.AcceptedCount)
		s.DB.Model(&models.Submission{}).
			Where("problem_id = ? AND status NOT IN ?", problem.ID, []models.SubmissionStatus{models.StatusQueued, models.StatusRunning, models.StatusPendingReview, models.StatusSystemError}).
			Count(&item.EvaluatedCount)
		if item.EvaluatedCount > 0 {
			item.PassRate = float64(item.AcceptedCount) * 100 / float64(item.EvaluatedCount)
		}
		items = append(items, item)
	}

	var all []models.Problem
	s.DB.Model(&models.Problem{}).Select("tags").Where("deleted_at IS NULL AND archived_at IS NULL").Where(publicProblemSQL()).Find(&all)
	availableTags := collectProblemTags(all)
	c.JSON(http.StatusOK, gin.H{
		"items": items, "total": total, "page": page, "page_size": pageSize,
		"available_tags": availableTags,
	})
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
