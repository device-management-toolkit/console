package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/device-management-toolkit/console/redfish/internal/controller/http/v1/generated"
	redfishv1 "github.com/device-management-toolkit/console/redfish/internal/entity/v1"
)

type graphicalConsoleTestRepo struct {
	system  *redfishv1.ComputerSystem
	bootErr error
	kvmErr  error
	solErr  error
	ccErr   error
}

func (r *graphicalConsoleTestRepo) GetAll(_ context.Context) ([]string, error) {
	return []string{"system-1"}, nil
}

func (r *graphicalConsoleTestRepo) GetByID(_ context.Context, _ string) (*redfishv1.ComputerSystem, error) {
	if r.system == nil {
		return nil, ErrSystemNotFound
	}

	s := *r.system

	return &s, nil
}

func (r *graphicalConsoleTestRepo) UpdatePowerState(_ context.Context, _ string, _ redfishv1.PowerState) error {
	return nil
}

func (r *graphicalConsoleTestRepo) GetBootSettings(_ context.Context, _ string) (*generated.ComputerSystemBoot, error) {
	if r.bootErr != nil {
		return nil, r.bootErr
	}

	return &generated.ComputerSystemBoot{}, nil
}

func (r *graphicalConsoleTestRepo) UpdateBootSettings(_ context.Context, _ string, _ *generated.ComputerSystemBoot) error {
	return nil
}

func (r *graphicalConsoleTestRepo) UpdateGraphicalConsoleServiceEnabled(_ context.Context, _ string, _ bool) error {
	return r.kvmErr
}

func (r *graphicalConsoleTestRepo) UpdateSerialConsoleServiceEnabled(_ context.Context, _ string, _ bool) error {
	return r.solErr
}

func (r *graphicalConsoleTestRepo) RequestKVMConsent(_ context.Context, _ string) error {
	return r.ccErr
}

func (r *graphicalConsoleTestRepo) SubmitKVMConsentCode(_ context.Context, _, _ string) error {
	return r.ccErr
}

func (r *graphicalConsoleTestRepo) CancelKVMConsent(_ context.Context, _ string) error {
	return r.ccErr
}

func TestConvertGraphicalConsoleToGeneratedNil(t *testing.T) {
	t.Parallel()

	uc := &ComputerSystemUseCase{}

	if got := uc.convertGraphicalConsoleToGenerated(nil); got != nil {
		t.Fatalf("expected nil GraphicalConsole, got %#v", got)
	}
}

func TestGetComputerSystemGraphicalConsoleMapping(t *testing.T) {
	t.Parallel()

	serviceEnabled := true
	maxSessions := int64(2)
	port := int64(5900)

	repo := &graphicalConsoleTestRepo{
		system: &redfishv1.ComputerSystem{
			ID:         "system-1",
			Name:       "Test System",
			SystemType: redfishv1.SystemTypePhysical,
			PowerState: redfishv1.PowerStateOn,
			GraphicalConsole: &redfishv1.ComputerSystemHostGraphicalConsole{
				ConnectTypesSupported: []string{"KVMIP", "OEM", "INVALID"},
				MaxConcurrentSessions: &maxSessions,
				Port:                  &port,
				ServiceEnabled:        &serviceEnabled,
			},
		},
	}

	uc := &ComputerSystemUseCase{Repo: repo}

	result, err := uc.GetComputerSystem(context.Background(), "system-1")
	if err != nil {
		t.Fatalf("GetComputerSystem returned error: %v", err)
	}

	if result.GraphicalConsole == nil {
		t.Fatal("expected GraphicalConsole to be present")
	}

	if result.GraphicalConsole.Port == nil || *result.GraphicalConsole.Port != 5900 {
		t.Fatalf("expected Port=5900, got %#v", result.GraphicalConsole.Port)
	}

	if result.GraphicalConsole.ServiceEnabled == nil || !*result.GraphicalConsole.ServiceEnabled {
		t.Fatalf("expected ServiceEnabled=true, got %#v", result.GraphicalConsole.ServiceEnabled)
	}

	if result.GraphicalConsole.MaxConcurrentSessions == nil || *result.GraphicalConsole.MaxConcurrentSessions != 2 {
		t.Fatalf("expected MaxConcurrentSessions=2, got %#v", result.GraphicalConsole.MaxConcurrentSessions)
	}

	if result.GraphicalConsole.ConnectTypesSupported == nil {
		t.Fatal("expected ConnectTypesSupported to be set")
	}

	got := *result.GraphicalConsole.ConnectTypesSupported
	if len(got) != 2 {
		t.Fatalf("expected 2 supported connect types, got %d: %#v", len(got), got)
	}

	if got[0] != generated.ComputerSystemGraphicalConnectTypesSupportedKVMIP {
		t.Fatalf("expected first connect type KVMIP, got %q", got[0])
	}

	if got[1] != generated.ComputerSystemGraphicalConnectTypesSupportedOEM {
		t.Fatalf("expected second connect type OEM, got %q", got[1])
	}
}

