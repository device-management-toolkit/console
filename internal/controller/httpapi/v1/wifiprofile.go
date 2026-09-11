package v1

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/usecase/devices/wsman"
	"github.com/device-management-toolkit/console/pkg/consoleerrors"
)

var errValidationWirelessProfile = dto.NotValidError{Console: consoleerrors.CreateConsoleError("WirelessProfileAPI")}

func (r *deviceManagementRoutes) getWirelessProfiles(c *gin.Context) {
	guid := c.Param("guid")
	tenantID := tenantIDFromRequest(c)

	response, err := r.d.GetWirelessProfiles(c.Request.Context(), guid, tenantID)
	if err != nil {
		r.l.Error(err, "http - v1 - getWirelessProfiles")
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusOK, response)
}

func (r *deviceManagementRoutes) addWirelessProfile(c *gin.Context) {
	guid := c.Param("guid")

	var req dto.WirelessProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validationErr := errValidationWirelessProfile.Wrap("addWirelessProfile", "ShouldBindJSON", err)
		ErrorResponse(c, validationErr)

		return
	}

	tenantID := tenantIDFromRequest(c)
	err := r.d.AddWirelessProfile(c.Request.Context(), guid, tenantID, req.ToWirelessProfile())
	if err != nil {
		r.l.Error(err, "http - v1 - addWirelessProfile")

		if errors.Is(err, wsman.ErrNoWiFiPort) {
			c.JSON(http.StatusNotFound, gin.H{
				errorKey: "Add Wireless Profile failed for guid: " + guid + ". - " + err.Error(),
			})

			return
		}

		ErrorResponse(c, err)

		return
	}

	c.Status(http.StatusNoContent)
}

func (r *deviceManagementRoutes) deleteWirelessProfile(c *gin.Context) {
	guid := c.Param("guid")
	profileName := c.Param("profileName")
	tenantID := tenantIDFromRequest(c)

	err := r.d.DeleteWirelessProfile(c.Request.Context(), guid, profileName, tenantID)
	if err != nil {
		r.l.Error(err, "http - v1 - deleteWirelessProfile")
		ErrorResponse(c, err)

		return
	}

	c.Status(http.StatusNoContent)
}

func (r *deviceManagementRoutes) updateWirelessProfile(c *gin.Context) {
	guid := c.Param("guid")

	var req dto.WirelessProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validationErr := errValidationWirelessProfile.Wrap("updateWirelessProfile", "ShouldBindJSON", err)
		ErrorResponse(c, validationErr)

		return
	}

	tenantID := tenantIDFromRequest(c)
	err := r.d.UpdateWirelessProfile(c.Request.Context(), guid, tenantID, req.ToWirelessProfile())
	if err != nil {
		r.l.Error(err, "http - v1 - updateWirelessProfile")

		if errors.Is(err, wsman.ErrNoWiFiPort) {
			c.JSON(http.StatusNotFound, gin.H{
				errorKey: "Update Wireless Profile failed for guid: " + guid + ". - " + err.Error(),
			})

			return
		}

		ErrorResponse(c, err)

		return
	}

	c.Status(http.StatusNoContent)
}
