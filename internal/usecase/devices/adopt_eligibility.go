package devices

import (
	"errors"
	"strconv"
	"strings"

	dto "github.com/device-management-toolkit/console/internal/entity/dto/v1"
)

const (
	adoptReasonNotActivated = "notActivated"
	adoptReasonUnsupported  = "unsupported"
	adoptReasonUnknown      = "unknown"

	fwSkuISMBit  = 0x10
	fwSkuVProBit = 0x08
)

// ErrAdoptNotEligible is returned when a device cannot be adopted: it must be
// activated and support Intel Standard Manageability (ISM) or Intel vPro.
var ErrAdoptNotEligible = errors.New("the device cannot be adopted: it must be activated and support ISM or vPro")

// adoptionFor reports whether a device can be adopted and, if not, why. The
// result is computed on read and never stored.
func adoptionFor(info *dto.DeviceInfo) *dto.Adoption {
	if info == nil {
		return &dto.Adoption{Reason: adoptReasonUnknown}
	}

	mode := controlModeOf(info.CurrentMode)
	if mode == adoptReasonNotActivated {
		return &dto.Adoption{Reason: adoptReasonNotActivated}
	}

	sku, err := strconv.Atoi(info.FWSku)
	if err != nil {
		return &dto.Adoption{Reason: adoptReasonUnknown}
	}

	if sku&(fwSkuISMBit|fwSkuVProBit) == 0 {
		return &dto.Adoption{Reason: adoptReasonUnsupported}
	}

	if mode == "" {
		return &dto.Adoption{Reason: adoptReasonUnknown}
	}

	return &dto.Adoption{Eligible: true}
}

// controlModeOf returns notActivated, or "activated" for admin/client control
// mode, or "" when the mode is not recognized.
func controlModeOf(currentMode string) string {
	normalized := strings.ToLower(strings.TrimSpace(currentMode))

	switch {
	case normalized == "" || normalized == "not activated" || normalized == "pre-provisioning state":
		return adoptReasonNotActivated
	case strings.Contains(normalized, "admin"), strings.Contains(normalized, "client"):
		return "activated"
	default:
		return ""
	}
}