func TestGetComputerSystemGraphicalConsoleDropsUnsupportedConnectTypes(t *testing.T) {
	t.Parallel()

	serviceEnabled := false

	repo := &graphicalConsoleTestRepo{
		system: &redfishv1.ComputerSystem{
			ID:         "system-1",
			Name:       "Test System",
			SystemType: redfishv1.SystemTypePhysical,
			PowerState: redfishv1.PowerStateOn,
			GraphicalConsole: &redfishv1.ComputerSystemHostGraphicalConsole{
				ConnectTypesSupported: []string{"INVALID"},
				ServiceEnabled:        &serviceEnabled,
			},
		},
		bootErr: errors.New("boot unavailable"),
	}

	uc := &ComputerSystemUseCase{Repo: repo}

	result, err := uc.GetComputerSystem(context.Background(), "system-1")
	if err != nil {
		t.Fatalf("GetComputerSystem returned error: %v", err)
	}

	if result.GraphicalConsole == nil {
		t.Fatal("expected GraphicalConsole to be present")
	}

	if result.GraphicalConsole.ConnectTypesSupported != nil {
		t.Fatalf("expected unsupported connect types to be omitted, got %#v", result.GraphicalConsole.ConnectTypesSupported)
	}
}

func TestConvertStateToGenerated(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		state     string
		wantNil   bool
		wantState generated.ResourceState
	}{
		{name: "StateEnabled", state: StateEnabled, wantState: generated.ResourceStateEnabled},
		{name: "StateDisabled", state: StateDisabled, wantState: generated.ResourceStateDisabled},
		{name: "StateStandbyOffline", state: StateStandbyOffline, wantState: generated.ResourceStateStandbyOffline},
		{name: "StateStandbySpare", state: StateStandbySpare, wantState: generated.ResourceStateStandbySpare},
		{name: "StateInTest", state: StateInTest, wantState: generated.ResourceStateInTest},
		{name: "StateStarting", state: StateStarting, wantState: generated.ResourceStateStarting},
		{name: "StateAbsent", state: StateAbsent, wantState: generated.ResourceStateAbsent},
		{name: "StateUnavailableOffline", state: StateUnavailableOffline, wantState: generated.ResourceStateUnavailableOffline},
		{name: "StateDeferring", state: StateDeferring, wantState: generated.ResourceStateDeferring},
		{name: "StateQuiesced", state: StateQuiesced, wantState: generated.ResourceStateQuiesced},
		{name: "StateUpdating", state: StateUpdating, wantState: generated.ResourceStateUpdating},
		{name: "StateDegraded", state: StateDegraded, wantState: generated.ResourceStateDegraded},
		{name: "UnknownState", state: "UnknownState", wantNil: true},
		{name: "EmptyState", state: "", wantNil: true},
	}

	uc := &ComputerSystemUseCase{}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			validateConvertStateResult(t, uc, tt.state, tt.wantNil, tt.wantState)
		})
	}
}

func validateConvertStateResult(t *testing.T, uc *ComputerSystemUseCase, state string, wantNil bool, wantState generated.ResourceState) {
	t.Helper()

	result := uc.convertStateToGenerated(state)

	if wantNil {
		if result != nil {
			t.Fatalf("convertStateToGenerated(%q) expected nil, got %v", state, result)
		}

		return
	}

	if result == nil {
		t.Fatalf("convertStateToGenerated(%q) expected non-nil result", state)

		return
	}

	got, err := result.AsResourceState()
	if err != nil {
		t.Fatalf("Failed to convert result back to ResourceState: %v", err)
	}

	if got != wantState {
		t.Errorf("convertStateToGenerated(%q) got %v, want %v", state, got, wantState)
	}
}

