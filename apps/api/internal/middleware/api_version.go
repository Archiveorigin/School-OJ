package middleware

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

var usageWriteFailures atomic.Uint64

// Persist route templates only, never request URLs, query values or identities.
// Daily counters survive process restarts and are shared by all API replicas.
var usageScript = redis.NewScript(`
redis.call('HINCRBY', KEYS[1], ARGV[1], 1)
redis.call('EXPIRE', KEYS[1], 7776000)
redis.call('HINCRBY', KEYS[2], ARGV[2], 1)
return 1
`)

func ObserveAPIVersion(clients ...*redis.Client) gin.HandlerFunc {
	var client *redis.Client
	if len(clients) > 0 {
		client = clients[0]
	}
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
		if client != nil {
			ctx, cancel := context.WithTimeout(c.Request.Context(), time.Second)
			_, err := usageScript.Run(ctx, client, []string{"oj:api:usage:" + time.Now().UTC().Format("2006-01-02"), "oj:metrics:api"}, version+" "+c.Request.Method+" "+path, "requests:"+version).Result()
			cancel()
			if err != nil {
				usageWriteFailures.Add(1)
				log.Printf("API usage persistence failed: %v", err)
				c.AbortWithStatusJSON(503, gin.H{"error": "API observation storage unavailable"})
				return
			}
		}
		c.Next()
	}
}

// Missing minutes are gaps, never evidence of zero usage. Each day needs all
// 1440 bits before it may count toward the 30-day legacy retirement window.
func ObserveAPICoverage(ctx context.Context, client *redis.Client) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		now := time.Now().UTC()
		writeCtx, cancel := context.WithTimeout(ctx, time.Second)
		_, err := client.TxPipelined(writeCtx, func(p redis.Pipeliner) error {
			key := "oj:api:coverage:" + now.Format("2006-01-02")
			p.SetBit(writeCtx, key, int64(now.Hour()*60+now.Minute()), 1)
			p.Expire(writeCtx, key, 90*24*time.Hour)
			return nil
		})
		cancel()
		if err != nil && ctx.Err() == nil {
			usageWriteFailures.Add(1)
			log.Printf("API coverage persistence failed: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func APIMetrics(client *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		if client == nil {
			c.Status(http.StatusServiceUnavailable)
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		values, err := client.HGetAll(ctx, "oj:metrics:api").Result()
		if err != nil {
			c.Status(http.StatusServiceUnavailable)
			return
		}
		c.Header("Content-Type", "text/plain; version=0.0.4")
		for key, value := range values {
			if _, err := strconv.ParseUint(value, 10, 64); err != nil {
				continue
			}
			kind, label, _ := strings.Cut(key, ":")
			switch kind {
			case "requests":
				fmt.Fprintf(c.Writer, "oj_api_requests_total{version=%q} %s\n", label, value)
			case "rate_limit":
				fmt.Fprintf(c.Writer, "oj_rate_limit_rejections_total{limiter=%q} %s\n", label, value)
			}
		}
		fmt.Fprintf(c.Writer, "oj_api_observation_write_failures_total %d\n", usageWriteFailures.Load())
	}
}
