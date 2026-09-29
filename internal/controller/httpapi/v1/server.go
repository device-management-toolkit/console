package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/device-management-toolkit/console/config"
	dto "github.com/device-management-toolkit/console/internal/entity/dto/v1"
)

type serverRoutes struct {
	config *config.Config
}

// NewServerRoutes registers server-level capability endpoints. These expose how
// Console was started so clients can adapt their UI (e.g. show/hide the CIRA tab).
func NewServerRoutes(handler *gin.RouterGroup, cfg *config.Config) {
	r := &serverRoutes{config: cfg}

	h := handler.Group("/server")
	{
		h.GET("features", r.getFeatures)
	}
}

// getFeatures returns the server-level feature flags.
func (r *serverRoutes) getFeatures(c *gin.Context) {
	c.JSON(http.StatusOK, dto.ServerFeatures{
		CIRAEnabled: !r.config.DisableCIRA,
	})
}
