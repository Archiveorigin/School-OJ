package handlers

import (
	"github.com/gin-gonic/gin"
	"school-oj/apps/api/internal/middleware"
	"school-oj/apps/api/internal/repositories"
	"school-oj/apps/api/internal/services"
)

func (s Server) teamAccess() services.TeamAccess {
	return services.TeamAccess{Store: repositories.TeamAccess{DB: s.DB}}
}

func (s Server) previewTeamJoin(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	value, ok := s.teamByParam(c)
	if !ok {
		return
	}
	view, err := (repositories.TeamAccess{DB: s.DB}).Preview(c.Request.Context(), value, user.ID)
	if err != nil {
		examMutationError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(200, view)
}

func (s Server) previewTeamInvitation(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	var req struct {
		JoinCode string `json:"join_code" binding:"required,max=24"`
	}
	if !bind(c, &req) {
		return
	}
	view, err := s.teamAccess().PreviewInvitation(c.Request.Context(), req.JoinCode, user.ID)
	if err != nil {
		examMutationError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(200, view)
}
