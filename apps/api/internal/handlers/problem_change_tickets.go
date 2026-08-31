package handlers

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"school-oj/apps/api/internal/middleware"
	"school-oj/apps/api/internal/models"
	"school-oj/apps/api/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const problemTicketAttachmentLimit = int64(32 << 20)

type problemChangeTicketInput struct {
	Action           models.ProblemChangeAction `json:"action"`
	ProblemID        *uint                      `json:"problem_id"`
	TargetScope      string                     `json:"target_scope"`
	TeamProblemSetID *uint                      `json:"team_problem_set_id"`
	Description      string                     `json:"description"`
}

func problemChangeTicketOnly(c *gin.Context) {
	c.JSON(http.StatusConflict, gin.H{
		"error":        "题目数据已统一由工单修改，请先提交工单，由管理员上传完整题包后执行",
		"ticket_route": "/problem-change-tickets/new",
	})
}

func (s Server) createProblemChangeTicket(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	if !canCreateProblems(user) {
		c.JSON(http.StatusForbidden, gin.H{"error": "problem author permission is required"})
		return
	}

	var req problemChangeTicketInput
	var attachmentBody []byte
	var attachmentName, attachmentType string
	if strings.HasPrefix(strings.ToLower(c.GetHeader("Content-Type")), "multipart/form-data") {
		req.Action = models.ProblemChangeAction(strings.TrimSpace(c.PostForm("action")))
		req.TargetScope = strings.TrimSpace(c.PostForm("target_scope"))
		req.Description = strings.TrimSpace(c.PostForm("description"))
		if value := strings.TrimSpace(c.PostForm("problem_id")); value != "" {
			id, err := strconv.ParseUint(value, 10, 64)
			if err != nil || id == 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "problem_id is invalid"})
				return
			}
			parsed := uint(id)
			req.ProblemID = &parsed
		}
		if value := strings.TrimSpace(c.PostForm("team_problem_set_id")); value != "" {
			id, err := strconv.ParseUint(value, 10, 64)
			if err != nil || id == 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "team_problem_set_id is invalid"})
				return
			}
			parsed := uint(id)
			req.TeamProblemSetID = &parsed
		}
		if file, err := c.FormFile("attachment"); err == nil {
			src, openErr := file.Open()
			if openErr != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": openErr.Error()})
				return
			}
			defer src.Close()
			attachmentBody, openErr = services.ReadLimited(src, problemTicketAttachmentLimit, "ticket attachment")
			if openErr != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": openErr.Error()})
				return
			}
			attachmentName = filepath.Base(file.Filename)
			attachmentType = file.Header.Get("Content-Type")
		}
	} else if !bind(c, &req) {
		return
	}

	if req.Action != models.ProblemChangeCreate && req.Action != models.ProblemChangeReplace && req.Action != models.ProblemChangeArchive {
		c.JSON(http.StatusBadRequest, gin.H{"error": "action must be create, replace or archive"})
		return
	}
	if len([]rune(strings.TrimSpace(req.Description))) < 8 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请至少用 8 个字说明修改需求"})
		return
	}
	if req.TargetScope == "" {
		req.TargetScope = "public"
	}
	if req.TargetScope != "public" && req.TargetScope != "prepared" && req.TargetScope != "team_problem_set" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "target_scope must be public, prepared or team_problem_set"})
		return
	}
	if req.Action == models.ProblemChangeCreate {
		if req.ProblemID != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "新增工单不能指定现有题目"})
			return
		}
		if req.TargetScope == "team_problem_set" {
			if req.TeamProblemSetID == nil || !s.canSubmitTicketToProblemSet(c, user, *req.TeamProblemSetID) {
				return
			}
		}
	} else {
		if req.ProblemID == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "替换或删除工单必须指定题目"})
			return
		}
		var problem models.Problem
		if err := s.DB.Where("deleted_at IS NULL AND archived_at IS NULL").First(&problem, *req.ProblemID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "problem not found"})
			return
		}
		if user.Role != models.RoleAdmin && problem.OwnerID != user.ID {
			c.JSON(http.StatusForbidden, gin.H{"error": "只能为自己拥有的题目发起工单"})
			return
		}
		if req.Action == models.ProblemChangeArchive {
			req.TargetScope = "public"
		} else if req.TargetScope == "team_problem_set" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "替换工单范围只能是公共题库或预备题库"})
			return
		}
		req.TeamProblemSetID = nil
	}

	ticket := models.ProblemChangeTicket{
		RequesterID:      user.ID,
		ProblemID:        req.ProblemID,
		Action:           req.Action,
		Status:           models.ProblemChangePending,
		TargetScope:      req.TargetScope,
		TeamProblemSetID: req.TeamProblemSetID,
		Description:      strings.TrimSpace(req.Description),
	}
	if len(attachmentBody) > 0 {
		object := fmt.Sprintf("problem-change-tickets/%d/%d-%s", user.ID, time.Now().UnixNano(), attachmentName)
		if _, err := s.MinIO.PutObject(c.Request.Context(), s.Cfg.MinIOBucket, object, bytes.NewReader(attachmentBody), int64(len(attachmentBody)), minio.PutObjectOptions{ContentType: attachmentType}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		ticket.AttachmentObject = object
		ticket.AttachmentName = attachmentName
	}
	if err := s.DB.Create(&ticket).Error; err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "idx_problem_change_tickets_active_problem") {
			c.JSON(http.StatusConflict, gin.H{"error": "该题目已有未完成的替换或删除工单"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	services.Audit(c, s.DB, "problem_change_ticket.create", "problem_change_ticket", ticket.ID, datatypes.JSONMap{"action": ticket.Action, "problem_id": ticket.ProblemID})
	s.loadProblemChangeTicket(&ticket)
	c.JSON(http.StatusCreated, ticket)
}

func (s Server) canSubmitTicketToProblemSet(c *gin.Context, user models.User, setID uint) bool {
	var set models.TeamProblemSet
	if err := s.DB.Where("deleted_at IS NULL").First(&set, setID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "problem set not found"})
		return false
	}
	var team models.Team
	if err := s.DB.First(&team, set.TeamID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "team not found"})
		return false
	}
	if !s.canTeamContentPermission(user, team) {
		c.JSON(http.StatusForbidden, gin.H{"error": "当前团队职级不能向该题单新增题目"})
		return false
	}
	return true
}

