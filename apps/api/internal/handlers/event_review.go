package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"school-oj/apps/api/internal/middleware"
	"school-oj/apps/api/internal/models"
	"school-oj/apps/api/internal/services"
)

type eventFailure struct {
	status  int
	message string
}

func (e *eventFailure) Error() string            { return e.message }
func eventFail(status int, message string) error { return &eventFailure{status, message} }
func eventError(c *gin.Context, err error) {
	if c.Writer.Written() {
		return
	}
	var failure *eventFailure
	if errors.As(err, &failure) {
		c.JSON(failure.status, gin.H{"error": failure.message})
		return
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(404, gin.H{"error": "场次不存在或已删除"})
		return
	}
	c.JSON(500, gin.H{"error": "场次操作失败，请刷新后重试"})
}

type eventPaper struct {
	kind    string
	id      uint
	exam    models.Exam
	contest models.TeamContest
}

func loadEventPaper(db *gorm.DB, kind string, id uint, lock bool) (eventPaper, error) {
	p := eventPaper{kind: kind, id: id}
	q := db.Where("deleted_at IS NULL")
	if lock {
		q = q.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	switch kind {
	case "exam":
		err := q.First(&p.exam, id).Error
		return p, err
	case "contest":
		err := q.First(&p.contest, id).Error
		return p, err
	default:
		return p, eventFail(400, "场次类型无效")
	}
}
func (p *eventPaper) review() *models.EventReview {
	if p.kind == "exam" {
		return &p.exam.EventReview
	}
	return &p.contest.EventReview
}
func (p eventPaper) model() any {
	if p.kind == "exam" {
		return &p.exam
	}
	return &p.contest
}
func (p eventPaper) starts() *time.Time {
	if p.kind == "exam" {
		return p.exam.StartsAt
	}
	return p.contest.StartsAt
}
func (p eventPaper) ends() *time.Time {
	if p.kind == "exam" {
		return p.exam.EndsAt
	}
	return contestEndAt(p.contest)
}
func (p eventPaper) archived(now time.Time) bool {
	if p.lifecycle(now) != "completed" {
		return false
	}
	if p.kind == "exam" {
		return p.exam.DeletedAt == nil
	}
	return p.contest.DeletedAt == nil && p.contest.State != models.TeamContestDraft
}
func (s Server) canManageEventPaper(user models.User, p eventPaper) bool {
	if p.kind == "exam" {
		return s.canManageCourse(user, p.exam.CourseID)
	}
	var team models.Team
	if s.DB.First(&team, p.contest.TeamID).Error != nil {
		return false
	}
	return s.canManageTeam(user, team) || p.contest.CreatedBy == user.ID && s.canTeamContentPermission(user, team)
}
func (s Server) canReadEventPaper(user models.User, p eventPaper, now time.Time) bool {
	if s.canManageEventPaper(user, p) {
		return true
	}
	if !p.review().Approved() || p.starts() != nil && now.Before(*p.starts()) {
		return false
	}
	if p.lifecycle(now) == "closed" {
		return false
	}
	if p.kind == "exam" {
		if !s.canAccessExam(user, p.exam) {
			return false
		}
		if p.archived(now) {
			return true
		}
		var count int64
		s.DB.Model(&models.ExamAttempt{}).Where("exam_id=? AND user_id=? AND finished_at IS NULL", p.id, user.ID).Count(&count)
		return count > 0
	}
	var member, participant int64
	s.DB.Model(&models.TeamMembership{}).Where("team_id=? AND user_id=?", p.contest.TeamID, user.ID).Count(&member)
	if member == 0 {
		return false
	}
	if p.archived(now) {
		return true
	}
	s.DB.Model(&models.TeamContestParticipant{}).Where("contest_id=? AND user_id=?", p.id, user.ID).Count(&participant)
	return participant > 0
}
func (s Server) eventPaperByParams(c *gin.Context, manage bool) (eventPaper, bool) {
	id, ok := idParam(c, "event_id")
	if !ok {
		return eventPaper{}, false
	}
	p, err := loadEventPaper(s.DB, c.Param("kind"), id, false)
	if err != nil {
		eventError(c, err)
		return p, false
	}
	u, _ := middleware.CurrentUser(c)
	if manage && !s.canManageEventPaper(u, p) || !manage && !s.canReadEventPaper(u, p, time.Now()) {
		c.JSON(403, gin.H{"error": "无权访问本场内容，或场次尚未审核开始"})
		return p, false
	}
	return p, true
}
func requireEventDraft(p eventPaper, revision int) error {
	if p.review().ReviewRevision != revision {
		return eventFail(409, "场次已变更，请刷新后重试")
	}
	if !p.review().Editable() {
		return eventFail(409, "整场内容已锁定；开赛前请先撤回审核再修改")
	}
	return nil
}
func bumpEventRevision(tx *gorm.DB, p eventPaper) error {
	return tx.Model(p.model()).Updates(map[string]any{"review_revision": gorm.Expr("review_revision + 1"), "review_status": "draft", "review_note": "", "reviewed_at": nil, "reviewed_by": nil}).Error
}

type eventPaperProblem struct {
	ProblemID         uint                               `json:"problem_id"`
	ProblemVersionID  uint                               `json:"problem_version_id"`
	Label             string                             `json:"label"`
	Score             int                                `json:"score"`
	Problem           models.Problem                     `json:"problem"`
	CaseCount         int                                `json:"case_count"`
	Checker           services.CheckerManifest           `json:"checker"`
	StatementSections *services.ProblemStatementSections `json:"statement_sections,omitempty"`
}

func eventStatementSections(problem models.Problem) *services.ProblemStatementSections {
	data, err := json.Marshal(problem.Manifest["statement_sections"])
	if err != nil {
		return nil
	}
	var sections *services.ProblemStatementSections
	if json.Unmarshal(data, &sections) != nil {
		return nil
	}
	return sections
}

func eventPaperProblems(db *gorm.DB, p eventPaper) ([]eventPaperProblem, error) {
	out := []eventPaperProblem{}
	if p.kind == "exam" {
		var links []models.ExamProblem
		if err := db.Preload("Problem").Preload("ProblemVersion").Where("exam_id=?", p.id).Order("sort_order,id").Find(&links).Error; err != nil {
			return nil, err
		}
		for _, link := range links {
			applyProblemVersion(&link.Problem, link.ProblemVersion)
			out = append(out, eventPaperProblem{link.ProblemID, link.ProblemVersionID, link.Label, link.Score, link.Problem, len(problemTestCases(link.Problem.Manifest)), eventProblemChecker(link.Problem), eventStatementSections(link.Problem)})
		}
	} else {
		var links []models.TeamContestProblem
		if err := db.Preload("Problem").Preload("ProblemVersion").Where("contest_id=?", p.id).Order("sort_order,id").Find(&links).Error; err != nil {
			return nil, err
		}
		normalizeTeamContestProblemLabels(links)
		for _, link := range links {
			applyProblemVersion(&link.Problem, link.ProblemVersion)
			out = append(out, eventPaperProblem{link.ProblemID, link.ProblemVersionID, link.Label, 100, link.Problem, len(problemTestCases(link.Problem.Manifest)), eventProblemChecker(link.Problem), eventStatementSections(link.Problem)})
		}
	}
	return out, nil
}
func eventSnapshot(tx *gorm.DB, p eventPaper) (datatypes.JSON, error) {
	problems, err := eventPaperProblems(tx, p)
	if err != nil {
		return nil, err
	}
	var files []models.EventAttachment
	if err = tx.Where("event_kind=? AND event_id=?", p.kind, p.id).Order("id").Find(&files).Error; err != nil {
		return nil, err
	}
	b, err := json.Marshal(gin.H{"event": p.model(), "problems": problems, "attachments": files})
	return datatypes.JSON(b), err
}
func recordEventReview(tx *gorm.DB, p eventPaper, userID uint, action, note string) error {
	snapshot, err := eventSnapshot(tx, p)
	if err != nil {
		return err
	}
	return tx.Create(&models.EventReviewLog{EventKind: p.kind, EventID: p.id, Revision: p.review().ReviewRevision, Action: action, ActorID: userID, Note: note, Snapshot: snapshot}).Error
}
func (s Server) getEventPaper(c *gin.Context) {
	p, ok := s.eventPaperByParams(c, true)
	if !ok {
		return
	}
	problems, err := eventPaperProblems(s.DB, p)
	if err != nil {
		eventError(c, err)
		return
	}
	var history []models.EventReviewLog
	if err = s.DB.Where("event_kind=? AND event_id=?", p.kind, p.id).Order("id desc").Find(&history).Error; err != nil {
		eventError(c, err)
		return
	}
	c.Header("Cache-Control", "private, no-store")
	checks, err := completionChecks(s.DB, p)
	if err != nil {
		eventError(c, err)
		return
	}
	c.JSON(200, gin.H{"event": p.model(), "kind": p.kind, "review": p.review(), "lifecycle": p.lifecycle(time.Now()), "completion_checks": checks, "problems": problems, "history": history, "can_edit": p.review().Editable()})
}
func (s Server) reviewEventPaper(c *gin.Context) {
	p, ok := s.eventPaperByParams(c, true)
	if !ok {
		return
	}
	var req struct {
		Revision int    `json:"revision"`
		Action   string `json:"action"`
		Note     string `json:"note"`
	}
	if !bind(c, &req) {
		return
	}
	req.Note = strings.TrimSpace(req.Note)
	u, _ := middleware.CurrentUser(c)
	if (req.Action == "approve" || req.Action == "reject" || req.Action == "complete" || req.Action == "reopen") && u.Role != models.RoleAdmin {
		c.JSON(403, gin.H{"error": "仅系统管理员可审核整场考试或比赛"})
		return
	}
	if len([]rune(req.Note)) > 2000 {
		c.JSON(400, gin.H{"error": "审核说明过长"})
		return
	}
	err := s.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		locked, err := loadEventPaper(tx, p.kind, p.id, true)
		if err != nil {
			return err
		}
		p = locked
		scoped := s
		scoped.DB = tx
		if !scoped.canManageEventPaper(u, p) {
			return eventFail(403, "场次管理权限已变化")
		}
		r := p.review()
		if r.ReviewRevision != req.Revision {
			return eventFail(409, "场次已变更，请刷新审核内容")
		}
		now := time.Now()
		updates := map[string]any{"review_revision": r.ReviewRevision + 1}
		switch req.Action {
		case "complete":
			if err = validateEventCompletion(tx, p, now); err != nil {
				return err
			}
			updates["completed_at"], updates["completed_by"], updates["completion_note"] = now, u.ID, req.Note
		case "reopen":
			if p.lifecycle(now) != "completed" {
				return eventFail(409, "仅已完结场次可以撤销完结")
			}
			if req.Note == "" {
				return eventFail(400, "请填写撤销完结的原因")
			}
			updates["completed_at"], updates["completed_by"], updates["completion_note"] = nil, nil, req.Note
		case "submit":
			if err = requireEventDraft(p, req.Revision); err != nil {
				return err
			}
			if p.starts() == nil || !p.starts().After(now) || p.ends() == nil || !p.ends().After(*p.starts()) {
				return eventFail(400, "请先设置有效的未来开始和结束时间")
			}
			problems, err := eventPaperProblems(tx, p)
			if err != nil {
				return err
			}
			if err = validateEventLabels(problems); err != nil {
				return err
			}
			if len(problems) == 0 || len(problems) > 100 {
				return eventFail(400, "整场需包含 1 至 100 道题目")
			}
			for _, problem := range problems {
				if problem.ProblemVersionID == 0 || problem.CaseCount == 0 || problem.Problem.DeletedAt != nil || problem.Problem.ArchivedAt != nil {
					return eventFail(409, "所选题目不可用，请重新组题")
				}
			}
			updates["review_status"] = "pending"
			updates["review_submitted_at"] = now
			updates["review_note"] = ""
			updates["reviewed_by"] = nil
			updates["reviewed_at"] = nil
		case "withdraw":
			if r.ReviewStatus != "pending" && r.ReviewStatus != "approved" {
				return eventFail(409, "当前状态不能撤回")
			}
			if r.Approved() && (p.starts() == nil || !now.Before(*p.starts())) {
				return eventFail(409, "已开始的场次不能撤回或修改题目")
			}
			updates["review_status"] = "draft"
			updates["review_note"] = "已撤回，修改完成后须重新整场审核"
			updates["reviewed_at"] = nil
			updates["reviewed_by"] = nil
			if p.kind == "exam" {
				updates["participants_frozen"] = false
				if err = tx.Where("exam_id=?", p.id).Delete(&models.ExamParticipant{}).Error; err != nil {
					return err
				}
			}

			if p.kind == "contest" {
				updates["state"] = models.TeamContestDraft
				updates["published_at"] = nil
				if err = tx.Where("contest_id=?", p.id).Delete(&models.TeamContestParticipant{}).Error; err != nil {
					return err
				}
			}
		case "approve", "reject":
			if r.ReviewStatus != "pending" {
				return eventFail(409, "该场次不再待审，请刷新")
			}
			updates["review_status"] = "rejected"
			updates["review_note"] = req.Note
			updates["reviewed_by"] = u.ID
			updates["reviewed_at"] = now
			if req.Action == "reject" && req.Note == "" {
				return eventFail(400, "请填写退回原因")
			}
			if req.Action == "approve" {
				if p.starts() == nil || !p.starts().After(now) || p.ends() == nil || !p.ends().After(*p.starts()) {
					return eventFail(409, "开始时间已到或时间无效，请退回修改，不能补审核后自动归档")
				}
				problems, err := eventPaperProblems(tx, p)
				if err != nil {
					return err
				}
				if err = validateEventLabels(problems); err != nil {
					return err
				}
				if len(problems) == 0 {
					return eventFail(409, "场次没有题目")
				}
				for _, problem := range problems {
					if problem.Problem.DeletedAt != nil || problem.Problem.ArchivedAt != nil || problem.ProblemVersionID == 0 || problem.CaseCount == 0 {
						return eventFail(409, "审核中的题目已不可用，请退回重新组题")
					}
				}
				updates["review_status"] = "approved"
				if p.kind == "exam" {
					if err = snapshotExamParticipants(tx, p.exam); err != nil {
						return err
					}
					updates["participants_frozen"] = true
				}
				if p.kind == "contest" {
					updates["state"] = models.TeamContestPublished
					updates["published_at"] = now
					if err = tx.Where("contest_id=?", p.id).Delete(&models.TeamContestParticipant{}).Error; err != nil {
						return err
					}
					if err = tx.Exec(`INSERT INTO team_contest_participants(contest_id,user_id) SELECT ?,m.user_id FROM team_memberships m JOIN users u ON u.id=m.user_id WHERE m.team_id=? AND u.account_deleted=false ON CONFLICT DO NOTHING`, p.id, p.contest.TeamID).Error; err != nil {
						return err
					}
				}
			}
		default:
			return eventFail(400, "审核操作无效")
		}
		if err = recordEventReview(tx, p, u.ID, req.Action, req.Note); err != nil {
			return err
		}
		return tx.Model(p.model()).Updates(updates).Error
	})
	if err != nil {
		eventError(c, err)
		return
	}
	services.Audit(c, s.DB, "event."+req.Action, p.kind, p.id, datatypes.JSONMap{"revision": req.Revision})
	c.JSON(http.StatusOK, gin.H{"updated": true, "revision": req.Revision + 1})
}

