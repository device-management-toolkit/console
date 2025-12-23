//go:build noui

package httpapi

import (
	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/pkg/logger"
	"github.com/gin-gonic/gin"
)

// setupUIRoutes is a no-op when building with the noui tag.
func setupUIRoutes(handler *gin.Engine, l logger.Interface, cfg *config.Config) {
	// No UI routes in noui build
}
