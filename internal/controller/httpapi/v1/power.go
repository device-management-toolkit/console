package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/device-management-toolkit/console/internal/controller/httpapi/middleware"
	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
)

func (r *deviceManagementRoutes) getPowerState(c *gin.Context) {
	guid := c.Param("guid")
	tenantID := middleware.TenantID(c)

	state, err := r.d.GetPowerState(c.Request.Context(), guid, tenantID)
	if err != nil {
		r.l.Error(err, "http - v1 - getPowerState")
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusOK, state)
}

func (r *deviceManagementRoutes) getPowerCapabilities(c *gin.Context) {
	guid := c.Param("guid")
	tenantID := middleware.TenantID(c)

	power, err := r.d.GetPowerCapabilities(c.Request.Context(), guid, tenantID)
	if err != nil {
		r.l.Error(err, "http - v1 - getPowerCapabilities")
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusOK, power)
}

func (r *deviceManagementRoutes) powerAction(c *gin.Context) {
	guid := c.Param("guid")

	var powerAction dto.PowerAction
	if err := c.ShouldBindJSON(&powerAction); err != nil {
		ErrorResponse(c, err)

		return
	}

	tenantID := middleware.TenantID(c)
	response, err := r.d.SendPowerAction(c.Request.Context(), guid, tenantID, powerAction.Action)
	if err != nil {
		r.l.Error(err, "http - v1 - powerAction")
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusOK, response)
}

func (r *deviceManagementRoutes) setBootOptions(c *gin.Context) {
	guid := c.Param("guid")

	var bootSetting dto.BootSetting
	if err := c.ShouldBindJSON(&bootSetting); err != nil {
		ErrorResponse(c, err)

		return
	}

	tenantID := middleware.TenantID(c)
	features, err := r.d.SetBootOptions(c.Request.Context(), guid, tenantID, bootSetting)
	if err != nil {
		r.l.Error(err, "http - v1 - setBootOptions")
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusOK, features)
}

func (r *deviceManagementRoutes) getBootSources(c *gin.Context) {
	guid := c.Param("guid")
	tenantID := middleware.TenantID(c)

	sources, err := r.d.GetBootSourceSetting(c.Request.Context(), guid, tenantID)
	if err != nil {
		r.l.Error(err, "http - v1 - getBootSources")
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusOK, sources)
}
