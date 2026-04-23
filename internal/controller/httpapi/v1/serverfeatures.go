// Separate from features.go, which handles per-device AMT features.
package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/device-management-toolkit/console/config"
	dto "github.com/device-management-toolkit/console/internal/entity/dto/v1"
)

type ServerFeaturesRoute struct {
	Config *config.Config
}

func NewServerFeaturesRoute(configData *config.Config) *ServerFeaturesRoute {
	return &ServerFeaturesRoute{Config: configData}
}

func (sfr ServerFeaturesRoute) Handler(c *gin.Context) {
	c.JSON(http.StatusOK, dto.ServerFeaturesResponse{
		CIRA: !sfr.Config.DisableCIRA,
	})
}