func (s Server) listMyProblemChangeTickets(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	var tickets []models.ProblemChangeTicket
	if err := s.DB.Where("requester_id = ?", user.ID).Order("created_at DESC").Find(&tickets).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s.loadProblemChangeTickets(tickets)
	c.JSON(http.StatusOK, tickets)
}

func (s Server) listProblemChangeTicketEligibleProblems(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	if !canCreateProblems(user) {
		c.JSON(http.StatusForbidden, gin.H{"error": "problem author permission is required"})
		return
	}
	query := s.DB.Model(&models.Problem{}).Where("deleted_at IS NULL AND archived_at IS NULL")
	if user.Role != models.RoleAdmin {
		query = query.Where("owner_id = ?", user.ID)
	}
	var problems []models.Problem
	if err := query.Order("updated_at DESC, id DESC").Find(&problems).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, problems)
}

func (s Server) listProblemChangeTickets(c *gin.Context) {
	q := s.DB.Model(&models.ProblemChangeTicket{})
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		q = q.Where("status = ?", status)
	}
	if action := strings.TrimSpace(c.Query("action")); action != "" {
		q = q.Where("action = ?", action)
	}
	var tickets []models.ProblemChangeTicket
	if err := q.Order("CASE status WHEN 'pending' THEN 0 WHEN 'processing' THEN 1 ELSE 2 END, created_at ASC").Find(&tickets).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s.loadProblemChangeTickets(tickets)
	c.JSON(http.StatusOK, tickets)
}

func (s Server) getProblemChangeTicket(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	ticket, ok := s.problemChangeTicketByParam(c)
	if !ok {
		return
	}
	if user.Role != models.RoleAdmin && ticket.RequesterID != user.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	if user.Role == models.RoleAdmin {
		s.loadProblemChangeTicketImpact(&ticket)
	}
	c.JSON(http.StatusOK, ticket)
}

func (s Server) downloadProblemChangeTicketAttachment(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	ticket, ok := s.problemChangeTicketByParam(c)
	if !ok {
		return
	}
	if user.Role != models.RoleAdmin && ticket.RequesterID != user.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	if ticket.AttachmentObject == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "attachment not found"})
		return
	}
	object, err := s.MinIO.GetObject(c.Request.Context(), s.Cfg.MinIOBucket, ticket.AttachmentObject, minio.GetObjectOptions{})
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "attachment not found"})
		return
	}
	defer object.Close()
	info, err := object.Stat()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "attachment not found"})
		return
	}
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, strings.ReplaceAll(ticket.AttachmentName, `"`, "")))
	c.DataFromReader(http.StatusOK, info.Size, info.ContentType, object, nil)
}

