package devices

import (
	"context"
	"errors"
	"strings"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/amterror"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/boot"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/redirection"
	cimBoot "github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/boot"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/kvm"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/ips/optin"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	dtov2 "github.com/device-management-toolkit/console/internal/entity/dto/v2"
	"github.com/device-management-toolkit/console/internal/usecase/devices/wsman"
	"github.com/device-management-toolkit/console/pkg/consoleerrors"
)

var ErrOCRNotSupportedUseCase = NotSupportedError{Console: consoleerrors.CreateConsoleError("One Click Recovery Unsupported")}

// AMT BootService EnabledState constants (CIM_BootService.RequestStateChange ValueMap).
const (
	enabledStateOCRAndRPEDisabled = 32768 // OCR disabled, RPE disabled
	enabledStateOCREnabled        = 32769 // OCR enabled,  RPE disabled
	enabledStateRPEEnabled        = 32770 // OCR disabled, RPE enabled
	enabledStateOCRAndRPEEnabled  = 32771 // OCR enabled,  RPE enabled
)

// User consent option string values.
const (
	userConsentKVM = "kvm"
)

// User consent option string values.
const (
	userConsentKVM = "kvm"
)

const (
	targetHTTPSBootInstanceID = "Intel(r) AMT: Force OCR UEFI HTTPS Boot"
	targetsPBAWinREInstanceID = "Intel(r) AMT: Force OCR UEFI Boot Option"
)

// PlatformErase capability bitmask bits (AMT_BootCapabilities.PlatformErase).
const (
	platformEraseRPESupport      = 0x01      // Bit 0: RPE overall support
	platformEraseSecureErase     = 0x04      // Bit 2: Secure Erase All SSDs
	platformEraseTPMClear        = 0x40      // Bit 6: TPM Clear
	platformEraseCSMEUnconfigure = 0x10000   // Bit 16: CSME Unconfigure (UI sentinel)
	platformEraseBIOSReload      = 0x4000000 // Bit 26: BIOS Reload of Golden Configuration
)

type BootConfiguration struct {
	bootService        cimBoot.BootService
	bootSourceSettings []cimBoot.BootSourceSetting
	capabilities       boot.BootCapabilitiesResponse
	bootData           boot.BootSettingDataResponse
}

func (uc *UseCase) GetFeatures(c context.Context, guid string) (settingsResults dto.Features, settingsResultsV2 dtov2.Features, err error) {
	item, err := uc.repo.GetByID(c, guid, "")
	if err != nil {
		return dto.Features{}, dtov2.Features{}, err
	}

	if item == nil || item.GUID == "" {
		return settingsResults, settingsResultsV2, ErrNotFound
	}

	device, err := uc.device.SetupWsmanClient(c, *item, false, true)
	if err != nil {
		return dto.Features{}, dtov2.Features{}, err
	}

	// Get redirection settings from AMT
	err = getRedirectionService(&settingsResultsV2, device)
	if err != nil {
		return settingsResults, settingsResultsV2, err
	}

	// translate v2 features to v1 - remove this once v1 is deprecated
	settingsResults.EnableSOL = settingsResultsV2.EnableSOL
	settingsResults.EnableIDER = settingsResultsV2.EnableIDER
	settingsResults.Redirection = settingsResultsV2.Redirection

	// Get optinservice settings from AMT
	err = getUserConsent(&settingsResultsV2, device)
	if err != nil {
		return settingsResults, settingsResultsV2, err
	}

	// translate v2 features to v1 - remove this once v1 is deprecated
	settingsResults.UserConsent = settingsResultsV2.UserConsent
	settingsResults.OptInState = settingsResultsV2.OptInState

	// Get KVM settings from AMT
	err = getKVM(&settingsResultsV2, device)
	if err != nil {
		return settingsResults, settingsResultsV2, err
	}

	// translate v2 features to v1 - remove this once v1 is deprecated
	settingsResults.EnableKVM = settingsResultsV2.EnableKVM
	settingsResults.KVMAvailable = settingsResultsV2.KVMAvailable

	// Get boot service related settings
	err = getBootConfigurationSettings(&settingsResultsV2, device)
	if err != nil {
		return dto.Features{}, dtov2.Features{}, err
	}

	settingsResults.OCR = settingsResultsV2.OCR
	settingsResults.RPE = settingsResultsV2.RPE
	settingsResults.HTTPSBootSupported = settingsResultsV2.HTTPSBootSupported
	settingsResults.WinREBootSupported = settingsResultsV2.WinREBootSupported
	settingsResults.LocalPBABootSupported = settingsResultsV2.LocalPBABootSupported
	settingsResults.RPESupported = settingsResultsV2.RPESupported

	uc.log.Debug("GetFeatures: RemoteErase (PlatformErase) support guid=%s RPESupported=%v RPE=%v", guid, settingsResultsV2.RPESupported, settingsResultsV2.RPE)

	return settingsResults, settingsResultsV2, nil
}

