package usecase

import (
	"testing"

	optin "github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/ips/optin"

	dtov2 "github.com/device-management-toolkit/console/internal/entity/dto/v2"
	"github.com/device-management-toolkit/console/pkg/logger"
	redfishv1 "github.com/device-management-toolkit/console/redfish/internal/entity/v1"
)

func TestDetermineKVMStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		enableKVM    bool
		kvmAvailable bool
		userConsent  string
		optInState   int
		want         string
	}{
		{
			name:         "disabled when KVM not available",
			enableKVM:    true,
			kvmAvailable: false,
			userConsent:  "kvm",
			optInState:   int(optin.InSession),
			want:         StateDisabled,
		},
		{
			name:         "disabled when KVM feature off",
			enableKVM:    false,
			kvmAvailable: true,
			userConsent:  "kvm",
			optInState:   int(optin.InSession),
			want:         StateDisabled,
		},
		{
			name:         "active when consent required and in session",
			enableKVM:    true,
			kvmAvailable: true,
			userConsent:  "kvm",
			optInState:   int(optin.InSession),
			want:         "Active",
		},
		{
			name:         "pending consent when requested",
			enableKVM:    true,
			kvmAvailable: true,
			userConsent:  "all",
			optInState:   int(optin.Requested),
			want:         "PendingConsent",
		},
		{
			name:         "enabled when consent received",
			enableKVM:    true,
			kvmAvailable: true,
			userConsent:  "all",
			optInState:   int(optin.Received),
			want:         StateEnabled,
		},
		{
			name:         "error when consent required and unknown opt-in state",
			enableKVM:    true,
			kvmAvailable: true,
			userConsent:  "kvm",
			optInState:   999,
			want:         "Error",
		},
		{
			name:         "enabled when consent not required",
			enableKVM:    true,
			kvmAvailable: true,
			userConsent:  "none",
			optInState:   int(optin.NotStarted),
			want:         StateEnabled,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := determineKVMStatus(tt.enableKVM, tt.kvmAvailable, tt.userConsent, tt.optInState)
			if got != tt.want {
				t.Fatalf("determineKVMStatus() = %q, want %q", got, tt.want)
			}
		})
	}
}

func assertGraphicalConsoleOEM(t *testing.T, got *redfishv1.ComputerSystemHostGraphicalConsole, wantKVMStatus string) {
	t.Helper()

	if got.OEM == nil || got.OEM.Intel == nil || got.OEM.Intel.AMT == nil {
		t.Fatal("OEM.Intel.AMT is nil")
	}

	amt := got.OEM.Intel.AMT

	if amt.KVMStatus != wantKVMStatus {
		t.Errorf("KVMStatus = %q, want %q", amt.KVMStatus, wantKVMStatus)
	}

	if amt.ControlMode != "ACM" {
		t.Errorf("ControlMode = %q, want %q", amt.ControlMode, "ACM")
	}

	if amt.UserConsentStatus != "NotRequired" {
		t.Errorf("UserConsentStatus = %q, want %q", amt.UserConsentStatus, "NotRequired")
	}
}

func assertGraphicalConsole(t *testing.T, got *redfishv1.ComputerSystemHostGraphicalConsole, wantEnabled bool, wantConnTypes []string, wantPort int64, wantKVMStatus string) {
	t.Helper()

	if got == nil {
		t.Fatal("buildGraphicalConsole() returned nil")
	}

	if got.ServiceEnabled == nil || *got.ServiceEnabled != wantEnabled {
		t.Errorf("ServiceEnabled = %v, want %v", got.ServiceEnabled, wantEnabled)
	}

	if wantConnTypes == nil {
		if len(got.ConnectTypesSupported) != 0 {
			t.Errorf("ConnectTypesSupported = %v, want empty", got.ConnectTypesSupported)
		}
	} else {
		if len(got.ConnectTypesSupported) != len(wantConnTypes) || got.ConnectTypesSupported[0] != wantConnTypes[0] {
			t.Errorf("ConnectTypesSupported = %v, want %v", got.ConnectTypesSupported, wantConnTypes)
		}
	}

	if wantPort == 0 {
		if got.Port != nil {
			t.Errorf("Port = %v, want nil", got.Port)
		}
	} else {
		if got.Port == nil || *got.Port != wantPort {
			t.Errorf("Port = %v, want %d", got.Port, wantPort)
		}
	}

	assertGraphicalConsoleOEM(t, got, wantKVMStatus)
}

func TestBuildGraphicalConsole(t *testing.T) {
	t.Parallel()

	repo := &WsmanComputerSystemRepo{log: logger.New("error")}
	kvmIP := []string{kvmConnectTypeKVMIP}

	tests := []struct {
		name          string
		useTLS        bool
		features      dtov2.Features
		wantEnabled   bool
		wantConnTypes []string
		wantPort      int64
		wantKVMStatus string
	}{
		{
			name:          "KVM not available - no connect types and no port",
			useTLS:        false,
			features:      dtov2.Features{EnableKVM: true, KVMAvailable: false, UserConsent: "none"},
			wantEnabled:   true,
			wantConnTypes: nil,
			wantPort:      0,
			wantKVMStatus: StateDisabled,
		},
		{
			name:          "KVM available non-TLS port",
			useTLS:        false,
			features:      dtov2.Features{EnableKVM: true, KVMAvailable: true, UserConsent: "none"},
			wantEnabled:   true,
			wantConnTypes: kvmIP,
			wantPort:      16994,
			wantKVMStatus: StateEnabled,
		},
		{
			name:          "KVM available TLS port",
			useTLS:        true,
			features:      dtov2.Features{EnableKVM: true, KVMAvailable: true, UserConsent: "none"},
			wantEnabled:   true,
			wantConnTypes: kvmIP,
			wantPort:      16995,
			wantKVMStatus: StateEnabled,
		},
		{
			name:          "KVM disabled",
			useTLS:        false,
			features:      dtov2.Features{EnableKVM: false, KVMAvailable: true, UserConsent: "none"},
			wantEnabled:   false,
			wantConnTypes: kvmIP,
			wantPort:      16994,
			wantKVMStatus: StateDisabled,
		},
		{
			name:          "consent required and in session - active",
			useTLS:        false,
			features:      dtov2.Features{EnableKVM: true, KVMAvailable: true, UserConsent: "kvm", OptInState: int(optin.InSession)},
			wantEnabled:   true,
			wantConnTypes: kvmIP,
			wantPort:      16994,
			wantKVMStatus: kvmStatusActive,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := repo.buildGraphicalConsole(tt.useTLS, tt.features)
			assertGraphicalConsole(t, got, tt.wantEnabled, tt.wantConnTypes, tt.wantPort, tt.wantKVMStatus)
		})
	}
}
