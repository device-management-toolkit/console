// Package redfish provides entity definitions for Redfish computer systems.
package redfish

// ComputerSystem represents a Redfish Computer System entity.
type ComputerSystem struct {
	ID               string                          `json:"Id"`
	Name             string                          `json:"Name"`
	Description      string                          `json:"Description,omitempty"`
	BiosVersion      string                          `json:"BiosVersion,omitempty"`
	HostName         string                          `json:"HostName,omitempty"`
	SystemType       SystemType                      `json:"SystemType"`
	Manufacturer     string                          `json:"Manufacturer"`
	Model            string                          `json:"Model"`
	SerialNumber     string                          `json:"SerialNumber"`
	PowerState       PowerState                      `json:"PowerState"`
	Status           *Status                         `json:"Status,omitempty"`
	MemorySummary    *ComputerSystemMemorySummary    `json:"MemorySummary,omitempty"`
	ProcessorSummary *ComputerSystemProcessorSummary `json:"ProcessorSummary,omitempty"`
	ODataID          string                          `json:"@odata.id"`
	ODataType        string                          `json:"@odata.type"`
}

// Status represents the status and health of a resource.
type Status struct {
	State        string `json:"State,omitempty"`
	Health       string `json:"Health,omitempty"`
	HealthRollup string `json:"HealthRollup,omitempty"`
}

// SystemType represents the type of computer system.
type SystemType string

const (
	// SystemTypePhysical indicates a physical computer system.
	SystemTypePhysical SystemType = "Physical"
	// SystemTypeVirtual indicates a virtual computer system.
	SystemTypeVirtual SystemType = "Virtual"
)

// PowerState represents the power state of a computer system.
type PowerState string

const (
	// PowerStateOn indicates that the system is powered on.
	PowerStateOn PowerState = "On"
	// PowerStateOff indicates that the system is powered off.
	PowerStateOff PowerState = "Off"
	// ResetTypeOn indicates a power on reset.
	ResetTypeOn PowerState = "On"
	// ResetTypeForceOff indicates a forced power off.
	ResetTypeForceOff PowerState = "ForceOff"
	// ResetTypeForceRestart indicates a forced restart.
	ResetTypeForceRestart PowerState = "ForceRestart"
	// ResetTypePowerCycle indicates a power cycle.
	ResetTypePowerCycle PowerState = "PowerCycle"
)

// MemoryMirroring represents the type of memory mirroring supported by the system.
type MemoryMirroring string

const (
	// MemoryMirroringSystem indicates system-level DIMM mirroring support.
	MemoryMirroringSystem MemoryMirroring = "System"
	// MemoryMirroringDIMM indicates DIMM-level mirroring support.
	MemoryMirroringDIMM MemoryMirroring = "DIMM"
	// MemoryMirroringHybrid indicates hybrid system and DIMM-level mirroring support.
	MemoryMirroringHybrid MemoryMirroring = "Hybrid"
	// MemoryMirroringNone indicates no DIMM mirroring support.
	MemoryMirroringNone MemoryMirroring = "None"
)

// ComputerSystemMemorySummary represents the memory summary of a computer system.
type ComputerSystemMemorySummary struct {
	TotalSystemMemoryGiB *float32        `json:"TotalSystemMemoryGiB"`
	Status               *Status         `json:"Status,omitempty"`
	MemoryMirroring      MemoryMirroring `json:"MemoryMirroring,omitempty"`
}

// ComputerSystemProcessorSummary represents the processor summary of a computer system.
type ComputerSystemProcessorSummary struct {
	Count                   *int    `json:"Count,omitempty"`
	CoreCount               *int    `json:"CoreCount,omitempty"`
	LogicalProcessorCount   *int    `json:"LogicalProcessorCount,omitempty"`
	Metrics                 *string `json:"Metrics,omitempty"`
	Model                   *string `json:"Model,omitempty"`
	Status                  *Status `json:"Status,omitempty"`
	StatusRedfishDeprecated *string `json:"Status@Redfish.Deprecated,omitempty"`
	ThreadingEnabled        *bool   `json:"ThreadingEnabled,omitempty"`
}