func getBootConfiguration(device wsman.Management) BootConfiguration {
	// These are non-fatal: if unavailable, OCR/RPE state and boot settings default to zero values
	bootService, _ := device.GetBootService()

	var bootSourceSettings cimBoot.Response

	bootSourceSettings, _ = device.GetCIMBootSourceSetting()

	// Non-fatal: older devices that do not support OCR/RPE will return an error here;
	// all capability-derived fields will default to false/zero.
	capabilities, _ := device.GetBootCapabilities()

	bootData, _ := device.GetBootData()

	return BootConfiguration{
		bootService:        bootService,
		bootSourceSettings: bootSourceSettings.Body.PullResponse.BootSourceSettingItems,
		capabilities:       capabilities,
		bootData:           bootData,
	}
}

func getBootConfigurationSettings(settingsResultsV2 *dtov2.Features, device wsman.Management) error {
	bootConfig := getBootConfiguration(device)

	isOCR := bootConfig.bootService.EnabledState == enabledStateOCREnabled || bootConfig.bootService.EnabledState == enabledStateOCRAndRPEEnabled
	isRPE := bootConfig.bootService.EnabledState == enabledStateRPEEnabled || bootConfig.bootService.EnabledState == enabledStateOCRAndRPEEnabled

	result := FindBootSettingInstances(bootConfig.bootSourceSettings)

	// AMT_BootSettingData.UEFIHTTPSBootEnabled is read-only. AMT_BootCapabilities instance is read-only.
	// So, these cannot be updated
	settingsResultsV2.OCR = isOCR
	settingsResultsV2.RPE = isRPE
	settingsResultsV2.HTTPSBootSupported = result.IsHTTPSBootExists && bootConfig.capabilities.ForceUEFIHTTPSBoot && bootConfig.bootData.UEFIHTTPSBootEnabled
	settingsResultsV2.WinREBootSupported = result.IsWinREExists && bootConfig.bootData.WinREBootEnabled && bootConfig.capabilities.ForceWinREBoot
	settingsResultsV2.LocalPBABootSupported = result.IsPBAExists && bootConfig.bootData.UEFILocalPBABootEnabled && bootConfig.capabilities.ForceUEFILocalPBABoot
	settingsResultsV2.RPESupported = bootConfig.capabilities.PlatformErase&platformEraseRPESupport != 0

	return nil
}

func FindBootSettingInstances(bootSourceSettings []cimBoot.BootSourceSetting) dtov2.BootSettings {
	result := dtov2.BootSettings{}

	for _, setting := range bootSourceSettings {
		instanceID := setting.InstanceID
		biosBootString := setting.BIOSBootString

		if strings.HasPrefix(instanceID, targetHTTPSBootInstanceID) {
			result.IsHTTPSBootExists = true
		}

		if strings.HasPrefix(instanceID, targetsPBAWinREInstanceID) && strings.Contains(biosBootString, "WinRe") {
			result.IsWinREExists = true
		}

		if strings.HasPrefix(instanceID, targetsPBAWinREInstanceID) && strings.Contains(biosBootString, "PBA") {
			result.IsPBAExists = true
		}

		if result.IsHTTPSBootExists && result.IsPBAExists && result.IsWinREExists {
			break
		}
	}

	return result
}

