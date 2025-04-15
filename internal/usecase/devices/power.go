package devices

import (
	"context"
	"encoding/base64"
	"strconv"
	"strings"

	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/amt/boot"
	cimBoot "github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/cim/boot"
	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/cim/power"
	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/cim/software"
	ipsPower "github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/ips/power"

	"github.com/open-amt-cloud-toolkit/console/internal/entity/dto/v1"
	"github.com/open-amt-cloud-toolkit/console/internal/usecase/devices/wsman"
	"github.com/open-amt-cloud-toolkit/console/pkg/consoleerrors"
)

const (
	BootActionHTTPSBoot         = 105
	BootActionPowerOnHTTPSBoot  = 106
	BootActionResetToIDERCDROM  = 202
	BootActionPowerOnIDERCDROM  = 203
	BootActionResetToBIOS       = 101
	BootActionResetToPXE        = 400
	BootActionPowerOnToPXE      = 401
	BootActionResetToDiag       = 301
	BootActionResetToIDERFloppy = 200
	OsToFullPower               = 500
	OsToPowerSaving             = 501
	CIMPMSPowerOn               = 2 // CIM > Power Management Service > Power On
)

var ErrValidationUseCase = ValidationError{Console: consoleerrors.CreateConsoleError("parameter validation failed")}

func (uc *UseCase) SendPowerAction(c context.Context, guid string, action int) (power.PowerActionResponse, error) {
	item, err := uc.repo.GetByID(c, guid, "")
	if err != nil {
		return power.PowerActionResponse{}, err
	}

	if item == nil || item.GUID == "" {
		return power.PowerActionResponse{}, ErrNotFound
	}

	device := uc.device.SetupWsmanClient(*item, false, true)

	if action == OsToFullPower || action == OsToPowerSaving {
		response, err := handleOSPowerSavingStateChange(device, action)
		if err != nil {
			return power.PowerActionResponse{}, err
		}

		return response, nil
	}

	if action == CIMPMSPowerOn {
		_, err := ensureFullPowerBeforeReset(device)
		if err != nil {
			return power.PowerActionResponse{}, err
		}
	}

	response, err := device.SendPowerAction(action)
	if err != nil {
		return power.PowerActionResponse{}, err
	}

	return response, nil
}

func handleOSPowerSavingStateChange(device wsman.Management, action int) (power.PowerActionResponse, error) {
	var targetStateValue int

	if action == OsToFullPower {
		targetStateValue = 2
	} else {
		targetStateValue = 3
	}

	currentState, err := device.GetOSPowerSavingState()
	if err != nil {
		return power.PowerActionResponse{}, err
	}

	if int(currentState) == targetStateValue {
		return power.PowerActionResponse{
			ReturnValue: power.ReturnValue(0),
		}, nil
	}

	response, err := device.RequestOSPowerSavingStateChange(ipsPower.OSPowerSavingState(targetStateValue))
	if err != nil {
		return power.PowerActionResponse{}, err
	}

	return power.PowerActionResponse{
		ReturnValue: power.ReturnValue(response.ReturnValue),
	}, nil
}

func ensureFullPowerBeforeReset(device wsman.Management) (power.PowerActionResponse, error) {
	res, err := handleOSPowerSavingStateChange(device, OsToFullPower)
	if err != nil {
		return power.PowerActionResponse{}, err
	}

	return res, nil
}

func (uc *UseCase) GetPowerState(c context.Context, guid string) (dto.PowerState, error) {
	item, err := uc.repo.GetByID(c, guid, "")
	if err != nil {
		return dto.PowerState{}, err
	}

	if item == nil || item.GUID == "" {
		return dto.PowerState{}, ErrNotFound
	}

	device := uc.device.SetupWsmanClient(*item, false, true)

	state, err := device.GetPowerState()
	if err != nil {
		return dto.PowerState{}, err
	}

	stateOS, err := device.GetOSPowerSavingState()
	if err != nil {
		return dto.PowerState{
			PowerState:         int(state[0].PowerState),
			OSPowerSavingState: 0, // UNKNOWN
		}, err
	}

	return dto.PowerState{
		PowerState:         int(state[0].PowerState),
		OSPowerSavingState: int(stateOS),
	}, nil
}

