package middleware

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestUsageSurvivesRouterRestartWithoutPersistingPrivateURLs(t *testing.T) {
	addr := os.Getenv("OJ_TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("requires isolated Redis")
	}
	if addr != "127.0.0.1:56379" {
		t.Fatal("unsafe Redis")
	}
	client := redis.NewClient(&redis.Options{Addr: addr, DB: 12})
	defer client.Close()
	ctx := context.Background()
	key := "oj:api:usage:" + time.Now().UTC().Format("2006-01-02")
	field := "legacy GET /api/observation-test/:id"
	client.HDel(ctx, key, field)
	defer client.HDel(ctx, key, field)
	for i := 0; i < 2; i++ {
		router := gin.New()
		router.Use(ObserveAPIVersion(client))
		router.GET("/api/observation-test/:id", func(c *gin.Context) { c.Status(200) })
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest("GET", "/api/observation-test/private-id?code=secret", nil))
		if recorder.Code != 200 {
			t.Fatal(recorder.Code)
		}
	}
	if client.HGet(ctx, key, field).Val() != "2" {
		t.Fatal("usage lost across routers")
	}
	for stored := range client.HGetAll(ctx, key).Val() {
		if strings.Contains(stored, "private-id") || strings.Contains(stored, "secret") {
			t.Fatal("private URL persisted")
		}
	}
}

func TestObservationFailureDoesNotExecuteUncountedBusinessRequest(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", MaxRetries: -1, DialTimeout: 10 * time.Millisecond})
	defer client.Close()
	router := gin.New()
	router.Use(ObserveAPIVersion(client))
	router.GET("/api/test", func(c *gin.Context) { t.Error("executed without durable observation") })
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest("GET", "/api/test", nil))
	if recorder.Code != 503 {
		t.Fatal(recorder.Code)
	}
}