func (uc *UseCase) SetFeatures(c context.Context, guid string, features dto.Features) (settingsResults dto.Features, settingsResultsV2 dtov2.Features, err error) {
	item, err := uc.repo.GetByID(c, guid, "")
	if err != nil {
		return settingsResults, settingsResultsV2, err
	}

	if item == nil || item.GUID == "" {
		return settingsResults, settingsResultsV2, ErrNotFound
	}

	device, err := uc.device.SetupWsmanClient(c, *item, false, true)
	if err != nil {
		return settingsResults, settingsResultsV2, err
	}

	// redirection
	state, listenerEnabled, err := redirectionRequestStateChange(features.EnableSOL, features.EnableIDER, &settingsResultsV2, device)
	if err != nil {
		return settingsResults, settingsResultsV2, err
	}

	// translate v2 features to v1 - remove this once v1 is deprecated
	settingsResults.EnableSOL = settingsResultsV2.EnableSOL
	settingsResults.EnableIDER = settingsResultsV2.EnableIDER

	// kvm
	kvmListenerEnabled, err := setKVM(features.EnableKVM, &settingsResultsV2, device)
	if err != nil {
		return settingsResults, settingsResultsV2, err
	}

	// translate v2 features to v1 - remove this once v1 is deprecated
	settingsResults.EnableKVM = settingsResultsV2.EnableKVM

	// get and put redirection
	err = setRedirectionService(state, listenerEnabled, kvmListenerEnabled, device)
	if err != nil {
		return settingsResults, settingsResultsV2, err
	}

	// Update Redirection, this is important when KVM, IDER and SOL are all false
	settingsResults.Redirection = listenerEnabled == 1 || kvmListenerEnabled == 1
	settingsResultsV2.Redirection = listenerEnabled == 1 || kvmListenerEnabled == 1

	// user consent
	err = setUserConsent(features.UserConsent, device)
	if err != nil {
		return settingsResults, settingsResultsV2, err
	}

	settingsResults.UserConsent = features.UserConsent
	settingsResultsV2.UserConsent = features.UserConsent

	// RPE: must run before BootServiceStateChange (OCR state change blocks the PUT)
	if err := setRPE(features.RPE, &settingsResultsV2, device); err != nil {
		return settingsResults, settingsResultsV2, err
	}

	// Remote Platform Erase (RPE) support and capabilities may be affected by the state change, so we need to get the settings again to return the correct values.
	syncRPEResults(&settingsResultsV2, &settingsResults)

	// Configure OCR/RPE boot service state
	// 32768 = both disabled, 32769 = OCR only, 32770 = RPE only, 32771 = both enabled
	requestedState := ocrBootState(features.OCR, features.RPE)

	_, err = device.BootServiceStateChange(requestedState)
	if err == nil {
		// Get OCR settings
		err = getBootConfigurationSettings(&settingsResultsV2, device)
		if err != nil {
			return settingsResults, settingsResultsV2, nil
		}

		settingsResults.OCR = settingsResultsV2.OCR
		settingsResults.HTTPSBootSupported = settingsResultsV2.HTTPSBootSupported
		settingsResults.WinREBootSupported = settingsResultsV2.WinREBootSupported
		settingsResults.LocalPBABootSupported = settingsResultsV2.LocalPBABootSupported

		return settingsResults, settingsResultsV2, err
	}

	return settingsResults, settingsResultsV2, nil
}

func syncRPEResults(src *dtov2.Features, dst *dto.Features) {
	dst.RPE = src.RPE
	dst.RPESupported = src.RPESupported
}

func setRPE(enableRemoteErase bool, settingsResultsV2 *dtov2.Features, device wsman.Management) error {
	bootCapabilities, err := device.GetBootCapabilities()
	if err != nil {
		return err
	}

	settingsResultsV2.RPE = enableRemoteErase
	settingsResultsV2.RPESupported = bootCapabilities.PlatformErase&platformEraseRPESupport != 0 // Bit 0: RPE overall support

	return nil
}

func ocrBootState(ocr, rpe bool) int {
	switch {
	case ocr && rpe:
		return enabledStateOCRAndRPEEnabled
	case ocr:
		return enabledStateOCREnabled
	case rpe:
		return enabledStateRPEEnabled
	default:
		return enabledStateOCRAndRPEDisabled
	}
}

func handleAMTKVMError(err error, results *dtov2.Features) bool {
	amtErr := &amterror.AMTError{}
	if errors.As(err, &amtErr) {
		if strings.Contains(amtErr.SubCode, "DestinationUnreachable") {
			results.EnableKVM = false
			results.KVMAvailable = false

			return true
		}
	}

	return false
}

func getSOLAndIDERState(enabledState redirection.EnabledState) (iderEnabled, solEnabled bool) {
	//nolint:exhaustive // we only care about IDER and SOL states. Other scenarios are handled by the default case.
	switch enabledState {
	case redirection.IDERAndSOLAreDisabled:
		return false, false
	case redirection.IDERIsEnabledAndSOLIsDisabled:
		return true, false
	case redirection.SOLIsEnabledAndIDERIsDisabled:
		return false, true
	case redirection.IDERAndSOLAreEnabled:
		return true, true
	default:
		return false, false // default case if state is invalid
	}
}

func redirectionRequestStateChange(enableSOL, enableIDER bool, results *dtov2.Features, w wsman.Management) (state redirection.EnabledState, listenerEnabled int, err error) {
	requestedState, listenerEnabled, err := w.RequestAMTRedirectionServiceStateChange(enableIDER, enableSOL)
	if err != nil {
		return 0, 0, err
	}

	state = redirection.EnabledState(requestedState)
	iderEnabled, solEnabled := getSOLAndIDERState(state)
	results.EnableSOL = solEnabled
	results.EnableIDER = iderEnabled
	results.Redirection = listenerEnabled == 1

	return state, listenerEnabled, nil
}

