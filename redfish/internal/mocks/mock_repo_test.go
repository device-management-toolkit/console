package mocks

import (
	"context"
	"errors"
	"testing"

	redfishv1 "github.com/device-management-toolkit/console/redfish/internal/entity/v1"
	"github.com/device-management-toolkit/console/redfish/internal/usecase"
)

const (
	testMockSystemID      = "550e8400-e29b-41d4-a716-446655440001"
	testMockUnknownSystem = "11111111-1111-1111-1111-111111111111"
)

// amtStatusFor reads the default mock system through the public repository API
// and returns its Intel AMT graphical-console status, failing the test if it is
// not populated.
func amtStatusFor(t *testing.T, repo *MockComputerSystemRepo) *redfishv1.ComputerSystemHostGraphicalConsoleAMT {
	t.Helper()

	system, err := repo.GetByID(context.Background(), testMockSystemID)
	if err != nil {
		t.Fatalf("GetByID(%q) returned error: %v", testMockSystemID, err)
	}

	if system.GraphicalConsole == nil || system.GraphicalConsole.OEM == nil ||
		system.GraphicalConsole.OEM.Intel == nil || system.GraphicalConsole.OEM.Intel.AMT == nil {
		t.Fatalf("GetByID(%q) did not populate GraphicalConsole AMT status", testMockSystemID)
	}

	return system.GraphicalConsole.OEM.Intel.AMT
}

func TestNewMockComputerSystemRepoInitialKVMState(t *testing.T) {
	t.Parallel()

	repo := NewMockComputerSystemRepo()

	system, err := repo.GetByID(context.Background(), testMockSystemID)
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}

	console := system.GraphicalConsole
	if console == nil {
		t.Fatal("expected GraphicalConsole to be populated")
	}

	if console.ServiceEnabled == nil || !*console.ServiceEnabled {
		t.Fatalf("expected ServiceEnabled=true, got %#v", console.ServiceEnabled)
	}

	if console.Port == nil || *console.Port != mockKVMRedirectionPort {
		t.Fatalf("expected Port=%d, got %#v", mockKVMRedirectionPort, console.Port)
	}

	if len(console.ConnectTypesSupported) != 1 || console.ConnectTypesSupported[0] != mockKVMConnectType {
		t.Fatalf("expected ConnectTypesSupported=[%q], got %#v", mockKVMConnectType, console.ConnectTypesSupported)
	}

	amt := console.OEM.Intel.AMT
	if amt.ControlMode != mockControlModeCCM {
		t.Fatalf("expected ControlMode=%q, got %q", mockControlModeCCM, amt.ControlMode)
	}

	if amt.KVMStatus != mockKVMStatusPendingConsent {
		t.Fatalf("expected initial KVMStatus=%q, got %q", mockKVMStatusPendingConsent, amt.KVMStatus)
	}

	if amt.UserConsentStatus != mockUserConsentRequired {
		t.Fatalf("expected initial UserConsentStatus=%q, got %q", mockUserConsentRequired, amt.UserConsentStatus)
	}
}

func TestMockKVMConsentLifecycleGranted(t *testing.T) {
	t.Parallel()

	repo := NewMockComputerSystemRepo()
	ctx := context.Background()

	// Request consent -> Requested / PendingConsent.
	if err := repo.RequestKVMConsent(ctx, testMockSystemID); err != nil {
		t.Fatalf("RequestKVMConsent returned error: %v", err)
	}

	amt := amtStatusFor(t, repo)
	if amt.UserConsentStatus != mockUserConsentRequested || amt.KVMStatus != mockKVMStatusPendingConsent {
		t.Fatalf("after request: got (KVMStatus=%q, UserConsentStatus=%q)", amt.KVMStatus, amt.UserConsentStatus)
	}

	// Submit the correct demo code -> Granted / Enabled.
	if err := repo.SubmitKVMConsentCode(ctx, testMockSystemID, mockConsentCodeGranted); err != nil {
		t.Fatalf("SubmitKVMConsentCode returned error: %v", err)
	}

	amt = amtStatusFor(t, repo)
	if amt.UserConsentStatus != mockUserConsentGranted || amt.KVMStatus != mockKVMStatusEnabled {
		t.Fatalf("after grant: got (KVMStatus=%q, UserConsentStatus=%q)", amt.KVMStatus, amt.UserConsentStatus)
	}

	// Cancel consent -> Required / PendingConsent.
	if err := repo.CancelKVMConsent(ctx, testMockSystemID); err != nil {
		t.Fatalf("CancelKVMConsent returned error: %v", err)
	}

	amt = amtStatusFor(t, repo)
	if amt.UserConsentStatus != mockUserConsentRequired || amt.KVMStatus != mockKVMStatusPendingConsent {
		t.Fatalf("after cancel: got (KVMStatus=%q, UserConsentStatus=%q)", amt.KVMStatus, amt.UserConsentStatus)
	}
}

