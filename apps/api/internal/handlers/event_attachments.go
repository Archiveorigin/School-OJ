package handlers

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"gorm.io/gorm"
	"school-oj/apps/api/internal/middleware"
	"school-oj/apps/api/internal/models"
	"school-oj/apps/api/internal/services"
)

const eventAttachmentLimit int64 = 32 << 20

func (s Server) listEventAttachments(c *gin.Context) {
	p, ok := s.eventPaperByParams(c, false)
	if !ok {
		return
	}
	items := []models.EventAttachment{}
	query := s.DB.Where("event_kind=? AND event_id=?", p.kind, p.id)
	if p.review().CompletedAt == nil && (p.ends() == nil || time.Now().Before(*p.ends())) {
		query = query.Where("announcement_id IS NULL")
	}
	if err := query.Order("id").Find(&items).Error; err != nil {
		eventError(c, err)
		return
	}
	c.Header("Cache-Control", "private, no-store")
	c.JSON(200, items)
}
func (s Server) uploadEventAttachment(c *gin.Context) {
	p, ok := s.eventPaperByParams(c, true)
	if !ok {
		return
	}
	if s.MinIO == nil {
		c.JSON(503, gin.H{"error": "附件存储暂不可用"})
		return
	}
	release, slotErr := acquireTicketPackage(c.Request.Context())
	if slotErr != nil {
		eventError(c, slotErr)
		return
	}
	defer release()
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, eventAttachmentLimit+(1<<20))
	defer func() {
		if c.Request.MultipartForm != nil {
			_ = c.Request.MultipartForm.RemoveAll()
		}
	}()
	if err := c.Request.ParseMultipartForm(4 << 20); err != nil {
		c.JSON(400, gin.H{"error": "附件单个不能超过 32 MiB"})
		return
	}
	announcementID, _ := strconv.ParseUint(c.PostForm("announcement_id"), 10, 64)
	revision, err := strconv.Atoi(c.PostForm("revision"))
	if err != nil && announcementID == 0 {
		c.JSON(400, gin.H{"error": "缺少场次修订号"})
		return
	}
	if err = s.requireAttachmentUpload(s.DB, p, revision, uint(announcementID)); err != nil {
		eventError(c, err)
		return
	}
	f, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(400, gin.H{"error": "请选择附件"})
		return
	}
	defer f.Close()
	name := path.Base(strings.ReplaceAll(header.Filename, "\\", "/"))
	if name == "." || strings.TrimSpace(name) == "" || len([]rune(name)) > 200 || strings.ContainsAny(name, "\r\n\x00") {
		c.JSON(400, gin.H{"error": "附件文件名无效"})
		return
	}
	if announcementID > 0 {
		allowed := map[string]bool{".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".webp": true, ".pdf": true, ".doc": true, ".docx": true, ".txt": true, ".xlsx": true, ".csv": true}
		if !allowed[strings.ToLower(path.Ext(name))] {
			c.JSON(400, gin.H{"error": "仅支持常用图片、PDF、Word、文本及表格文件"})
			return
		}
	}
	body, err := services.ReadLimited(f, eventAttachmentLimit, "附件")
	if err != nil || len(body) == 0 {
		c.JSON(400, gin.H{"error": "附件不能为空，且单个不能超过 32 MiB"})
		return
	}
	var nonce [16]byte
	if _, err = rand.Read(nonce[:]); err != nil {
		eventError(c, err)
		return
	}
	object := fmt.Sprintf("event-attachments/%s/%d/%s", p.kind, p.id, hex.EncodeToString(nonce[:]))
	if _, err = s.MinIO.PutObject(c.Request.Context(), s.Cfg.MinIOBucket, object, bytes.NewReader(body), int64(len(body)), minio.PutObjectOptions{ContentType: "application/octet-stream"}); err != nil {
		s.cleanupTicketUploads([]models.ProblemTicketAttachment{{Object: object}})
		c.JSON(503, gin.H{"error": "附件上传失败"})
		return
	}
	u, _ := middleware.CurrentUser(c)
	sum := sha256.Sum256(body)
	item := models.EventAttachment{EventKind: p.kind, EventID: p.id, Name: name, Object: object, Size: int64(len(body)), Checksum: hex.EncodeToString(sum[:]), CreatedBy: u.ID}
	if announcementID > 0 {
		aid := uint(announcementID)
		item.AnnouncementID = &aid
	}
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
		if e = scoped.requireAttachmentUpload(tx, p, revision, uint(announcementID)); e != nil {
			return e
		}
		var totals struct {
			Count int64
			Size  int64
		}
		if e = tx.Model(&models.EventAttachment{}).Select("count(*) AS count, COALESCE(sum(size),0) AS size").Where("event_kind=? AND event_id=?", p.kind, p.id).Scan(&totals).Error; e != nil {
			return e
		}
		if totals.Count >= 20 || totals.Size+item.Size > 96<<20 {
			return eventFail(400, "每场最多 20 个附件，总计 96 MiB")
		}
		var duplicate int64
		if e = tx.Model(&models.EventAttachment{}).Where("event_kind=? AND event_id=? AND lower(name)=lower(?)", p.kind, p.id, name).Count(&duplicate).Error; e != nil {
			return e
		}
		if duplicate > 0 {
			return eventFail(409, "已有同名附件，请先移除旧附件")
		}
		if e = tx.Create(&item).Error; e != nil {
			return e
		}
		if announcementID > 0 {
			return nil
		}
		return bumpEventRevision(tx, p)
	})
	if err != nil {
		s.cleanupTicketUploads([]models.ProblemTicketAttachment{{Object: object}})
		eventError(c, err)
		return
	}
	services.Audit(c, s.DB, "event.attachment.upload", p.kind, p.id, nil)
	c.JSON(201, item)
}
func (s Server) deleteEventAttachment(c *gin.Context) {
	p, ok := s.eventPaperByParams(c, true)
	if !ok {
		return
	}
	id, ok := idParam(c, "attachment_id")
	if !ok {
		return
	}
	var req struct {
		Revision int `json:"revision"`
	}
	if !bind(c, &req) {
		return
	}
	u, _ := middleware.CurrentUser(c)
	var item models.EventAttachment
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
		if e = tx.Where("event_kind=? AND event_id=?", p.kind, p.id).First(&item, id).Error; e != nil {
			return e
		}
		if e = tx.Delete(&item).Error; e != nil {
			return e
		}
		return bumpEventRevision(tx, p)
	})
	if err != nil {
		eventError(c, err)
		return
	}
	s.cleanupTicketUploads([]models.ProblemTicketAttachment{{Object: item.Object}})
	services.Audit(c, s.DB, "event.attachment.delete", p.kind, p.id, nil)
	c.JSON(200, gin.H{"deleted": true})
}
func (s Server) downloadEventAttachment(c *gin.Context) {
	p, ok := s.eventPaperByParams(c, false)
	if !ok {
		return
	}
	id, ok := idParam(c, "attachment_id")
	if !ok {
		return
	}
	var item models.EventAttachment
	if err := s.DB.Where("event_kind=? AND event_id=?", p.kind, p.id).First(&item, id).Error; err != nil {
		eventError(c, err)
		return
	}
	if s.MinIO == nil {
		c.JSON(503, gin.H{"error": "附件存储暂不可用"})
		return
	}
	src, err := s.MinIO.GetObject(c.Request.Context(), s.Cfg.MinIOBucket, item.Object, minio.GetObjectOptions{})
	if err != nil {
		c.JSON(503, gin.H{"error": "附件读取失败"})
		return
	}
	defer src.Close()
	stat, err := src.Stat()
	if err != nil {
		c.JSON(404, gin.H{"error": "附件不存在"})
		return
	}
	// Revalidate after storage lookup so deletion/withdrawal cannot authorize a
	// newly starting download with a stale parent snapshot.
	fresh, err := loadEventPaper(s.DB, p.kind, p.id, false)
	u, _ := middleware.CurrentUser(c)
	if err != nil || !s.canReadEventPaper(u, fresh, time.Now()) {
		c.JSON(403, gin.H{"error": "场次已删除或下载权限已变化"})
		return
	}
	c.Header("Cache-Control", "private, no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	var exists int64
	if err := s.DB.Model(&models.EventAttachment{}).Where("id=? AND event_kind=? AND event_id=?", id, p.kind, p.id).Count(&exists).Error; err != nil || exists == 0 {
		c.JSON(404, gin.H{"error": "附件已移除"})
		return
	}
	c.Header("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": item.Name}))
	c.DataFromReader(200, stat.Size, "application/octet-stream", src, nil)
}