func (uc *UseCase) GetPowerCapabilities(c context.Context, guid string) (dto.PowerCapabilities, error) {
	item, err := uc.repo.GetByID(c, guid, "")
	if err != nil {
		return dto.PowerCapabilities{}, err
	}

	if item == nil || item.GUID == "" {
		return dto.PowerCapabilities{}, ErrNotFound
	}

	device := uc.device.SetupWsmanClient(*item, false, true)

	version, err := device.GetAMTVersion()
	if err != nil {
		return dto.PowerCapabilities{}, err
	}

	capabilities, err := device.GetPowerCapabilities()
	if err != nil {
		return dto.PowerCapabilities{}, err
	}

	amtversion, err := parseVersion(version)
	if err != nil {
		return dto.PowerCapabilities{}, err
	}

	response := determinePowerCapabilities(amtversion, capabilities)

	return response, nil
}

func determinePowerCapabilities(amtversion int, capabilities boot.BootCapabilitiesResponse) dto.PowerCapabilities {
	response := dto.PowerCapabilities{
		PowerUp:    2,
		PowerCycle: 5,
		PowerDown:  8,
		Reset:      10,
	}

	if amtversion > MinAMTVersion {
		response.SoftOff = 12
		response.SoftReset = 14
		response.Sleep = 4
		response.Hibernate = 7
	}

	if capabilities.BIOSSetup {
		response.PowerOnToBIOS = 100
		response.ResetToBIOS = 101
	}

	if capabilities.SecureErase {
		response.ResetToSecureErase = 104
	}

	response.ResetToIDERFloppy = 200
	response.PowerOnToIDERFloppy = 201
	response.ResetToIDERCDROM = 202
	response.PowerOnToIDERCDROM = 203

	if capabilities.ForceDiagnosticBoot {
		response.PowerOnToDiagnostic = 300
		response.ResetToDiagnostic = 301
	}

	response.ResetToPXE = 400
	response.PowerOnToPXE = 401

	return response
}

func (uc *UseCase) SetBootOptions(c context.Context, guid string, bootSetting dto.BootSetting) (power.PowerActionResponse, error) {
	item, err := uc.repo.GetByID(c, guid, "")
	if err != nil {
		return power.PowerActionResponse{}, err
	}

	if item == nil || item.GUID == "" {
		return power.PowerActionResponse{}, ErrNotFound
	}

	device := uc.device.SetupWsmanClient(*item, false, true)

	bootData, err := device.GetBootData()
	if err != nil {
		return power.PowerActionResponse{}, err
	}

	newData := boot.BootSettingDataRequest{
		BIOSLastStatus:         bootData.BIOSLastStatus,
		BIOSPause:              false,
		BIOSSetup:              bootSetting.Action < 104,
		BootMediaIndex:         0,
		BootguardStatus:        bootData.BootguardStatus,
		ConfigurationDataReset: false,
		ElementName:            bootData.ElementName,
		EnforceSecureBoot:      bootData.EnforceSecureBoot,
		FirmwareVerbosity:      0,
		ForcedProgressEvents:   false,
		InstanceID:             bootData.InstanceID,
		LockKeyboard:           false,
		LockPowerButton:        false,
		LockResetButton:        false,
		LockSleepButton:        false,
		OptionsCleared:         true,
		OwningEntity:           bootData.OwningEntity,
		ReflashBIOS:            false,
		UseIDER:                bootSetting.Action > 199 && bootSetting.Action < 300,
		UseSOL:                 bootSetting.UseSOL,
		UseSafeMode:            false,
		UserPasswordBypass:     false,
		SecureErase:            false,
	}

	// boot on ider
	// boot on floppy
	err = determineBootDevice(bootSetting, &newData)
	if err != nil {
		return power.PowerActionResponse{}, err
	}

	bootSource := getBootSource(bootSetting)

	_, err = device.ChangeBootOrder("")
	if err != nil {
		return power.PowerActionResponse{}, err
	}

	_, err = device.SetBootData(newData)
	if err != nil {
		return power.PowerActionResponse{}, err
	}

	// set boot config role
	_, err = device.SetBootConfigRole(1)
	if err != nil {
		return power.PowerActionResponse{}, err
	}

	_, err = device.ChangeBootOrder(bootSource)
	if err != nil {
		return power.PowerActionResponse{}, err
	}

	// reset
	// power on
	determineBootAction(&bootSetting)

	powerActionResult, err := device.SendPowerAction(bootSetting.Action)
	if err != nil {
		return power.PowerActionResponse{}, err
	}

	return powerActionResult, nil
}