func (s Server) cancelProblemChangeTicket(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	ticket, ok := s.problemChangeTicketByParam(c)
	if !ok {
		return
	}
	if user.Role != models.RoleAdmin && ticket.RequesterID != user.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	result := s.DB.Model(&models.ProblemChangeTicket{}).
		Where("id = ? AND status = ?", ticket.ID, models.ProblemChangePending).
		Updates(map[string]any{"status": models.ProblemChangeCancelled, "processed_at": time.Now()})
	if result.Error != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": result.Error.Error()})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "只有待处理工单可以取消"})
		return
	}
	services.Audit(c, s.DB, "problem_change_ticket.cancel", "problem_change_ticket", ticket.ID, nil)
	ticket, _ = s.problemChangeTicket(ticket.ID)
	c.JSON(http.StatusOK, ticket)
}

func (s Server) rejectProblemChangeTicket(c *gin.Context) {
	admin, _ := middleware.CurrentUser(c)
	var req struct {
		Note string `json:"note"`
	}
	if !bind(c, &req) {
		return
	}
	note := strings.TrimSpace(req.Note)
	if note == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请填写驳回原因"})
		return
	}
	id, ok := idParam(c, "id")
	if !ok {
		return
	}
	now := time.Now()
	result := s.DB.Model(&models.ProblemChangeTicket{}).
		Where("id = ? AND status IN ?", id, []models.ProblemChangeStatus{models.ProblemChangePending, models.ProblemChangeProcessing}).
		Updates(map[string]any{"status": models.ProblemChangeRejected, "resolution_note": note, "processed_by": admin.ID, "processed_at": now})
	if result.Error != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": result.Error.Error()})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "工单已处理或不存在"})
		return
	}
	services.Audit(c, s.DB, "problem_change_ticket.reject", "problem_change_ticket", id, datatypes.JSONMap{"note": note})
	ticket, _ := s.problemChangeTicket(id)
	c.JSON(http.StatusOK, ticket)
}

func (s Server) applyProblemChangeTicket(c *gin.Context) {
	admin, _ := middleware.CurrentUser(c)
	id, ok := idParam(c, "id")
	if !ok {
		return
	}
	ticket, found := s.problemChangeTicket(id)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "ticket not found"})
		return
	}
	if ticket.Status != models.ProblemChangePending {
		c.JSON(http.StatusConflict, gin.H{"error": "工单已处理或正在处理"})
		return
	}
	note := strings.TrimSpace(c.PostForm("resolution_note"))
	if ticket.Action == models.ProblemChangeArchive {
		if err := s.applyArchiveProblemTicket(ticket, admin, note); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	} else {
		file, err := c.FormFile("package")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "新增或替换工单必须上传完整最终题包"})
			return
		}
		src, err := file.Open()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		body, err := services.ReadLimited(src, services.MaxProblemPackageSize, "problem package")
		_ = src.Close()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		pkg, err := services.ParseProblemPackage(body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		tags := tagsJSONMap(parseTagFields(c.PostFormArray("tags"), c.PostForm("tags")))
		difficulty := strings.TrimSpace(c.PostForm("difficulty"))
		if err := s.applyProblemPackageTicket(c, ticket, admin, body, pkg, tags, difficulty, note); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	services.Audit(c, s.DB, "problem_change_ticket.apply", "problem_change_ticket", ticket.ID, datatypes.JSONMap{"action": ticket.Action})
	applied, _ := s.problemChangeTicket(ticket.ID)
	c.JSON(http.StatusOK, applied)
}

func (s Server) applyArchiveProblemTicket(ticket models.ProblemChangeTicket, admin models.User, note string) error {
	if ticket.ProblemID == nil {
		return errors.New("ticket has no problem")
	}
	now := time.Now()
	return s.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&models.ProblemChangeTicket{}).
			Where("id = ? AND status = ?", ticket.ID, models.ProblemChangePending).
			Update("status", models.ProblemChangeProcessing)
		if result.Error != nil || result.RowsAffected != 1 {
			return errors.New("工单已被其他管理员处理")
		}
		archiveResult := tx.Model(&models.Problem{}).Where("id = ? AND deleted_at IS NULL AND archived_at IS NULL", *ticket.ProblemID).Update("archived_at", now)
		if archiveResult.Error != nil {
			return archiveResult.Error
		}
		if archiveResult.RowsAffected != 1 {
			return errors.New("题目不存在或已归档")
		}
		if err := tx.Model(&models.PreparedProblem{}).Where("problem_id = ?", *ticket.ProblemID).Update("archived", true).Error; err != nil {
			return err
		}
		return tx.Model(&models.ProblemChangeTicket{}).Where("id = ?", ticket.ID).Updates(map[string]any{
			"status": models.ProblemChangeCompleted, "resolution_note": note,
			"processed_by": admin.ID, "processed_at": now,
		}).Error
	})
}

