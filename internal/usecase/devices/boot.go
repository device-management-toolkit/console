package devices

import (
	"context"
	"errors"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/boot"

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

	return remoteEraseStatusFromBIOS(data.BIOSLastStatus), nil
}

func remoteEraseStatusFromBIOS(status []uint16) dtov2.RemoteEraseStatus {
	result := dtov2.RemoteEraseStatus{Status: dtov2.RemoteEraseStateUnknown, Message: "BIOS erase outcome is unconfirmed."}
	if len(status) > 0 {
		result.BIOSStatus = &status[0]
	}

	if len(status) < biosStatusWordCount {
		return result
	}

	result.BIOSDetail = &status[1]

	switch status[0] {
	case biosStatusSuccess:
		if status[1] == biosDetailSuccess {
			result.Status, result.Message = dtov2.RemoteEraseStateSucceeded, "BIOS reported the last boot completed without error. This does not confirm completion of the current erase request."
		}
	case biosStatusInProgress:
		result.Status, result.Message = dtov2.RemoteEraseStatePending, "Waiting for the BIOS erase result."
	case biosStatusNotUpdated:
		result.Message = "BIOS status has not been updated. Completion of the requested erase is unconfirmed."
	case biosStatusFailed:
		result.Status, result.Message = dtov2.RemoteEraseStateFailed, "BIOS reported that the erase failed."

		switch status[1] {
		case biosDetailDriveFailure:
			result.Message = "BIOS erase failed: general drive failure."
		case biosDetailAuthenticationFailure:
			result.Message = "BIOS erase failed: drive password/authentication failure."
		case biosDetailUnsupportedFeature:
			result.Message = "BIOS erase failed: feature is not supported."
		}
	}

	return result
}

func (uc *UseCase) SetRemoteEraseOptions(c context.Context, guid, tenantID string, req dto.RemoteEraseRequest) error {
	if err := validateRemoteEraseCredentials(req); err != nil {
		return err
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

	if err := device.SetRemoteEraseOptions(eraseMask, deviceManagement.RPECredentials{
		SSDPassword: req.SSDPassword,
		PSID:        req.PSID,
	}); err != nil {
		if errors.Is(err, deviceManagement.ErrRPENotEnabled) {
			return NotSupportedError{Console: consoleerrors.CreateConsoleError("Remote Platform Erase is not enabled by the BIOS on this device")}
		}

		return err
	}

	return nil
}

func validateRemoteEraseCredentials(req dto.RemoteEraseRequest) error {
	if len(req.SSDPassword) > boot.RPEMaxSizes[boot.RPE_SSD_MASTER_PASSWORD] || len(req.PSID) > boot.RPEMaxSizes[boot.RPE_PSID] {
		return ValidationError{}.Wrap("SetRemoteEraseOptions", "check RPE credentials", deviceManagement.ErrRPECredentialTooLong.Error())
	}

	if !req.SecureEraseAllSSDs && (req.SSDPassword != "" || req.PSID != "") {
		return ValidationError{}.Wrap("SetRemoteEraseOptions", "check RPE credentials", "ssdPassword and psid require secureEraseAllSSDs to be selected")
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
