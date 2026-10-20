package handlers

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"school-oj/apps/api/internal/adapters/imageimport"
)

func (s Server) importEventProblemImage(c *gin.Context) {
	p, ok := s.eventPaperByParams(c, true)
	if !ok {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 8192)
	var req struct {
		URL      string `json:"url"`
		Revision int    `json:"revision"`
	}
	if !bind(c, &req) {
		return
	}
	if err := requireEventDraft(p, req.Revision); err != nil {
		eventError(c, err)
		return
	}
	asset, err := imageimport.FetchProblemImage(c.Request.Context(), req.URL)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	// Preview only: the asset is persisted with the question in its existing
	// transactional save, which rechecks permission, editability and revision.
	c.Header("Cache-Control", "private, no-store")
	c.JSON(200, asset)
}