func (s Server) applyProblemPackageTicket(c *gin.Context, ticket models.ProblemChangeTicket, admin models.User, body []byte, pkg services.ParsedProblemPackage, tags datatypes.JSONMap, difficulty, note string) error {
	if ticket.Action == models.ProblemChangeReplace || (ticket.Action == models.ProblemChangeCreate && ticket.ProblemID != nil) {
		if ticket.ProblemID == nil {
			return errors.New("ticket has no problem")
		}
		var problem models.Problem
		if err := s.DB.First(&problem, *ticket.ProblemID).Error; err != nil {
			return err
		}
		pkg.Manifest.Slug = problem.Slug
		rebuiltBody, rebuiltPkg, err := services.RebuildProblemPackage(body, pkg.Manifest, nil)
		if err != nil {
			return err
		}
		artifacts, ok := s.uploadProblemPackageArtifacts(c, rebuiltBody, &rebuiltPkg)
		if !ok {
			return errors.New("upload problem package artifacts")
		}
		if len(tags) == 0 {
			tags = problem.Tags
		}
		if difficulty == "" {
			difficulty = problem.Difficulty
		}
		return s.replaceProblemVersion(ticket, admin, rebuiltPkg, artifacts, tags, difficulty, note)
	}

	return s.createProblemFromTicket(c, ticket, admin, body, pkg, tags, difficulty, note)
}

func (s Server) replaceProblemVersion(ticket models.ProblemChangeTicket, admin models.User, pkg services.ParsedProblemPackage, artifacts problemPackageArtifacts, tags datatypes.JSONMap, difficulty, note string) error {
	now := time.Now()
	return s.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&models.ProblemChangeTicket{}).
			Where("id = ? AND status = ?", ticket.ID, models.ProblemChangePending).
			Update("status", models.ProblemChangeProcessing)
		if result.Error != nil || result.RowsAffected != 1 {
			return errors.New("工单已被其他管理员处理")
		}
		var problem models.Problem
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&problem, *ticket.ProblemID).Error; err != nil {
			return err
		}
		var versionNo int
		if err := tx.Model(&models.ProblemVersion{}).Where("problem_id = ?", problem.ID).Select("COALESCE(MAX(version), 0) + 1").Scan(&versionNo).Error; err != nil {
			return err
		}
		version := models.ProblemVersion{
			ProblemID: problem.ID, Version: versionNo, Title: pkg.Manifest.Title,
			Statement: pkg.Manifest.Statement, Tags: tags,
			Difficulty:  normalizeProblemDifficulty(difficulty, tags),
			TimeLimitMS: pkg.Manifest.TimeLimitMS, MemoryLimitMB: pkg.Manifest.MemoryLimitMB,
			OutputLimitKB: pkg.Manifest.OutputLimitKB, PackageObject: artifacts.Object,
			PackageChecksum: artifacts.Checksum, Manifest: artifacts.Manifest, CreatedBy: admin.ID,
		}
		if err := tx.Create(&version).Error; err != nil {
			return err
		}
		updates := map[string]any{
			"title": version.Title, "statement": version.Statement, "tags": version.Tags,
			"difficulty": version.Difficulty, "time_limit_ms": version.TimeLimitMS,
			"memory_limit_mb": version.MemoryLimitMB, "output_limit_kb": version.OutputLimitKB,
			"package_object": version.PackageObject, "package_checksum": version.PackageChecksum,
			"manifest": version.Manifest, "current_version_id": version.ID, "updated_at": now,
		}
		if err := tx.Model(&models.Problem{}).Where("id = ?", problem.ID).Updates(updates).Error; err != nil {
			return err
		}
		if ticket.TargetScope == "public" {
			if err := tx.Model(&models.PreparedProblem{}).Where("problem_id = ? AND published_at IS NULL", problem.ID).Update("published_at", now).Error; err != nil {
				return err
			}
		}
		if ticket.Action == models.ProblemChangeCreate {
			if err := tx.Model(&models.ProblemReview{}).Where("problem_id = ? AND status = ?", problem.ID, models.ProblemReviewPending).Updates(map[string]any{
				"status": models.ProblemReviewApproved, "review_note": note,
				"reviewed_by": admin.ID, "reviewed_at": now,
			}).Error; err != nil {
				return err
			}
		}
		if err := tx.Model(&models.ExamProblem{}).
			Where("problem_id = ? AND exam_id IN (?)", problem.ID, tx.Model(&models.Exam{}).Select("id").Where("deleted_at IS NULL AND (starts_at IS NULL OR starts_at > ?)", now)).
			Update("problem_version_id", version.ID).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.TeamContestProblem{}).
			Where("problem_id = ? AND contest_id IN (?)", problem.ID, tx.Model(&models.TeamContest{}).Select("id").Where("deleted_at IS NULL AND state IN ? AND (starts_at IS NULL OR starts_at > ?)", []string{models.TeamContestDraft, models.TeamContestPublished}, now)).
			Update("problem_version_id", version.ID).Error; err != nil {
			return err
		}
		return tx.Model(&models.ProblemChangeTicket{}).Where("id = ?", ticket.ID).Updates(map[string]any{
			"status": models.ProblemChangeCompleted, "resolution_note": note,
			"applied_version_id": version.ID, "processed_by": admin.ID, "processed_at": now,
		}).Error
	})
}

