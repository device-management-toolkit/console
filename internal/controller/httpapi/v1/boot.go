package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
)

func (r *deviceManagementRoutes) getBootCapabilities(c *gin.Context) {
	guid := c.Param("guid")

	capabilities, err := r.d.GetBootCapabilities(c.Request.Context(), guid)
	if err != nil {
		r.l.Error(err, "http - v1 - getBootCapabilities")
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusOK, capabilities)
}

func (r *deviceManagementRoutes) setRPEEnabled(c *gin.Context) {
	guid := c.Param("guid")

	var req dto.RPERequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, err)

		return
	}

	if err := r.d.SetRPEEnabled(c.Request.Context(), guid, req.Enabled); err != nil {
		r.l.Error(err, "http - v1 - setRPEEnabled")
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusOK, nil)
}

func (r *deviceManagementRoutes) sendRemoteErase(c *gin.Context) {
	guid := c.Param("guid")

	var req dto.RemoteEraseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, err)

		return
	}

	if err := r.d.SendRemoteErase(c.Request.Context(), guid, req.EraseMask); err != nil {
		r.l.Error(err, "http - v1 - sendRemoteErase")
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusOK, nil)
}