func TestConvertSerialConsoleToGeneratedNil(t *testing.T) {
	t.Parallel()

	uc := &ComputerSystemUseCase{}

	if got := uc.convertSerialConsoleToGenerated(nil); got != nil {
		t.Fatalf("expected nil SerialConsole, got %#v", got)
	}
}

func TestGetComputerSystemSerialConsoleMapping(t *testing.T) {
	t.Parallel()

	serviceEnabled := true
	interactive := true
	maxSessions := int64(1)
	consoleURI := "/relay/webrelay.ashx?host=system-1&mode=sol"

	repo := &graphicalConsoleTestRepo{
		system: &redfishv1.ComputerSystem{
			ID:         "system-1",
			Name:       "Test System",
			SystemType: redfishv1.SystemTypePhysical,
			PowerState: redfishv1.PowerStateOn,
			SerialConsole: &redfishv1.ComputerSystemHostSerialConsole{
				MaxConcurrentSessions: &maxSessions,
				WebSocket: &redfishv1.ComputerSystemHostWebSocketConsole{
					ConsoleURI:     &consoleURI,
					Interactive:    &interactive,
					ServiceEnabled: &serviceEnabled,
				},
				OEM: &redfishv1.ComputerSystemHostSerialConsoleOEM{
					Intel: &redfishv1.ComputerSystemHostSerialConsoleIntel{
						AMT: &redfishv1.ComputerSystemHostSerialConsoleAMT{
							ControlMode:       "ACM",
							SOLStatus:         "Enabled",
							UserConsentStatus: "NotRequired",
						},
					},
				},
			},
		},
	}

	uc := &ComputerSystemUseCase{Repo: repo}

	result, err := uc.GetComputerSystem(context.Background(), "system-1")
	if err != nil {
		t.Fatalf("GetComputerSystem returned error: %v", err)
	}

	assertGeneratedSerialConsoleMapping(t, result, consoleURI)
}

func assertGeneratedSerialConsoleMapping(t *testing.T, result *generated.ComputerSystemComputerSystem, consoleURI string) {
	t.Helper()
	serialConsole := assertGeneratedSerialConsoleCore(t, result, consoleURI)
	assertGeneratedSerialConsoleAMT(t, serialConsole)
}

func assertGeneratedSerialConsoleCore(t *testing.T, result *generated.ComputerSystemComputerSystem, consoleURI string) *generated.ComputerSystemHostSerialConsole {
	t.Helper()

	if result.SerialConsole == nil {
		t.Fatal("expected SerialConsole to be present")
	}

	if result.SerialConsole.MaxConcurrentSessions == nil || *result.SerialConsole.MaxConcurrentSessions != 1 {
		t.Fatalf("expected MaxConcurrentSessions=1, got %#v", result.SerialConsole.MaxConcurrentSessions)
	}

	if result.SerialConsole.WebSocket == nil {
		t.Fatal("expected WebSocket to be present")
	}

	if result.SerialConsole.WebSocket.ConsoleURI == nil || *result.SerialConsole.WebSocket.ConsoleURI != consoleURI {
		t.Fatalf("expected ConsoleURI=%q, got %#v", consoleURI, result.SerialConsole.WebSocket.ConsoleURI)
	}

	if result.SerialConsole.WebSocket.ServiceEnabled == nil || !*result.SerialConsole.WebSocket.ServiceEnabled {
		t.Fatalf("expected ServiceEnabled=true, got %#v", result.SerialConsole.WebSocket.ServiceEnabled)
	}

	return result.SerialConsole
}

