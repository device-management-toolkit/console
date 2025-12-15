package v1

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/usecase/devices/wsman"
)

// setLinkPreference sets the link preference (ME or Host) on a device's WiFi port.
func (r *deviceManagementRoutes) setLinkPreference(c *gin.Context) {
	guid := c.Param("guid")

	var req dto.LinkPreferenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, err)

		return
	}

	response, err := r.d.SetLinkPreference(c.Request.Context(), guid, req)

	if err != nil {
		r.l.Error(err, "http - v1 - setLinkPreference")
		// Handle no WiFi port error with 404 and error message
		if errors.Is(err, wsman.ErrNoWiFiPort) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Set Link Preference failed: No WiFi port found for guid : " + guid + ".",
			})
			return
		}
		// For other errors (device not found, validation, etc.), use standard error response
		ErrorResponse(c, err)
		return
	}

	// Map AMT return value to HTTP status code
	// Non-zero return value -> 400 Bad Request with error message
	// 0 -> 200 OK with success response
	if response.ReturnValue != 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Set Link Preference failed for guid : " + guid + ".",
		})
		return
	}

	c.JSON(http.StatusOK, response)
}
