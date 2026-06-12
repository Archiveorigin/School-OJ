package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"school-oj/apps/api/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func TestAuthRejectsUnexpectedSigningMethod(t *testing.T) {
	gin.SetMode(gin.TestMode)
	token, err := jwt.NewWithClaims(jwt.SigningMethodNone, Claims{
		UserID: 1,
		Role:   models.RoleStudent,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatal(err)
	}

	router := gin.New()
	router.Use(Auth(nil, "secret"))
	router.GET("/protected", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}
