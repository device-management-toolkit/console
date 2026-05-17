package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/pkg/consoleerrors"
)

var errValidationWirelessProfile = dto.NotValidError{Console: consoleerrors.CreateConsoleError("WirelessProfileAPI")}

func (r *deviceManagementRoutes) getWirelessProfiles(c *gin.Context) {
	guid := c.Param("guid")

	response, err := r.d.GetWirelessProfiles(c.Request.Context(), guid)
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

	err := r.d.AddWirelessProfile(c.Request.Context(), guid, req.ToWirelessProfile())
	if err != nil {
		r.l.Error(err, "http - v1 - addWirelessProfile")
		ErrorResponse(c, err)

		return
	}

	c.Status(http.StatusNoContent)
}

func (r *deviceManagementRoutes) deleteWirelessProfile(c *gin.Context) {
	guid := c.Param("guid")
	profileName := c.Param("profileName")

	err := r.d.DeleteWirelessProfile(c.Request.Context(), guid, profileName)
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

	err := r.d.UpdateWirelessProfile(c.Request.Context(), guid, req.ToWirelessProfile())
	if err != nil {
		r.l.Error(err, "http - v1 - updateWirelessProfile")
		ErrorResponse(c, err)

		return
	}

	c.Status(http.StatusNoContent)
}
