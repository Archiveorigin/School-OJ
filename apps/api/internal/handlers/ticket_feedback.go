package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"school-oj/apps/api/internal/middleware"
	"school-oj/apps/api/internal/models"
	"school-oj/apps/api/internal/services"
)

const problemTicketAttachmentLimit int64 = 32 << 20
const ticketTotalAttachmentLimit int64 = 96 << 20

func ticketFileLimit(action models.ProblemChangeAction) int {
	if action == models.ProblemChangeCreate {
		return 30
	}
	return 20
}

type problemChangeTicketInput struct {
	Action              models.ProblemChangeAction `json:"action"`
	ProblemID           *uint                      `json:"problem_id"`
	ExamID              *uint                      `json:"exam_id"`
	ContestID           *uint                      `json:"contest_id"`
	TargetScope         string                     `json:"target_scope"`
	TeamProblemSetID    *uint                      `json:"team_problem_set_id"`
	Description         string                     `json:"description"`
	DraftTitle          string                     `json:"draft_title"`
	DraftStatement      string                     `json:"draft_statement"`
	Revision            int                        `json:"revision"`
	RetainAttachmentIDs []uint                     `json:"retain_attachment_ids"`
}

type ticketUpload struct {
	name, contentType string
	body              []byte
}
type ticketScope struct{ problemID, examID, contestID *uint }

func problemChangeTicketOnly(c *gin.Context) {
	c.JSON(http.StatusConflict, gin.H{"error": "请提交工单，由管理员审核后执行题库变更", "ticket_route": "/problem-changes/new"})
}

func ticketRouteScope(c *gin.Context) (ticketScope, error) {
	var scope ticketScope
	path := examAPIPath(c.FullPath())
	var target **uint
	var param string
	switch {
	case strings.HasPrefix(path, "/exams/"):
		target, param = &scope.examID, "id"
	case strings.HasPrefix(path, "/contests/"):
		target, param = &scope.contestID, "contest_id"
	case strings.HasPrefix(path, "/problems/"):
		target, param = &scope.problemID, "id"
	default:
		return scope, nil
	}
	n, err := strconv.ParseUint(c.Param(param), 10, 64)
	if err != nil || n == 0 {
		return scope, errors.New("invalid context")
	}
	id := uint(n)
	*target = &id
	return scope, nil
}

func ticketIDParam(c *gin.Context) (uint, bool) {
	key := "id"
	if c.Param("ticket_id") != "" {
		key = "ticket_id"
	}
	return idParam(c, key)
}

func sameTicketID(a, b *uint) bool { return a == nil && b == nil || a != nil && b != nil && *a == *b }

// Context access is checked on every operation, including attachment reads.
// Existing private feedback remains readable by its requester after an event ends.
func (s Server) ticketContextAccess(user models.User, ticket models.ProblemChangeTicket, writing bool) (bool, bool) {
	if ticket.ExamID != nil {
		var exam models.Exam
		if s.DB.Where("deleted_at IS NULL").First(&exam, *ticket.ExamID).Error != nil {
			return false, false
		}
		manager := s.canManageCourse(user, exam.CourseID)
		if manager {
			return true, true
		}
		if !s.canAccessExam(user, exam) {
			return false, false
		}
		if exam.StartsAt != nil && time.Now().Before(*exam.StartsAt) {
			return false, false
		}
		if writing {
			if reason, _ := studentExamEntryDecision(exam, time.Now(), s.examFinishedAt(exam.ID, user.ID)); reason != "" {
				return false, false
			}
			var count int64
			if s.DB.Model(&models.ExamAttempt{}).Where("exam_id = ? AND user_id = ? AND finished_at IS NULL", exam.ID, user.ID).Count(&count).Error != nil || count == 0 {
				return false, false
			}
		}
		return true, false
	}
	if ticket.ContestID != nil {
		var contest models.TeamContest
		var team models.Team
		if s.DB.Where("deleted_at IS NULL").First(&contest, *ticket.ContestID).Error != nil || s.DB.First(&team, contest.TeamID).Error != nil {
			return false, false
		}
		manager := s.canManageTeam(user, team)
		if manager {
			return true, true
		}
		if _, joined := s.teamMembership(team.ID, user.ID); !joined {
			return false, false
		}
		_, status := teamContestWindow(contest, time.Now())
		if status == models.TeamContestDraft || status == models.TeamContestPublished {
			return false, false
		}
		return true, false
	}
	if ticket.ProblemID == nil {
		return user.ID != 0 && (ticket.RequesterID == user.ID || user.Role == models.RoleAdmin), user.Role == models.RoleAdmin
	}
	var problem models.Problem
	if s.DB.Where("deleted_at IS NULL").First(&problem, *ticket.ProblemID).Error != nil {
		return false, false
	}
	manager := user.Role == models.RoleAdmin
	accessible := s.isProblemPublic(problem.ID) || user.ID != 0 && (s.canManageProblemData(user, problem) || s.canAccessTeamProblem(user, problem))
	if writing && problem.ArchivedAt != nil {
		accessible = false
	}
	return accessible, manager
}

