package devices

import (
	"errors"
	"strconv"
	"strings"

	dto "github.com/device-management-toolkit/console/internal/entity/dto/v1"
)

const (
	manageReasonNotActivated = "notActivated"
	manageReasonUnsupported  = "unsupported"
	manageReasonUnknown      = "unknown"

	fwSkuISMBit  = 0x10
	fwSkuVProBit = 0x08
)

// ErrManageNotEligible is returned when a device cannot be managed: it is already
// managed, or it does not support Intel Standard Manageability (ISM) or Intel vPro.
var ErrManageNotEligible = errors.New("the device cannot be managed: it must support ISM or vPro and be in the Discovered state")

// managementFor reports whether a device can be managed. The result is computed
// on read and never stored. Only a device that definitely lacks ISM and vPro
// support is ineligible, since it can never succeed. Discovery data can be stale,
// so a device that appears not activated or reported nothing stays eligible: Manage
// verifies the AMT credentials against the device, which is the authoritative
// check. Reason is still set for those devices so the UI can show a soft hint.
func managementFor(info *dto.DeviceInfo) *dto.Management {
	m := classifyManagement(info)
	m.Eligible = m.Reason != manageReasonUnsupported

	return m
}

// classifyManagement works out why a device may not be manageable. A SKU that
// lacks ISM and vPro wins over the control mode: such a device can never be
// managed, so telling the user to activate it would send them down a dead end.
func classifyManagement(info *dto.DeviceInfo) *dto.Management {
	if info == nil {
		return &dto.Management{Reason: manageReasonUnknown}
	}

	sku, skuErr := strconv.Atoi(info.FWSku)
	if skuErr == nil && sku&(fwSkuISMBit|fwSkuVProBit) == 0 {
		return &dto.Management{Reason: manageReasonUnsupported}
	}

	mode := controlModeOf(info.CurrentMode)
	if mode == manageReasonNotActivated && strings.TrimSpace(info.CurrentMode) != "" {
		return &dto.Management{Reason: manageReasonNotActivated}
	}

	// No SKU means discovery learned nothing about AMT (no MEI access, or no AMT
	// at all), so neither "not activated" nor "unsupported" can be claimed.
	if skuErr != nil {
		return &dto.Management{Reason: manageReasonUnknown}
	}

	switch mode {
	case manageReasonNotActivated:
		return &dto.Management{Reason: manageReasonNotActivated}
	case "":
		return &dto.Management{Reason: manageReasonUnknown}
	}

	return &dto.Management{}
}

// controlModeOf returns notActivated, or "activated" for admin/client control
// mode, or "" when the mode is not recognized.
func controlModeOf(currentMode string) string {
	normalized := strings.ToLower(strings.TrimSpace(currentMode))

	switch {
	case normalized == "" || normalized == "not activated" || normalized == "pre-provisioning state":
		return manageReasonNotActivated
	case strings.Contains(normalized, "admin"), strings.Contains(normalized, "client"):
		return "activated"
	default:
		return ""
	}
}
