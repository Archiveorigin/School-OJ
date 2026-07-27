package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

const tokenBucketScript = `
local now = redis.call("TIME")
local now_ms = tonumber(now[1]) * 1000 + math.floor(tonumber(now[2]) / 1000)
local capacity = tonumber(ARGV[1])
local window_ms = tonumber(ARGV[2])
local values = redis.call("HMGET", KEYS[1], "tokens", "updated_at")
local tokens = tonumber(values[1])
local updated_at = tonumber(values[2])
if tokens == nil or updated_at == nil then
  tokens = capacity
  updated_at = now_ms
end
local elapsed = math.max(0, now_ms - updated_at)
local refill_per_ms = capacity / window_ms
tokens = math.min(capacity, tokens + elapsed * refill_per_ms)
local allowed = 0
local retry_after_ms = 0
if tokens >= 1 then
  allowed = 1
  tokens = tokens - 1
else
  retry_after_ms = math.ceil((1 - tokens) / refill_per_ms)
end
redis.call("HSET", KEYS[1], "tokens", tokens, "updated_at", now_ms)
redis.call("PEXPIRE", KEYS[1], math.ceil(window_ms * 2))
return {allowed, math.floor(tokens), retry_after_ms}
`

var tokenBucket = redis.NewScript(tokenBucketScript)

// SecurityHeaders applies conservative API headers that are safe for JSON, SSE,
// and authenticated object responses.
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "no-referrer")
		c.Header("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		c.Header("Cross-Origin-Resource-Policy", "same-site")
		c.Next()
	}
}

// RateLimitIP enforces a distributed token bucket keyed by the trusted client IP.
func RateLimitIP(client redis.Scripter, name string, capacity int, window time.Duration) gin.HandlerFunc {
	return redisRateLimit(client, name, capacity, window, func(c *gin.Context) string {
		return c.ClientIP()
	})
}

// RateLimitUser enforces a distributed token bucket keyed by the authenticated
// user. It falls back to the trusted client IP if authentication context is absent.
func RateLimitUser(client redis.Scripter, name string, capacity int, window time.Duration) gin.HandlerFunc {
	return redisRateLimit(client, name, capacity, window, func(c *gin.Context) string {
		if user, ok := CurrentUser(c); ok {
			return fmt.Sprintf("user:%d", user.ID)
		}
		return "ip:" + c.ClientIP()
	})
}

func redisRateLimit(client redis.Scripter, name string, capacity int, window time.Duration, identity func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if capacity <= 0 || window <= 0 {
			c.Next()
			return
		}
		if client == nil {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "rate limiter unavailable"})
			return
		}
		key := rateLimitKey(name, identity(c))
		allowed, remaining, retryAfter, err := takeToken(c.Request.Context(), client, key, capacity, window)
		if err != nil {
			log.Printf("rate limiter %s failed: %v", name, err)
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "rate limiter unavailable"})
			return
		}
		c.Header("X-RateLimit-Limit", strconv.Itoa(capacity))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		if !allowed {
			seconds := int(math.Ceil(retryAfter.Seconds()))
			if seconds < 1 {
				seconds = 1
			}
			c.Header("Retry-After", strconv.Itoa(seconds))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":               "too many requests",
				"retry_after_seconds": seconds,
			})
			return
		}
		c.Next()
	}
}

func takeToken(ctx context.Context, client redis.Scripter, key string, capacity int, window time.Duration) (bool, int, time.Duration, error) {
	values, err := tokenBucket.Run(ctx, client, []string{key}, capacity, window.Milliseconds()).Slice()
	if err != nil {
		return false, 0, 0, err
	}
	if len(values) != 3 {
		return false, 0, 0, fmt.Errorf("unexpected token bucket result length %d", len(values))
	}
	allowed, err := redisResultInt64(values[0])
	if err != nil {
		return false, 0, 0, err
	}
	remaining, err := redisResultInt64(values[1])
	if err != nil {
		return false, 0, 0, err
	}
	retryAfterMS, err := redisResultInt64(values[2])
	if err != nil {
		return false, 0, 0, err
	}
	return allowed == 1, int(remaining), time.Duration(retryAfterMS) * time.Millisecond, nil
}

func redisResultInt64(value any) (int64, error) {
	switch typed := value.(type) {
	case int64:
		return typed, nil
	case string:
		return strconv.ParseInt(typed, 10, 64)
	case []byte:
		return strconv.ParseInt(string(typed), 10, 64)
	default:
		return 0, fmt.Errorf("unexpected redis integer type %T", value)
	}
}

func rateLimitKey(name, identity string) string {
	name = strings.Trim(strings.ToLower(strings.TrimSpace(name)), ":")
	sum := sha256.Sum256([]byte(strings.TrimSpace(identity)))
	return "oj:rate:" + name + ":" + hex.EncodeToString(sum[:16])
}