func assertGeneratedSerialConsoleAMT(t *testing.T, serialConsole *generated.ComputerSystemHostSerialConsole) {
	t.Helper()

	if serialConsole.Oem == nil || serialConsole.Oem.Intel == nil || serialConsole.Oem.Intel.AMT == nil {
		t.Fatal("expected SerialConsole.Oem.Intel.AMT to be present")
	}

	gotControlMode, err := serialConsole.Oem.Intel.AMT.ControlMode.AsComputerSystemOemIntelAMTControlMode()
	if err != nil {
		t.Fatalf("failed to decode ControlMode: %v", err)
	}

	if gotControlMode != generated.ACM {
		t.Fatalf("expected ControlMode=%q, got %q", generated.ACM, gotControlMode)
	}

	gotSOLStatus, err := serialConsole.Oem.Intel.AMT.SOLStatus.AsComputerSystemOemIntelAMTSOLStatus()
	if err != nil {
		t.Fatalf("failed to decode SOLStatus: %v", err)
	}

	if gotSOLStatus != generated.ComputerSystemOemIntelAMTSOLStatusEnabled {
		t.Fatalf("expected SOLStatus=%q, got %q", generated.ComputerSystemOemIntelAMTSOLStatusEnabled, gotSOLStatus)
	}

	gotConsentStatus, err := serialConsole.Oem.Intel.AMT.UserConsentStatus.AsComputerSystemOemIntelAMTUserConsentStatus()
	if err != nil {
		t.Fatalf("failed to decode UserConsentStatus: %v", err)
	}

	if gotConsentStatus != generated.NotRequired {
		t.Fatalf("expected UserConsentStatus=%q, got %q", generated.NotRequired, gotConsentStatus)
	}
}

func TestConvertSerialConsoleToGenerated_WithNilWebSocket(t *testing.T) {
	t.Parallel()

	uc := &ComputerSystemUseCase{}

	maxSessions := int64(1)
	console := &redfishv1.ComputerSystemHostSerialConsole{
		MaxConcurrentSessions: &maxSessions,
		WebSocket:             nil,
		OEM:                   nil,
	}

	got := uc.convertSerialConsoleToGenerated(console)

	if got == nil {
		t.Fatal("expected non-nil result")

		return
	}

	if got.MaxConcurrentSessions == nil || *got.MaxConcurrentSessions != 1 {
		t.Fatalf("expected MaxConcurrentSessions=1, got %#v", got.MaxConcurrentSessions)
	}

	if got.WebSocket != nil {
		t.Fatalf("expected WebSocket to be nil, got %#v", got.WebSocket)
	}

	if got.Oem != nil {
		t.Fatalf("expected Oem to be nil, got %#v", got.Oem)
	}
}

func TestConvertSerialConsoleOEMToGenerated_WithNilIntel(t *testing.T) {
	t.Parallel()

	uc := &ComputerSystemUseCase{}

	oem := &redfishv1.ComputerSystemHostSerialConsoleOEM{
		Intel: nil,
	}

	got := uc.convertSerialConsoleOEMToGenerated(oem)

	if got == nil {
		t.Fatal("expected non-nil result")

		return
	}

	if got.Intel != nil {
		t.Fatalf("expected Intel to be nil when input Intel is nil, got %#v", got.Intel)
	}
}

func TestConvertSerialConsoleOEMToGenerated_WithNilAMT(t *testing.T) {
	t.Parallel()

	uc := &ComputerSystemUseCase{}

	oem := &redfishv1.ComputerSystemHostSerialConsoleOEM{
		Intel: &redfishv1.ComputerSystemHostSerialConsoleIntel{
			AMT: nil,
		},
	}

	got := uc.convertSerialConsoleOEMToGenerated(oem)

	if got == nil {
		t.Fatal("expected non-nil result")

		return
	}

	if got.Intel == nil {
		t.Fatal("expected Intel to be non-nil")

		return
	}

	if got.Intel.AMT != nil {
		t.Fatalf("expected Intel.AMT to be nil, got %#v", got.Intel.AMT)
	}
}

func TestConvertSerialConsoleOEMToGenerated_Nil(t *testing.T) {
	t.Parallel()

	uc := &ComputerSystemUseCase{}

	got := uc.convertSerialConsoleOEMToGenerated(nil)

	if got != nil {
		t.Fatalf("expected nil, got %#v", got)
	}
}

func TestConvertSerialControlModeToGenerated_EmptyValue(t *testing.T) {
	t.Parallel()

	got := convertSerialControlModeToGenerated("")

	if got != nil {
		t.Fatalf("expected nil for empty value, got %#v", got)
	}
}

func TestConvertSOLStatusToGenerated_EmptyValue(t *testing.T) {
	t.Parallel()

	got := convertSOLStatusToGenerated("")

	if got != nil {
		t.Fatalf("expected nil for empty value, got %#v", got)
	}
}

