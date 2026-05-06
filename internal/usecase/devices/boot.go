package devices

import (
	"context"
	"errors"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	deviceManagement "github.com/device-management-toolkit/console/internal/usecase/devices/wsman"
	"github.com/device-management-toolkit/console/pkg/consoleerrors"
)

func (uc *UseCase) GetRemoteEraseCapabilities(c context.Context, guid string) (dto.BootCapabilities, error) {
	item, err := uc.repo.GetByID(c, guid, "")
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

	capabilities, err := device.GetBootCapabilities()
	if err != nil {
		return dto.BootCapabilities{}, err
	}

	uc.log.Debug("getRemoteEraseCapabilities: PlatformErase capability", "guid", guid, "PlatformErase", capabilities.PlatformErase, "supported", capabilities.PlatformErase != 0)

	return dto.BootCapabilities{
		SecureEraseAllSSDs: capabilities.PlatformErase&platformEraseSecureErase != 0,
		TPMClear:           capabilities.PlatformErase&platformEraseTPMClear != 0,
		RestoreBIOSToEOM:   capabilities.PlatformErase&platformEraseBIOSReload != 0,
		UnconfigureCSME:    capabilities.PlatformErase != 0,
	}, nil
}

func (uc *UseCase) SetRemoteEraseOptions(c context.Context, guid string, req dto.RemoteEraseRequest) error {
	item, err := uc.repo.GetByID(c, guid, "")
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

	capabilities, err := device.GetBootCapabilities()
	if err != nil {
		return err
	}

	if capabilities.PlatformErase == 0 {
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
		eraseMask |= platformEraseCSMEUnconfigure
	}

	uc.log.Debug("SetRemoteEraseOptions guid=%s eraseMask=0x%x secureErase=%v tpmClear=%v biosReload=%v csmeReset=%v",
		guid, eraseMask,
		req.SecureEraseAllSSDs,
		req.TPMClear,
		req.RestoreBIOSToEOM,
		req.UnconfigureCSME,
	)

	if err := device.SetRemoteEraseOptions(eraseMask); err != nil {
		if errors.Is(err, deviceManagement.ErrRPENotEnabled) {
			return NotSupportedError{Console: consoleerrors.CreateConsoleError("Remote Platform Erase is not enabled by the BIOS on this device")}
		}

		return err
	}

	return nil
}
