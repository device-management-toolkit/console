package devices

import (
	"context"
	"errors"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/boot"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	deviceManagement "github.com/device-management-toolkit/console/internal/usecase/devices/wsman"
	"github.com/device-management-toolkit/console/pkg/consoleerrors"
)

func (uc *UseCase) GetRemoteEraseCapabilities(c context.Context, guid, tenantID string) (dto.BootCapabilities, error) {
	item, err := uc.repo.GetByID(c, guid, tenantID)
	if err != nil {
		return dto.BootCapabilities{}, err
	}

	if item == nil || item.GUID == "" {
		return dto.BootCapabilities{}, ErrNotFound
	}

	device, err := uc.device.SetupWsmanClient(c, *item, false, true)
	if err != nil {
		return dto.BootCapabilities{}, err
	}

	capabilities, err := device.GetPowerCapabilities()
	if err != nil {
		return dto.BootCapabilities{}, err
	}

	uc.log.Debug("getRemoteEraseCapabilities: PlatformErase capability", "guid", guid, "PlatformErase", capabilities.PlatformErase, "supported", capabilities.PlatformErase != 0, "ConfigurationDataReset", capabilities.ConfigurationDataReset)

	return dto.BootCapabilities{
		SecureEraseAllSSDs: capabilities.PlatformErase&platformEraseSecureErase != 0,
		TPMClear:           capabilities.PlatformErase&platformEraseTPMClear != 0,
		RestoreBIOSToEOM:   capabilities.PlatformErase&platformEraseBIOSReload != 0,
		UnconfigureCSME:    capabilities.ConfigurationDataReset,
	}, nil
}

func (uc *UseCase) SetRemoteEraseOptions(c context.Context, guid, tenantID string, req dto.RemoteEraseRequest) error {
	item, err := uc.repo.GetByID(c, guid, tenantID)
	if err != nil {
		return err
	}

	if item == nil || item.GUID == "" {
		return ErrNotFound
	}

	device, err := uc.device.SetupWsmanClient(c, *item, false, true)
	if err != nil {
		return err
	}

	capabilities, err := device.GetPowerCapabilities()
	if err != nil {
		return err
	}

	if capabilities.PlatformErase == 0 && !capabilities.ConfigurationDataReset {
		return ValidationError{}.Wrap("SetRemoteEraseOptions", "check boot capabilities", "device does not support Remote Platform Erase")
	}

	eraseMask := 0
	if req.SecureEraseAllSSDs {
		eraseMask |= platformEraseSecureErase
	}

	if req.TPMClear {
		eraseMask |= platformEraseTPMClear
	}

	if req.RestoreBIOSToEOM {
		eraseMask |= platformEraseBIOSReload
	}

	if req.UnconfigureCSME {
		eraseMask |= deviceManagement.RPEConfigurationDataResetSignalBit
	}

	if eraseMask == 0 {
		return ValidationError{}.Wrap("SetRemoteEraseOptions", "check erase options", "at least one erase option must be selected")
	}

	uc.log.Debug(
		"SetRemoteEraseOptions guid=%s eraseMask=0x%x secureErase=%v tpmClear=%v biosReload=%v csmeReset=%v",
		guid, eraseMask,
		req.SecureEraseAllSSDs,
		req.TPMClear,
		req.RestoreBIOSToEOM,
		req.UnconfigureCSME,
	)

	if err := device.SetRemoteEraseOptions(eraseMask, req.SSDPassword); err != nil {
		if errors.Is(err, deviceManagement.ErrRPENotEnabled) {
			return NotSupportedError{Console: consoleerrors.CreateConsoleError("Remote Platform Erase is not enabled by the BIOS on this device")}
		}

		return err
	}

	return nil
}

// setupDeviceClient retrieves a device by GUID and sets up the wsman client.
func (uc *UseCase) setupDeviceClient(c context.Context, guid string) (deviceManagement.Management, error) {
	item, err := uc.repo.GetByID(c, guid, "")
	if err != nil {
		return nil, err
	}

	if item == nil || item.GUID == "" {
		return nil, ErrNotFound
	}

	device, err := uc.device.SetupWsmanClient(c, *item, false, true)
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