func TestConvertSerialUserConsentStatusToGenerated_EmptyValue(t *testing.T) {
	t.Parallel()

	got := convertSerialUserConsentStatusToGenerated("")

	if got != nil {
		t.Fatalf("expected nil for empty value, got %#v", got)
	}
}

func TestConvertSerialControlModeToGenerated_ValidValue(t *testing.T) {
	t.Parallel()

	got := convertSerialControlModeToGenerated("ACM")

	if got == nil {
		t.Fatal("expected non-nil result")

		return
	}

	mode, err := got.AsComputerSystemOemIntelAMTControlMode()
	if err != nil {
		t.Fatalf("failed to decode ControlMode: %v", err)
	}

	if mode != generated.ACM {
		t.Fatalf("expected ACM, got %q", mode)
	}
}

func TestConvertSOLStatusToGenerated_ValidValue(t *testing.T) {
	t.Parallel()

	got := convertSOLStatusToGenerated("Enabled")

	if got == nil {
		t.Fatal("expected non-nil result")

		return
	}

	status, err := got.AsComputerSystemOemIntelAMTSOLStatus()
	if err != nil {
		t.Fatalf("failed to decode SOLStatus: %v", err)
	}

	if status != generated.ComputerSystemOemIntelAMTSOLStatusEnabled {
		t.Fatalf("expected Enabled, got %q", status)
	}
}

func TestConvertSerialUserConsentStatusToGenerated_ValidValue(t *testing.T) {
	t.Parallel()

	got := convertSerialUserConsentStatusToGenerated("NotRequired")

	if got == nil {
		t.Fatal("expected non-nil result")

		return
	}

	consent, err := got.AsComputerSystemOemIntelAMTUserConsentStatus()
	if err != nil {
		t.Fatalf("failed to decode UserConsentStatus: %v", err)
	}

	if consent != generated.NotRequired {
		t.Fatalf("expected NotRequired, got %q", consent)
	}
}

