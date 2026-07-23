package mcp

import (
	"sort"

	dto "github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/usecase/devices"
)

// Power action names used by MCP clients. They are stable identifiers that map
// to the integer codes devices.Feature.SendPowerAction expects.
const (
	actionPowerOn         = "power_on"
	actionSleep           = "sleep"
	actionPowerCycle      = "power_cycle"
	actionHibernate       = "hibernate"
	actionPowerOff        = "power_off"
	actionPowerOffSoft    = "power_off_soft"
	actionReset           = "reset"
	actionSoftOff         = "soft_off"
	actionSoftReset       = "soft_reset"
	actionOSToFullPower   = "os_to_full_power"
	actionOSToPowerSaving = "os_to_power_saving"
)

// powerActionNames maps human-friendly action names (used by MCP clients) to
// the integer action codes that devices.Feature.SendPowerAction expects. The
// codes mirror the CIM RequestPowerStateChange values and the OS power-saving
// transitions the use case handles, so behavior is identical to the REST API.
//
// Boot-target actions (PXE, BIOS, diagnostics, IDE-R, HTTPS boot) are not
// listed here: they require the boot-configuration flow and are exposed
// through the REST power/bootOptions endpoint instead.
var powerActionNames = map[string]int{
	actionPowerOn:         2,  // CIM: verified hardware power on
	actionSleep:           4,  // CIM: sleep deep
	actionPowerCycle:      5,  // CIM: power cycle (off then on)
	actionHibernate:       7,  // CIM: hibernate
	actionPowerOff:        8,  // CIM: verified hardware power off (hard)
	actionPowerOffSoft:    9,  // CIM: soft power off
	actionReset:           10, // CIM: master bus reset (reboot)
	actionSoftOff:         12, // CIM: soft off graceful
	actionSoftReset:       14, // CIM: master bus reset graceful
	actionOSToFullPower:   devices.OsToFullPower,
	actionOSToPowerSaving: devices.OsToPowerSaving,
}

// powerActionCode resolves a named action to its integer code. The second
// return value is false when the name is unknown.
func powerActionCode(name string) (int, bool) {
	code, ok := powerActionNames[name]

	return code, ok
}

// sortedPowerActionNames returns every supported action name in a stable order,
// for use in tool descriptions and validation error messages.
func sortedPowerActionNames() []string {
	names := make([]string, 0, len(powerActionNames))
	for name := range powerActionNames {
		names = append(names, name)
	}

	sort.Strings(names)

	return names
}

// capabilityActionNames translates a dto.PowerCapabilities response into the
// set of power_action names the device reports as supported, so an MCP client
// can discover valid actions before calling power_action. A capability is
// considered supported when its code is non-zero and maps to a known action.
func capabilityActionNames(caps dto.PowerCapabilities) []string {
	codeToName := make(map[int]string, len(powerActionNames))
	for name, code := range powerActionNames {
		if _, exists := codeToName[code]; !exists {
			codeToName[code] = name
		}
	}

	capCodes := []int{
		caps.PowerUp, caps.PowerCycle, caps.PowerDown, caps.Reset,
		caps.SoftOff, caps.SoftReset, caps.Sleep, caps.Hibernate,
	}

	seen := make(map[string]bool)
	names := make([]string, 0, len(capCodes))

	for _, code := range capCodes {
		if code == 0 {
			continue
		}

		if name, ok := codeToName[code]; ok && !seen[name] {
			seen[name] = true

			names = append(names, name)
		}
	}

	sort.Strings(names)

	return names
}
