package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
)

func (r *deviceManagementRoutes) getNetworkSettings(c *gin.Context) {
	guid := c.Param("guid")

	network, err := r.d.GetNetworkSettings(c.Request.Context(), guid)
	if err != nil {
		r.l.Error(err, "http - v1 - getNetworkSettings")
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusOK, network)
}

// getWiredNetworkSettings returns the wired network settings for a device.
func (r *deviceManagementRoutes) getWiredNetworkSettings(c *gin.Context) {
	guid := c.Param("guid")

	wired, err := r.d.GetWiredNetworkSettings(c.Request.Context(), guid)
	if err != nil {
		r.l.Error(err, "http - v1 - getWiredNetworkSettings")
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusOK, wired)
}

// patchWiredNetworkSettings updates the wired IPv4 configuration for a device.
func (r *deviceManagementRoutes) patchWiredNetworkSettings(c *gin.Context) {
	guid := c.Param("guid")

	var req dto.WiredNetworkConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, err)

		return
	}

	if err := r.d.PatchWiredNetworkSettings(c.Request.Context(), guid, req); err != nil {
		r.l.Error(err, "http - v1 - patchWiredNetworkSettings")
		ErrorResponse(c, err)

		return
	}

	c.Status(http.StatusNoContent)
}
