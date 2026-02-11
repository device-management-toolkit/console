// Package redfish defines constants and types for Redfish v1 entities.
package redfish

// CIM (Common Information Model) Power State constants
// These constants are from the DMTF CIM_AssociatedPowerManagementService specification
// Reference: https://www.dmtf.org/sites/default/files/standards/documents/DSP0004_3.0.1.pdf
const (
	// CIMPowerStateUnknown indicates the power state is unknown (0).
	CIMPowerStateUnknown = 0

	// CIMPowerStateOther indicates a power state not defined in the CIM spec (1).
	CIMPowerStateOther = 1

	// CIMPowerStateOn indicates the system is fully powered on (2).
	CIMPowerStateOn = 2

	// CIMPowerStateSleepLight indicates the system is in light sleep mode (3).
	CIMPowerStateSleepLight = 3

	// CIMPowerStateSleepDeep indicates the system is in deep sleep mode (4).
	CIMPowerStateSleepDeep = 4

	// CIMPowerStatePowerCycleOffSoft indicates soft power cycle off (5).
	CIMPowerStatePowerCycleOffSoft = 5

	// CIMPowerStateOffHard indicates hard power off (6).
	CIMPowerStateOffHard = 6

	// CIMPowerStateHibernate indicates the system is in hibernate state (7).
	CIMPowerStateHibernate = 7

	// CIMPowerStateOffSoft indicates soft power off (8).
	CIMPowerStateOffSoft = 8

	// CIMPowerStatePowerCycleOffHard indicates hard power cycle off (9).
	CIMPowerStatePowerCycleOffHard = 9

	// CIMPowerStateMasterBusReset indicates master bus reset (10).
	CIMPowerStateMasterBusReset = 10

	// CIMPowerStateDiagnosticInterrupt indicates diagnostic interrupt/NMI (11).
	CIMPowerStateDiagnosticInterrupt = 11

	// CIMPowerStateOffSoftGraceful indicates graceful soft power off (12).
	CIMPowerStateOffSoftGraceful = 12

	// CIMPowerStateOffHardGraceful indicates graceful hard power off (13).
	CIMPowerStateOffHardGraceful = 13

	// CIMPowerStateMasterBusResetGraceful indicates graceful master bus reset (14).
	CIMPowerStateMasterBusResetGraceful = 14

	// CIMPowerStatePowerCycleOffSoftGraceful indicates graceful soft power cycle (15).
	CIMPowerStatePowerCycleOffSoftGraceful = 15

	// CIMPowerStatePowerCycleOffHardGraceful indicates graceful hard power cycle (16).
	CIMPowerStatePowerCycleOffHardGraceful = 16
)

// CIM_PowerManagementService RequestPowerStateChange action values.
// These are the most commonly used power action constants from the CIM spec.
const (
	// CIMPowerActionOn powers on the system (same as CIMPowerStateOn).
	CIMPowerActionOn = CIMPowerStateOn

	// CIMPowerActionCycle performs a power cycle (soft).
	CIMPowerActionCycle = CIMPowerStatePowerCycleOffSoft

	// CIMPowerActionOffHard performs a hard power off.
	CIMPowerActionOffHard = CIMPowerStateOffHard

	// CIMPowerActionOffSoft performs a soft power off.
	CIMPowerActionOffSoft = CIMPowerStateOffSoft

	// CIMPowerActionReset performs a master bus reset/reboot.
	CIMPowerActionReset = CIMPowerStateMasterBusReset
)

// Redfish service retry constants.
const (
	// ServiceUnavailableRetryAfterSeconds is the default retry-after duration in seconds
	// for 503 Service Unavailable responses.
	ServiceUnavailableRetryAfterSeconds = 60
)

// IsPowerOn returns true if the power state indicates the system is powered on.
func IsPowerOn(state int) bool {
	return state == CIMPowerStateOn
}

// IsPowerOff returns true if the power state indicates the system is powered off.
func IsPowerOff(state int) bool {
	return state == CIMPowerStateOffHard || state == CIMPowerStateOffSoft ||
		state == CIMPowerStateOffSoftGraceful || state == CIMPowerStateOffHardGraceful
}

// IsSleepState returns true if the power state indicates a sleep/hibernate state.
func IsSleepState(state int) bool {
	return state == CIMPowerStateSleepLight || state == CIMPowerStateSleepDeep || state == CIMPowerStateHibernate
}