func (s Server) ticketReadable(user models.User, ticket models.ProblemChangeTicket) (bool, bool) {
	access, manager := s.ticketContextAccess(user, ticket, false)
	return access && (manager || user.ID != 0 && ticket.RequesterID == user.ID || ticket.IsPublic && ticket.ExamID == nil && ticket.ContestID == nil), manager
}

func (s Server) problemChangeTicket(id uint) (models.ProblemChangeTicket, bool) {
	var ticket models.ProblemChangeTicket
	err := s.DB.Preload("Requester").Preload("Attachments", func(db *gorm.DB) *gorm.DB { return db.Order("id") }).Where("deleted_at IS NULL").First(&ticket, id).Error
	return ticket, err == nil
}

func (s Server) authorizedTicket(c *gin.Context) (models.ProblemChangeTicket, bool, bool) {
	id, ok := ticketIDParam(c)
	if !ok {
		return models.ProblemChangeTicket{}, false, false
	}
	ticket, found := s.problemChangeTicket(id)
	scope, err := ticketRouteScope(c)
	if !found || err != nil || scope.examID != nil && !sameTicketID(scope.examID, ticket.ExamID) || scope.contestID != nil && !sameTicketID(scope.contestID, ticket.ContestID) || scope.problemID != nil && (!sameTicketID(scope.problemID, ticket.ProblemID) || ticket.ExamID != nil || ticket.ContestID != nil) {
		c.JSON(404, gin.H{"error": "工单不存在"})
		return ticket, false, false
	}
	user, _ := middleware.CurrentUser(c)
	readable, manager := s.ticketReadable(user, ticket)
	if !readable {
		c.JSON(403, gin.H{"error": "无权访问此工单"})
		return ticket, false, false
	}
	return ticket, manager, true
}

// Never serialize User/Problem ORM records through a ticket endpoint.
func (s Server) ticketView(user models.User, ticket models.ProblemChangeTicket) gin.H {
	_, manager := s.ticketReadable(user, ticket)
	private := manager || user.ID != 0 && ticket.RequesterID == user.ID
	canWrite, _ := s.ticketContextAccess(user, ticket, true)
	view := gin.H{
		"id": ticket.ID, "requester_id": ticket.RequesterID, "requester": gin.H{"name": ticket.Requester.Name},
		"problem_id": ticket.ProblemID, "exam_id": ticket.ExamID, "contest_id": ticket.ContestID,
		"problem_version_id": ticket.ProblemVersionID, "action": ticket.Action, "status": ticket.Status,
		"target_scope": ticket.TargetScope, "team_problem_set_id": ticket.TeamProblemSetID,
		"description": ticket.Description, "resolution_note": ticket.ResolutionNote, "is_public": ticket.IsPublic,
		"revision": ticket.Revision, "created_at": ticket.CreatedAt, "updated_at": ticket.UpdatedAt,
		"processed_at": ticket.ProcessedAt, "applied_version_id": ticket.AppliedVersionID,
		"attachment_count": len(ticket.Attachments), "attachments_private": !private,
		"can_edit":   canWrite && user.ID != 0 && user.ID == ticket.RequesterID && ticket.Status == models.ProblemChangePending,
		"can_manage": manager, "can_apply": user.Role == models.RoleAdmin && ticket.Action != models.ProblemChangeSuggest && (ticket.ProblemID != nil || ticket.Action == models.ProblemChangeCreate),
		"can_resolve": manager && (ticket.Action == models.ProblemChangeSuggest || ticket.ProblemID == nil && ticket.Action == models.ProblemChangeReplace),
	}
	if ticket.ProblemID != nil {
		var p struct {
			ID               uint
			Title            string
			DisplayCode      string
			Difficulty       string
			CurrentVersionID *uint
		}
		if ticket.ProblemVersionID != nil {
			s.DB.Model(&models.ProblemVersion{}).Select("problem_id AS id, title, difficulty").Where("id = ?", *ticket.ProblemVersionID).Scan(&p)
		} else {
			s.DB.Model(&models.Problem{}).Select("id, title, display_code, difficulty").Where("id = ?", *ticket.ProblemID).Scan(&p)
		}
		view["problem"] = gin.H{"id": p.ID, "title": p.Title, "display_code": p.DisplayCode, "difficulty": p.Difficulty}
		if user.Role == models.RoleAdmin {
			s.DB.Model(&models.Problem{}).Select("current_version_id").Where("id = ?", *ticket.ProblemID).Scan(&p)
			view["current_version_id"] = p.CurrentVersionID
		}
	}
	if private {
		view["attachments"] = ticket.Attachments
		if ticket.Action == models.ProblemChangeCreate {
			view["draft_title"], view["draft_statement"] = ticket.DraftTitle, ticket.DraftStatement
		}
		if len(ticket.Attachments) > 0 {
			view["attachment_name"] = ticket.Attachments[0].Name
		}
	}
	if user.Role == models.RoleAdmin {
		s.loadProblemChangeTicketImpact(&ticket)
		view["impact_summary"] = ticket.ImpactSummary
	}
	return view
}