func (s Server) getEventProblemAsset(c *gin.Context) {
	p, ok := s.eventPaperByParams(c, false)
	if !ok {
		return
	}
	id, ok := idParam(c, "problem_id")
	if !ok {
		return
	}
	asset, err := services.NormalizeAssetPath(strings.TrimPrefix(c.Param("asset_path"), "/"))
	if err != nil {
		c.JSON(400, gin.H{"error": "图片路径无效"})
		return
	}
	links, err := eventPaperProblems(s.DB, p)
	if err != nil {
		eventError(c, err)
		return
	}
	for _, link := range links {
		if link.ProblemID == id {
			statement, _ := url.PathUnescape(link.Problem.Statement)
			if statement == "" {
				statement = link.Problem.Statement
			}
			object, contentType := problemAssetObject(link.Problem.Manifest, asset)
			if object == "" || !strings.Contains(statement, asset) || !strings.HasPrefix(contentType, "image/") {
				break
			}
			if s.MinIO == nil {
				c.JSON(503, gin.H{"error": "图片暂不可用"})
				return
			}
			src, e := s.MinIO.GetObject(c.Request.Context(), s.Cfg.MinIOBucket, object, minio.GetObjectOptions{})
			if e != nil {
				eventError(c, e)
				return
			}
			defer src.Close()
			stat, e := src.Stat()
			if e != nil {
				c.JSON(404, gin.H{"error": "图片不存在"})
				return
			}
			c.Header("Cache-Control", "private, no-store")
			c.Header("X-Content-Type-Options", "nosniff")
			c.DataFromReader(200, stat.Size, contentType, src, nil)
			return
		}
	}
	c.JSON(404, gin.H{"error": "图片不存在"})
}

func (s Server) requireAttachmentUpload(tx *gorm.DB, p eventPaper, revision int, announcementID uint) error {
	if announcementID == 0 {
		return requireEventDraft(p, revision)
	}
	if !p.review().Approved() || p.review().CompletedAt != nil {
		return eventFail(409, "仅未完结的已审核场次支持公告附件")
	}
	var count int64
	if err := tx.Table("event_announcements").Where("id=? AND event_kind=? AND event_id=?", announcementID, p.kind, p.id).Count(&count).Error; err != nil {
		return err
	}
	if count != 1 {
		return eventFail(400, "公告不属于本场")
	}
	return nil
}
