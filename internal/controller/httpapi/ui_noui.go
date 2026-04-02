//go:build noui

package httpapi

import (
	"net/http"

	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/pkg/logger"
	"github.com/gin-gonic/gin"
)

// setupUIRoutes handles UI routes when building with the noui tag.
// If UI.ExternalURL is configured, requests to UI paths will be redirected there.
// Otherwise, they will return 404.
func setupUIRoutes(handler *gin.Engine, l logger.Interface, cfg *config.Config) {
	if cfg.UI.ExternalURL != "" {
		l.Info("UI disabled, redirecting UI requests to: " + cfg.UI.ExternalURL)
		// Redirect all UI-related paths to external UI
		handler.NoRoute(func(c *gin.Context) {
			path := c.Request.URL.Path
			// Only redirect likely UI paths, not API endpoints
			if !isAPIPath(path) {
				c.Redirect(http.StatusMovedPermanently, cfg.UI.ExternalURL+path)
				return
			}
			// For API paths that don't exist, return 404
			c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
		})
	} else {
		l.Info("UI disabled, no external UI configured")
	}
}

// isAPIPath checks if the path is an API endpoint that should not be redirected.
func isAPIPath(path string) bool {
	apiPrefixes := []string{"/api/", "/healthz", "/metrics", "/version"}
	for _, prefix := range apiPrefixes {
		if len(path) >= len(prefix) && path[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}