func parseTicketInput(c *gin.Context) (problemChangeTicketInput, []ticketUpload, error) {
	var req problemChangeTicketInput
	var uploads []ticketUpload
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, ticketTotalAttachmentLimit+(2<<20))
	if !strings.HasPrefix(strings.ToLower(c.GetHeader("Content-Type")), "multipart/form-data") {
		err := json.NewDecoder(io.LimitReader(c.Request.Body, 1<<20)).Decode(&req)
		return req, uploads, err
	}
	if err := c.Request.ParseMultipartForm(4 << 20); err != nil {
		return req, nil, err
	}
	defer c.Request.MultipartForm.RemoveAll()
	if draft := c.PostForm("draft"); draft != "" {
		if err := json.Unmarshal([]byte(draft), &req); err != nil {
			return req, nil, err
		}
	} else {
		req.Action = models.ProblemChangeAction(c.PostForm("action"))
		req.Description = c.PostForm("description")
		req.TargetScope = c.PostForm("target_scope")
		req.DraftTitle = c.PostForm("draft_title")
		req.DraftStatement = c.PostForm("draft_statement")
		req.Revision, _ = strconv.Atoi(c.PostForm("revision"))
		for key, target := range map[string]**uint{"problem_id": &req.ProblemID, "team_problem_set_id": &req.TeamProblemSetID, "exam_id": &req.ExamID, "contest_id": &req.ContestID} {
			if raw := c.PostForm(key); raw != "" {
				n, err := strconv.ParseUint(raw, 10, 64)
				if err != nil || n == 0 {
					return req, nil, fmt.Errorf("%s 无效", key)
				}
				id := uint(n)
				*target = &id
			}
		}
		if raw := c.PostForm("retain_attachment_ids"); raw != "" {
			if err := json.Unmarshal([]byte(raw), &req.RetainAttachmentIDs); err != nil {
				return req, nil, err
			}
		}
	}
	files := append(c.Request.MultipartForm.File["attachments"], c.Request.MultipartForm.File["attachment"]...)
	if limit := ticketFileLimit(req.Action); len(files) > limit {
		return req, nil, fmt.Errorf("最多上传 %d 个附件", limit)
	}
	var total int64
	for _, file := range files {
		if file.Size <= 0 || file.Size > problemTicketAttachmentLimit || total+file.Size > ticketTotalAttachmentLimit {
			return req, nil, errors.New("附件必须非空，单个不超过 32 MiB，总计不超过 96 MiB")
		}
		stream, err := file.Open()
		if err != nil {
			return req, nil, err
		}
		body, err := services.ReadLimited(stream, problemTicketAttachmentLimit, "工单附件")
		stream.Close()
		if err != nil {
			return req, nil, err
		}
		name := filepath.Base(strings.ReplaceAll(file.Filename, "\\", "/"))
		if err = validateProblemChangeTicketAttachment(name, body); err != nil {
			return req, nil, err
		}
		total += int64(len(body))
		if total > ticketTotalAttachmentLimit {
			return req, nil, errors.New("附件总计超过 96 MiB")
		}
		contentType := "application/octet-stream"
		if config, format, err := image.DecodeConfig(bytes.NewReader(body)); err == nil && config.Width > 0 && config.Height > 0 && int64(config.Width)*int64(config.Height) <= 40000000 {
			switch format {
			case "png", "jpeg", "gif":
				contentType = "image/" + format
			}
		}
		uploads = append(uploads, ticketUpload{name: name, contentType: contentType, body: body})
	}
	return req, uploads, nil
}

func validateProblemChangeTicketAttachment(name string, body []byte) error {
	if name == "" && len(body) == 0 {
		return nil
	}
	if strings.TrimSpace(name) == "" || len([]rune(name)) > 255 || strings.ContainsAny(name, "\r\n\x00") || len(body) == 0 {
		return errors.New("附件名称无效或文件为空")
	}
	if int64(len(body)) > problemTicketAttachmentLimit {
		return errors.New("单个附件不能超过 32 MiB")
	}
	return nil
}

