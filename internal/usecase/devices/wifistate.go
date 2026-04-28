package devices

import (
	"context"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/wifi"

	wsman "github.com/device-management-toolkit/console/internal/usecase/devices/wsman"
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

	enumerateResponse, err := device.EnumerateWiFiPort()
	if err != nil {
		return 0, err
	}

	pullResponse, err := device.PullWiFiPort(enumerateResponse.Body.EnumerateResponse.EnumerationContext)
	if err != nil {
		return 0, err
	}

	if len(pullResponse.Body.PullResponse.WiFiPortItems) == 0 {
		return 0, wsman.ErrNoWiFiPort
	}

	return pullResponse.Body.PullResponse.WiFiPortItems[0].EnabledState, nil
}
