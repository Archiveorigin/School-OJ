package services

import (
	"strconv"

	"school-oj/apps/api/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func Audit(c *gin.Context, db *gorm.DB, action, resourceType string, resourceID any, meta datatypes.JSONMap) {
	var actorID *uint
	if value, ok := c.Get("user"); ok {
		if user, ok := value.(models.User); ok {
			actorID = &user.ID
		}
	}
	resource := ""
	switch v := resourceID.(type) {
	case nil:
	case string:
		resource = v
	case uint:
		resource = strconv.FormatUint(uint64(v), 10)
	case int:
		resource = strconv.Itoa(v)
	default:
		resource = strconv.Itoa(0)
	}
	_ = db.Create(&models.AuditLog{
		ActorUserID: actorID,
		Action:      action,
		ResourceType: resourceType,
		ResourceID:   resource,
		IP:           c.ClientIP(),
		UserAgent:    c.Request.UserAgent(),
		Meta:         meta,
	}).Error
}