func validateTicketText(req problemChangeTicketInput, user models.User) error {
	if req.Action != models.ProblemChangeCreate && (req.DraftTitle != "" || req.DraftStatement != "") {
		return errors.New("只有新增题目工单可以包含 Markdown 草稿")
	}
	switch req.Action {
	case models.ProblemChangeCreate, models.ProblemChangeArchive:
		if !canCreateProblems(user) {
			return errors.New("新增和删除题目申请需要出题资格")
		}
	case models.ProblemChangeReplace, models.ProblemChangeSuggest:
	default:
		return errors.New("工单类型无效")
	}
	if n := len([]rune(strings.TrimSpace(req.Description))); n < 4 || n > 10000 {
		return errors.New("修改说明须为 4 至 10000 个字")
	}
	if req.Action == models.ProblemChangeCreate && (strings.TrimSpace(req.DraftTitle) == "" || strings.TrimSpace(req.DraftStatement) == "") {
		return errors.New("新增题目必须填写标题和 Markdown 题面")
	}
	if len([]rune(req.DraftTitle)) > 200 || len(req.DraftStatement) > 200000 {
		return errors.New("题目标题或题面过长")
	}
	return nil
}

func (s Server) prepareTicket(c *gin.Context, user models.User, req problemChangeTicketInput) (models.ProblemChangeTicket, error) {
	scope, err := ticketRouteScope(c)
	if err != nil {
		return models.ProblemChangeTicket{}, err
	}
	ticket := models.ProblemChangeTicket{RequesterID: user.ID, Action: req.Action, Status: models.ProblemChangePending, Revision: 1, ProblemID: req.ProblemID, ExamID: scope.examID, ContestID: scope.contestID, Description: strings.TrimSpace(req.Description), DraftTitle: strings.TrimSpace(req.DraftTitle), DraftStatement: req.DraftStatement, TargetScope: "public"}
	if req.ExamID != nil || req.ContestID != nil {
		return ticket, errors.New("场次必须由当前页面指定")
	}
	if scope.problemID != nil {
		if req.ProblemID != nil && !sameTicketID(req.ProblemID, scope.problemID) {
			return ticket, errors.New("题目与当前页面不匹配")
		}
		ticket.ProblemID = scope.problemID
	}
	if err = validateTicketText(req, user); err != nil {
		return ticket, err
	}
	if req.Action == models.ProblemChangeCreate {
		if ticket.ProblemID != nil || ticket.ExamID != nil || ticket.ContestID != nil {
			return ticket, errors.New("请从个人入口提交新增题目")
		}
		ticket.TargetScope = req.TargetScope
		if ticket.TargetScope == "" {
			ticket.TargetScope = "public"
		}
		switch ticket.TargetScope {
		case "public", "prepared":
		case "team_problem_set":
			if req.TeamProblemSetID == nil {
				return ticket, errors.New("请选择团队题单")
			}
			ticket.TeamProblemSetID = req.TeamProblemSetID
		default:
			return ticket, errors.New("新增题目范围无效")
		}
		return ticket, nil
	}
	if ticket.ProblemID == nil && (ticket.ExamID == nil && ticket.ContestID == nil || ticket.Action == models.ProblemChangeArchive) {
		return ticket, errors.New("此类工单必须指定题目")
	}
	if access, _ := s.ticketContextAccess(user, ticket, true); !access {
		return ticket, errors.New("无权在当前题目或场次提交工单")
	}
	if ticket.ProblemID != nil {
		if ticket.ExamID != nil {
			var link models.ExamProblem
			if s.DB.Where("exam_id = ? AND problem_id = ?", *ticket.ExamID, *ticket.ProblemID).First(&link).Error != nil {
				return ticket, errors.New("题目不属于当前考试")
			}
			ticket.ProblemVersionID = &link.ProblemVersionID
		} else if ticket.ContestID != nil {
			var link models.TeamContestProblem
			if s.DB.Where("contest_id = ? AND problem_id = ?", *ticket.ContestID, *ticket.ProblemID).First(&link).Error != nil {
				return ticket, errors.New("题目不属于当前比赛")
			}
			ticket.ProblemVersionID = &link.ProblemVersionID
		} else {
			var p models.Problem
			if s.DB.First(&p, *ticket.ProblemID).Error != nil {
				return ticket, errors.New("题目不存在")
			}
			ticket.ProblemVersionID = p.CurrentVersionID
			ticket.IsPublic = true
		}
	}
	// Only new-problem tickets may request publication; feedback cannot publish a private problem.
	ticket.TargetScope = "prepared"
	return ticket, nil
}

func (s Server) storeTicketUploads(c *gin.Context, uploads []ticketUpload) ([]models.ProblemTicketAttachment, error) {
	records := make([]models.ProblemTicketAttachment, 0, len(uploads))
	if len(uploads) > 0 && s.MinIO == nil {
		return records, errors.New("附件服务暂不可用")
	}
	for i, file := range uploads {
		object := fmt.Sprintf("ticket-files/%d-%d", time.Now().UnixNano(), i)
		if _, err := s.MinIO.PutObject(c.Request.Context(), s.Cfg.MinIOBucket, object, bytes.NewReader(file.body), int64(len(file.body)), minio.PutObjectOptions{ContentType: file.contentType}); err != nil {
			s.cleanupTicketUploads(append(records, models.ProblemTicketAttachment{Object: object}))
			return nil, err
		}
		records = append(records, models.ProblemTicketAttachment{Name: file.name, Object: object, ContentType: file.contentType, Size: int64(len(file.body))})
	}
	return records, nil
}