func getKVM(results *dtov2.Features, w wsman.Management) error {
	kvmResult, err := w.GetKVMRedirection()
	if err != nil {
		isAMTErr := handleAMTKVMError(err, results)
		if !isAMTErr {
			return err
		}
	} else {
		results.EnableKVM = kvmResult.Body.GetResponse.EnabledState == kvm.EnabledState(redirection.Enabled) || kvmResult.Body.GetResponse.EnabledState == kvm.EnabledState(redirection.EnabledButOffline)
		results.KVMAvailable = true
	}

	return nil
}

func setKVM(enableKVM bool, results *dtov2.Features, w wsman.Management) (kvmListenerEnabled int, err error) {
	kvmListenerEnabled, err = w.SetKVMRedirection(enableKVM)
	if err != nil {
		isAMTErr := handleAMTKVMError(err, results)
		if !isAMTErr {
			return 0, err
		}
	} else {
		results.EnableKVM = kvmListenerEnabled == 1
		results.KVMAvailable = true
	}

	return kvmListenerEnabled, nil
}

func getRedirectionService(results *dtov2.Features, w wsman.Management) error {
	redirectionResult, err := w.GetAMTRedirectionService()
	if err != nil {
		return err
	}

	iderEnabled, solEnabled := getSOLAndIDERState(redirectionResult.Body.GetAndPutResponse.EnabledState)
	results.EnableSOL = solEnabled
	results.EnableIDER = iderEnabled
	results.Redirection = redirectionResult.Body.GetAndPutResponse.ListenerEnabled

	return nil
}

func setRedirectionService(state redirection.EnabledState, listenerEnabled, kvmListenerEnabled int, w wsman.Management) error {
	currentRedirection, err := w.GetAMTRedirectionService()
	if err != nil {
		return err
	}

	request := &redirection.RedirectionRequest{
		CreationClassName:       currentRedirection.Body.GetAndPutResponse.CreationClassName,
		ElementName:             currentRedirection.Body.GetAndPutResponse.ElementName,
		EnabledState:            state,
		ListenerEnabled:         listenerEnabled == 1 || kvmListenerEnabled == 1,
		Name:                    currentRedirection.Body.GetAndPutResponse.Name,
		SystemCreationClassName: currentRedirection.Body.GetAndPutResponse.SystemCreationClassName,
		SystemName:              currentRedirection.Body.GetAndPutResponse.SystemName,
	}

	_, err = w.SetAMTRedirectionService(request)
	if err != nil {
		return err
	}

	return nil
}

func getUserConsent(results *dtov2.Features, w wsman.Management) error {
	optServiceResult, err := w.GetIPSOptInService()
	if err != nil {
		return err
	}

	results.UserConsent = UserConsentOptions[int(optServiceResult.Body.GetAndPutResponse.OptInRequired)]
	results.OptInState = optServiceResult.Body.GetAndPutResponse.OptInState

	return nil
}

func setUserConsent(userConsent string, w wsman.Management) error {
	optInResponse, err := w.GetIPSOptInService()
	if err != nil {
		return err
	}

	optinRequest := optin.OptInServiceRequest{
		CreationClassName:       optInResponse.Body.GetAndPutResponse.CreationClassName,
		ElementName:             optInResponse.Body.GetAndPutResponse.ElementName,
		Name:                    optInResponse.Body.GetAndPutResponse.Name,
		OptInCodeTimeout:        optInResponse.Body.GetAndPutResponse.OptInCodeTimeout,
		OptInDisplayTimeout:     optInResponse.Body.GetAndPutResponse.OptInDisplayTimeout,
		OptInRequired:           determineConsentCode(userConsent),
		SystemName:              optInResponse.Body.GetAndPutResponse.SystemName,
		SystemCreationClassName: optInResponse.Body.GetAndPutResponse.SystemCreationClassName,
	}

	err = w.SetIPSOptInService(optinRequest)
	if err != nil {
		return err
	}

	return nil
}

func determineConsentCode(consent string) int {
	consentCode := optin.OptInRequiredAll // default to all if not valid user consent

	consent = strings.ToLower(consent)

	switch consent {
	case userConsentKVM:
		consentCode = optin.OptInRequiredKVM
	case "all":
		consentCode = optin.OptInRequiredAll
	case "none":
		consentCode = optin.OptInRequiredNone
	}

	return int(consentCode)
}

var UserConsentOptions = map[int]string{
	0:          "none",
	1:          userConsentKVM,
	4294967295: "all",
}