func TestMockSubmitKVMConsentCodeOutcomes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		code            string
		wantReturnValue int
		wantKVMStatus   string
		wantUserConsent string
	}{
		{
			name:            "denied demo code",
			code:            mockConsentCodeDenied,
			wantReturnValue: mockConsentReturnCodeInvalid,
			wantKVMStatus:   mockKVMStatusError,
			wantUserConsent: mockUserConsentDenied,
		},
		{
			name:            "timeout demo code",
			code:            mockConsentCodeTimeout,
			wantReturnValue: mockConsentReturnCodeInvalid,
			wantKVMStatus:   mockKVMStatusError,
			wantUserConsent: mockUserConsentTimeout,
		},
		{
			name:            "incorrect code keeps prompt active",
			code:            "654321",
			wantReturnValue: mockConsentReturnCodeInvalid,
			wantKVMStatus:   mockKVMStatusPendingConsent,
			wantUserConsent: mockUserConsentRequested,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := NewMockComputerSystemRepo()
			ctx := context.Background()

			if err := repo.RequestKVMConsent(ctx, testMockSystemID); err != nil {
				t.Fatalf("RequestKVMConsent returned error: %v", err)
			}

			err := repo.SubmitKVMConsentCode(ctx, testMockSystemID, tt.code)

			var consentErr *usecase.ConsentFailedError
			if !errors.As(err, &consentErr) {
				t.Fatalf("expected *usecase.ConsentFailedError, got %v", err)
			}

			if consentErr.ReturnValue != tt.wantReturnValue {
				t.Fatalf("expected ReturnValue=%d, got %d", tt.wantReturnValue, consentErr.ReturnValue)
			}

			amt := amtStatusFor(t, repo)
			if amt.KVMStatus != tt.wantKVMStatus || amt.UserConsentStatus != tt.wantUserConsent {
				t.Fatalf("got (KVMStatus=%q, UserConsentStatus=%q), want (%q, %q)",
					amt.KVMStatus, amt.UserConsentStatus, tt.wantKVMStatus, tt.wantUserConsent)
			}
		})
	}
}

func TestMockSubmitKVMConsentCodeInvalidFormat(t *testing.T) {
	t.Parallel()

	invalidCodes := []string{"", "12345", "1234567", "abcdef", "12 456", "１２３４５６"}

	for _, code := range invalidCodes {
		code := code

		t.Run("code="+code, func(t *testing.T) {
			t.Parallel()

			repo := NewMockComputerSystemRepo()
			ctx := context.Background()

			if err := repo.RequestKVMConsent(ctx, testMockSystemID); err != nil {
				t.Fatalf("RequestKVMConsent returned error: %v", err)
			}

			err := repo.SubmitKVMConsentCode(ctx, testMockSystemID, code)
			if !errors.Is(err, usecase.ErrInvalidConsentCode) {
				t.Fatalf("expected ErrInvalidConsentCode for %q, got %v", code, err)
			}
		})
	}
}

func TestMockSubmitKVMConsentCodeRequiresActivePrompt(t *testing.T) {
	t.Parallel()

	repo := NewMockComputerSystemRepo()
	ctx := context.Background()

	// No RequestKVMConsent call: the opt-in flow has not started.
	err := repo.SubmitKVMConsentCode(ctx, testMockSystemID, mockConsentCodeGranted)

	var consentErr *usecase.ConsentFailedError
	if !errors.As(err, &consentErr) {
		t.Fatalf("expected *usecase.ConsentFailedError, got %v", err)
	}

	if consentErr.ReturnValue != mockConsentReturnInvalidState {
		t.Fatalf("expected ReturnValue=%d, got %d", mockConsentReturnInvalidState, consentErr.ReturnValue)
	}
}