func (s Server) cleanupTicketUploads(records []models.ProblemTicketAttachment) {
	if s.MinIO == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for _, file := range records {
		_ = s.MinIO.RemoveObject(ctx, s.Cfg.MinIOBucket, file.Object, minio.RemoveObjectOptions{})
	}
}

func (s Server) createProblemChangeTicket(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	req, uploads, err := parseTicketInput(c)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	ticket, err := s.prepareTicket(c, user, req)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if ticket.TeamProblemSetID != nil && !s.canSubmitTicketToProblemSet(c, user, *ticket.TeamProblemSetID) {
		return
	}
	records, err := s.storeTicketUploads(c, uploads)
	if err != nil {
		c.JSON(503, gin.H{"error": "附件上传失败，请重试"})
		return
	}
	err = s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&ticket).Error; err != nil {
			return err
		}
		for i := range records {
			records[i].TicketID = ticket.ID
		}
		if len(records) > 0 {
			return tx.Create(&records).Error
		}
		return nil
	})
	if err != nil {
		s.cleanupTicketUploads(records)
		c.JSON(400, gin.H{"error": "工单保存失败"})
		return
	}
	services.Audit(c, s.DB, "problem_change_ticket.create", "problem_change_ticket", ticket.ID, datatypes.JSONMap{"action": ticket.Action, "exam_id": ticket.ExamID, "contest_id": ticket.ContestID})
	ticket, _ = s.problemChangeTicket(ticket.ID)
	c.JSON(201, s.ticketView(user, ticket))
}

func (s Server) canSubmitTicketToProblemSet(c *gin.Context, user models.User, setID uint) bool {
	var set models.TeamProblemSet
	var team models.Team
	if s.DB.Where("deleted_at IS NULL").First(&set, setID).Error != nil || s.DB.First(&team, set.TeamID).Error != nil {
		c.JSON(404, gin.H{"error": "题单不存在"})
		return false
	}
	if !s.canTeamContentPermission(user, team) {
		c.JSON(403, gin.H{"error": "无权向该团队题单新增题目"})
		return false
	}
	return true
}

