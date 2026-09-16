package middleware

import (
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"testing"
)

func TestAPIVersionObservationKeepsBothPrefixes(t *testing.T) {
	router := gin.New()
	router.Use(ObserveAPIVersion())
	for _, prefix := range []string{"/api", "/api/v1"} {
		router.GET(prefix+"/teams/:id", func(c *gin.Context) { c.Status(200) })
	}
	for path, version := range map[string]string{"/api/teams/private?code=secret": "legacy", "/api/v1/teams/private": "v1", "/unknown": ""} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest("GET", path, nil))
		if got := recorder.Header().Get("X-API-Version"); got != version {
			t.Fatalf("%s version=%q", path, got)
		}
	}
}
