package devices

import (
	"context"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/boot"

	wsmanAPI "github.com/device-management-toolkit/console/internal/usecase/devices/wsman"
)

// setupDeviceClient retrieves a device by GUID and sets up the wsman client.
func (uc *UseCase) setupDeviceClient(c context.Context, guid string) (wsmanAPI.Management, error) {
	item, err := uc.repo.GetByID(c, guid, "")
	if err != nil {
		return nil, err
	}

	if item == nil || item.GUID == "" {
		return nil, ErrNotFound
	}

	device, err := uc.device.SetupWsmanClient(*item, false, true)
	if err != nil {
		return nil, err
	}

	return device, nil
}

// GetBootData retrieves the current boot settings from a device.
func (uc *UseCase) GetBootData(c context.Context, guid string) (boot.BootSettingDataResponse, error) {
	device, err := uc.setupDeviceClient(c, guid)
	if err != nil {
		return boot.BootSettingDataResponse{}, err
	}

	bootData, err := device.GetBootData()
	if err != nil {
		return boot.BootSettingDataResponse{}, err
	}

	return bootData, nil
}

// SetBootData configures boot settings for a device.
func (uc *UseCase) SetBootData(c context.Context, guid string, bootData boot.BootSettingDataRequest) error {
	device, err := uc.setupDeviceClient(c, guid)
	if err != nil {
		return err
	}

	// Clear existing boot order
	_, err = device.ChangeBootOrder("")
	if err != nil {
		return err
	}

	// Set new boot data
	_, err = device.SetBootData(bootData)
	if err != nil {
		return err
	}

	// Enable boot configuration
	_, err = device.SetBootConfigRole(1)
	if err != nil {
		return err
	}

	return nil
}

// ChangeBootOrder sets the boot order for a device.
func (uc *UseCase) ChangeBootOrder(c context.Context, guid, bootSource string) error {
	device, err := uc.setupDeviceClient(c, guid)
	if err != nil {
		return err
	}

	_, err = device.ChangeBootOrder(bootSource)

	return err
}