func (s Server) listMyProblemChangeTickets(c *gin.Context)     { s.listTicketFeedback(c, "mine") }
func (s Server) listProblemChangeTickets(c *gin.Context)       { s.listTicketFeedback(c, "manage") }
func (s Server) listScopedProblemChangeTickets(c *gin.Context) { s.listTicketFeedback(c, "scope") }
func (s Server) listTicketFeedback(c *gin.Context, mode string) {
	user, _ := middleware.CurrentUser(c)
	scope, err := ticketRouteScope(c)
	if err != nil {
		c.JSON(400, gin.H{"error": "范围无效"})
		return
	}
	query := s.DB.Model(&models.ProblemChangeTicket{}).Where("deleted_at IS NULL")
	switch mode {
	case "mine":
		query = query.Where("requester_id = ?", user.ID)
	case "scope":
		contextTicket := models.ProblemChangeTicket{ProblemID: scope.problemID, ExamID: scope.examID, ContestID: scope.contestID}
		access, manager := s.ticketContextAccess(user, contextTicket, false)
		if !access {
			c.JSON(403, gin.H{"error": "无权访问当前工单"})
			return
		}
		if scope.examID != nil {
			query = query.Where("exam_id = ?", *scope.examID)
		} else if scope.contestID != nil {
			query = query.Where("contest_id = ?", *scope.contestID)
		} else {
			query = query.Where("problem_id = ? AND exam_id IS NULL AND contest_id IS NULL", *scope.problemID)
		}
		if !manager {
			query = query.Where("requester_id = ? OR is_public = true", user.ID)
		}
		if pid := c.Query("problem_id"); pid != "" {
			if pid == "0" {
				query = query.Where("problem_id IS NULL")
			} else {
				n, e := strconv.ParseUint(pid, 10, 64)
				if e != nil {
					c.JSON(400, gin.H{"error": "题目编号无效"})
					return
				}
				query = query.Where("problem_id = ?", n)
			}
		}
	}
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if action := c.Query("action"); action != "" {
		query = query.Where("action = ?", action)
	}
	if mode == "manage" {
		switch c.Query("context") {
		case "problem":
			query = query.Where("exam_id IS NULL AND contest_id IS NULL AND action <> 'create'")
		case "exam":
			query = query.Where("exam_id IS NOT NULL")
		case "contest":
			query = query.Where("contest_id IS NOT NULL")
		case "new":
			query = query.Where("action = 'create'")
		}
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	var total int64
	var tickets []models.ProblemChangeTicket
	if query.Count(&total).Error != nil || query.Preload("Requester").Preload("Attachments").Order("created_at DESC, id DESC").Offset((page-1)*size).Limit(size).Find(&tickets).Error != nil {
		c.JSON(500, gin.H{"error": "工单读取失败"})
		return
	}
	items := make([]gin.H, 0, len(tickets))
	for _, ticket := range tickets {
		if readable, _ := s.ticketReadable(user, ticket); readable {
			items = append(items, s.ticketView(user, ticket))
		}
	}
	c.JSON(200, gin.H{"items": items, "total": total, "page": page, "page_size": size})
}

func (s Server) listProblemChangeTicketEligibleProblems(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	var problems []struct {
		ID          uint   `json:"id"`
		Title       string `json:"title"`
		DisplayCode string `json:"display_code"`
	}
	q := s.DB.Model(&models.Problem{}).Select("id,title,display_code").Where("deleted_at IS NULL AND archived_at IS NULL")
	if user.Role != models.RoleAdmin {
		q = q.Where("owner_id = ?", user.ID)
	}
	if q.Order("id DESC").Limit(100).Scan(&problems).Error != nil {
		c.JSON(500, gin.H{"error": "题目读取失败"})
		return
	}
	c.JSON(200, problems)
}

func (s Server) getProblemChangeTicket(c *gin.Context) {
	ticket, _, ok := s.authorizedTicket(c)
	if !ok {
		return
	}
	user, _ := middleware.CurrentUser(c)
	c.JSON(200, s.ticketView(user, ticket))
}

func (s Server) downloadProblemChangeTicketAttachment(c *gin.Context) {
	ticket, manager, ok := s.authorizedTicket(c)
	if !ok {
		return
	}
	user, _ := middleware.CurrentUser(c)
	if !manager && ticket.RequesterID != user.ID {
		c.JSON(403, gin.H{"error": "附件仅申请人和有权管理者可见"})
		return
	}
	var file models.ProblemTicketAttachment
	if raw := c.Param("attachment_id"); raw != "" {
		id, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			c.JSON(400, gin.H{"error": "附件编号无效"})
			return
		}
		for _, a := range ticket.Attachments {
			if uint(id) == a.ID {
				file = a
				break
			}
		}
	} else if len(ticket.Attachments) > 0 {
		file = ticket.Attachments[0]
	}
	if file.ID == 0 || s.MinIO == nil {
		c.JSON(404, gin.H{"error": "附件不存在"})
		return
	}
	object, err := s.MinIO.GetObject(c.Request.Context(), s.Cfg.MinIOBucket, file.Object, minio.GetObjectOptions{})
	if err != nil {
		c.JSON(404, gin.H{"error": "附件不存在"})
		return
	}
	defer object.Close()
	info, err := object.Stat()
	if err != nil {
		c.JSON(404, gin.H{"error": "附件不存在"})
		return
	}
	disposition, contentType := "attachment", "application/octet-stream"
	if c.Query("preview") == "1" {
		// Validate legacy objects too; object metadata and file extensions are not authoritative.
		head, err := io.ReadAll(io.LimitReader(object, problemTicketAttachmentLimit+1))
		if err != nil || int64(len(head)) > problemTicketAttachmentLimit {
			c.JSON(400, gin.H{"error": "图片预览不可用"})
			return
		}
		config, format, err := image.DecodeConfig(bytes.NewReader(head))
		if err != nil || int64(config.Width)*int64(config.Height) > 40000000 || (format != "png" && format != "jpeg" && format != "gif") {
			c.JSON(400, gin.H{"error": "此附件不支持图片预览"})
			return
		}
		c.Header("Content-Disposition", mime.FormatMediaType("inline", map[string]string{"filename": file.Name}))
		c.Header("Cache-Control", "private, no-store")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Data(200, "image/"+format, head)
		return
	}
	c.Header("Content-Disposition", mime.FormatMediaType(disposition, map[string]string{"filename": file.Name}))
	c.Header("Cache-Control", "private, no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	c.DataFromReader(200, info.Size, contentType, object, nil)
}

func (s Server) updateProblemChangeTicket(c *gin.Context) {
	ticket, _, ok := s.authorizedTicket(c)
	if !ok {
		return
	}
	user, _ := middleware.CurrentUser(c)
	writable, _ := s.ticketContextAccess(user, ticket, true)
	if ticket.RequesterID != user.ID || !writable {
		c.JSON(403, gin.H{"error": "只能编辑本人当前可访问的工单"})
		return
	}
	req, uploads, err := parseTicketInput(c)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if req.Action != ticket.Action || !sameTicketID(req.ProblemID, ticket.ProblemID) || req.ExamID != nil || req.ContestID != nil {
		c.JSON(400, gin.H{"error": "编辑不能改变工单类型或所属对象"})
		return
	}
	if err = validateTicketText(req, user); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	keep := map[uint]bool{}
	var size int64
	for _, id := range req.RetainAttachmentIDs {
		found := false
		for _, a := range ticket.Attachments {
			if a.ID == id && !keep[id] {
				// Migrated single attachments have unknown size. Resolve it before
				// admitting more files so they cannot bypass the aggregate quota.
				if a.Size == 0 {
					if s.MinIO == nil {
						c.JSON(503, gin.H{"error": "无法核对历史附件大小，请稍后重试"})
						return
					}
					info, err := s.MinIO.StatObject(c.Request.Context(), s.Cfg.MinIOBucket, a.Object, minio.StatObjectOptions{})
					if err != nil || info.Size <= 0 || info.Size > problemTicketAttachmentLimit {
						c.JSON(400, gin.H{"error": "历史附件不可用或超过大小限制，请移除后重试"})
						return
					}
					a.Size = info.Size
				}
				found = true
				keep[id] = true
				size += a.Size
				break
			}
		}
		if !found {
			c.JSON(400, gin.H{"error": "保留附件无效"})
			return
		}
	}
	for _, u := range uploads {
		size += int64(len(u.body))
	}
	if len(keep)+len(uploads) > ticketFileLimit(ticket.Action) || size > ticketTotalAttachmentLimit {
		c.JSON(400, gin.H{"error": "附件超过数量或总大小限制"})
		return
	}
	if req.Revision != ticket.Revision || ticket.Status != models.ProblemChangePending {
		c.JSON(409, gin.H{"error": "工单已变更或处理，请刷新"})
		return
	}
	records, err := s.storeTicketUploads(c, uploads)
	if err != nil {
		c.JSON(503, gin.H{"error": "附件上传失败"})
		return
	}
	removed := []models.ProblemTicketAttachment{}
	err = s.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&models.ProblemChangeTicket{}).Where("id = ? AND revision = ? AND status = ? AND deleted_at IS NULL", ticket.ID, req.Revision, models.ProblemChangePending).Updates(map[string]any{"description": strings.TrimSpace(req.Description), "draft_title": strings.TrimSpace(req.DraftTitle), "draft_statement": req.DraftStatement, "revision": gorm.Expr("revision + 1"), "attachment_name": "", "attachment_object": ""})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.New("工单已变更，请刷新")
		}
		for _, a := range ticket.Attachments {
			if !keep[a.ID] {
				removed = append(removed, a)
				if err := tx.Delete(&a).Error; err != nil {
					return err
				}
			}
		}
		for i := range records {
			records[i].TicketID = ticket.ID
		}
		if len(records) > 0 {
			return tx.Create(&records).Error
		}
		return nil
	})
	if err != nil {
		s.cleanupTicketUploads(records)
		c.JSON(409, gin.H{"error": "工单已变更，保存失败，请刷新"})
		return
	}
	s.cleanupTicketUploads(removed)
	services.Audit(c, s.DB, "problem_change_ticket.update", "problem_change_ticket", ticket.ID, datatypes.JSONMap{"revision": req.Revision, "previous_description": ticket.Description})
	ticket, _ = s.problemChangeTicket(ticket.ID)
	c.JSON(200, s.ticketView(user, ticket))
}

