package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"school-oj/apps/api/internal/middleware"
	"school-oj/apps/api/internal/models"
	"school-oj/apps/api/internal/streams"

	"github.com/gin-gonic/gin"
)

func (s Server) submissionEvents(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	id, ok := idParam(c, "id")
	if !ok {
		return
	}

	var initial models.Submission
	if err := s.DB.First(&initial, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "submission not found"})
		return
	}
	if !s.canAccessSubmission(user, initial) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	pubsub := s.Redis.Subscribe(c.Request.Context(), streams.SubmissionChannel(id))
	defer pubsub.Close()
	if _, err := pubsub.Receive(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "submission event service unavailable"})
		return
	}
	notifications := pubsub.Channel()

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache, no-store")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	last := ""
	sendStatus := func() bool {
		var sub models.Submission
		if err := s.DB.First(&sub, id).Error; err != nil {
			writeSSE(c, "error", gin.H{"error": "submission not found"})
			return true
		}
		payload := gin.H{
			"id":               sub.ID,
			"status":           sub.Status,
			"score":            sub.Score,
			"manual_score":     sub.ManualScore,
			"manual_graded_at": sub.ManualGradedAt,
			"time_ms":          sub.TimeMS,
			"memory_kb":        sub.MemoryKB,
			"message":          sub.Message,
			"updated_at":       sub.UpdatedAt,
		}
		raw, _ := json.Marshal(payload)
		if string(raw) != last {
			last = string(raw)
			writeSSE(c, "status", payload)
		}
		return terminal(sub.Status)
	}
	if sendStatus() {
		return
	}

	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()
	fallback := time.NewTicker(30 * time.Second)
	defer fallback.Stop()
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case _, open := <-notifications:
			if !open {
				writeSSE(c, "error", gin.H{"error": "submission event service unavailable"})
				return
			}
			if sendStatus() {
				return
			}
		case <-fallback.C:
			// Pub/Sub is intentionally ephemeral. A low-frequency reconciliation
			// covers a lost notification without returning to one-second DB polling.
			if sendStatus() {
				return
			}
		case now := <-heartbeat.C:
			writeSSE(c, "ping", gin.H{"time": now.UTC().Format(time.RFC3339)})
		}
	}
}
