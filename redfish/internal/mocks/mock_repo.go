// Package mocks provides mock implementations for testing.
package mocks

import (
	"context"
	"fmt"
	"strings"

	"github.com/device-management-toolkit/console/redfish/internal/controller/http/v1/generated"
	redfishv1 "github.com/device-management-toolkit/console/redfish/internal/entity/v1"
	"github.com/device-management-toolkit/console/redfish/internal/usecase"
)

// MockComputerSystemRepo implements ComputerSystemRepository with in-memory test data.
type MockComputerSystemRepo struct {
	systems  map[string]*redfishv1.ComputerSystem
	kvmState map[string]*mockKVMState
}

const (
	// Default test system memory in GiB.
	testSystemMemoryGiB = 32.0

	// Default test processor count (only value available from CIM_Processor enumeration).
	mockProcessorCount = 2

	// State and health strings.
	enabledState = "Enabled"
	okHealth     = "OK"
)

// KVM consent demo state machine.
//
// The mock provisions systems in CCM (Client Control Mode), where Intel AMT
// always requires user consent before a KVM session can start. The consent
// flow is modeled as a deterministic state machine so demo mode
// (REDFISH_USE_MOCK=true) can exercise every KVMStatus/UserConsentStatus
// combination. The optInState values mirror IPS_OptInService.OptInState
// reported by real Intel AMT hardware.
//
//	optInState        KVMStatus        UserConsentStatus
//	---------------   --------------   -----------------
//	NotStarted (0)    PendingConsent   Required
//	Requested  (1)    PendingConsent   Requested
//	Displayed  (2)    PendingConsent   Requested
//	Received   (3)    Enabled          Granted
//	InSession  (4)    Active           Granted
//	Denied     (5)    Error            Denied
//	Timeout    (6)    Error            Timeout
const (
	mockOptInNotStarted = 0
	mockOptInRequested  = 1
	mockOptInDisplayed  = 2
	mockOptInReceived   = 3
	mockOptInInSession  = 4
	mockOptInDenied     = 5
	mockOptInTimeout    = 6

	mockControlModeACM = "ACM"
	mockControlModeCCM = "CCM"

	mockKVMConnectType       = "KVMIP"
	mockKVMRedirectionPort   = 16994
	mockKVMConsentCodeLength = 6

	// KVMStatus values (mirror generated ComputerSystemOemIntelAMTKVMStatus).
	mockKVMStatusActive         = "Active"
	mockKVMStatusDisabled       = "Disabled"
	mockKVMStatusEnabled        = "Enabled"
	mockKVMStatusError          = "Error"
	mockKVMStatusPendingConsent = "PendingConsent"

	// UserConsentStatus values (mirror generated ComputerSystemOemIntelAMTUserConsentStatus).
	mockUserConsentDenied      = "Denied"
	mockUserConsentGranted     = "Granted"
	mockUserConsentNotRequired = "NotRequired"
	mockUserConsentRequested   = "Requested"
	mockUserConsentRequired    = "Required"
	mockUserConsentTimeout     = "Timeout"

	// Demo consent codes recognized by SubmitKVMConsentCode so the UI/demo can
	// deterministically reach the granted, denied, and timeout outcomes.
	mockConsentCodeGranted = "123456"
	mockConsentCodeDenied  = "000000"
	mockConsentCodeTimeout = "111111"

	mockConsentOperationSubmit    = "SubmitKVMConsentCode"
	mockConsentOperationSolSubmit = "SubmitSolConsentCode"

	// Consent ReturnValues mirror the values used by the real WSMAN repository.
	mockConsentReturnInvalidState = 2
	mockConsentReturnCodeInvalid  = 2066
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
		systems:  make(map[string]*redfishv1.ComputerSystem),
		kvmState: make(map[string]*mockKVMState),
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
			State:  enabledState,
			Health: okHealth,
		},
		MemorySummary: &redfishv1.ComputerSystemMemorySummary{
			TotalSystemMemoryGiB: float32Ptr(testSystemMemoryGiB),
			Status: &redfishv1.Status{
				State:  enabledState,
				Health: okHealth,
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
				State:        enabledState,
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
	repo.kvmState[testSystem.ID] = newMockKVMState()

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

	// Populate the GraphicalConsole from the current KVM state so that reads
	// reflect the latest consent-flow transitions in demo mode.
	systemCopy.GraphicalConsole = r.kvmStateFor(systemID).buildGraphicalConsole()

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

	if _, ok := r.kvmState[system.ID]; !ok {
		r.kvmState[system.ID] = newMockKVMState()
	}
}

// RemoveSystem removes a system from the mock repository (for testing).
func (r *MockComputerSystemRepo) RemoveSystem(systemID string) {
	delete(r.systems, systemID)
	delete(r.kvmState, systemID)
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

// UpdateGraphicalConsoleServiceEnabled updates the mock KVM service enabled state for a system.
func (r *MockComputerSystemRepo) UpdateGraphicalConsoleServiceEnabled(_ context.Context, systemID string, enabled bool) error {
	if _, exists := r.systems[systemID]; !exists {
		return usecase.ErrSystemNotFound
	}

	r.kvmStateFor(systemID).enableKVM = enabled

	return nil
}

// UpdateSerialConsoleServiceEnabled updates the mock SOL service enabled state for a system.
func (r *MockComputerSystemRepo) UpdateSerialConsoleServiceEnabled(_ context.Context, systemID string, enabled bool) error {
	system, exists := r.systems[systemID]
	if !exists {
		return usecase.ErrSystemNotFound
	}

	if system.SerialConsole == nil {
		system.SerialConsole = &redfishv1.ComputerSystemHostSerialConsole{}
	}

	if system.SerialConsole.WebSocket == nil {
		system.SerialConsole.WebSocket = &redfishv1.ComputerSystemHostWebSocketConsole{}
	}

	system.SerialConsole.WebSocket.ServiceEnabled = &enabled

	return nil
}

// RequestKVMConsent starts a mock KVM consent flow for an existing system,
// transitioning the consent state to "Requested".
func (r *MockComputerSystemRepo) RequestKVMConsent(_ context.Context, systemID string) error {
	if _, exists := r.systems[systemID]; !exists {
		return usecase.ErrSystemNotFound
	}

	state := r.kvmStateFor(systemID)
	if strings.EqualFold(strings.TrimSpace(state.controlMode), mockControlModeACM) {
		return usecase.ErrKVMConsentNotRequiredInACM
	}

	state.optInState = mockOptInRequested

	return nil
}

// SubmitKVMConsentCode accepts a mock six-digit consent code for an existing
// system and transitions the consent state based on the code submitted.
func (r *MockComputerSystemRepo) SubmitKVMConsentCode(_ context.Context, systemID, code string) error {
	if _, exists := r.systems[systemID]; !exists {
		return usecase.ErrSystemNotFound
	}

	if !isSixDigitConsentCode(code) {
		return usecase.ErrInvalidConsentCode
	}

	state := r.kvmStateFor(systemID)

	// A consent code can only be submitted while a prompt is active.
	if state.optInState != mockOptInRequested && state.optInState != mockOptInDisplayed {
		return &usecase.ConsentFailedError{Operation: mockConsentOperationSubmit, ReturnValue: mockConsentReturnInvalidState}
	}

	if code == mockConsentCodeGranted {
		state.optInState = mockOptInReceived

		return nil
	}

	// Demo trigger codes drive the denied/timeout outcomes; any other code is
	// treated as an incorrect entry and leaves the prompt active for retry.
	switch code {
	case mockConsentCodeDenied:
		state.optInState = mockOptInDenied
	case mockConsentCodeTimeout:
		state.optInState = mockOptInTimeout
	}

	return &usecase.ConsentFailedError{Operation: mockConsentOperationSubmit, ReturnValue: mockConsentReturnCodeInvalid}
}

// CancelKVMConsent cancels a mock KVM consent flow for an existing system,
// resetting the consent state back to "Required".
func (r *MockComputerSystemRepo) CancelKVMConsent(_ context.Context, systemID string) error {
	if _, exists := r.systems[systemID]; !exists {
		return usecase.ErrSystemNotFound
	}

	r.kvmStateFor(systemID).optInState = mockOptInNotStarted

	return nil
}

// mockKVMState captures the per-system Intel AMT KVM runtime state used to build
// a realistic GraphicalConsole resource in demo mode.
type mockKVMState struct {
	enableKVM    bool
	kvmAvailable bool
	controlMode  string
	optInState   int
}

// newMockKVMState returns the default KVM state for a freshly provisioned mock
// system: KVM available and enabled, CCM control mode, consent not yet started.
func newMockKVMState() *mockKVMState {
	return &mockKVMState{
		enableKVM:    true,
		kvmAvailable: true,
		controlMode:  mockControlModeCCM,
		optInState:   mockOptInNotStarted,
	}
}

// kvmStateFor returns the KVM state for systemID, lazily creating a default
// state for systems added without explicit KVM configuration.
func (r *MockComputerSystemRepo) kvmStateFor(systemID string) *mockKVMState {
	state, ok := r.kvmState[systemID]
	if !ok {
		state = newMockKVMState()
		r.kvmState[systemID] = state
	}

	return state
}

// kvmStatus derives the Intel AMT KVMStatus from the current state, mirroring
// usecase.determineKVMStatus.
func (s *mockKVMState) kvmStatus() string {
	if !s.kvmAvailable || !s.enableKVM {
		return mockKVMStatusDisabled
	}

	if strings.EqualFold(strings.TrimSpace(s.controlMode), mockControlModeACM) {
		if s.optInState == mockOptInInSession {
			return mockKVMStatusActive
		}

		return mockKVMStatusEnabled
	}

	switch s.optInState {
	case mockOptInInSession:
		return mockKVMStatusActive
	case mockOptInNotStarted, mockOptInRequested, mockOptInDisplayed:
		return mockKVMStatusPendingConsent
	case mockOptInReceived:
		return mockKVMStatusEnabled
	default:
		return mockKVMStatusError
	}
}

// userConsentStatus derives the Intel AMT UserConsentStatus from the current
// state, mirroring usecase.determineKVMUserConsentStatus.
func (s *mockKVMState) userConsentStatus() string {
	if strings.EqualFold(strings.TrimSpace(s.controlMode), mockControlModeACM) {
		return mockUserConsentNotRequired
	}

	switch s.optInState {
	case mockOptInNotStarted:
		return mockUserConsentRequired
	case mockOptInRequested, mockOptInDisplayed:
		return mockUserConsentRequested
	case mockOptInReceived, mockOptInInSession:
		return mockUserConsentGranted
	case mockOptInDenied:
		return mockUserConsentDenied
	case mockOptInTimeout:
		return mockUserConsentTimeout
	default:
		return mockUserConsentRequested
	}
}

// buildGraphicalConsole constructs a GraphicalConsole resource reflecting the
// current KVM state, mirroring the real WSMAN repository output.
func (s *mockKVMState) buildGraphicalConsole() *redfishv1.ComputerSystemHostGraphicalConsole {
	serviceEnabled := s.enableKVM

	var connectTypes []string

	var port *int64

	if s.kvmAvailable {
		connectTypes = []string{mockKVMConnectType}
		redirectionPort := int64(mockKVMRedirectionPort)
		port = &redirectionPort
	}

	return &redfishv1.ComputerSystemHostGraphicalConsole{
		ConnectTypesSupported: connectTypes,
		Port:                  port,
		ServiceEnabled:        &serviceEnabled,
		OEM: &redfishv1.ComputerSystemHostGraphicalConsoleOEM{
			Intel: &redfishv1.ComputerSystemHostGraphicalConsoleIntel{
				AMT: &redfishv1.ComputerSystemHostGraphicalConsoleAMT{
					ControlMode:       s.controlMode,
					KVMStatus:         s.kvmStatus(),
					UserConsentStatus: s.userConsentStatus(),
				},
			},
		},
	}
}

// isSixDigitConsentCode reports whether code is exactly six numeric digits,
// mirroring the validation performed by the real WSMAN repository.
func isSixDigitConsentCode(code string) bool {
	if len(code) != mockKVMConsentCodeLength {
		return false
	}

	for _, c := range code {
		if c < '0' || c > '9' {
			return false
		}
	}

	return true
}

// RequestSolConsent starts a mock SOL consent flow for an existing system,
// transitioning the consent state to "Requested".
func (r *MockComputerSystemRepo) RequestSolConsent(_ context.Context, systemID string) error {
	if _, exists := r.systems[systemID]; !exists {
		return usecase.ErrSystemNotFound
	}

	state := r.kvmStateFor(systemID)
	if strings.EqualFold(strings.TrimSpace(state.controlMode), mockControlModeACM) {
		return usecase.ErrSOLConsentNotRequiredInACM
	}

	state.optInState = mockOptInRequested

	return nil
}

// SubmitSolConsentCode accepts a mock six-digit consent code for SOL for an existing
// system and transitions the consent state based on the code submitted.
func (r *MockComputerSystemRepo) SubmitSolConsentCode(_ context.Context, systemID, code string) error {
	if _, exists := r.systems[systemID]; !exists {
		return usecase.ErrSystemNotFound
	}

	if !isSixDigitConsentCode(code) {
		return usecase.ErrInvalidConsentCode
	}

	state := r.kvmStateFor(systemID)

	// A consent code can only be submitted while a prompt is active.
	if state.optInState != mockOptInRequested && state.optInState != mockOptInDisplayed {
		return &usecase.ConsentFailedError{Operation: mockConsentOperationSolSubmit, ReturnValue: mockConsentReturnInvalidState}
	}

	if code == mockConsentCodeGranted {
		state.optInState = mockOptInReceived

		return nil
	}

	// Demo trigger codes drive the denied/timeout outcomes; any other code is
	// treated as a security failure.
	if code == mockConsentCodeDenied {
		state.optInState = mockOptInDenied

		return nil
	}

	if code == mockConsentCodeTimeout {
		state.optInState = mockOptInTimeout

		return nil
	}

	return &usecase.ConsentFailedError{Operation: mockConsentOperationSolSubmit, ReturnValue: mockConsentReturnCodeInvalid}
}

// CancelSolConsent cancels a mock SOL consent flow for an existing system,
// resetting the opt-in state back to "NotStarted".
func (r *MockComputerSystemRepo) CancelSolConsent(_ context.Context, systemID string) error {
	if _, exists := r.systems[systemID]; !exists {
		return usecase.ErrSystemNotFound
	}

	r.kvmStateFor(systemID).optInState = mockOptInNotStarted

	return nil
}
