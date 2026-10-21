package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"school-oj/apps/api/internal/authz"
	"school-oj/apps/api/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

type Claims struct {
	TokenVersion int64       `json:"token_version,omitempty"`
	Roles        []string    `json:"roles,omitempty"`
	PermVersion  int64       `json:"perm_version,omitempty"`
	UserID       uint        `json:"user_id"`
	Role         models.Role `json:"role"`
	jwt.RegisteredClaims
}

func SignToken(secret string, user models.User) (string, error) {
	claims := Claims{
		TokenVersion: max(user.TokenVersion, 1),
		Roles:        authz.FromUser(user).Roles,
		PermVersion:  user.PermVersion,
		UserID:       user.ID,
		Role:         user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.Email,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(12 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

func Auth(db *gorm.DB, secret string) gin.HandlerFunc {
	return authenticate(db, secret, true)
}

func OptionalAuth(db *gorm.DB, secret string) gin.HandlerFunc {
	return authenticate(db, secret, false)
}

func authenticate(db *gorm.DB, secret string, required bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if len(header) > 8192 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		tokenString := ""
		if strings.HasPrefix(header, "Bearer ") {
			tokenString = strings.TrimPrefix(header, "Bearer ")
		}
		if tokenString == "" {
			if header != "" {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header"})
				return
			}
			if required {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
				return
			}
			c.Next()
			return
		}
		token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, jwt.ErrTokenSignatureInvalid
			}
			return []byte(secret), nil
		}, jwt.WithExpirationRequired(), jwt.WithValidMethods([]string{"HS256"}))
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		claims, ok := token.Claims.(*Claims)
		if !ok || claims.UserID == 0 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
			return
		}
		var user models.User
		identityContext, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		err = db.WithContext(identityContext).Where("account_deleted = false").First(&user, claims.UserID).Error
		cancel()
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
			} else {
				c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "身份验证暂不可用", "code": "AUTH_STATE_UNAVAILABLE"})
			}
			return
		}
		// Pre-migration tokens are accepted only until the first credential
		// change. Permission changes refresh capabilities without ending login.
		if max(claims.TokenVersion, 1) != max(user.TokenVersion, 1) {
			c.Set("authorization.reason", "SESSION_REVOKED")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "登录已失效，请重新登录", "code": "SESSION_REVOKED"})
			return
		}
		if !authz.ValidBaseRole(user.Role) {
			c.Set("authorization.reason", "IDENTITY_UNDEFINED")
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "账号角色未定义", "code": "IDENTITY_UNDEFINED"})
			return
		}
		// Live database state is authoritative, including for legacy tokens.
		// No decision cache: permission revocations apply on the next request.
		c.Set("user", user)
		c.Set("authenticated_context", authz.FromUser(user))
		c.Next()
	}
}

func CurrentUser(c *gin.Context) (models.User, bool) {
	value, ok := c.Get("user")
	if !ok {
		return models.User{}, false
	}
	user, ok := value.(models.User)
	return user, ok
}