func (s Server) createProblemFromTicket(c *gin.Context, ticket models.ProblemChangeTicket, admin models.User, body []byte, pkg services.ParsedProblemPackage, tags datatypes.JSONMap, difficulty, note string) error {
	now := time.Now()
	return s.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&models.ProblemChangeTicket{}).
			Where("id = ? AND status = ?", ticket.ID, models.ProblemChangePending).
			Update("status", models.ProblemChangeProcessing)
		if result.Error != nil || result.RowsAffected != 1 {
			return errors.New("工单已被其他管理员处理")
		}
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", int64(734956411)).Error; err != nil {
			return err
		}
		slug, err := nextProblemInternalSlug(tx)
		if err != nil {
			return err
		}
		displayCode, err := nextProblemDisplayCode(tx)
		if err != nil {
			return err
		}
		pkg.Manifest.Slug = slug
		rebuiltBody, rebuiltPkg, err := services.RebuildProblemPackage(body, pkg.Manifest, nil)
		if err != nil {
			return err
		}
		artifacts, ok := s.uploadProblemPackageArtifacts(c, rebuiltBody, &rebuiltPkg)
		if !ok {
			return errors.New("upload problem package artifacts")
		}
		problem := models.Problem{
			OwnerID: ticket.RequesterID, DisplayCode: displayCode, Slug: slug,
			Title: rebuiltPkg.Manifest.Title, Statement: rebuiltPkg.Manifest.Statement,
			Tags: tags, Difficulty: normalizeProblemDifficulty(difficulty, tags),
			TimeLimitMS: rebuiltPkg.Manifest.TimeLimitMS, MemoryLimitMB: rebuiltPkg.Manifest.MemoryLimitMB,
			OutputLimitKB: rebuiltPkg.Manifest.OutputLimitKB, PackageObject: artifacts.Object,
			PackageChecksum: artifacts.Checksum, Manifest: artifacts.Manifest,
		}
		if ticket.TargetScope == "team_problem_set" && ticket.TeamProblemSetID != nil {
			var set models.TeamProblemSet
			if err := tx.First(&set, *ticket.TeamProblemSetID).Error; err != nil {
				return err
			}
			problem.TeamID = &set.TeamID
		}
		if err := tx.Create(&problem).Error; err != nil {
			return err
		}
		version := models.ProblemVersion{
			ProblemID: problem.ID, Version: 1, Title: problem.Title, Statement: problem.Statement,
			Tags: problem.Tags, Difficulty: problem.Difficulty, TimeLimitMS: problem.TimeLimitMS,
			MemoryLimitMB: problem.MemoryLimitMB, OutputLimitKB: problem.OutputLimitKB,
			PackageObject: problem.PackageObject, PackageChecksum: problem.PackageChecksum,
			Manifest: problem.Manifest, CreatedBy: admin.ID,
		}
		if err := tx.Create(&version).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.Problem{}).Where("id = ?", problem.ID).Update("current_version_id", version.ID).Error; err != nil {
			return err
		}
		if ticket.TargetScope == "prepared" {
			prepared := models.PreparedProblem{ProblemID: problem.ID, OwnerID: ticket.RequesterID, Difficulty: problem.Difficulty}
			if err := tx.Create(&prepared).Error; err != nil {
				return err
			}
		}
		if ticket.TargetScope == "team_problem_set" && ticket.TeamProblemSetID != nil {
			var count int64
			if err := tx.Model(&models.TeamProblemSetProblem{}).Where("problem_set_id = ?", *ticket.TeamProblemSetID).Count(&count).Error; err != nil {
				return err
			}
			link := models.TeamProblemSetProblem{ProblemSetID: *ticket.TeamProblemSetID, ProblemID: problem.ID, SortOrder: int(count)}
			if err := tx.Create(&link).Error; err != nil {
				return err
			}
		}
		return tx.Model(&models.ProblemChangeTicket{}).Where("id = ?", ticket.ID).Updates(map[string]any{
			"problem_id": problem.ID, "status": models.ProblemChangeCompleted,
			"resolution_note": note, "applied_version_id": version.ID,
			"processed_by": admin.ID, "processed_at": now,
		}).Error
	})
}

