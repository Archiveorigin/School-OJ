package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRateLimitKeyDoesNotExposeIdentity(t *testing.T) {
	key := rateLimitKey("AUTH:LOGIN", "203.0.113.4")
	if !strings.HasPrefix(key, "oj:rate:auth:login:") {
		t.Fatalf("unexpected key prefix: %s", key)
	}
	if strings.Contains(key, "203.0.113.4") {
		t.Fatalf("rate limit key exposes identity: %s", key)
	}
	if key != rateLimitKey("auth:login", "203.0.113.4") {
		t.Fatal("rate limit key must be deterministic")
	}
}

func TestSecurityHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(SecurityHeaders())
	router.GET("/test", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/test", nil))
	for _, header := range []string{
		"X-Content-Type-Options",
		"X-Frame-Options",
		"Referrer-Policy",
		"Permissions-Policy",
		"Cross-Origin-Resource-Policy",
	} {
		if response.Header().Get(header) == "" {
			t.Fatalf("missing security header %s", header)
		}
	}
}
