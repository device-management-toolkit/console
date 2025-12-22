package devices

import (
	"context"

	dto "github.com/device-management-toolkit/console/internal/entity/dto/v1"
)

// SetLinkPreference sets the link preference (ME or Host) on a device's WiFi interface.
func (uc *UseCase) SetLinkPreference(c context.Context, guid string, req dto.LinkPreferenceRequest) (dto.LinkPreferenceResponse, error) {
	item, err := uc.repo.GetByID(c, guid, "")
	if err != nil {
		return dto.LinkPreferenceResponse{}, err
	}

	if item == nil || item.GUID == "" {
		return dto.LinkPreferenceResponse{}, ErrNotFound
	}

	// Validate link preference value
	if req.LinkPreference != dto.LinkPreferenceME && req.LinkPreference != dto.LinkPreferenceHost {
		return dto.LinkPreferenceResponse{}, ErrValidationUseCase.Wrap("SetLinkPreference", "validate link preference", "linkPreference must be 1 (ME) or 2 (Host)")
	}

	// Validate timeout
	const maxTimeout = 65535

	if req.Timeout > maxTimeout {
		return dto.LinkPreferenceResponse{}, ErrValidationUseCase.Wrap("SetLinkPreference", "validate timeout", "timeout max value is 65535")
	}

	device, _ := uc.device.SetupWsmanClient(*item, false, true)

	returnValue, err := device.SetLinkPreference(req.LinkPreference, req.Timeout)
	if err != nil {
		return dto.LinkPreferenceResponse{ReturnValue: returnValue}, err
	}

	return dto.LinkPreferenceResponse{ReturnValue: returnValue}, nil
}
