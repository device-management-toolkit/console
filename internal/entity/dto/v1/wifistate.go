package dto

import (
	"github.com/go-playground/validator/v10"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/wifi"
)

type WirelessState string

type WirelessStateChangeRequest struct {
	State WirelessState `json:"state" binding:"required,wifistate"`
}

type WirelessStateResponse struct {
	State WirelessState `json:"state"`
}

// ValidateWirelessState verifies the API string can be parsed by the shared wifi enum parser.
var ValidateWirelessState validator.Func = func(fl validator.FieldLevel) bool {
	_, ok := wifi.ParseRequestedState(fl.Field().String())

	return ok
}
