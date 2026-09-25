package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"school-oj/apps/api/internal/models"
)

func (s Server) addTeamProblemSetBatch(c *gin.Context, user models.User, set models.TeamProblemSet, team models.Team, requested []uint) {
	if len(requested) == 0 || len(requested) > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "每次请选择 1 至 100 道题目"})
		return
	}
	ids, seen := []uint{}, map[uint]bool{}
	for _, id := range requested {
		if id == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "题目编号无效"})
			return
		}
		if !seen[id] {
			ids = append(ids, id)
			seen[id] = true
		}
	}
	var problems []models.Problem
	if err := s.DB.Where("id IN ? AND deleted_at IS NULL AND archived_at IS NULL", ids).Find(&problems).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取题目失败"})
		return
	}
	if len(problems) != len(ids) {
		c.JSON(http.StatusNotFound, gin.H{"error": "部分题目已删除或归档，请刷新后重试"})
		return
	}
	for _, problem := range problems {
		if problem.TeamID != nil && *problem.TeamID != team.ID || problem.TeamID == nil && !s.isProblemPublic(problem.ID) && !s.canManageProblemData(user, problem) {
			c.JSON(http.StatusForbidden, gin.H{"error": "包含无权添加的题目，本次未添加任何题目"})
			return
		}
	}
	added, skipped := 0, 0
	err := s.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		// Serialize batches with deletion and other additions, then compute ordering.
		var locked models.TeamProblemSet
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND deleted_at IS NULL", set.ID).First(&locked).Error; err != nil {
			return err
		}
		var existing []models.TeamProblemSetProblem
		if err := tx.Where("problem_set_id = ?", set.ID).Find(&existing).Error; err != nil {
			return err
		}
		present, maxOrder := map[uint]bool{}, 0
		for _, link := range existing {
			present[link.ProblemID] = true
			if link.SortOrder > maxOrder {
				maxOrder = link.SortOrder
			}
		}
		links := []models.TeamProblemSetProblem{}
		for _, id := range ids {
			if present[id] {
				skipped++
				continue
			}
			maxOrder++
			links = append(links, models.TeamProblemSetProblem{ProblemSetID: set.ID, ProblemID: id, SortOrder: maxOrder})
		}
		added = len(links)
		if added == 0 {
			return nil
		}
		return tx.Create(&links).Error
	})
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "题单已变更，未添加题目，请刷新后重试"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"added": added, "skipped": skipped})
}
