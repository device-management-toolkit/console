// Package usecase provides a mock implementation of ComputerSystemRepository for testing.
package usecase

import (
	"context"
	"fmt"

	redfishv1 "github.com/device-management-toolkit/console/redfish/internal/entity/v1"
)

// MockComputerSystemRepo implements ComputerSystemRepository with in-memory test data.
type MockComputerSystemRepo struct {
	systems map[string]*redfishv1.ComputerSystem
}

// NewMockComputerSystemRepo creates a new mock repository with sample test data.
func NewMockComputerSystemRepo() *MockComputerSystemRepo {
	repo := &MockComputerSystemRepo{
		systems: make(map[string]*redfishv1.ComputerSystem),
	}

	// Add default test system
	testSystem := &redfishv1.ComputerSystem{
		ID:           "test-system-1",
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
		ODataID:   "/redfish/v1/Systems/test-system-1",
		ODataType: "#ComputerSystem.v1_22_0.ComputerSystem",
	}

	repo.systems["test-system-1"] = testSystem

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
		return nil, ErrSystemNotFound
	}

	// Return a copy to prevent external modifications
	systemCopy := *system

	return &systemCopy, nil
}

// UpdatePowerState updates the power state of a system.
func (r *MockComputerSystemRepo) UpdatePowerState(_ context.Context, systemID string, state redfishv1.PowerState) error {
	system, exists := r.systems[systemID]
	if !exists {
		return ErrSystemNotFound
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
			return fmt.Errorf("%w: system is already on", ErrPowerStateConflict)
		}
	case redfishv1.ResetTypeForceOff:
		if current == redfishv1.PowerStateOff {
			return fmt.Errorf("%w: system is already off", ErrPowerStateConflict)
		}
	case redfishv1.PowerStateOff:
		if current == redfishv1.PowerStateOff {
			return fmt.Errorf("%w: system is already off", ErrPowerStateConflict)
		}
	case redfishv1.ResetTypeForceRestart, redfishv1.ResetTypePowerCycle:
		// These can always be performed
	default:
		return fmt.Errorf("%w: %s", ErrInvalidPowerState, target)
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