func TestUpdateGraphicalConsoleServiceEnabled(t *testing.T) {
	t.Parallel()

	errAMT := errors.New("amt refused")

	tests := []struct {
		name    string
		kvmErr  error
		enabled bool
		wantErr error
	}{
		{
			name:    "success",
			enabled: true,
		},
		{
			name:    "repo error",
			kvmErr:  errAMT,
			enabled: false,
			wantErr: errAMT,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			uc := &ComputerSystemUseCase{Repo: &graphicalConsoleTestRepo{kvmErr: tt.kvmErr}}

			err := uc.UpdateGraphicalConsoleServiceEnabled(context.Background(), "system-1", tt.enabled)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("UpdateGraphicalConsoleServiceEnabled() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUpdateSerialConsoleServiceEnabled(t *testing.T) {
	t.Parallel()

	errAMT := errors.New("amt refused")

	tests := []struct {
		name    string
		solErr  error
		enabled bool
		wantErr error
	}{
		{
			name:    "enable success",
			enabled: true,
		},
		{
			name:    "disable success",
			enabled: false,
		},
		{
			name:    "repo error",
			solErr:  errAMT,
			enabled: false,
			wantErr: errAMT,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			uc := &ComputerSystemUseCase{Repo: &graphicalConsoleTestRepo{solErr: tt.solErr}}

			err := uc.UpdateSerialConsoleServiceEnabled(context.Background(), "system-1", tt.enabled)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("UpdateSerialConsoleServiceEnabled() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRequestKVMConsent(t *testing.T) {
	t.Parallel()

	errAMT := errors.New("amt refused")

	tests := []struct {
		name    string
		repoErr error
		wantErr error
	}{
		{name: "success"},
		{name: "repo error", repoErr: errAMT, wantErr: errAMT},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			uc := &ComputerSystemUseCase{Repo: &graphicalConsoleTestRepo{ccErr: tt.repoErr}}

			err := uc.RequestKVMConsent(context.Background(), "system-1")
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("RequestKVMConsent() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSubmitKVMConsentCode(t *testing.T) {
	t.Parallel()

	errAMT := errors.New("amt refused")

	tests := []struct {
		name    string
		repoErr error
		wantErr error
	}{
		{name: "success"},
		{name: "repo error", repoErr: errAMT, wantErr: errAMT},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			uc := &ComputerSystemUseCase{Repo: &graphicalConsoleTestRepo{ccErr: tt.repoErr}}

			err := uc.SubmitKVMConsentCode(context.Background(), "system-1", "123456")
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("SubmitKVMConsentCode() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCancelKVMConsent(t *testing.T) {
	t.Parallel()

	errAMT := errors.New("amt refused")

	tests := []struct {
		name    string
		repoErr error
		wantErr error
	}{
		{name: "success"},
		{name: "repo error", repoErr: errAMT, wantErr: errAMT},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			uc := &ComputerSystemUseCase{Repo: &graphicalConsoleTestRepo{ccErr: tt.repoErr}}

			err := uc.CancelKVMConsent(context.Background(), "system-1")
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("CancelKVMConsent() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetComputerSystemIncludesGenerateRedirectionTokenAction(t *testing.T) {
	t.Parallel()

	const systemID = "system-1"

	repo := &graphicalConsoleTestRepo{
		system: &redfishv1.ComputerSystem{
			ID:         systemID,
			Name:       "Test System",
			SystemType: redfishv1.SystemTypePhysical,
			PowerState: redfishv1.PowerStateOn,
		},
	}

	uc := &ComputerSystemUseCase{Repo: repo}

	result, err := uc.GetComputerSystem(context.Background(), systemID)
	if err != nil {
		t.Fatalf("GetComputerSystem returned error: %v", err)
	}

	if result.Actions == nil {
		t.Fatal("expected Actions to be present")
	}

	if result.Actions.HashComputerSystemReset == nil {
		t.Fatal("expected #ComputerSystem.Reset action to be present")
	}

	if result.Actions.Oem == nil {
		t.Fatal("expected OEM actions to be present")
	}

	if result.Actions.Oem.HashOemIntelAMTGenerateRedirectionToken == nil {
		t.Fatal("expected #Oem.Intel.AMT.GenerateRedirectionToken action to be present")
	}

	if result.Actions.Oem.HashOemIntelAMTRequestKVMConsent == nil {
		t.Fatal("expected #Oem.Intel.AMT.RequestKVMConsent action to be present")
	}

	if result.Actions.Oem.HashOemIntelAMTSubmitKVMConsentCode == nil {
		t.Fatal("expected #Oem.Intel.AMT.SubmitKVMConsentCode action to be present")
	}

	if result.Actions.Oem.HashOemIntelAMTCancelKVMConsent == nil {
		t.Fatal("expected #Oem.Intel.AMT.CancelKVMConsent action to be present")
	}

	action := result.Actions.Oem.HashOemIntelAMTGenerateRedirectionToken
	expectedTarget := "/redfish/v1/Systems/system-1/Actions/Oem/IntelComputerSystem.GenerateRedirectionToken"

	if action.Target == nil || *action.Target != expectedTarget {
		t.Fatalf("expected action target %q, got %#v", expectedTarget, action.Target)
	}

	if action.Title == nil || *action.Title != "Generate Redirection Token" {
		t.Fatalf("expected action title %q, got %#v", "Generate Redirection Token", action.Title)
	}

	requestAction := result.Actions.Oem.HashOemIntelAMTRequestKVMConsent
	requestTarget := "/redfish/v1/Systems/system-1/Actions/Oem/IntelComputerSystem.RequestKVMConsent"

	submitAction := result.Actions.Oem.HashOemIntelAMTSubmitKVMConsentCode
	submitTarget := "/redfish/v1/Systems/system-1/Actions/Oem/IntelComputerSystem.SubmitKVMConsentCode"

	cancelAction := result.Actions.Oem.HashOemIntelAMTCancelKVMConsent
	cancelTarget := "/redfish/v1/Systems/system-1/Actions/Oem/IntelComputerSystem.CancelKVMConsent"

	assertActionTarget(t, "request", requestAction.Target, requestTarget)
	assertActionTarget(t, "submit", submitAction.Target, submitTarget)
	assertActionTarget(t, "cancel", cancelAction.Target, cancelTarget)
}

func assertActionTarget(t *testing.T, actionName string, got *string, want string) {
	t.Helper()

	if got == nil || *got != want {
		t.Fatalf("expected %s action target %q, got %#v", actionName, want, got)
	}
}
