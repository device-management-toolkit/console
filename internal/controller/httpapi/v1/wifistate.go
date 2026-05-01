package v1

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/wifi"

	dto "github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/usecase/devices/wsman"
	"github.com/device-management-toolkit/console/pkg/consoleerrors"
)

var (
	errUnsupportedRequestedWirelessState = errors.New("unsupported requested wireless state")
	errUnsupportedEnabledWirelessState   = errors.New("unsupported enabled wireless state")
	errValidationWiFiState               = dto.NotValidError{Console: consoleerrors.CreateConsoleError("WifiStateAPI")}
)

func (r *deviceManagementRoutes) requestWirelessStateChange(c *gin.Context) {
	guid := c.Param("guid")

	var req dto.WirelessStateChangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validationErr := errValidationWiFiState.Wrap("requestWirelessStateChange", "ShouldBindJSON", err)
		ErrorResponse(c, validationErr)

		return
	}

	requestedState, ok := wifi.ParseRequestedState(string(req.State))
	if !ok {
		msg := "unsupported wireless state " + string(req.State)
		c.AbortWithStatusJSON(http.StatusBadRequest, response{Error: msg, Message: msg})

		return
	}

	returnedRequestedState, err := r.d.RequestWirelessStateChange(c.Request.Context(), guid, requestedState)
	if err != nil {
		r.l.Error(err, "http - v1 - requestWirelessStateChange")
		ErrorResponse(c, err)

		return
	}

	state := dto.WirelessState(returnedRequestedState.String())
	if string(state) == wifi.ValueNotFound {
		r.l.Error(errUnsupportedRequestedWirelessState, "http - v1 - requestWirelessStateChange - requested state conversion")
		ErrorResponse(c, errUnsupportedRequestedWirelessState)

		return
	}

	c.JSON(http.StatusOK, dto.WirelessStateResponse{State: state})
}

func (r *deviceManagementRoutes) getWirelessState(c *gin.Context) {
	guid := c.Param("guid")

	returnedEnabledState, err := r.d.GetWirelessState(c.Request.Context(), guid)
	if err != nil {
		r.l.Error(err, "http - v1 - getWirelessState")

		if errors.Is(err, wsman.ErrNoWiFiPort) {
			c.JSON(http.StatusNotFound, gin.H{
				errorKey: "Get Wireless State failed for guid: " + guid + ". - " + err.Error(),
			})

			return
		}

		ErrorResponse(c, err)

		return
	}

	state := dto.WirelessState(returnedEnabledState.String())
	if string(state) == wifi.ValueNotFound {
		r.l.Error(errUnsupportedEnabledWirelessState, "http - v1 - getWirelessState - enabled state conversion")
		ErrorResponse(c, errUnsupportedEnabledWirelessState)

		return
	}

	c.JSON(http.StatusOK, dto.WirelessStateResponse{State: state})
}
