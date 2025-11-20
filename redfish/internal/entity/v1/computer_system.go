// Package redfish provides entity definitions for Redfish computer systems.
package redfish

// ComputerSystem represents a Redfish Computer System entity.
type ComputerSystem struct {
	ID           string     `json:"Id"`
	Name         string     `json:"Name"`
	SystemType   SystemType `json:"SystemType"`
	Manufacturer string     `json:"Manufacturer"`
	Model        string     `json:"Model"`
	SerialNumber string     `json:"SerialNumber"`
	PowerState   PowerState `json:"PowerState"`
	Status       *Status    `json:"Status,omitempty"`
	ODataID      string     `json:"@odata.id"`
	ODataType    string     `json:"@odata.type"`
}

// Status represents the status and health of a resource.
type Status struct {
	State  string `json:"State,omitempty"`
	Health string `json:"Health,omitempty"`
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
