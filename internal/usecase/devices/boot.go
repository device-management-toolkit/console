package devices

import (
	"context"
	"errors"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	dtov2 "github.com/device-management-toolkit/console/internal/entity/dto/v2"
	deviceManagement "github.com/device-management-toolkit/console/internal/usecase/devices/wsman"
	"github.com/device-management-toolkit/console/pkg/consoleerrors"
)

const (
	biosStatusWordCount             = 2
	biosStatusSuccess               = 0
	biosStatusInProgress            = 1
	biosStatusNotUpdated            = 2
	biosStatusFailed                = 0xffff
	biosDetailSuccess               = 0
	biosDetailDriveFailure          = 1
	biosDetailAuthenticationFailure = 2
	biosDetailUnsupportedFeature    = 3
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

func (uc *UseCase) GetRemoteEraseStatus(c context.Context, guid, tenantID string) (dtov2.RemoteEraseStatus, error) {
	item, err := uc.repo.GetByID(c, guid, tenantID)
	if err != nil {
		return dtov2.RemoteEraseStatus{}, err
	}

	if item == nil || item.GUID == "" {
		return dtov2.RemoteEraseStatus{}, ErrNotFound
	}

	device, err := uc.device.SetupWsmanClient(c, *item, false, true)
	if err != nil {
		return dtov2.RemoteEraseStatus{}, err
	}

	data, err := device.GetBootData()
	if err != nil {
		return dtov2.RemoteEraseStatus{}, err
	}

	result := dtov2.RemoteEraseStatus{Status: "unknown", Message: "BIOS erase outcome is unconfirmed."}
	if len(data.BIOSLastStatus) < biosStatusWordCount {
		return result, nil
	}

	switch data.BIOSLastStatus[0] {
	case biosStatusSuccess:
		if data.BIOSLastStatus[1] == biosDetailSuccess {
			result.Status, result.Message = "succeeded", "BIOS reported that the erase completed successfully."
		}
	case biosStatusInProgress, biosStatusNotUpdated:
		result.Status, result.Message = "pending", "Waiting for the BIOS erase result."
	case biosStatusFailed:
		result.Status, result.Message = "failed", "BIOS reported that the erase failed."

		switch data.BIOSLastStatus[1] {
		case biosDetailDriveFailure:
			result.Message = "BIOS erase failed: general drive failure."
		case biosDetailAuthenticationFailure:
			result.Message = "BIOS erase failed: drive password/authentication failure."
		case biosDetailUnsupportedFeature:
			result.Message = "BIOS erase failed: feature is not supported."
		}
	}

	return result, nil
}

func (uc *UseCase) SetRemoteEraseOptions(c context.Context, guid, tenantID string, req dto.RemoteEraseRequest) error {
	if len(req.SSDPassword) > deviceManagement.RPECredentialMaxBytes || len(req.PSID) > deviceManagement.RPECredentialMaxBytes {
		return ValidationError{}.Wrap("SetRemoteEraseOptions", "check RPE credentials", deviceManagement.ErrRPECredentialTooLong.Error())
	}

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

	eraseMask := remoteEraseMask(req)
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

	if err := device.SetRemoteEraseOptions(eraseMask, req.SSDPassword, req.PSID); err != nil {
		if errors.Is(err, deviceManagement.ErrRPENotEnabled) {
			return NotSupportedError{Console: consoleerrors.CreateConsoleError("Remote Platform Erase is not enabled by the BIOS on this device")}
		}

		return err
	}

	return nil
}

func remoteEraseMask(req dto.RemoteEraseRequest) int {
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

	return eraseMask
}