func determineBootDevice(bootSetting dto.BootSetting, newData *boot.BootSettingDataRequest) error {
	switch bootSetting.Action {
	case BootActionHTTPSBoot, BootActionPowerOnHTTPSBoot:
		typeLengthValueBuffer, params, err := validateHTTPBootParams(bootSetting.BootDetails.URL, bootSetting.BootDetails.Username, bootSetting.BootDetails.Password)
		if err != nil {
			return err
		}

		newData.BIOSLastStatus = nil
		newData.UseIDER = false
		newData.BIOSSetup = false
		newData.UseSOL = false
		newData.BootMediaIndex = 0
		newData.EnforceSecureBoot = bootSetting.BootDetails.EnforceSecureBoot
		newData.UserPasswordBypass = false
		newData.UefiBootNumberOfParams = params
		newData.UefiBootParametersArray = base64.StdEncoding.EncodeToString(typeLengthValueBuffer)
		newData.ForcedProgressEvents = true
	case BootActionResetToIDERCDROM, BootActionPowerOnIDERCDROM:
		newData.IDERBootDevice = 1
	default:
		newData.IDERBootDevice = 0
	}

	return nil
}

func validateHTTPBootParams(url, username, password string) (buffer []byte, paramCount int, err error) {
	// Example: Create TLV parameters for HTTPS boot
	parameters := []boot.TLVParameter{}

	// Create a network device path (URI to HTTPS server)
	networkPathParam, err := boot.NewStringParameter(
		boot.OCR_EFI_NETWORK_DEVICE_PATH,
		url,
	)
	if err != nil {
		return nil, 0, err
	}

	parameters = append(parameters, networkPathParam)

	// Set sync Root CA flag to true
	syncRootCAParam := boot.NewBoolParameter(
		boot.OCR_HTTPS_CERT_SYNC_ROOT_CA,
		true,
	)
	parameters = append(parameters, syncRootCAParam)

	// user name
	if username != "" {
		usernameParam, err := boot.NewStringParameter(
			boot.OCR_HTTPS_USER_NAME,
			username,
		)
		if err != nil {
			return nil, 0, err
		}

		parameters = append(parameters, usernameParam)
	}

	// password
	if password != "" {
		passwordParam, err := boot.NewStringParameter(
			boot.OCR_HTTPS_PASSWORD,
			password,
		)
		if err != nil {
			return nil, 0, err
		}

		parameters = append(parameters, passwordParam)
	}

	// Validate the parameters before creating the buffer
	valid, _ := boot.ValidateParameters(parameters)
	if !valid {
		return nil, 0, ErrValidationUseCase
	}

	// Create the TLV buffer
	tlvBuffer, err := boot.CreateTLVBuffer(parameters)
	if err != nil {
		return nil, 0, err
	}

	return tlvBuffer, len(parameters), nil
}

// "Intel(r) AMT: Force PXE Boot".
// "Intel(r) AMT: Force CD/DVD Boot".
func getBootSource(bootSetting dto.BootSetting) string {
	switch bootSetting.Action {
	case BootActionResetToPXE, BootActionPowerOnToPXE:
		return string(cimBoot.PXE)
	case BootActionResetToIDERCDROM, BootActionPowerOnIDERCDROM:
		return string(cimBoot.CD)
	case BootActionHTTPSBoot, BootActionPowerOnHTTPSBoot:
		return string(cimBoot.OCRUEFIHTTPS)
	default:
		return ""
	}
}

func determineBootAction(bootSetting *dto.BootSetting) {
	switch bootSetting.Action {
	case BootActionResetToBIOS, BootActionHTTPSBoot, BootActionResetToIDERFloppy,
		BootActionResetToIDERCDROM, BootActionResetToDiag, BootActionResetToPXE:
		bootSetting.Action = int(power.MasterBusReset)
	default:
		bootSetting.Action = int(power.PowerOn)
	}
}

func parseVersion(version []software.SoftwareIdentity) (int, error) {
	amtversion := 0

	var err error

	for _, v := range version {
		if v.InstanceID == "AMT" {
			splitversion := strings.Split(v.VersionString, ".")

			amtversion, err = strconv.Atoi(splitversion[0])
			if err != nil {
				return 0, err
			}
		}
	}

	return amtversion, nil
}