func (s Server) updateEventSettings(c *gin.Context) {
	p, ok := s.eventPaperByParams(c, true)
	if !ok {
		return
	}
	var req struct {
		Revision    int       `json:"revision"`
		Title       string    `json:"title"`
		Description string    `json:"description"`
		StartsAt    time.Time `json:"starts_at"`
		EndsAt      time.Time `json:"ends_at"`
	}
	if !bind(c, &req) {
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" || len([]rune(req.Title)) > 200 || len(req.Description) > 20000 || req.StartsAt.IsZero() || !req.EndsAt.After(req.StartsAt) || req.EndsAt.Sub(req.StartsAt) > 30*24*time.Hour {
		c.JSON(400, gin.H{"error": "请填写有效标题和时间范围（最长 30 天）"})
		return
	}
	u, _ := middleware.CurrentUser(c)
	err := s.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		locked, err := loadEventPaper(tx, p.kind, p.id, true)
		if err != nil {
			return err
		}
		p = locked
		scoped := s
		scoped.DB = tx
		if !scoped.canManageEventPaper(u, p) {
			return eventFail(403, "管理权限已变化")
		}
		if err = requireEventDraft(p, req.Revision); err != nil {
			return err
		}
		updates := map[string]any{"title": req.Title, "description": req.Description, "starts_at": req.StartsAt}
		if p.kind == "exam" {
			updates["ends_at"] = req.EndsAt
			if p.exam.FreezeEnabled && req.EndsAt.Sub(req.StartsAt) <= time.Duration(p.exam.FreezeDurationMinutes)*time.Minute {
				return eventFail(400, "封榜时长须小于场次时长")
			}
		} else {
			duration := int(req.EndsAt.Sub(req.StartsAt) / time.Minute)
			if duration < 1 || p.contest.FreezeEnabled && duration <= p.contest.FreezeDurationMinutes {
				return eventFail(400, "比赛时长无效")
			}
			updates["duration_minutes"] = duration
		}
		if err = tx.Model(p.model()).Updates(updates).Error; err != nil {
			return err
		}
		return bumpEventRevision(tx, p)
	})
	if err != nil {
		eventError(c, err)
		return
	}
	c.JSON(200, gin.H{"updated": true})
}

