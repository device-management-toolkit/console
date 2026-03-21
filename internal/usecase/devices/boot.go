package devices

import (
	"context"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
)

func (uc *UseCase) GetBootCapabilities(c context.Context, guid string) (dto.BootCapabilities, error) {
	item, err := uc.repo.GetByID(c, guid, "")
	if err != nil {
		return dto.BootCapabilities{}, err
	}

	if item == nil || item.GUID == "" {
		return dto.BootCapabilities{}, ErrNotFound
	}

	device, err := uc.device.SetupWsmanClient(*item, false, true)
	if err != nil {
		return dto.BootCapabilities{}, err
	}

	capabilities, err := device.GetBootCapabilities()
	if err != nil {
		return dto.BootCapabilities{}, err
	}

	uc.log.Debug("GetBootCapabilities: PlatformErase capability", "guid", guid, "PlatformErase", capabilities.PlatformErase, "supported", capabilities.PlatformErase != 0)

	return dto.BootCapabilities{
		IDER:                               capabilities.IDER,
		SOL:                                capabilities.SOL,
		BIOSReflash:                        capabilities.BIOSReflash,
		BIOSSetup:                          capabilities.BIOSSetup,
		BIOSPause:                          capabilities.BIOSPause,
		ForcePXEBoot:                       capabilities.ForcePXEBoot,
		ForceHardDriveBoot:                 capabilities.ForceHardDriveBoot,
		ForceHardDriveSafeModeBoot:         capabilities.ForceHardDriveSafeModeBoot,
		ForceDiagnosticBoot:                capabilities.ForceDiagnosticBoot,
		ForceCDorDVDBoot:                   capabilities.ForceCDorDVDBoot,
		VerbosityScreenBlank:               capabilities.VerbosityScreenBlank,
		PowerButtonLock:                    capabilities.PowerButtonLock,
		ResetButtonLock:                    capabilities.ResetButtonLock,
		KeyboardLock:                       capabilities.KeyboardLock,
		SleepButtonLock:                    capabilities.SleepButtonLock,
		UserPasswordBypass:                 capabilities.UserPasswordBypass,
		ForcedProgressEvents:               capabilities.ForcedProgressEvents,
		VerbosityVerbose:                   capabilities.VerbosityVerbose,
		VerbosityQuiet:                     capabilities.VerbosityQuiet,
		ConfigurationDataReset:             capabilities.ConfigurationDataReset,
		BIOSSecureBoot:                     capabilities.BIOSSecureBoot,
		SecureErase:                        capabilities.SecureErase,
		ForceWinREBoot:                     capabilities.ForceWinREBoot,
		ForceUEFILocalPBABoot:              capabilities.ForceUEFILocalPBABoot,
		ForceUEFIHTTPSBoot:                 capabilities.ForceUEFIHTTPSBoot,
		AMTSecureBootControl:               capabilities.AMTSecureBootControl,
		UEFIWiFiCoExistenceAndProfileShare: capabilities.UEFIWiFiCoExistenceAndProfileShare,
		PlatformErase:                      capabilities.PlatformErase,
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

	device, err := uc.device.SetupWsmanClient(*item, false, true)
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

	device, err := uc.device.SetupWsmanClient(*item, false, true)
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

	if eraseMask != 0 && (capabilities.PlatformErase&eraseMask) == 0 {
		return ValidationError{}.Wrap("SendRemoteErase", "validate erase mask", "requested erase capabilities are not supported by this device")
	}

	return device.SendRemoteErase(eraseMask)
}
