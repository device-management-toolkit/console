package devices

import (
	"context"
	"errors"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	deviceManagement "github.com/device-management-toolkit/console/internal/usecase/devices/wsman"
	"github.com/device-management-toolkit/console/pkg/consoleerrors"
)

func (uc *UseCase) GetBootCapabilities(c context.Context, guid string) (dto.BootCapabilities, error) {
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

	uc.log.Debug("GetBootCapabilities: PlatformErase capability", "guid", guid, "PlatformErase", capabilities.PlatformErase, "supported", capabilities.PlatformErase != 0)

	return dto.BootCapabilities{
		PlatformErase: capabilities.PlatformErase,
	}, nil
}

func (uc *UseCase) SetRPEEnabled(c context.Context, guid string, enabled bool) error {
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
		return ValidationError{}.Wrap("SetRPEEnabled", "check boot capabilities", "device does not support Remote Platform Erase")
	}

	return device.SetRPEEnabled(enabled)
}

func (uc *UseCase) SendRemoteErase(c context.Context, guid string, eraseMask int) error {
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
		return ValidationError{}.Wrap("SendRemoteErase", "check boot capabilities", "device does not support Remote Platform Erase")
	}

	uc.log.Debug("SendRemoteErase guid=%s eraseMask=%d secureErase=%v ecStorage=%v storageDrives=%v meRegion=%v",
		guid, eraseMask, eraseMask&0x01 != 0, eraseMask&0x02 != 0, eraseMask&0x04 != 0, eraseMask&0x08 != 0)

	if err := device.SendRemoteErase(eraseMask); err != nil {
		if errors.Is(err, deviceManagement.ErrRPENotEnabled) {
			return NotSupportedError{Console: consoleerrors.CreateConsoleError("Remote Platform Erase is not enabled by the BIOS on this device")}
		}

		return err
	}

	return nil
}