func eventProblemChecker(p models.Problem) services.CheckerManifest {
	data, _ := json.Marshal(p.Manifest)
	var m services.ProblemManifest
	_ = json.Unmarshal(data, &m)
	if m.Checker.Type == "" {
		m.Checker.Type = "exact"
	}
	return m.Checker
}

func snapshotExamParticipants(tx *gorm.DB, exam models.Exam) error {
	if err := tx.Where("exam_id=?", exam.ID).Delete(&models.ExamParticipant{}).Error; err != nil {
		return err
	}
	return tx.Exec(`INSERT INTO exam_participants(exam_id,user_id)
SELECT ?,u.id FROM users u WHERE u.role='student' AND u.account_deleted=false AND (
 (?::bigint IS NOT NULL AND EXISTS(SELECT 1 FROM class_memberships m WHERE m.user_id=u.id AND m.class_id=?)) OR
 (?::bigint IS NULL AND EXISTS(SELECT 1 FROM course_memberships m WHERE m.user_id=u.id AND m.course_id=? AND m.role='student'))
) ON CONFLICT DO NOTHING`, exam.ID, exam.ClassID, exam.ClassID, exam.ClassID, exam.CourseID).Error
}

func validateEventLabels(problems []eventPaperProblem) error {
	seen := map[string]bool{}
	for _, p := range problems {
		label := strings.ToLower(strings.TrimSpace(p.Label))
		if label == "" || len(label) > 16 || seen[label] || p.Score < 1 || p.Score > 1000 {
			return eventFail(409, "题号须唯一且不超过 16 字节，分值须为 1 至 1000；请重新组题")
		}
		seen[label] = true
	}
	return nil
}
