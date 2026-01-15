package v1

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/usecase/devices"
)

// getKVMDisplays returns current IPS_ScreenSettingData for the device
func (r *deviceManagementRoutes) getKVMDisplays(c *gin.Context) {
	start := time.Now()

	defer func() {
		devices.RecordAPIRequest(time.Since(start), "kvm_displays")
	}()

	guid := c.Param("guid")

	settings, err := r.d.GetKVMScreenSettings(c.Request.Context(), guid)
	if err != nil {
		r.l.Error(err, "http - v1 - getKVMDisplays")
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusOK, settings)
}

// setKVMDisplays updates IPS_ScreenSettingData for the device
func (r *deviceManagementRoutes) setKVMDisplays(c *gin.Context) {
	guid := c.Param("guid")

	var req dto.KVMScreenSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, err)

		return
	}

	settings, err := r.d.SetKVMScreenSettings(c.Request.Context(), guid, req)
	if err != nil {
		r.l.Error(err, "http - v1 - setKVMDisplays")
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusOK, settings)
}