func (s Server) cancelProblemChangeTicket(c *gin.Context)  { s.finishTicketFeedback(c, "cancel") }
func (s Server) deleteProblemChangeTicket(c *gin.Context)  { s.finishTicketFeedback(c, "delete") }
func (s Server) rejectProblemChangeTicket(c *gin.Context)  { s.finishTicketFeedback(c, "reject") }
func (s Server) resolveProblemChangeTicket(c *gin.Context) { s.finishTicketFeedback(c, "resolve") }
func (s Server) finishTicketFeedback(c *gin.Context, operation string) {
	ticket, manager, ok := s.authorizedTicket(c)
	if !ok {
		return
	}
	user, _ := middleware.CurrentUser(c)
	var req struct {
		Note     string `json:"note"`
		Revision int    `json:"revision"`
	}
	if !bind(c, &req) {
		return
	}
	if operation == "cancel" || operation == "delete" {
		writable, _ := s.ticketContextAccess(user, ticket, true)
		if user.ID != ticket.RequesterID || !writable {
			c.JSON(403, gin.H{"error": "只能删除或撤回本人当前可访问的工单"})
			return
		}
	} else {
		if !manager {
			c.JSON(403, gin.H{"error": "无权处理此工单"})
			return
		}
		if strings.TrimSpace(req.Note) == "" || len([]rune(req.Note)) > 10000 {
			c.JSON(400, gin.H{"error": "请填写处理原因（最多 10000 字）"})
			return
		}
		if operation == "resolve" && ticket.Action != models.ProblemChangeSuggest && !(ticket.Action == models.ProblemChangeReplace && ticket.ProblemID == nil) {
			c.JSON(400, gin.H{"error": "题库变更须由系统管理员执行"})
			return
		}
	}
	status := models.ProblemChangeCancelled
	if operation == "reject" {
		status = models.ProblemChangeRejected
	}
	if operation == "resolve" {
		status = models.ProblemChangeCompleted
	}
	updates := map[string]any{"status": status, "processed_at": time.Now(), "processed_by": user.ID, "revision": gorm.Expr("revision + 1")}
	if operation == "delete" {
		updates["deleted_at"] = time.Now()
	}
	if operation == "reject" || operation == "resolve" {
		updates["resolution_note"] = strings.TrimSpace(req.Note)
	}
	result := s.DB.Model(&models.ProblemChangeTicket{}).Where("id = ? AND revision = ? AND status = ? AND deleted_at IS NULL", ticket.ID, req.Revision, models.ProblemChangePending).Updates(updates)
	if result.Error != nil || result.RowsAffected != 1 {
		c.JSON(409, gin.H{"error": "工单已变更或处理，请刷新"})
		return
	}
	services.Audit(c, s.DB, "problem_change_ticket."+operation, "problem_change_ticket", ticket.ID, datatypes.JSONMap{"note": req.Note, "revision": req.Revision})
	if operation == "delete" {
		c.JSON(200, gin.H{"deleted": true})
		return
	}
	ticket, _ = s.problemChangeTicket(ticket.ID)
	c.JSON(200, s.ticketView(user, ticket))
}

