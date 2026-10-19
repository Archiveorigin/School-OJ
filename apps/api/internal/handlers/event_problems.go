package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"school-oj/apps/api/internal/middleware"
	"school-oj/apps/api/internal/models"
	"school-oj/apps/api/internal/services"
)

type eventProblemInput struct {
	services.ProblemPackageDraft
	Revision          int    `json:"revision"`
	ExpectedVersionID uint   `json:"expected_version_id"`
	TestMode          string `json:"test_mode"`
	Label             string `json:"label"`
	Score             int    `json:"score"`
}

func (s Server) saveEventProblem(c *gin.Context) {
	p, ok := s.eventPaperByParams(c, true)
	if !ok {
		return
	}
	release, err := acquireTicketPackage(c.Request.Context())
	if err != nil {
		eventError(c, err)
		return
	}
	defer release()
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, services.MaxProblemPackageSize)
	defer func() {
		if c.Request.MultipartForm != nil {
			_ = c.Request.MultipartForm.RemoveAll()
		}
	}()
	var req eventProblemInput
	if strings.Contains(c.ContentType(), "multipart/form-data") {
		if err := c.Request.ParseMultipartForm(4 << 20); err != nil {
			c.JSON(400, gin.H{"error": "题目文件过大或格式错误"})
			return
		}
		if err := json.Unmarshal([]byte(c.PostForm("draft")), &req); err != nil {
			c.JSON(400, gin.H{"error": "题目草稿格式错误"})
			return
		}
		uploads, present, err := optionalTestPointUploadsFromMultipart(c.Request.MultipartForm)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		if present {
			req.Cases, err = services.BuildNamedProblemCasesFromTestPointFiles(uploads)
			if err != nil {
				c.JSON(400, gin.H{"error": err.Error()})
				return
			}
		}
	} else if !bind(c, &req) {
		return
	}
	if err := requireEventDraft(p, req.Revision); err != nil {
		eventError(c, err)
		return
	}
	if req.Score == 0 {
		req.Score = 100
	}
	req.Label = strings.TrimSpace(req.Label)
	if req.Score < 1 || req.Score > 1000 || len(req.Label) > 16 || strings.TrimSpace(req.Title) == "" || len([]rune(req.Title)) > 200 || strings.TrimSpace(req.Statement) == "" || len(req.Statement) > 200000 {
		c.JSON(400, gin.H{"error": "请填写有效标题、题面、题号和分值"})
		return
	}
	if req.TimeLimitMS < 1 || req.TimeLimitMS > 60000 || req.MemoryLimitMB < 1 || req.MemoryLimitMB > 4096 || req.OutputLimitKB < 1 || req.OutputLimitKB > 65536 {
		c.JSON(400, gin.H{"error": "判题限制超出允许范围"})
		return
	}
	var existing models.Problem
	if raw := c.Param("problem_id"); raw != "" {
		id, err := strconv.ParseUint(raw, 10, 64)
		if err != nil || id == 0 {
			c.JSON(400, gin.H{"error": "题目编号无效"})
			return
		}
		if err = s.DB.Where("event_kind=? AND event_id=? AND deleted_at IS NULL", p.kind, p.id).First(&existing, id).Error; err != nil {
			eventError(c, err)
			return
		}
		if existing.CurrentVersionID == nil || *existing.CurrentVersionID != req.ExpectedVersionID {
			c.JSON(409, gin.H{"error": "题目版本已变更，请刷新"})
			return
		}
	}
	if req.TestMode == "" {
		req.TestMode = "preserve"
	}
	if req.TestMode != "preserve" && req.TestMode != "merge" && req.TestMode != "replace" {
		c.JSON(400, gin.H{"error": "请选择保留、合并或完整替换测试点"})
		return
	}
	if existing.ID == 0 && (req.TestMode == "preserve" || len(req.Cases) == 0) {
		c.JSON(400, gin.H{"error": "新题必须提供至少一组完整测试点"})
		return
	}
	if req.TestMode == "preserve" && len(req.Cases) > 0 {
		c.JSON(400, gin.H{"error": "已上传测试点，请明确选择合并或完整替换"})
		return
	}
	if req.TestMode != "preserve" && len(req.Cases) == 0 {
		c.JSON(400, gin.H{"error": "请选择要合并或替换的测试文件"})
		return
	}
	var body []byte
	var pkg services.ParsedProblemPackage
	if existing.ID == 0 {
		var nonce [16]byte
		if _, err = rand.Read(nonce[:]); err != nil {
			eventError(c, err)
			return
		}
		req.Slug = "event-" + hex.EncodeToString(nonce[:])
		req.TeamID = nil
		req.ProblemSetID = nil
		body, pkg, err = services.BuildProblemPackage(req.ProblemPackageDraft)
	} else {
		var base []byte
		base, err = s.readTicketObject(c.Request.Context(), existing.PackageObject, services.MaxProblemPackageSize)
		if err == nil {
			var old services.ParsedProblemPackage
			old, err = services.ParseProblemPackage(base)
			if err == nil {
				manifest := old.Manifest
				manifest.Title = req.Title
				manifest.Statement = req.Statement
				manifest.TimeLimitMS = req.TimeLimitMS
				manifest.MemoryLimitMB = req.MemoryLimitMB
				manifest.OutputLimitKB = req.OutputLimitKB
				manifest.Checker = req.Checker
				cases := req.Cases
				if req.TestMode == "preserve" {
					cases = nil
				} else if req.TestMode == "merge" {
					cases, err = services.MergeProblemCases(base, cases)
				}
				var assets []services.ParsedProblemAsset
				for _, asset := range req.Assets {
					if err != nil {
						break
					}
					var data []byte
					data, err = services.DecodeProblemAssetData(asset.Data)
					if err != nil {
						break
					}
					assetPath := asset.Path
					if assetPath == "" {
						assetPath = "assets/" + asset.Name
					}
					assetPath, err = services.NormalizeAssetPath(assetPath)
					if err != nil {
						break
					}
					var contentType string
					contentType, err = services.ValidateProblemAsset(assetPath, data)
					if err != nil {
						break
					}
					assets = append(assets, services.ParsedProblemAsset{Name: asset.Name, Path: assetPath, ContentType: contentType, Body: data, Size: int64(len(data))})
				}
				if err == nil {
					body, pkg, err = services.RebuildProblemPackage(base, manifest, cases, assets...)
				}
			}
		}
	}
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	artifacts, ok := s.uploadProblemPackageArtifacts(c, body, &pkg)
	if !ok {
		return
	}
	u, _ := middleware.CurrentUser(c)
	var saved models.Problem
	err = s.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		locked, e := loadEventPaper(tx, p.kind, p.id, true)
		if e != nil {
			return e
		}
		p = locked
		scoped := s
		scoped.DB = tx
		if !scoped.canManageEventPaper(u, p) {
			return eventFail(403, "管理权限已变化")
		}
		if e = requireEventDraft(p, req.Revision); e != nil {
			return e
		}
		links, e := eventPaperProblems(tx, p)
		if e != nil {
			return e
		}
		if existing.ID == 0 && len(links) >= 100 {
			return eventFail(400, "每场最多 100 道题目")
		}
		if req.Label == "" {
			req.Label = defaultProblemLabel(len(links))
		}
		for _, link := range links {
			if link.ProblemID != existing.ID && strings.EqualFold(link.Label, req.Label) {
				return eventFail(400, "题号重复")
			}
		}
		saved = existing
		if existing.ID == 0 {
			saved = models.Problem{OwnerID: u.ID, EventKind: p.kind, EventID: &p.id, Slug: pkg.Manifest.Slug, DisplayCode: "E" + strings.TrimPrefix(pkg.Manifest.Slug, "event-")[:14]}
		} else {
			if e = tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&saved, existing.ID).Error; e != nil {
				return e
			}
			if saved.CurrentVersionID == nil || *saved.CurrentVersionID != req.ExpectedVersionID {
				return eventFail(409, "题目版本已变更")
			}
			found := false
			for _, link := range links {
				if link.ProblemID == saved.ID {
					found = true
				}
			}
			if !found {
				return eventFail(409, "题目已从本场移除")
			}
		}
		saved.Title = pkg.Manifest.Title
		saved.Statement = pkg.Manifest.Statement
		saved.TimeLimitMS = pkg.Manifest.TimeLimitMS
		saved.MemoryLimitMB = pkg.Manifest.MemoryLimitMB
		saved.OutputLimitKB = pkg.Manifest.OutputLimitKB
		saved.Tags = tagsJSONMap(req.Tags)
		saved.Difficulty = normalizeProblemDifficulty(req.Difficulty, saved.Tags)
		saved.PackageObject = artifacts.Object
		saved.PackageChecksum = artifacts.Checksum
		saved.Manifest = artifacts.Manifest
		if existing.ID == 0 {
			if e = tx.Create(&saved).Error; e != nil {
				return e
			}
		}
		var number int
		if e = tx.Model(&models.ProblemVersion{}).Where("problem_id=?", saved.ID).Select("COALESCE(MAX(version),0)+1").Scan(&number).Error; e != nil {
			return e
		}
		v := models.ProblemVersion{ProblemID: saved.ID, Version: number, Title: saved.Title, Statement: saved.Statement, Tags: saved.Tags, Difficulty: saved.Difficulty, TimeLimitMS: saved.TimeLimitMS, MemoryLimitMB: saved.MemoryLimitMB, OutputLimitKB: saved.OutputLimitKB, PackageObject: saved.PackageObject, PackageChecksum: saved.PackageChecksum, Manifest: saved.Manifest, CreatedBy: u.ID}
		if e = tx.Create(&v).Error; e != nil {
			return e
		}
		saved.CurrentVersionID = &v.ID
		if e = tx.Omit("CurrentVersion").Save(&saved).Error; e != nil {
			return e
		}
		if p.kind == "exam" {
			if existing.ID == 0 {
				e = tx.Create(&models.ExamProblem{ExamID: p.id, ProblemID: saved.ID, ProblemVersionID: v.ID, Label: req.Label, Score: req.Score, SortOrder: len(links)}).Error
			} else {
				e = tx.Model(&models.ExamProblem{}).Where("exam_id=? AND problem_id=?", p.id, saved.ID).Updates(map[string]any{"problem_version_id": v.ID, "label": req.Label, "score": req.Score}).Error
			}
		} else {
			if existing.ID == 0 {
				e = tx.Create(&models.TeamContestProblem{ContestID: p.id, ProblemID: saved.ID, ProblemVersionID: v.ID, Label: req.Label, SortOrder: len(links)}).Error
			} else {
				e = tx.Model(&models.TeamContestProblem{}).Where("contest_id=? AND problem_id=?", p.id, saved.ID).Updates(map[string]any{"problem_version_id": v.ID, "label": req.Label}).Error
			}
		}
		if e != nil {
			return e
		}
		return bumpEventRevision(tx, p)
	})
	if err != nil {
		s.cleanupTicketPackage(artifacts)
		eventError(c, err)
		return
	}
	services.Audit(c, s.DB, "event.problem.save", p.kind, p.id, nil)
	c.JSON(200, saved)
}

