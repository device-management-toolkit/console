package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
)

// Reuse the same action codes used in Redfish systems
const (
	actionPowerUp   = 2
	actionPowerDown = 8
)

func (r *deviceManagementRoutes) getPowerState(c *gin.Context) {
	guid := c.Param("guid")

	state, err := r.d.GetPowerState(c.Request.Context(), guid)
	if err != nil {
		r.l.Error(err, "http - v1 - getPowerState")
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusOK, state)
}

func (r *deviceManagementRoutes) getPowerCapabilities(c *gin.Context) {
	guid := c.Param("guid")

	power, err := r.d.GetPowerCapabilities(c.Request.Context(), guid)
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

	response, err := r.d.SendPowerAction(c.Request.Context(), guid, powerAction.Action)
	if err != nil {
		r.l.Error(err, "http - v1 - powerAction")
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusOK, response)
}

// powerOn maps POST /amt/power/on/:guid to SendPowerAction with action=2 (Power Up)
func (r *deviceManagementRoutes) powerOn(c *gin.Context) {
	guid := c.Param("guid")

	// Power Up
	response, err := r.d.SendPowerAction(c.Request.Context(), guid, actionPowerUp)
	if err != nil {
		r.l.Error(err, "http - v1 - powerOn")
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusOK, response)
}

// powerOff maps POST /amt/power/off/:guid to SendPowerAction with action=8 (Power Down)
func (r *deviceManagementRoutes) powerOff(c *gin.Context) {
	guid := c.Param("guid")

	// Power Down
	response, err := r.d.SendPowerAction(c.Request.Context(), guid, actionPowerDown)
	if err != nil {
		r.l.Error(err, "http - v1 - powerOff")
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

	features, err := r.d.SetBootOptions(c.Request.Context(), guid, bootSetting)
	if err != nil {
		r.l.Error(err, "http - v1 - setBootOptions")
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusOK, features)
}