func (s Server) problemChangeTicketByParam(c *gin.Context) (models.ProblemChangeTicket, bool) {
	id, ok := idParam(c, "id")
	if !ok {
		return models.ProblemChangeTicket{}, false
	}
	ticket, found := s.problemChangeTicket(id)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "ticket not found"})
		return models.ProblemChangeTicket{}, false
	}
	return ticket, true
}

func (s Server) problemChangeTicket(id uint) (models.ProblemChangeTicket, bool) {
	var ticket models.ProblemChangeTicket
	if err := s.DB.Preload("Requester").Preload("Problem").First(&ticket, id).Error; err != nil {
		return models.ProblemChangeTicket{}, false
	}
	return ticket, true
}

func (s Server) loadProblemChangeTicket(ticket *models.ProblemChangeTicket) {
	s.DB.Preload("Requester").Preload("Problem").First(ticket, ticket.ID)
}

func (s Server) loadProblemChangeTickets(tickets []models.ProblemChangeTicket) {
	for index := range tickets {
		s.loadProblemChangeTicket(&tickets[index])
	}
}

func (s Server) loadProblemChangeTicketImpact(ticket *models.ProblemChangeTicket) {
	if ticket.ProblemID == nil {
		return
	}
	now := time.Now()
	impact := &models.ProblemChangeImpactSummary{}
	s.DB.Model(&models.ExamProblem{}).
		Joins("JOIN exams ON exams.id = exam_problems.exam_id").
		Where("exam_problems.problem_id = ? AND exams.deleted_at IS NULL AND (exams.starts_at IS NULL OR exams.starts_at > ?)", *ticket.ProblemID, now).
		Count(&impact.FutureExams)
	s.DB.Model(&models.ExamProblem{}).
		Joins("JOIN exams ON exams.id = exam_problems.exam_id").
		Where("exam_problems.problem_id = ? AND exams.deleted_at IS NULL AND exams.starts_at IS NOT NULL AND exams.starts_at <= ?", *ticket.ProblemID, now).
		Count(&impact.PinnedExams)
	s.DB.Model(&models.TeamContestProblem{}).
		Joins("JOIN team_contests ON team_contests.id = team_contest_problems.contest_id").
		Where("team_contest_problems.problem_id = ? AND team_contests.deleted_at IS NULL AND team_contests.state IN ? AND (team_contests.starts_at IS NULL OR team_contests.starts_at > ?)", *ticket.ProblemID, []string{models.TeamContestDraft, models.TeamContestPublished}, now).
		Count(&impact.FutureContests)
	s.DB.Model(&models.TeamContestProblem{}).
		Joins("JOIN team_contests ON team_contests.id = team_contest_problems.contest_id").
		Where("team_contest_problems.problem_id = ? AND team_contests.deleted_at IS NULL AND (team_contests.state NOT IN ? OR (team_contests.starts_at IS NOT NULL AND team_contests.starts_at <= ?))", *ticket.ProblemID, []string{models.TeamContestDraft, models.TeamContestPublished}, now).
		Count(&impact.PinnedContests)
	s.DB.Model(&models.Submission{}).Where("problem_id = ?", *ticket.ProblemID).Count(&impact.HistoricalSubmits)
	ticket.ImpactSummary = impact
}
