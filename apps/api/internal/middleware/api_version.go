package middleware

import (
	"log"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// ObserveAPIVersion records route templates only: never raw URLs, query values,
// invitation codes, identities or IPs. Counters reset on process restart and are
// an observation aid, not sufficient evidence by themselves to retire /api.
func ObserveAPIVersion() gin.HandlerFunc {
	type usage struct {
		started time.Time
		count   uint64
	}
	var mu sync.Mutex
	counts := make(map[string]usage)
	return func(c *gin.Context) {
		path := c.FullPath()
		version := "legacy"
		if strings.HasPrefix(path, "/api/v1/") {
			version = "v1"
		} else if !strings.HasPrefix(path, "/api/") {
			c.Next()
			return
		}
		c.Header("X-API-Version", version)
		key := version + " " + c.Request.Method + " " + path
		now := time.Now()
		mu.Lock()
		entry := counts[key]
		if entry.started.IsZero() || now.Sub(entry.started) >= time.Minute {
			if entry.count > 0 {
				log.Printf("api_usage version=%s method=%s route=%s count=%d window_seconds=%.0f", version, c.Request.Method, path, entry.count, now.Sub(entry.started).Seconds())
			}
			if entry.started.IsZero() {
				log.Printf("api_usage_first version=%s method=%s route=%s", version, c.Request.Method, path)
			}
			entry = usage{started: now}
		}
		entry.count++
		counts[key] = entry
		mu.Unlock()
		c.Next()
	}
}
