// Package mocks provides mock implementations for testing.
package mocks

import (
	"context"
	"fmt"

	"github.com/device-management-toolkit/console/redfish/internal/controller/http/v1/generated"
	redfishv1 "github.com/device-management-toolkit/console/redfish/internal/entity/v1"
	"github.com/device-management-toolkit/console/redfish/internal/usecase"
)

// MockComputerSystemRepo implements ComputerSystemRepository with in-memory test data.
type MockComputerSystemRepo struct {
	systems map[string]*redfishv1.ComputerSystem
}

const (
	// Default test system memory in GiB.
	testSystemMemoryGiB = 32.0

	// Default test processor count (only value available from CIM_Processor enumeration).
	mockProcessorCount = 2
)

// float32Ptr creates a pointer to a float32 value.
func float32Ptr(f float32) *float32 {
	return &f
}

// intPtr creates a pointer to an int value.
func intPtr(i int) *int {
	return &i
}

// stringPtr creates a pointer to a string value.
func stringPtr(s string) *string {
	return &s
}

// NewMockComputerSystemRepo creates a new mock repository with sample test data.
func NewMockComputerSystemRepo() *MockComputerSystemRepo {
	repo := &MockComputerSystemRepo{
		systems: make(map[string]*redfishv1.ComputerSystem),
	}

	// Add default test system
	testSystem := &redfishv1.ComputerSystem{
		ID:           "550e8400-e29b-41d4-a716-446655440001",
		Name:         "Test System 1",
		SystemType:   redfishv1.SystemTypePhysical,
		Manufacturer: "Intel Corporation",
		Model:        "vPro Test System",
		SerialNumber: "TEST-SN-001",
		PowerState:   redfishv1.PowerStateOn,
		Status: &redfishv1.Status{
			State:  "Enabled",
			Health: "OK",
		},
		MemorySummary: &redfishv1.ComputerSystemMemorySummary{
			TotalSystemMemoryGiB: float32Ptr(testSystemMemoryGiB),
			Status: &redfishv1.Status{
				State:  "Enabled",
				Health: "OK",
			},
		},
		ProcessorSummary: &redfishv1.ComputerSystemProcessorSummary{
			Count: intPtr(mockProcessorCount),
			// CoreCount, LogicalProcessorCount, Model, and ThreadingEnabled are nil
			// because CIM_Processor doesn't provide these in Intel AMT WSMAN implementation
			CoreCount:             nil,
			LogicalProcessorCount: nil,
			Model:                 nil,
			Status: &redfishv1.Status{
				State:        "Enabled",
				Health:       "OK",
				HealthRollup: "OK",
			},
			StatusRedfishDeprecated: stringPtr("Please migrate to use Status in the individual Processor resources"),
			ThreadingEnabled:        nil,
		},
		ODataID:   "/redfish/v1/Systems/550e8400-e29b-41d4-a716-446655440001",
		ODataType: "#ComputerSystem.v1_22_0.ComputerSystem",
	}

	repo.systems["550e8400-e29b-41d4-a716-446655440001"] = testSystem

	return repo
}

// GetAll retrieves all computer system IDs.
func (r *MockComputerSystemRepo) GetAll(_ context.Context) ([]string, error) {
	systemIDs := make([]string, 0, len(r.systems))
	for id := range r.systems {
		systemIDs = append(systemIDs, id)
	}

	return systemIDs, nil
}

// GetByID retrieves a computer system by its ID.
func (r *MockComputerSystemRepo) GetByID(_ context.Context, systemID string) (*redfishv1.ComputerSystem, error) {
	system, exists := r.systems[systemID]
	if !exists {
		return nil, usecase.ErrSystemNotFound
	}

	// Return a copy to prevent external modifications
	systemCopy := *system

	return &systemCopy, nil
}

// UpdatePowerState updates the power state of a system.
func (r *MockComputerSystemRepo) UpdatePowerState(_ context.Context, systemID string, state redfishv1.PowerState) error {
	system, exists := r.systems[systemID]
	if !exists {
		return usecase.ErrSystemNotFound
	}

	// Validate the state transition
	if err := r.validatePowerStateTransition(system.PowerState, state); err != nil {
		return err
	}

	// Update the power state
	system.PowerState = state

	return nil
}

// validatePowerStateTransition checks if a power state transition is valid.
func (r *MockComputerSystemRepo) validatePowerStateTransition(current, target redfishv1.PowerState) error {
	// For mock purposes, allow most transitions
	// In production, you'd enforce actual constraints
	switch target {
	case redfishv1.ResetTypeOn:
		if current == redfishv1.PowerStateOn {
			return fmt.Errorf("%w: system is already on", usecase.ErrPowerStateConflict)
		}
	case redfishv1.ResetTypeForceOff:
		if current == redfishv1.PowerStateOff {
			return fmt.Errorf("%w: system is already off", usecase.ErrPowerStateConflict)
		}
	case redfishv1.PowerStateOff:
		if current == redfishv1.PowerStateOff {
			return fmt.Errorf("%w: system is already off", usecase.ErrPowerStateConflict)
		}
	case redfishv1.ResetTypeForceRestart, redfishv1.ResetTypePowerCycle:
		// These can always be performed
	default:
		return fmt.Errorf("%w: %s", usecase.ErrInvalidPowerState, target)
	}

	return nil
}

// AddSystem adds a new system to the mock repository (for testing).
func (r *MockComputerSystemRepo) AddSystem(system *redfishv1.ComputerSystem) {
	r.systems[system.ID] = system
}

// RemoveSystem removes a system from the mock repository (for testing).
func (r *MockComputerSystemRepo) RemoveSystem(systemID string) {
	delete(r.systems, systemID)
}

// GetBootSettings retrieves the current boot configuration for a system (mock implementation).
func (r *MockComputerSystemRepo) GetBootSettings(_ context.Context, systemID string) (*generated.ComputerSystemBoot, error) {
	_, exists := r.systems[systemID]
	if !exists {
		return nil, usecase.ErrSystemNotFound
	}

	// Return mock boot settings - defaults to disabled override
	boot := &generated.ComputerSystemBoot{}

	enabled := generated.ComputerSystemBoot_BootSourceOverrideEnabled{}
	_ = enabled.FromComputerSystemBootSourceOverrideEnabled(generated.ComputerSystemBootSourceOverrideEnabledDisabled)
	boot.BootSourceOverrideEnabled = &enabled

	target := generated.ComputerSystemBoot_BootSourceOverrideTarget{}
	_ = target.FromComputerSystemBootSource(generated.ComputerSystemBootSourceNone)
	boot.BootSourceOverrideTarget = &target

	mode := generated.ComputerSystemBoot_BootSourceOverrideMode{}
	_ = mode.FromComputerSystemBootSourceOverrideMode(generated.UEFI)
	boot.BootSourceOverrideMode = &mode

	return boot, nil
}

// UpdateBootSettings updates the boot configuration for a system (mock implementation).
func (r *MockComputerSystemRepo) UpdateBootSettings(_ context.Context, systemID string, boot *generated.ComputerSystemBoot) error {
	system, exists := r.systems[systemID]
	if !exists {
		return usecase.ErrSystemNotFound
	}

	// For mock purposes, just log that boot settings were updated
	// In a real implementation, this would update the system's boot configuration
	_ = system
	_ = boot

	// Mock implementation accepts any valid boot settings
	return nil
}