func TestMockKVMConsentSystemNotFound(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		call func(repo *MockComputerSystemRepo) error
	}{
		{
			name: "RequestKVMConsent",
			call: func(repo *MockComputerSystemRepo) error {
				return repo.RequestKVMConsent(context.Background(), testMockUnknownSystem)
			},
		},
		{
			name: "SubmitKVMConsentCode",
			call: func(repo *MockComputerSystemRepo) error {
				return repo.SubmitKVMConsentCode(context.Background(), testMockUnknownSystem, mockConsentCodeGranted)
			},
		},
		{
			name: "CancelKVMConsent",
			call: func(repo *MockComputerSystemRepo) error {
				return repo.CancelKVMConsent(context.Background(), testMockUnknownSystem)
			},
		},
		{
			name: "UpdateGraphicalConsoleServiceEnabled",
			call: func(repo *MockComputerSystemRepo) error {
				return repo.UpdateGraphicalConsoleServiceEnabled(context.Background(), testMockUnknownSystem, true)
			},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := NewMockComputerSystemRepo()
			if err := tt.call(repo); !errors.Is(err, usecase.ErrSystemNotFound) {
				t.Fatalf("expected ErrSystemNotFound, got %v", err)
			}
		})
	}
}

func TestMockUpdateGraphicalConsoleServiceEnabledStatus(t *testing.T) {
	t.Parallel()

	repo := NewMockComputerSystemRepo()
	ctx := context.Background()

	// Disabling KVM forces KVMStatus to Disabled while leaving the consent
	// status intact.
	if err := repo.UpdateGraphicalConsoleServiceEnabled(ctx, testMockSystemID, false); err != nil {
		t.Fatalf("UpdateGraphicalConsoleServiceEnabled(false) returned error: %v", err)
	}

	amt := amtStatusFor(t, repo)
	if amt.KVMStatus != mockKVMStatusDisabled {
		t.Fatalf("expected KVMStatus=%q after disable, got %q", mockKVMStatusDisabled, amt.KVMStatus)
	}

	// Re-enabling restores the consent-driven status.
	if err := repo.UpdateGraphicalConsoleServiceEnabled(ctx, testMockSystemID, true); err != nil {
		t.Fatalf("UpdateGraphicalConsoleServiceEnabled(true) returned error: %v", err)
	}

	amt = amtStatusFor(t, repo)
	if amt.KVMStatus != mockKVMStatusPendingConsent {
		t.Fatalf("expected KVMStatus=%q after re-enable, got %q", mockKVMStatusPendingConsent, amt.KVMStatus)
	}
}

func TestMockRequestKVMConsentACM(t *testing.T) {
	t.Parallel()

	repo := NewMockComputerSystemRepo()
	repo.kvmStateFor(testMockSystemID).controlMode = mockControlModeACM

	err := repo.RequestKVMConsent(context.Background(), testMockSystemID)
	if !errors.Is(err, usecase.ErrKVMConsentNotRequiredInACM) {
		t.Fatalf("expected ErrKVMConsentNotRequiredInACM, got %v", err)
	}

	amt := amtStatusFor(t, repo)
	if amt.UserConsentStatus != mockUserConsentNotRequired {
		t.Fatalf("expected UserConsentStatus=%q in ACM, got %q", mockUserConsentNotRequired, amt.UserConsentStatus)
	}

	if amt.KVMStatus != mockKVMStatusEnabled {
		t.Fatalf("expected KVMStatus=%q in ACM, got %q", mockKVMStatusEnabled, amt.KVMStatus)
	}
}

