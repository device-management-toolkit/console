package devices

import (
	"context"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/ips/kvmredirection"

	dto "github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/pkg/consoleerrors"
)

var ErrNotSupportedUseCase = NotSupportedError{Console: consoleerrors.CreateConsoleError("Not Supported")}

// GetKVMScreenSettings returns IPS_ScreenSettingData for the device.
func (uc *UseCase) GetKVMScreenSettings(c context.Context, guid string) (dto.KVMScreenSettings, error) {
	item, err := uc.repo.GetByID(c, guid, "")
	if err != nil {
		return dto.KVMScreenSettings{}, err
	}

	if item == nil || item.GUID == "" {
		return dto.KVMScreenSettings{}, ErrNotFound
	}

	device := uc.device.SetupWsmanClient(*item, false, true)

	resp, err := device.GetIPSScreenSettingData()
	if err != nil {
		return dto.KVMScreenSettings{}, err
	}

	redirResp, err := device.GetIPSKVMRedirectionSettingData()
	if err != nil {
		return dto.KVMScreenSettings{}, err
	}

	defaultScreen := redirResp.Body.KVMRedirectionSettingsResponse.DefaultScreen

	// Map raw ScreenSettingData response into a user-friendly displays array.
	const defaultDisplayCapacity = 4

	displays := make([]dto.KVMScreenDisplay, 0, defaultDisplayCapacity)
	items := resp.Body.PullResponse.ScreenSettingDataItems

	for i := range items { // avoid copying large struct values
		it := &items[i]
		// could be problematic if arrays are of different lengths from firmware, but as of now AMT provides 4 for each.
		count := len(it.IsActive)
		for idx := 0; idx < count; idx++ {
			isActive := it.IsActive[idx]
			// Role assignment reflects AMT firmware configuration, not current activity status
			// An inactive display can still be designated as "primary" by the firmware
			role := getRoleForIndex(idx, it.PrimaryIndex, it.SecondaryIndex, it.TertiaryIndex, it.QuadraryIndex)

			displays = append(displays, dto.KVMScreenDisplay{
				DisplayIndex: idx,
				IsActive:     isActive,
				UpperLeftX:   safeIndex(it.UpperLeftX, idx),
				UpperLeftY:   safeIndex(it.UpperLeftY, idx),
				ResolutionX:  safeIndex(it.ResolutionX, idx),
				ResolutionY:  safeIndex(it.ResolutionY, idx),
				Role:         role,
				IsDefault:    idx == int(defaultScreen),
			})
		}
	}

	return dto.KVMScreenSettings{Displays: displays}, nil
}

// SetKVMScreenSettings updates IPS_ScreenSettingData; currently not supported via wsman lib
// We accept payload but return NotSupported to preserve API contract for future.
func (uc *UseCase) SetKVMScreenSettings(c context.Context, guid string, reqData dto.KVMScreenSettingsRequest) (dto.KVMScreenSettings, error) {
	item, err := uc.repo.GetByID(c, guid, "")
	if err != nil {
		return dto.KVMScreenSettings{}, err
	}

	if item == nil || item.GUID == "" {
		return dto.KVMScreenSettings{}, ErrNotFound
	}

	device := uc.device.SetupWsmanClient(*item, false, true)

	pull, err := device.GetIPSKVMRedirectionSettingData()
	if err != nil {
		return dto.KVMScreenSettings{}, err
	}

	redirectionPull := pull.Body.PullResponse.KVMRedirectionSettingsItems

	// Validate selected display index fits into uint8 range
	if reqData.DisplayIndex < 0 || reqData.DisplayIndex > 255 {
		return dto.KVMScreenSettings{}, ErrValidationUseCase.Wrap("SetKVMScreenSettings", "validate display index", "display index out of range")
	}

	kvmRequest := &kvmredirection.KVMRedirectionSettingsRequest{
		XMLName:                        redirectionPull[0].XMLName,
		ElementName:                    redirectionPull[0].ElementName,
		InstanceID:                     redirectionPull[0].InstanceID,
		OptInPolicy:                    redirectionPull[0].OptInPolicy,
		SessionTimeout:                 redirectionPull[0].SessionTimeout,
		RFBPassword:                    redirectionPull[0].RFBPassword,
		DefaultScreen:                  uint8(reqData.DisplayIndex),
		InitialDecimationModeForLowRes: redirectionPull[0].InitialDecimationModeForLowRes,
		GreyscalePixelFormatSupported:  redirectionPull[0].GreyscalePixelFormatSupported,
		ZlibControlSupported:           redirectionPull[0].ZlibControlSupported,
		DoubleBufferMode:               redirectionPull[0].DoubleBufferMode,
		DoubleBufferState:              redirectionPull[0].DoubleBufferState,
		EnabledByMEBx:                  redirectionPull[0].EnabledByMEBx,
		Is5900PortEnabled:              redirectionPull[0].Is5900PortEnabled,
		BackToBackFbMode:               redirectionPull[0].BackToBackFbMode,
	}

	_, err = device.SetIPSKVMRedirectionSettingData(kvmRequest)
	if err != nil {
		return dto.KVMScreenSettings{}, err
	}

	// Read-only for now
	return uc.GetKVMScreenSettings(c, guid)
}

// Helper functions.
func safeIndex(a []int, i int) int {
	if i < len(a) {
		return a[i]
	}

	return 0
}

func getRoleForIndex(i, primary, secondary, tertiary, quaternary int) string {
	// AMT uses 1-based indexing for role assignments
	// Convert 0-based array index to 1-based for comparison
	displayNum := i + 1

	// Check each role assignment, treating 0 as "not assigned"
	switch {
	case primary != 0 && displayNum == primary:
		return "primary"
	case secondary != 0 && displayNum == secondary:
		return "secondary"
	case tertiary != 0 && displayNum == tertiary:
		return "tertiary"
	case quaternary != 0 && displayNum == quaternary:
		return "quaternary"
	default:
		return ""
	}
}
