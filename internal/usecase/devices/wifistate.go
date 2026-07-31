package devices

import (
	"context"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/wifi"
)

func (uc *UseCase) RequestWirelessStateChange(c context.Context, guid string, requestedState wifi.RequestedState) (wifi.RequestedState, error) {
	if !isWirelessRequestedStateSupported(requestedState) {
		return 0, ErrValidationUseCase.Wrap("RequestWirelessStateChange", "validate requested state", "state must be one of 3, 32768, 32769")
	}

	item, err := uc.repo.GetByID(c, guid, "")
	if err != nil {
		return 0, err
	}

	if item == nil || item.GUID == "" {
		return 0, ErrNotFound
	}

	device, err := uc.device.SetupWsmanClient(c, *item, false, true)
	if err != nil {
		return 0, err
	}

	ports, err := device.GetWiFiPorts()
	if err != nil {
		return 0, err
	}

	if len(ports) > 0 && ports[0].EnabledState == wifi.EnabledState(requestedState) {
		return requestedState, nil
	}

	if err := device.WiFiRequestStateChange(requestedState); err != nil {
		return 0, err
	}

	return requestedState, nil
}

func isWirelessRequestedStateSupported(requestedState wifi.RequestedState) bool {
	switch requestedState {
	case wifi.RequestedStateWifiDisabled, wifi.RequestedStateWifiEnabledS0, wifi.RequestedStateWifiEnabledS0SxAC:
		return true
	default:
		return false
	}
}

func (uc *UseCase) GetWirelessState(c context.Context, guid string) (wifi.EnabledState, error) {
	item, err := uc.repo.GetByID(c, guid, "")
	if err != nil {
		return 0, err
	}

	if item == nil || item.GUID == "" {
		return 0, ErrNotFound
	}

	device, err := uc.device.SetupWsmanClient(c, *item, false, true)
	if err != nil {
		return 0, err
	}

	ports, err := device.GetWiFiPorts()
	if err != nil {
		return 0, err
	}

	return ports[0].EnabledState, nil
}
