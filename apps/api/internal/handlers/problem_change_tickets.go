package handlers

import (
	"errors"
	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"school-oj/apps/api/internal/models"
	"school-oj/apps/api/internal/services"
	"strings"
	"time"
)

func validateProblemChangeTicketOperation(action models.ProblemChangeAction, mode string) error {
	if action != models.ProblemChangeReplace && action != models.ProblemChangeArchive {
		return nil
	}
	if strings.TrimSpace(mode) != "overwrite" {
		return errors.New("替换或删除工单必须选择覆盖性操作")
	}
	return nil
}

func (s Server) applyArchiveProblemTicket(ticket models.ProblemChangeTicket, admin models.User, note string) error {
	if ticket.ProblemID == nil {
		return errors.New("ticket has no problem")
	}
	now := time.Now()
	return s.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&models.ProblemChangeTicket{}).
			Where("id = ? AND status = ? AND revision = ? AND deleted_at IS NULL", ticket.ID, models.ProblemChangePending, ticket.Revision).
			Updates(map[string]any{"status": models.ProblemChangeProcessing, "revision": gorm.Expr("revision + 1")})
		if result.Error != nil || result.RowsAffected != 1 {
			return errors.New("工单已被其他管理员处理")
		}
		archiveResult := tx.Model(&models.Problem{}).Where("id = ? AND current_version_id = ? AND deleted_at IS NULL AND archived_at IS NULL", *ticket.ProblemID, ticket.ExpectedVersionID).Update("archived_at", now)
		if archiveResult.Error != nil {
			return archiveResult.Error
		}
		if archiveResult.RowsAffected != 1 {
			return errors.New("题目版本已变更或题目已归档，请刷新核对后执行")
		}
		if err := tx.Model(&models.PreparedProblem{}).Where("problem_id = ?", *ticket.ProblemID).Update("archived", true).Error; err != nil {
			return err
		}
		return tx.Model(&models.ProblemChangeTicket{}).Where("id = ?", ticket.ID).Updates(map[string]any{
			"status": models.ProblemChangeCompleted, "resolution_note": note,
			"history":      ticketHistory(ticket, admin, "apply", note),
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
		err = s.replaceProblemVersion(ticket, admin, rebuiltPkg, artifacts, tags, difficulty, note)
		if err != nil {
			s.cleanupTicketPackage(artifacts)
		}
		return err
	}

	return s.createProblemFromTicket(c, ticket, admin, body, pkg, tags, difficulty, note)
}

func (s Server) replaceProblemVersion(ticket models.ProblemChangeTicket, admin models.User, pkg services.ParsedProblemPackage, artifacts problemPackageArtifacts, tags datatypes.JSONMap, difficulty, note string) error {
	now := time.Now()
	return s.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&models.ProblemChangeTicket{}).
			Where("id = ? AND status = ? AND revision = ? AND deleted_at IS NULL", ticket.ID, models.ProblemChangePending, ticket.Revision).
			Updates(map[string]any{"status": models.ProblemChangeProcessing, "revision": gorm.Expr("revision + 1")})
		if result.Error != nil || result.RowsAffected != 1 {
			return errors.New("工单已被其他管理员处理")
		}
		var problem models.Problem
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&problem, *ticket.ProblemID).Error; err != nil {
			return err
		}
		if problem.ArchivedAt != nil || problem.DeletedAt != nil || !sameTicketID(ticket.ExpectedVersionID, problem.CurrentVersionID) {
			return errors.New("题目版本已变更或已归档，请刷新核对后执行")
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
		if ticket.Action == models.ProblemChangeCreate && ticket.TargetScope == "public" {
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
			"history":            ticketHistory(ticket, admin, "apply", note),
			"applied_version_id": version.ID, "processed_by": admin.ID, "processed_at": now,
		}).Error
	})
}

func (s Server) createProblemFromTicket(c *gin.Context, ticket models.ProblemChangeTicket, admin models.User, body []byte, pkg services.ParsedProblemPackage, tags datatypes.JSONMap, difficulty, note string) error {
	now := time.Now()
	var uploaded problemPackageArtifacts
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&models.ProblemChangeTicket{}).
			Where("id = ? AND status = ? AND revision = ? AND deleted_at IS NULL", ticket.ID, models.ProblemChangePending, ticket.Revision).
			Updates(map[string]any{"status": models.ProblemChangeProcessing, "revision": gorm.Expr("revision + 1")})
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
		uploaded = artifacts
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
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("deleted_at IS NULL").First(&set, *ticket.TeamProblemSetID).Error; err != nil {
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
			var nextOrder int
			if err := tx.Model(&models.TeamProblemSetProblem{}).Where("problem_set_id = ?", *ticket.TeamProblemSetID).Select("COALESCE(MAX(sort_order), 0) + 1").Scan(&nextOrder).Error; err != nil {
				return err
			}
			link := models.TeamProblemSetProblem{ProblemSetID: *ticket.TeamProblemSetID, ProblemID: problem.ID, SortOrder: nextOrder}
			if err := tx.Create(&link).Error; err != nil {
				return err
			}
		}
		return tx.Model(&models.ProblemChangeTicket{}).Where("id = ?", ticket.ID).Updates(map[string]any{
			"problem_id": problem.ID, "status": models.ProblemChangeCompleted,
			"resolution_note": note, "applied_version_id": version.ID,
			"history":      ticketHistory(ticket, admin, "apply", note),
			"processed_by": admin.ID, "processed_at": now,
		}).Error
	})
	if err != nil {
		s.cleanupTicketPackage(uploaded)
	}
	return err
}

func (s Server) cleanupTicketPackage(artifacts problemPackageArtifacts) {
	if artifacts.Object == "" {
		return
	}
	records := []models.ProblemTicketAttachment{{Object: artifacts.Object}}
	if assets, ok := artifacts.Manifest["assets"].([]any); ok {
		for _, value := range assets {
			if asset, ok := value.(map[string]any); ok {
				if object, ok := asset["object"].(string); ok && object != "" {
					records = append(records, models.ProblemTicketAttachment{Object: object})
				}
			}
		}
	}
	s.cleanupTicketUploads(records)
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