func (s Server) updateEventProblemSelection(c *gin.Context) {
	p, ok := s.eventPaperByParams(c, true)
	if !ok {
		return
	}
	var req struct {
		Revision int `json:"revision"`
		Problems []struct {
			ProblemID        uint   `json:"problem_id"`
			ProblemVersionID uint   `json:"problem_version_id"`
			Label            string `json:"label"`
			Score            int    `json:"score"`
			UseLatest        bool   `json:"use_latest"`
		} `json:"problems"`
	}
	if !bind(c, &req) {
		return
	}
	if len(req.Problems) > 100 {
		c.JSON(400, gin.H{"error": "每场最多 100 道题目"})
		return
	}
	u, _ := middleware.CurrentUser(c)
	err := s.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		locked, e := loadEventPaper(tx, p.kind, p.id, true)
		if e != nil {
			return e
		}
		p = locked
		scoped := s
		scoped.DB = tx
		if !scoped.canManageEventPaper(u, p) {
			return eventFail(403, "管理权限已变化")
		}
		if e = requireEventDraft(p, req.Revision); e != nil {
			return e
		}
		old, e := eventPaperProblems(tx, p)
		if e != nil {
			return e
		}
		versions := map[uint]uint{}
		for _, link := range old {
			versions[link.ProblemID] = link.ProblemVersionID
		}
		seen := map[uint]bool{}
		labels := map[string]bool{}
		examLinks := []models.ExamProblem{}
		contestLinks := []models.TeamContestProblem{}
		for index, item := range req.Problems {
			label := strings.TrimSpace(item.Label)
			if label == "" {
				label = fmt.Sprintf("%c", 'A'+index%26)
				if index >= 26 {
					label = fmt.Sprintf("P%d", index+1)
				}
			}
			if item.ProblemID == 0 || seen[item.ProblemID] || labels[strings.ToLower(label)] || len(label) > 16 {
				return eventFail(400, "题目或题号重复、无效")
			}
			seen[item.ProblemID] = true
			labels[strings.ToLower(label)] = true
			var problem models.Problem
			if e = tx.Where("deleted_at IS NULL AND archived_at IS NULL").First(&problem, item.ProblemID).Error; e != nil {
				return eventFail(400, "所选题目不可用")
			}
			if problem.EventKind != "" {
				if problem.EventKind != p.kind || problem.EventID == nil || *problem.EventID != p.id {
					return eventFail(403, "不能使用其他场次的专属题目")
				}
			} else if !scoped.isProblemPublic(problem.ID) && !scoped.canManageProblemData(u, problem) && !(p.kind == "contest" && problem.TeamID != nil && *problem.TeamID == p.contest.TeamID) {
				return eventFail(403, "无权使用该题目")
			}
			versionID := versions[problem.ID]
			if versionID == 0 || item.UseLatest {
				if problem.CurrentVersionID == nil {
					return eventFail(409, "题目没有可用版本")
				}
				versionID = *problem.CurrentVersionID
			}
			if item.ProblemVersionID != 0 && item.ProblemVersionID != versions[problem.ID] && !item.UseLatest {
				return eventFail(409, "题目版本已变化，请刷新")
			}
			if item.Score == 0 {
				item.Score = 100
			}
			if item.Score < 1 || item.Score > 1000 {
				return eventFail(400, "分值范围为 1 至 1000")
			}
			examLinks = append(examLinks, models.ExamProblem{ExamID: p.id, ProblemID: problem.ID, ProblemVersionID: versionID, Label: label, Score: item.Score, SortOrder: index})
			contestLinks = append(contestLinks, models.TeamContestProblem{ContestID: p.id, ProblemID: problem.ID, ProblemVersionID: versionID, Label: label, SortOrder: index})
		}
		if p.kind == "exam" {
			if e = tx.Where("exam_id=?", p.id).Delete(&models.ExamProblem{}).Error; e != nil {
				return e
			}
			if len(examLinks) > 0 {
				e = tx.Create(&examLinks).Error
			}
		} else {
			if e = tx.Where("contest_id=?", p.id).Delete(&models.TeamContestProblem{}).Error; e != nil {
				return e
			}
			if len(contestLinks) > 0 {
				e = tx.Create(&contestLinks).Error
			}
		}
		if e != nil {
			return e
		}
		return bumpEventRevision(tx, p)
	})
	if err != nil {
		eventError(c, err)
		return
	}
	c.JSON(200, gin.H{"updated": true})
}

