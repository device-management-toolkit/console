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
// managed, or it is not activated or does not support Intel Standard Manageability (ISM) or Intel vPro.
var ErrManageNotEligible = errors.New("the device cannot be managed: it must be activated, support ISM or vPro, and be in the Discovered state")

// managementFor reports whether a device can be managed. The result is computed
// on read and never stored. A device is eligible only when discovery shows it
// supports ISM or vPro and is activated (admin or client control mode). A device
// that is not activated, reported no AMT data, or lacks ISM and vPro is not
// eligible: Manage could only fail for it, so the UI must not offer it. Reason
// says why, and a device that was deactivated and re-discovered is reported as
// not activated until it is activated again and synced.
func managementFor(info *dto.DeviceInfo) *dto.Management {
	m := classifyManagement(info)
	m.Eligible = m.Reason == ""

	return m
}

// classifyManagement works out why a device may not be manageable. A SKU that
// lacks ISM and vPro wins over the control mode: such a device can never be
// managed, so telling the user to activate it would send them down a dead end.
func classifyManagement(info *dto.DeviceInfo) *dto.Management {
	if info == nil {
		return &dto.Management{Reason: manageReasonUnknown}
	}

	mode := strings.ToLower(strings.TrimSpace(info.CurrentMode))

	// A host that reported neither a SKU nor a control mode has no AMT to manage (for
	// example no MEI device), so treat it like a device without ISM or vPro.
	if strings.TrimSpace(info.FWSku) == "" && mode == "" {
		return &dto.Management{Reason: manageReasonUnsupported}
	}

	sku, skuErr := strconv.Atoi(info.FWSku)
	if skuErr == nil && sku&(fwSkuISMBit|fwSkuVProBit) == 0 {
		return &dto.Management{Reason: manageReasonUnsupported}
	}

	switch {
	case mode == "not activated" || mode == "pre-provisioning state":
		return &dto.Management{Reason: manageReasonNotActivated}
	// No SKU means discovery learned nothing about AMT (no MEI access, or no AMT
	// at all), so an empty mode cannot be read as "not activated".
	case skuErr != nil:
		return &dto.Management{Reason: manageReasonUnknown}
	case mode == "":
		return &dto.Management{Reason: manageReasonNotActivated}
	case strings.Contains(mode, "admin"), strings.Contains(mode, "client"):
		return &dto.Management{}
	default:
		return &dto.Management{Reason: manageReasonUnknown}
	}
}
