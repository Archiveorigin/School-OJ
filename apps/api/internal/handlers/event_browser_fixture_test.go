package handlers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"school-oj/apps/api/internal/models"
)

// Opt-in fixture on an isolated database/schema and a loopback-only listener.
// Clock controls are compiled only into the test executable.
func TestEventBrowserFixture(t *testing.T) {
	addr := os.Getenv("OJ_EVENT_BROWSER_FIXTURE_ADDR")
	if addr == "" {
		t.Skip("opt-in browser fixture")
	}
	if addr != "127.0.0.1:18089" {
		t.Fatal("fixture must bind isolated loopback port")
	}
	f := newExamFixture(t)
	setupTicketObjects(t, f)
	_, contestID := eventTestPaper(t, f, "contest")
	f.s.Cfg.CORSOrigins = []string{"http://127.0.0.1:4179"}
	f.router = f.s.Router()
	server := &http.Server{Addr: addr, Handler: f.router, ReadHeaderTimeout: 10 * time.Second}
	f.router.GET("/__qa/info", func(c *gin.Context) {
		c.JSON(200, gin.H{"course_id": f.exam.CourseID, "contest_id": contestID, "team_slug": "review-qa"})
	})
	f.router.POST("/__qa/clock/:kind/:id/:phase", func(c *gin.Context) {
		id, ok := idParam(c, "id")
		if !ok {
			return
		}
		p, err := loadEventPaper(f.s.DB, c.Param("kind"), id, false)
		if err != nil {
			eventError(c, err)
			return
		}
		updates := map[string]any{}
		switch c.Param("phase") {
		case "start":
			updates["starts_at"] = time.Now().Add(-time.Minute)
			if p.kind == "exam" {
				updates["ends_at"] = time.Now().Add(time.Hour)
			}
		case "end":
			updates["starts_at"] = time.Now().Add(-2 * time.Hour)
			if p.kind == "exam" {
				updates["ends_at"] = time.Now().Add(-time.Minute)
			}
		default:
			c.Status(400)
			return
		}
		if err = f.s.DB.Model(p.model()).Updates(updates).Error; err != nil {
			eventError(c, err)
			return
		}
		c.JSON(200, gin.H{"updated": true})
	})
	f.router.POST("/__qa/stop", func(c *gin.Context) {
		c.Status(200)
		go func() { time.Sleep(50 * time.Millisecond); _ = server.Shutdown(context.Background()) }()
	})
	// Expired unpublished papers must not appear in the browser archive.
	start, end := time.Now().Add(-2*time.Hour), time.Now().Add(-time.Hour)
	f.s.DB.Create(&models.Exam{CourseID: f.exam.CourseID, Title: "不应归档的过期草稿", StartsAt: &start, EndsAt: &end, EventReview: models.EventReview{ReviewStatus: "draft", ReviewRevision: 1}})
	f.s.DB.Model(&f.exam).Update("deleted_at", time.Now())
	f.s.DB.Model(&f.other).Update("deleted_at", time.Now())
	t.Log(fmt.Sprintf("EVENT_FIXTURE_READY %s", addr))
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		t.Fatal(err)
	}
}