func (s Server) downloadEventProblemPackage(c *gin.Context) {
	p, ok := s.eventPaperByParams(c, true)
	if !ok {
		return
	}
	id, ok := idParam(c, "problem_id")
	if !ok {
		return
	}
	problems, err := eventPaperProblems(s.DB, p)
	if err != nil {
		eventError(c, err)
		return
	}
	for _, link := range problems {
		if link.ProblemID == id {
			body, ok := s.problemPackageBody(c, link.Problem)
			if !ok {
				return
			}
			c.Header("Cache-Control", "private, no-store")
			c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="problem-%d-v%d.zip"`, id, link.ProblemVersionID))
			c.Data(200, "application/zip", body)
			return
		}
	}
	c.JSON(404, gin.H{"error": "题目不属于本场"})
}

func (s Server) listEventProblemChoices(c *gin.Context) {
	p, ok := s.eventPaperByParams(c, true)
	if !ok {
		return
	}
	u, _ := middleware.CurrentUser(c)
	q := s.DB.Model(&models.Problem{}).Where("problems.event_kind = '' AND problems.deleted_at IS NULL AND problems.archived_at IS NULL")
	if u.Role != models.RoleAdmin {
		allowed := s.DB.Where(publicProblemSQL()).Or("problems.owner_id = ?", u.ID)
		if p.kind == "contest" {
			allowed = allowed.Or("problems.team_id = ?", p.contest.TeamID)
		}
		q = q.Where(allowed)
	}
	if search := strings.TrimSpace(c.Query("q")); search != "" {
		search = "%" + strings.NewReplacer("\\", "\\\\", "%", "\\%", "_", "\\_").Replace(search) + "%"
		q = q.Where("problems.title ILIKE ? OR problems.display_code ILIKE ?", search, search)
	}
	items := []models.Problem{}
	if err := q.Order("problems.id desc").Limit(50).Find(&items).Error; err != nil {
		eventError(c, err)
		return
	}
	c.Header("Cache-Control", "private, no-store")
	c.JSON(200, items)
}
