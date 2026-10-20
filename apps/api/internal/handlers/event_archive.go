package handlers

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"school-oj/apps/api/internal/middleware"
	"school-oj/apps/api/internal/models"
	"strconv"
	"strings"
	"time"
)

type eventListRow struct {
	Kind           string     `json:"kind"`
	ID             uint       `json:"id"`
	Title          string     `json:"title"`
	ContextName    string     `json:"context_name"`
	StartsAt       *time.Time `json:"starts_at"`
	EndsAt         *time.Time `json:"ends_at"`
	ScoringRule    string     `json:"scoring_rule"`
	ReviewStatus   string     `json:"review_status"`
	ReviewRevision int        `json:"review_revision"`
	CompletedAt    *time.Time `json:"completed_at"`
	Lifecycle      string     `json:"lifecycle"`
	ProblemCount   int64      `json:"problem_count"`
}

func (s Server) eventListQuery(user models.User, archive bool) *gorm.DB {
	exams := s.DB.Table("exams e").Select(`'exam' AS kind,e.id,e.title,c.name AS context_name,e.starts_at,e.ends_at,e.scoring_rule,e.review_status,e.review_revision,e.completed_at,(SELECT count(*) FROM exam_problems ep WHERE ep.exam_id=e.id) AS problem_count`).Joins("JOIN courses c ON c.id=e.course_id").Where("e.deleted_at IS NULL")
	contests := s.DB.Table("team_contests e").Select(`'contest' AS kind,e.id,e.title,t.name AS context_name,e.starts_at,e.starts_at+e.duration_minutes*interval '1 minute' AS ends_at,e.scoring_rule,e.review_status,e.review_revision,e.completed_at,(SELECT count(*) FROM team_contest_problems ep WHERE ep.contest_id=e.id) AS problem_count`).Joins("JOIN teams t ON t.id=e.team_id").Where("e.deleted_at IS NULL")
	if archive {
		exams = exams.Where("e.completed_at IS NOT NULL AND e.review_status='approved' AND e.ends_at IS NOT NULL AND e.ends_at<=?", time.Now())
		contests = contests.Where("e.completed_at IS NOT NULL AND e.review_status='approved' AND e.state<>'draft' AND e.starts_at IS NOT NULL AND e.starts_at+e.duration_minutes*interval '1 minute'<=?", time.Now())
		if user.Role != models.RoleAdmin {
			if user.Role == models.RoleTeacher {
				exams = exams.Where("c.teacher_id=? OR EXISTS(SELECT 1 FROM course_memberships m WHERE m.course_id=e.course_id AND m.user_id=? AND m.role IN ?)", user.ID, user.ID, courseTeachingRoles())
			} else {
				exams = exams.Where("(e.class_id IS NOT NULL AND EXISTS(SELECT 1 FROM class_memberships m WHERE m.class_id=e.class_id AND m.user_id=?)) OR (e.class_id IS NULL AND EXISTS(SELECT 1 FROM course_memberships m WHERE m.course_id=e.course_id AND m.user_id=? AND m.role='student'))", user.ID, user.ID)
			}
			contests = contests.Where("EXISTS(SELECT 1 FROM team_memberships m WHERE m.team_id=e.team_id AND m.user_id=?)", user.ID)
		}
	}
	union := s.DB.Table("(? UNION ALL ?) AS events", exams, contests).Select(`events.*, CASE WHEN completed_at IS NOT NULL THEN 'completed' WHEN review_status<>'approved' THEN review_status WHEN starts_at>NOW() THEN 'upcoming' WHEN ends_at<=NOW() THEN CASE WHEN completed_at IS NULL THEN 'closed' ELSE 'completed' END ELSE 'running' END AS lifecycle`)
	return s.DB.Table("(?) AS event_states", union)
}
func (s Server) listEventArchives(c *gin.Context) { s.listEvents(c, true) }
func (s Server) listEventReviews(c *gin.Context)  { s.listEvents(c, false) }
func (s Server) listEvents(c *gin.Context, archive bool) {
	u, _ := middleware.CurrentUser(c)
	q := s.eventListQuery(u, archive)
	page, size := eventPage(c)
	if kind := c.Query("kind"); kind != "" {
		if kind != "exam" && kind != "contest" {
			c.JSON(400, gin.H{"error": "类型无效"})
			return
		}
		q = q.Where("kind=?", kind)
	}
	if status := c.Query("status"); status != "" && !archive {
		if status == "approved" {
			q = q.Where("review_status=?", status)
		} else {
			q = q.Where("lifecycle=?", status)
		}
	}
	if search := strings.TrimSpace(c.Query("q")); search != "" {
		q = q.Where("title ILIKE ?", "%"+strings.NewReplacer("\\", "\\\\", "%", "\\%", "_", "\\_").Replace(search)+"%")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		eventError(c, err)
		return
	}
	items := []eventListRow{}
	order := "ends_at desc,id desc,kind"
	if !archive {
		order = "CASE WHEN lifecycle='pending' THEN 0 WHEN lifecycle='closed' THEN 1 ELSE 2 END, starts_at asc NULLS LAST,id desc,kind"
	}
	if err := q.Order(order).Offset((page - 1) * size).Limit(size).Scan(&items).Error; err != nil {
		eventError(c, err)
		return
	}
	c.Header("Cache-Control", "private, no-store")
	c.JSON(200, gin.H{"items": items, "total": total, "page": page, "page_size": size})
}
func eventPage(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if page > 100000 {
		page = 100000
	}
	if size < 1 {
		size = 20
	}
	if size > 100 {
		size = 100
	}
	return page, size
}
func (s Server) getEventArchive(c *gin.Context) {
	p, ok := s.eventPaperByParams(c, false)
	if !ok {
		return
	}
	if !p.archived(time.Now()) {
		c.JSON(404, gin.H{"error": "该场次尚未有效完结，不能归档"})
		return
	}
	problems, err := eventPaperProblems(s.DB, p)
	if err != nil {
		eventError(c, err)
		return
	}
	u, _ := middleware.CurrentUser(c)
	canRank := true
	if p.kind == "exam" {
		canRank = s.canManageCourse(u, p.exam.CourseID) || examRankingVisible(p.exam) && examScoringRule(p.exam) != "oi"
	} else if p.contest.ScoringRule == "oi" {
		canRank = s.canManageEventPaper(u, p)
	}
	c.Header("Cache-Control", "private, no-store")
	c.JSON(200, gin.H{"event": p.model(), "kind": p.kind, "problems": problems, "can_view_ranking": canRank, "can_submit": false})
}
