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
// result is computed on read and never stored. A SKU that lacks ISM and vPro
// wins over the control mode: such a device can never be managed, so telling the
// user to activate it would send them down a dead end.
func adoptionFor(info *dto.DeviceInfo) *dto.Adoption {
	if info == nil {
		return &dto.Adoption{Reason: adoptReasonUnknown}
	}

	sku, skuErr := strconv.Atoi(info.FWSku)
	if skuErr == nil && sku&(fwSkuISMBit|fwSkuVProBit) == 0 {
		return &dto.Adoption{Reason: adoptReasonUnsupported}
	}

	mode := controlModeOf(info.CurrentMode)
	if mode == adoptReasonNotActivated && strings.TrimSpace(info.CurrentMode) != "" {
		return &dto.Adoption{Reason: adoptReasonNotActivated}
	}

	// No SKU means discovery learned nothing about AMT (no MEI access, or no AMT
	// at all), so neither "not activated" nor "unsupported" can be claimed.
	if skuErr != nil {
		return &dto.Adoption{Reason: adoptReasonUnknown}
	}

	switch mode {
	case adoptReasonNotActivated:
		return &dto.Adoption{Reason: adoptReasonNotActivated}
	case "":
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