func (s Server) applyProblemChangeTicket(c *gin.Context) {
	ticket, _, ok := s.authorizedTicket(c)
	if !ok {
		return
	}
	admin, _ := middleware.CurrentUser(c)
	if admin.Role != models.RoleAdmin {
		c.JSON(403, gin.H{"error": "题库变更仅系统管理员可执行"})
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, services.MaxProblemPackageSize+(2<<20))
	if err := c.Request.ParseMultipartForm(4 << 20); err != nil {
		c.JSON(400, gin.H{"error": "表单无效或题包过大"})
		return
	}
	defer c.Request.MultipartForm.RemoveAll()
	revision, _ := strconv.Atoi(c.PostForm("revision"))
	if revision != ticket.Revision || ticket.Status != models.ProblemChangePending {
		c.JSON(409, gin.H{"error": "工单已变更或处理，请刷新"})
		return
	}
	if ticket.Action == models.ProblemChangeSuggest || ticket.Action == models.ProblemChangeReplace && ticket.ProblemID == nil {
		c.JSON(400, gin.H{"error": "此工单应填写处理结果后完成"})
		return
	}
	if ticket.ProblemID != nil {
		n, err := strconv.ParseUint(c.PostForm("expected_version_id"), 10, 64)
		if err != nil || n == 0 {
			c.JSON(400, gin.H{"error": "请核对当前题目版本"})
			return
		}
		id := uint(n)
		ticket.ExpectedVersionID = &id
	}
	if err := validateProblemChangeTicketOperation(ticket.Action, c.PostForm("operation_mode")); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	note := strings.TrimSpace(c.PostForm("resolution_note"))
	if len([]rune(note)) > 10000 {
		c.JSON(400, gin.H{"error": "处理说明过长"})
		return
	}
	var err error
	if ticket.Action == models.ProblemChangeArchive {
		err = s.applyArchiveProblemTicket(ticket, admin, note)
	} else {
		file, e := c.FormFile("package")
		if e != nil {
			c.JSON(400, gin.H{"error": "请上传管理员最终完整题包"})
			return
		}
		src, e := file.Open()
		if e != nil {
			c.JSON(400, gin.H{"error": "无法读取题包"})
			return
		}
		body, e := services.ReadLimited(src, services.MaxProblemPackageSize, "题包")
		src.Close()
		if e != nil {
			c.JSON(400, gin.H{"error": e.Error()})
			return
		}
		pkg, e := services.ParseProblemPackage(body)
		if e != nil {
			c.JSON(400, gin.H{"error": e.Error()})
			return
		}
		tags := tagsJSONMap(parseTagFields(c.PostFormArray("tags"), c.PostForm("tags")))
		err = s.applyProblemPackageTicket(c, ticket, admin, body, pkg, tags, strings.TrimSpace(c.PostForm("difficulty")), note)
	}
	if err != nil {
		if !c.Writer.Written() {
			c.JSON(409, gin.H{"error": err.Error()})
		}
		return
	}
	services.Audit(c, s.DB, "problem_change_ticket.apply", "problem_change_ticket", ticket.ID, datatypes.JSONMap{"action": ticket.Action, "revision": revision})
	ticket, _ = s.problemChangeTicket(ticket.ID)
	c.JSON(200, s.ticketView(admin, ticket))
}
