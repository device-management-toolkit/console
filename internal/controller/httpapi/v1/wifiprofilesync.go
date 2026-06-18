package v1

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/usecase/devices"
	"github.com/device-management-toolkit/console/internal/usecase/devices/wsman"
	"github.com/device-management-toolkit/console/pkg/consoleerrors"
)

var errValidationWirelessProfileSync = dto.NotValidError{Console: consoleerrors.CreateConsoleError("WirelessProfileSyncAPI")}

func (r *deviceManagementRoutes) getWirelessProfileSync(c *gin.Context) {
	guid := c.Param("guid")

	response, err := r.d.GetWirelessProfileSync(c.Request.Context(), guid)
	if err != nil {
		r.l.Error(err, "http - v1 - getWirelessProfileSync")

		if errors.Is(err, wsman.ErrNoWiFiPort) {
			c.JSON(http.StatusNotFound, gin.H{
				errorKey: "Get Wireless Profile Sync failed for guid: " + guid + ". - " + err.Error(),
			})

			return
		}

		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusOK, response)
}

func (r *deviceManagementRoutes) setWirelessProfileSync(c *gin.Context) {
	guid := c.Param("guid")

	var req dto.WirelessProfileSyncRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validationErr := errValidationWirelessProfileSync.Wrap("setWirelessProfileSync", "ShouldBindJSON", err)
		ErrorResponse(c, validationErr)

		return
	}

	response, err := r.d.SetWirelessProfileSync(c.Request.Context(), guid, req)
	if err != nil {
		r.l.Error(err, "http - v1 - setWirelessProfileSync")

		if errors.Is(err, wsman.ErrNoWiFiPort) {
			c.JSON(http.StatusNotFound, gin.H{
				errorKey: "Set Wireless Profile Sync failed for guid: " + guid + ". - " + err.Error(),
			})

			return
		}

		if errors.Is(err, devices.ErrUEFIProfileSyncNotSupported) {
			c.JSON(http.StatusConflict, gin.H{
				errorKey: "Set Wireless Profile Sync failed for guid: " + guid + ". - " + err.Error(),
			})

			return
		}

		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusOK, response)
}
