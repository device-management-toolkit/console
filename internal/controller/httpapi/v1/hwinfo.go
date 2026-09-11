package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (r *deviceManagementRoutes) getHardwareInfo(c *gin.Context) {
	guid := c.Param("guid")
	tenantID := tenantIDFromRequest(c)

	hwInfo, err := r.d.GetHardwareInfo(c.Request.Context(), guid, tenantID)
	if err != nil {
		r.l.Error(err, "http - v1 - getHardwareInfo")
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusOK, hwInfo)
}

func (r *deviceManagementRoutes) getDiskInfo(c *gin.Context) {
	guid := c.Param("guid")
	tenantID := tenantIDFromRequest(c)

	diskInfo, err := r.d.GetDiskInfo(c.Request.Context(), guid, tenantID)
	if err != nil {
		r.l.Error(err, "http - v1 - getHardwareInfo")
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusOK, diskInfo)
}

func (r *deviceManagementRoutes) getGeneralSettings(c *gin.Context) {
	guid := c.Param("guid")
	tenantID := tenantIDFromRequest(c)

	generalSettings, err := r.d.GetGeneralSettings(c.Request.Context(), guid, tenantID)
	if err != nil {
		r.l.Error(err, "http - v1 - getGeneralSettings")
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusOK, generalSettings)
}