func TestMockKVMStateStatusMapping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		state           mockKVMState
		wantKVMStatus   string
		wantUserConsent string
	}{
		{
			name:            "CCM not started",
			state:           mockKVMState{enableKVM: true, kvmAvailable: true, controlMode: mockControlModeCCM, optInState: mockOptInNotStarted},
			wantKVMStatus:   mockKVMStatusPendingConsent,
			wantUserConsent: mockUserConsentRequired,
		},
		{
			name:            "CCM requested",
			state:           mockKVMState{enableKVM: true, kvmAvailable: true, controlMode: mockControlModeCCM, optInState: mockOptInRequested},
			wantKVMStatus:   mockKVMStatusPendingConsent,
			wantUserConsent: mockUserConsentRequested,
		},
		{
			name:            "CCM displayed",
			state:           mockKVMState{enableKVM: true, kvmAvailable: true, controlMode: mockControlModeCCM, optInState: mockOptInDisplayed},
			wantKVMStatus:   mockKVMStatusPendingConsent,
			wantUserConsent: mockUserConsentRequested,
		},
		{
			name:            "CCM received",
			state:           mockKVMState{enableKVM: true, kvmAvailable: true, controlMode: mockControlModeCCM, optInState: mockOptInReceived},
			wantKVMStatus:   mockKVMStatusEnabled,
			wantUserConsent: mockUserConsentGranted,
		},
		{
			name:            "CCM in session",
			state:           mockKVMState{enableKVM: true, kvmAvailable: true, controlMode: mockControlModeCCM, optInState: mockOptInInSession},
			wantKVMStatus:   mockKVMStatusActive,
			wantUserConsent: mockUserConsentGranted,
		},
		{
			name:            "CCM denied",
			state:           mockKVMState{enableKVM: true, kvmAvailable: true, controlMode: mockControlModeCCM, optInState: mockOptInDenied},
			wantKVMStatus:   mockKVMStatusError,
			wantUserConsent: mockUserConsentDenied,
		},
		{
			name:            "CCM timeout",
			state:           mockKVMState{enableKVM: true, kvmAvailable: true, controlMode: mockControlModeCCM, optInState: mockOptInTimeout},
			wantKVMStatus:   mockKVMStatusError,
			wantUserConsent: mockUserConsentTimeout,
		},
		{
			name:            "KVM unavailable",
			state:           mockKVMState{enableKVM: true, kvmAvailable: false, controlMode: mockControlModeCCM, optInState: mockOptInNotStarted},
			wantKVMStatus:   mockKVMStatusDisabled,
			wantUserConsent: mockUserConsentRequired,
		},
		{
			name:            "KVM disabled",
			state:           mockKVMState{enableKVM: false, kvmAvailable: true, controlMode: mockControlModeCCM, optInState: mockOptInReceived},
			wantKVMStatus:   mockKVMStatusDisabled,
			wantUserConsent: mockUserConsentGranted,
		},
		{
			name:            "ACM not started",
			state:           mockKVMState{enableKVM: true, kvmAvailable: true, controlMode: mockControlModeACM, optInState: mockOptInNotStarted},
			wantKVMStatus:   mockKVMStatusEnabled,
			wantUserConsent: mockUserConsentNotRequired,
		},
		{
			name:            "ACM in session",
			state:           mockKVMState{enableKVM: true, kvmAvailable: true, controlMode: mockControlModeACM, optInState: mockOptInInSession},
			wantKVMStatus:   mockKVMStatusActive,
			wantUserConsent: mockUserConsentNotRequired,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			state := tt.state
			if got := state.kvmStatus(); got != tt.wantKVMStatus {
				t.Fatalf("kvmStatus() = %q, want %q", got, tt.wantKVMStatus)
			}

			if got := state.userConsentStatus(); got != tt.wantUserConsent {
				t.Fatalf("userConsentStatus() = %q, want %q", got, tt.wantUserConsent)
			}
		})
	}
}

func TestMockKVMStateUnknownOptInState(t *testing.T) {
	t.Parallel()

	// An opt-in state outside the documented 0..6 range must fall back to the
	// defensive Error / Requested defaults rather than panicking.
	const unknownOptInState = 99

	state := mockKVMState{
		enableKVM:    true,
		kvmAvailable: true,
		controlMode:  mockControlModeCCM,
		optInState:   unknownOptInState,
	}

	if got := state.kvmStatus(); got != mockKVMStatusError {
		t.Fatalf("kvmStatus() = %q, want %q", got, mockKVMStatusError)
	}

	if got := state.userConsentStatus(); got != mockUserConsentRequested {
		t.Fatalf("userConsentStatus() = %q, want %q", got, mockUserConsentRequested)
	}
}

func TestMockKVMStateForLazilyCreatesDefault(t *testing.T) {
	t.Parallel()

	repo := NewMockComputerSystemRepo()

	// Simulate a system that exists without an explicit KVM entry to exercise
	// the lazy default-state creation in kvmStateFor.
	delete(repo.kvmState, testMockSystemID)

	state := repo.kvmStateFor(testMockSystemID)
	if state == nil {
		t.Fatal("kvmStateFor returned nil for a system without explicit KVM state")
	}

	// The lazily created default must be persisted for stable subsequent reads.
	if _, ok := repo.kvmState[testMockSystemID]; !ok {
		t.Fatal("kvmStateFor did not persist the lazily created default state")
	}

	if !state.enableKVM || !state.kvmAvailable ||
		state.controlMode != mockControlModeCCM || state.optInState != mockOptInNotStarted {
		t.Fatalf("unexpected lazily created default state: %#v", state)
	}
}

func TestMockGetByIDUnknownSystem(t *testing.T) {
	t.Parallel()

	repo := NewMockComputerSystemRepo()

	if _, err := repo.GetByID(context.Background(), testMockUnknownSystem); !errors.Is(err, usecase.ErrSystemNotFound) {
		t.Fatalf("expected ErrSystemNotFound, got %v", err)
	}
}
