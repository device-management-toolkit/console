package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
)

func (r *deviceManagementRoutes) getRemoteEraseCapabilities(c *gin.Context) {
	guid := c.Param("guid")

	capabilities, err := r.d.GetRemoteEraseCapabilities(c.Request.Context(), guid)
	if err != nil {
		r.l.Error(err, "http - v1 - getRemoteEraseCapabilities")
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusOK, capabilities)
}

func (r *deviceManagementRoutes) setRemoteEraseOptions(c *gin.Context) {
	guid := c.Param("guid")

	var req dto.RemoteEraseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, err)

		return
	}

	if err := r.d.SetRemoteEraseOptions(c.Request.Context(), guid, req); err != nil {
		r.l.Error(err, "http - v1 - setRemoteEraseOptions")
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusOK, nil)
}
