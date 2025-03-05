package devices

import (
	"context"
	"strconv"
	"strings"

	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/amt/boot"
	cimBoot "github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/cim/boot"
	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/cim/power"
	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/cim/software"

	"github.com/open-amt-cloud-toolkit/console/internal/entity/dto/v1"
)

func (uc *UseCase) SendPowerAction(c context.Context, guid string, action int) (power.PowerActionResponse, error) {
	item, err := uc.repo.GetByID(c, guid, "")
	if err != nil {
		return power.PowerActionResponse{}, err
	}

	if item == nil || item.GUID == "" {
		return power.PowerActionResponse{}, ErrNotFound
	}

	device := uc.device.SetupWsmanClient(c, *item, false, true)

	response, err := device.SendPowerAction(action)
	if err != nil {
		return power.PowerActionResponse{}, err
	}

	return response, nil
}

func (uc *UseCase) GetPowerState(c context.Context, guid string) (dto.PowerState, error) {
	item, err := uc.repo.GetByID(c, guid, "")
	if err != nil {
		return dto.PowerState{}, err
	}

	if item == nil || item.GUID == "" {
		return dto.PowerState{}, ErrNotFound
	}

	device := uc.device.SetupWsmanClient(c, *item, false, true)

	state, err := device.GetPowerState()
	if err != nil {
		return dto.PowerState{}, err
	}

	return dto.PowerState{
		PowerState: int(state[0].PowerState),
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

	device := uc.device.SetupWsmanClient(c, *item, false, true)

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

	device := uc.device.SetupWsmanClient(c, *item, false, true)

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
	determineIDERBootDevice(bootSetting, &newData)
	// force boot mode
	_, err = device.SetBootConfigRole(1)
	if err != nil {
		return power.PowerActionResponse{}, err
	}

	bootSource := getBootSource(bootSetting)
	if bootSource != "" {
		_, err = device.ChangeBootOrder(bootSource)
		if err != nil {
			return power.PowerActionResponse{}, err
		}
	}

	_, err = device.SetBootData(newData)
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

func determineIDERBootDevice(bootSetting dto.BootSetting, newData *boot.BootSettingDataRequest) {
	if bootSetting.Action == 202 || bootSetting.Action == 203 {
		newData.IDERBootDevice = 1
	} else {
		newData.IDERBootDevice = 0
	}
}

// "Intel(r) AMT: Force PXE Boot".
// "Intel(r) AMT: Force CD/DVD Boot".
func getBootSource(bootSetting dto.BootSetting) string {
	if bootSetting.Action == 400 || bootSetting.Action == 401 {
		return string(cimBoot.PXE)
	} else if bootSetting.Action == 202 || bootSetting.Action == 203 {
		return string(cimBoot.CD)
	}

	return ""
}

func determineBootAction(bootSetting *dto.BootSetting) {
	if bootSetting.Action == 101 || bootSetting.Action == 200 || bootSetting.Action == 202 || bootSetting.Action == 301 || bootSetting.Action == 400 {
		bootSetting.Action = int(power.MasterBusReset)
	} else {
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
