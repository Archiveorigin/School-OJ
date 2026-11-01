package handlers

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"school-oj/apps/api/internal/middleware"
	"school-oj/apps/api/internal/models"
	"school-oj/apps/api/internal/services"
	"strings"
	"time"
)

type eventAnnouncement struct {
	Attachments []models.EventAttachment `json:"attachments" gorm:"-"`
	ID          uint                     `json:"id"`
	EventKind   string                   `json:"event_kind"`
	EventID     uint                     `json:"event_id"`
	ProblemID   *uint                    `json:"problem_id"`
	Body        string                   `json:"body"`
	CreatedBy   uint                     `json:"created_by"`
	CreatedAt   time.Time                `json:"created_at"`
}

func (s Server) listEventAnnouncements(c *gin.Context) {
	p, ok := s.eventPaperByParams(c, false)
	if !ok {
		return
	}
	items := []eventAnnouncement{}
	if err := s.DB.Where("event_kind=? AND event_id=?", p.kind, p.id).Order("id desc").Find(&items).Error; err != nil {
		eventError(c, err)
		return
	}
	ids := make([]uint, len(items))
	for i := range items {
		ids[i] = items[i].ID
		items[i].Attachments = []models.EventAttachment{}
	}
	if len(ids) > 0 {
		var attachments []models.EventAttachment
		if err := s.DB.Where("announcement_id IN ? AND event_kind=? AND event_id=?", ids, p.kind, p.id).Order("id").Find(&attachments).Error; err != nil {
			eventError(c, err)
			return
		}
		byAnnouncement := make(map[uint][]models.EventAttachment, len(ids))
		for _, attachment := range attachments {
			byAnnouncement[*attachment.AnnouncementID] = append(byAnnouncement[*attachment.AnnouncementID], attachment)
		}
		for i := range items {
			if found := byAnnouncement[items[i].ID]; found != nil {
				items[i].Attachments = found
			}
		}
	}
	c.Header("Cache-Control", "private, no-store")
	c.JSON(200, items)
}
func (s Server) publishEventAnnouncement(c *gin.Context) {
	p, ok := s.eventPaperByParams(c, true)
	if !ok {
		return
	}
	var req struct {
		Body      string `json:"body"`
		ProblemID *uint  `json:"problem_id"`
	}
	if !bind(c, &req) {
		return
	}
	req.Body = strings.TrimSpace(req.Body)
	if len([]rune(req.Body)) == 0 || len([]rune(req.Body)) > 4000 {
		c.JSON(400, gin.H{"error": "公告正文须为 1 至 4000 字"})
		return
	}
	u, _ := middleware.CurrentUser(c)
	item := eventAnnouncement{EventKind: p.kind, EventID: p.id, ProblemID: req.ProblemID, Body: req.Body, CreatedBy: u.ID, CreatedAt: time.Now()}
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		locked, err := loadEventPaper(tx, p.kind, p.id, true)
		if err != nil {
			return err
		}
		scoped := s
		scoped.DB = tx
		if !scoped.canManageEventPaper(u, locked) {
			return eventFail(403, "管理权限已变化")
		}
		if !locked.review().Approved() || locked.review().CompletedAt != nil {
			return eventFail(409, "仅已审核且未完结场次可以发布公告")
		}
		if req.ProblemID != nil {
			table, field := "exam_problems", "exam_id"
			if p.kind == "contest" {
				table, field = "team_contest_problems", "contest_id"
			}
			var count int64
			if err := tx.Table(table).Where(field+"=? AND problem_id=?", p.id, *req.ProblemID).Count(&count).Error; err != nil {
				return err
			}
			if count == 0 {
				return eventFail(400, "关联题目不属于本场")
			}
		}
		return tx.Create(&item).Error
	})
	if err != nil {
		eventError(c, err)
		return
	}
	services.Audit(c, s.DB, "event.announcement.publish", p.kind, p.id, nil)
	c.JSON(200, item)
}
