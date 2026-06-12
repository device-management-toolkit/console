package usecase

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	gomock "go.uber.org/mock/gomock"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/boot"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/redirection"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/setupandconfiguration"
	cimBoot "github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/boot"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/kvm"
	cimmodels "github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/models"
	cimservice "github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/service"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/software"
	optin "github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/ips/optin"
	ipspower "github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/ips/power"

	"github.com/device-management-toolkit/console/internal/entity"
	dto "github.com/device-management-toolkit/console/internal/entity/dto/v1"
	dtov2 "github.com/device-management-toolkit/console/internal/entity/dto/v2"
	"github.com/device-management-toolkit/console/internal/mocks"
	"github.com/device-management-toolkit/console/internal/usecase/devices"
	wsmanAPI "github.com/device-management-toolkit/console/internal/usecase/devices/wsman"
	"github.com/device-management-toolkit/console/pkg/logger"
	redfishv1 "github.com/device-management-toolkit/console/redfish/internal/entity/v1"
)

func TestMapProvisioningModeToControlMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		mode setupandconfiguration.ProvisioningModeValue
		want string
	}{
		{name: "admin mode maps to ACM", mode: setupandconfiguration.AdminControlMode, want: controlModeACM},
		{name: "client mode maps to CCM", mode: setupandconfiguration.ClientControlMode, want: controlModeCCM},
		{name: "unknown mode omitted", mode: 999, want: ""},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := mapProvisioningModeToControlMode(tt.mode)
			if got != tt.want {
				t.Fatalf("mapProvisioningModeToControlMode() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRequestKVMConsentRepo_ReturnValueFailure(t *testing.T) {
	t.Parallel()

	device := &entity.Device{GUID: "system-1", TenantID: "tenant-1"}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repoMock := mocks.NewMockDeviceManagementRepository(ctrl)
	wsmanMock := mocks.NewMockWSMAN(ctrl)
	wsmanMock.EXPECT().Worker().Return().AnyTimes()

	management := mocks.NewMockManagement(ctrl)

	uc := devices.New(repoMock, wsmanMock, mocks.NewMockRedirection(ctrl), logger.New("error"), mocks.MockCrypto{})
	repo := &WsmanComputerSystemRepo{usecase: uc, log: logger.New("error")}

	gomock.InOrder(
		repoMock.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil),
		wsmanMock.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(management, nil),
		management.EXPECT().GetAMTVersion().Return([]software.SoftwareIdentity{}, nil),
		management.EXPECT().GetSetupAndConfiguration().Return([]setupandconfiguration.SetupAndConfigurationServiceResponse{{ProvisioningMode: setupandconfiguration.ClientControlMode}}, nil),
		repoMock.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil),
		wsmanMock.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(management, nil),
		management.EXPECT().GetUserConsentCode().Return(optin.Response{
			Body: optin.Body{StartOptInResponse: optin.StartOptIn_OUTPUT{ReturnValue: 5}},
		}, nil),
	)

	err := repo.RequestKVMConsent(context.Background(), device.GUID)

	var consentErr *ConsentFailedError
	if !errors.As(err, &consentErr) {
		t.Fatalf("RequestKVMConsent() error type = %T, want *ConsentFailedError", err)
	}
}

func TestRequestKVMConsentRepo_ACMReturnsNotRequiredError(t *testing.T) {
	t.Parallel()

	device := &entity.Device{GUID: "system-1", TenantID: "tenant-1"}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repoMock := mocks.NewMockDeviceManagementRepository(ctrl)
	wsmanMock := mocks.NewMockWSMAN(ctrl)
	wsmanMock.EXPECT().Worker().Return().AnyTimes()

	management := mocks.NewMockManagement(ctrl)

	uc := devices.New(repoMock, wsmanMock, mocks.NewMockRedirection(ctrl), logger.New("error"), mocks.MockCrypto{})
	repo := &WsmanComputerSystemRepo{usecase: uc, log: logger.New("error")}

	gomock.InOrder(
		repoMock.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil),
		wsmanMock.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(management, nil),
		management.EXPECT().GetAMTVersion().Return([]software.SoftwareIdentity{}, nil),
		management.EXPECT().GetSetupAndConfiguration().Return([]setupandconfiguration.SetupAndConfigurationServiceResponse{{ProvisioningMode: setupandconfiguration.AdminControlMode}}, nil),
	)

	err := repo.RequestKVMConsent(context.Background(), device.GUID)
	if !errors.Is(err, ErrKVMConsentNotRequiredInACM) {
		t.Fatalf("RequestKVMConsent() error = %v, wantErr %v", err, ErrKVMConsentNotRequiredInACM)
	}
}

func TestMapControlModeFromVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		version dto.Version
		want    string
	}{
		{
			name:    "admin mode from version",
			version: dto.Version{AMTSetupAndConfigurationService: dto.SetupAndConfigurationServiceResponses{Response: dto.SetupAndConfigurationServiceResponse{ProvisioningMode: setupandconfiguration.AdminControlMode}}},
			want:    controlModeACM,
		},
		{
			name:    "client mode from version",
			version: dto.Version{AMTSetupAndConfigurationService: dto.SetupAndConfigurationServiceResponses{Response: dto.SetupAndConfigurationServiceResponse{ProvisioningMode: setupandconfiguration.ClientControlMode}}},
			want:    controlModeCCM,
		},
		{
			name:    "unknown mode omitted",
			version: dto.Version{},
			want:    "",
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := mapControlModeFromVersion(tt.version)
			if got != tt.want {
				t.Fatalf("mapControlModeFromVersion() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSubmitKVMConsentCodeRepo_ReturnValueFailure(t *testing.T) {
	t.Parallel()

	device := &entity.Device{GUID: "system-1", TenantID: "tenant-1"}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repoMock := mocks.NewMockDeviceManagementRepository(ctrl)
	wsmanMock := mocks.NewMockWSMAN(ctrl)
	wsmanMock.EXPECT().Worker().Return().AnyTimes()

	management := mocks.NewMockManagement(ctrl)

	uc := devices.New(repoMock, wsmanMock, mocks.NewMockRedirection(ctrl), logger.New("error"), mocks.MockCrypto{})
	repo := &WsmanComputerSystemRepo{usecase: uc, log: logger.New("error")}

	repoMock.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
	wsmanMock.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(management, nil)
	management.EXPECT().SendConsentCode(123456).Return(optin.Response{
		Body: optin.Body{SendOptInCodeResponse: optin.SendOptInCode_OUTPUT{ReturnValue: 7}},
	}, nil)

	err := repo.SubmitKVMConsentCode(context.Background(), device.GUID, "123456")

	var consentErr *ConsentFailedError
	if !errors.As(err, &consentErr) {
		t.Fatalf("SubmitKVMConsentCode() error type = %T, want *ConsentFailedError", err)
	}
}

func TestGetAMTControlMode(t *testing.T) {
	t.Parallel()

	device := &entity.Device{GUID: "system-1", TenantID: "tenant-1"}

	tests := []struct {
		name     string
		setup    func(*testing.T, *mocks.MockDeviceManagementRepository, *mocks.MockWSMAN, *mocks.MockManagement, *entity.Device)
		wantMode string
	}{
		{
			name: "client control mode from version",
			setup: func(t *testing.T, repo *mocks.MockDeviceManagementRepository, wsmanMock *mocks.MockWSMAN, management *mocks.MockManagement, device *entity.Device) {
				t.Helper()

				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
				wsmanMock.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(management, nil)
				management.EXPECT().GetAMTVersion().Return([]software.SoftwareIdentity{}, nil)
				management.EXPECT().GetSetupAndConfiguration().Return([]setupandconfiguration.SetupAndConfigurationServiceResponse{{ProvisioningMode: setupandconfiguration.ClientControlMode}}, nil)
			},
			wantMode: controlModeCCM,
		},
		{
			name: "failure returns empty control mode",
			setup: func(t *testing.T, repo *mocks.MockDeviceManagementRepository, wsmanMock *mocks.MockWSMAN, management *mocks.MockManagement, device *entity.Device) {
				t.Helper()

				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
				wsmanMock.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(management, nil)
				management.EXPECT().GetAMTVersion().Return(nil, errors.New("boom"))
			},
			wantMode: "",
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			repoMock := mocks.NewMockDeviceManagementRepository(ctrl)
			wsmanMock := mocks.NewMockWSMAN(ctrl)
			wsmanMock.EXPECT().Worker().Return().AnyTimes()

			redirectionMock := mocks.NewMockRedirection(ctrl)
			management := mocks.NewMockManagement(ctrl)

			uc := devices.New(repoMock, wsmanMock, redirectionMock, logger.New("error"), mocks.MockCrypto{})
			repo := &WsmanComputerSystemRepo{usecase: uc, log: logger.New("error")}

			tt.setup(t, repoMock, wsmanMock, management, device)

			got := repo.getAMTControlMode(context.Background(), device.GUID)
			if got != tt.wantMode {
				t.Fatalf("getAMTControlMode() = %q, want %q", got, tt.wantMode)
			}
		})
	}
}

func TestCancelKVMConsentRepo_ReturnValueFailure(t *testing.T) {
	t.Parallel()

	device := &entity.Device{GUID: "system-1", TenantID: "tenant-1"}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repoMock := mocks.NewMockDeviceManagementRepository(ctrl)
	wsmanMock := mocks.NewMockWSMAN(ctrl)
	wsmanMock.EXPECT().Worker().Return().AnyTimes()

	management := mocks.NewMockManagement(ctrl)

	uc := devices.New(repoMock, wsmanMock, mocks.NewMockRedirection(ctrl), logger.New("error"), mocks.MockCrypto{})
	repo := &WsmanComputerSystemRepo{usecase: uc, log: logger.New("error")}

	repoMock.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
	wsmanMock.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(management, nil)
	management.EXPECT().CancelUserConsentRequest().Return(optin.Response{
		Body: optin.Body{CancelOptInResponse: optin.CancelOptIn_OUTPUT{ReturnValue: 9}},
	}, nil)

	err := repo.CancelKVMConsent(context.Background(), device.GUID)

	var consentErr *ConsentFailedError
	if !errors.As(err, &consentErr) {
		t.Fatalf("CancelKVMConsent() error type = %T, want *ConsentFailedError", err)
	}
}

func TestDetermineKVMStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		enableKVM    bool
		kvmAvailable bool
		userConsent  string
		optInState   int
		controlMode  string
		want         string
	}{
		{
			name:         "disabled when KVM not available",
			enableKVM:    true,
			kvmAvailable: false,
			userConsent:  "kvm",
			optInState:   int(optin.InSession),
			controlMode:  controlModeACM,
			want:         StateDisabled,
		},
		{
			name:         "disabled when KVM feature off",
			enableKVM:    false,
			kvmAvailable: true,
			userConsent:  "kvm",
			optInState:   int(optin.InSession),
			controlMode:  controlModeACM,
			want:         StateDisabled,
		},
		{
			name:         "active when consent required and in session",
			enableKVM:    true,
			kvmAvailable: true,
			userConsent:  "kvm",
			optInState:   int(optin.InSession),
			controlMode:  controlModeACM,
			want:         "Active",
		},
		{
			name:         "pending consent when requested",
			enableKVM:    true,
			kvmAvailable: true,
			userConsent:  "all",
			optInState:   int(optin.Requested),
			controlMode:  controlModeACM,
			want:         StateEnabled,
		},
		{
			name:         "enabled when consent received",
			enableKVM:    true,
			kvmAvailable: true,
			userConsent:  "all",
			optInState:   int(optin.Received),
			controlMode:  controlModeACM,
			want:         StateEnabled,
		},
		{
			name:         "error when consent required and unknown opt-in state",
			enableKVM:    true,
			kvmAvailable: true,
			userConsent:  "kvm",
			optInState:   999,
			controlMode:  controlModeACM,
			want:         StateEnabled,
		},
		{
			name:         "enabled when consent not required",
			enableKVM:    true,
			kvmAvailable: true,
			userConsent:  "none",
			optInState:   int(optin.NotStarted),
			controlMode:  controlModeACM,
			want:         StateEnabled,
		},
		{
			name:         "pending when consent flow is requested even if policy is none",
			enableKVM:    true,
			kvmAvailable: true,
			userConsent:  "none",
			optInState:   int(optin.Requested),
			controlMode:  controlModeACM,
			want:         StateEnabled,
		},
		{
			name:         "CCM requires consent even when configured none",
			enableKVM:    true,
			kvmAvailable: true,
			userConsent:  "none",
			optInState:   int(optin.Requested),
			controlMode:  controlModeCCM,
			want:         statusPendingConsent,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := determineKVMStatus(tt.enableKVM, tt.kvmAvailable, tt.userConsent, tt.optInState, tt.controlMode)
			if got != tt.want {
				t.Fatalf("determineKVMStatus() = %q, want %q", got, tt.want)
			}
		})
	}
}

func assertGraphicalConsoleOEM(t *testing.T, got *redfishv1.ComputerSystemHostGraphicalConsole, wantKVMStatus, wantUserConsentStatus string) {
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

	if amt.UserConsentStatus != wantUserConsentStatus {
		t.Errorf("UserConsentStatus = %q, want %q", amt.UserConsentStatus, wantUserConsentStatus)
	}
}

func assertGraphicalConsole(t *testing.T, got *redfishv1.ComputerSystemHostGraphicalConsole, wantEnabled bool, wantConnTypes []string, wantPort int64, wantKVMStatus, wantUserConsentStatus string) {
	t.Helper()

	if got == nil {
		t.Fatal("buildGraphicalConsole() returned nil")

		return
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

	assertGraphicalConsoleOEM(t, got, wantKVMStatus, wantUserConsentStatus)
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
		wantConsent   string
	}{
		{
			name:          "KVM not available - no connect types and no port",
			useTLS:        false,
			features:      dtov2.Features{EnableKVM: true, KVMAvailable: false, UserConsent: "none"},
			wantEnabled:   true,
			wantConnTypes: nil,
			wantPort:      0,
			wantKVMStatus: StateDisabled,
			wantConsent:   userConsentNotRequired,
		},
		{
			name:          "KVM available non-TLS port",
			useTLS:        false,
			features:      dtov2.Features{EnableKVM: true, KVMAvailable: true, UserConsent: "none"},
			wantEnabled:   true,
			wantConnTypes: kvmIP,
			wantPort:      16994,
			wantKVMStatus: StateEnabled,
			wantConsent:   userConsentNotRequired,
		},
		{
			name:          "KVM available TLS port",
			useTLS:        true,
			features:      dtov2.Features{EnableKVM: true, KVMAvailable: true, UserConsent: "none"},
			wantEnabled:   true,
			wantConnTypes: kvmIP,
			wantPort:      16995,
			wantKVMStatus: StateEnabled,
			wantConsent:   userConsentNotRequired,
		},
		{
			name:          "KVM disabled",
			useTLS:        false,
			features:      dtov2.Features{EnableKVM: false, KVMAvailable: true, UserConsent: "none"},
			wantEnabled:   false,
			wantConnTypes: kvmIP,
			wantPort:      16994,
			wantKVMStatus: StateDisabled,
			wantConsent:   userConsentNotRequired,
		},
		{
			name:          "consent required and in session - active",
			useTLS:        false,
			features:      dtov2.Features{EnableKVM: true, KVMAvailable: true, UserConsent: "kvm", OptInState: int(optin.InSession)},
			wantEnabled:   true,
			wantConnTypes: kvmIP,
			wantPort:      16994,
			wantKVMStatus: kvmStatusActive,
			wantConsent:   userConsentNotRequired,
		},
		{
			name:          "consent requested maps to Requested",
			useTLS:        false,
			features:      dtov2.Features{EnableKVM: true, KVMAvailable: true, UserConsent: "kvm", OptInState: int(optin.Requested)},
			wantEnabled:   true,
			wantConnTypes: kvmIP,
			wantPort:      16994,
			wantKVMStatus: StateEnabled,
			wantConsent:   userConsentNotRequired,
		},
		{
			name:          "consent flow requested with none policy is pending",
			useTLS:        false,
			features:      dtov2.Features{EnableKVM: true, KVMAvailable: true, UserConsent: "none", OptInState: int(optin.Requested)},
			wantEnabled:   true,
			wantConnTypes: kvmIP,
			wantPort:      16994,
			wantKVMStatus: StateEnabled,
			wantConsent:   userConsentNotRequired,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := repo.buildGraphicalConsole(tt.useTLS, tt.features, controlModeACM)
			assertGraphicalConsole(t, got, tt.wantEnabled, tt.wantConnTypes, tt.wantPort, tt.wantKVMStatus, tt.wantConsent)
		})
	}
}

func TestDetermineKVMUserConsentStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		userConsent string
		optInState  int
		controlMode string
		want        string
	}{
		{name: "kvm requested status", userConsent: "kvm", optInState: int(optin.Requested), want: userConsentRequested},
		{name: "all requested status", userConsent: "all", optInState: int(optin.Displayed), want: userConsentRequested},
		{name: "kvm uppercase required status", userConsent: "KVM", optInState: int(optin.NotStarted), want: userConsentRequired},
		{name: "all with spaces required status", userConsent: "  all  ", optInState: int(optin.NotStarted), want: userConsentRequired},
		{name: "received maps to granted", userConsent: "kvm", optInState: int(optin.Received), want: userConsentGranted},
		{name: "in-session maps to granted", userConsent: "kvm", optInState: int(optin.InSession), want: userConsentGranted},
		{name: "raw denied state maps to denied", userConsent: "kvm", optInState: optInStateDeniedRaw, want: userConsentDenied},
		{name: "raw timeout state maps to timeout", userConsent: "kvm", optInState: optInStateTimeoutRaw, want: userConsentTimeout},
		{name: "unknown required state falls back to requested", userConsent: "kvm", optInState: 999, want: userConsentRequested},
		{name: "none not required", userConsent: "none", want: userConsentNotRequired},
		{name: "none with requested flow maps to requested", userConsent: "none", optInState: int(optin.Requested), want: userConsentRequested},
		{name: "none with received flow maps to granted", userConsent: "none", optInState: int(optin.Received), want: userConsentGranted},
		{name: "empty not required", userConsent: "", want: userConsentNotRequired},
		{name: "CCM requires required when none configured", userConsent: "none", optInState: int(optin.NotStarted), controlMode: controlModeCCM, want: userConsentRequired},
		{name: "CCM with received maps to granted", userConsent: "none", optInState: int(optin.Received), controlMode: controlModeCCM, want: userConsentGranted},
		{name: "ACM requested remains not required", userConsent: "kvm", optInState: int(optin.Requested), controlMode: controlModeACM, want: userConsentNotRequired},
		{name: "ACM in-session remains not required", userConsent: "kvm", optInState: int(optin.InSession), controlMode: controlModeACM, want: userConsentNotRequired},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := determineKVMUserConsentStatus(tt.userConsent, tt.optInState, tt.controlMode)
			if got != tt.want {
				t.Fatalf("determineKVMUserConsentStatus() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDetermineSOLUserConsentStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		userConsent string
		optInState  int
		controlMode string
		want        string
	}{
		{name: "sol requested status", userConsent: "sol", optInState: int(optin.Requested), want: userConsentRequested},
		{name: "all requested status", userConsent: "all", optInState: int(optin.Displayed), want: userConsentRequested},
		{name: "sol uppercase required status", userConsent: "SOL", optInState: int(optin.NotStarted), want: userConsentRequired},
		{name: "all with spaces required status", userConsent: "  all  ", optInState: int(optin.NotStarted), want: userConsentRequired},
		{name: "received maps to granted", userConsent: "sol", optInState: int(optin.Received), want: userConsentGranted},
		{name: "in-session maps to granted", userConsent: "sol", optInState: int(optin.InSession), want: userConsentGranted},
		{name: "raw denied state maps to denied", userConsent: "sol", optInState: optInStateDeniedRaw, want: userConsentDenied},
		{name: "raw timeout state maps to timeout", userConsent: "sol", optInState: optInStateTimeoutRaw, want: userConsentTimeout},
		{name: "unknown required state falls back to requested", userConsent: "sol", optInState: 999, want: userConsentRequested},
		{name: "none not required", userConsent: "none", want: userConsentNotRequired},
		{name: "none with requested flow maps to requested", userConsent: "none", optInState: int(optin.Requested), want: userConsentRequested},
		{name: "none with received flow maps to granted", userConsent: "none", optInState: int(optin.Received), want: userConsentGranted},
		{name: "empty not required", userConsent: "", want: userConsentNotRequired},
		{name: "CCM requires required when none configured", userConsent: "none", optInState: int(optin.NotStarted), controlMode: controlModeCCM, want: userConsentRequired},
		{name: "CCM with received maps to granted", userConsent: "none", optInState: int(optin.Received), controlMode: controlModeCCM, want: userConsentGranted},
		{name: "ACM requested remains not required", userConsent: "sol", optInState: int(optin.Requested), controlMode: controlModeACM, want: userConsentNotRequired},
		{name: "ACM in-session remains not required", userConsent: "sol", optInState: int(optin.InSession), controlMode: controlModeACM, want: userConsentNotRequired},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := determineSOLUserConsentStatus(tt.userConsent, tt.optInState, tt.controlMode)
			if got != tt.want {
				t.Fatalf("determineSOLUserConsentStatus() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDetermineSOLStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		enableSOL    bool
		solAvailable bool
		userConsent  string
		optInState   int
		want         string
	}{
		{
			name:         "disabled when SOL not available",
			enableSOL:    true,
			solAvailable: false,
			userConsent:  "sol",
			optInState:   int(optin.InSession),
			want:         StateDisabled,
		},
		{
			name:         "disabled when SOL feature off",
			enableSOL:    false,
			solAvailable: true,
			userConsent:  "sol",
			optInState:   int(optin.InSession),
			want:         StateDisabled,
		},
		{
			name:         "active when consent required and in session",
			enableSOL:    true,
			solAvailable: true,
			userConsent:  "sol",
			optInState:   int(optin.InSession),
			want:         solStatusActive,
		},
		{
			name:         "pending consent when requested",
			enableSOL:    true,
			solAvailable: true,
			userConsent:  "all",
			optInState:   int(optin.Requested),
			want:         "PendingConsent",
		},
		{
			name:         "enabled when consent received",
			enableSOL:    true,
			solAvailable: true,
			userConsent:  "all",
			optInState:   int(optin.Received),
			want:         StateEnabled,
		},
		{
			name:         "error when consent required and unknown opt-in state",
			enableSOL:    true,
			solAvailable: true,
			userConsent:  "sol",
			optInState:   999,
			want:         "Error",
		},
		{
			name:         "enabled when consent not required",
			enableSOL:    true,
			solAvailable: true,
			userConsent:  "none",
			optInState:   int(optin.NotStarted),
			want:         StateEnabled,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := determineSOLStatus(tt.enableSOL, tt.solAvailable, tt.userConsent, tt.optInState)
			if got != tt.want {
				t.Fatalf("determineSOLStatus() = %q, want %q", got, tt.want)
			}
		})
	}
}

func boolToListener(enabled bool) int {
	if enabled {
		return 1
	}

	return 0
}

func expectGetFeaturesSuccess(t *testing.T, repo *mocks.MockDeviceManagementRepository, wsmanMock *mocks.MockWSMAN, management *mocks.MockManagement, device *entity.Device) {
	t.Helper()

	repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
	wsmanMock.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(management, nil)
	management.EXPECT().GetAMTRedirectionService().Return(redirection.Response{
		Body: redirection.Body{
			GetAndPutResponse: redirection.RedirectionResponse{
				EnabledState:    32771,
				ListenerEnabled: true,
			},
		},
	}, nil)
	management.EXPECT().GetIPSOptInService().Return(optin.Response{
		Body: optin.Body{
			GetAndPutResponse: optin.OptInServiceResponse{
				OptInRequired: 1,
				OptInState:    1,
			},
		},
	}, nil)

	management.EXPECT().GetKVMRedirection().Return(kvm.Response{
		Body: kvm.Body{
			GetResponse: kvm.KVMRedirectionSAP{EnabledState: kvm.EnabledState(0)},
		},
	}, nil)
	management.EXPECT().GetBootService().Return(cimBoot.BootService{EnabledState: 32769}, nil)
	management.EXPECT().GetCIMBootSourceSetting().Return(cimBoot.Response{
		Body: cimBoot.Body{
			PullResponse: cimBoot.PullResponse{
				BootSourceSettingItems: []cimBoot.BootSourceSetting{
					{InstanceID: "Intel(r) AMT: Force OCR UEFI HTTPS Boot"},
					{InstanceID: "Intel(r) AMT: Force OCR UEFI Boot Option"},
				},
			},
		},
	}, nil)
	management.EXPECT().GetPowerCapabilities().Return(boot.BootCapabilitiesResponse{
		ForceUEFIHTTPSBoot:    true,
		ForceWinREBoot:        false,
		ForceUEFILocalPBABoot: false,
	}, nil)
	management.EXPECT().GetBootData().Return(boot.BootSettingDataResponse{
		UEFIHTTPSBootEnabled:    true,
		WinREBootEnabled:        false,
		UEFILocalPBABootEnabled: false,
	}, nil)
}

func expectSetFeaturesSuccess(t *testing.T, repo *mocks.MockDeviceManagementRepository, wsmanMock *mocks.MockWSMAN, management *mocks.MockManagement, device *entity.Device, enableKVM bool) {
	t.Helper()

	repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
	wsmanMock.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(management, nil)
	management.EXPECT().RequestAMTRedirectionServiceStateChange(true, true).Return(redirection.EnableIDERAndSOL, 1, nil)
	management.EXPECT().SetKVMRedirection(enableKVM).Return(boolToListener(enableKVM), nil)
	management.EXPECT().GetAMTRedirectionService().Return(redirection.Response{
		Body: redirection.Body{
			GetAndPutResponse: redirection.RedirectionResponse{
				EnabledState:    32771,
				ListenerEnabled: true,
			},
		},
	}, nil)
	management.EXPECT().SetAMTRedirectionService(&redirection.RedirectionRequest{
		EnabledState:    redirection.EnabledState(redirection.EnableIDERAndSOL),
		ListenerEnabled: true,
	}).Return(redirection.Response{}, nil)
	management.EXPECT().GetIPSOptInService().Return(optin.Response{
		Body: optin.Body{
			GetAndPutResponse: optin.OptInServiceResponse{
				OptInRequired: 1,
				OptInState:    0,
			},
		},
	}, nil)
	management.EXPECT().SetIPSOptInService(optin.OptInServiceRequest{
		OptInRequired: 1,
		OptInState:    0,
	}).Return(nil)
	management.EXPECT().BootServiceStateChange(32769).Return(cimBoot.BootService{}, nil)
	management.EXPECT().GetBootService().Return(cimBoot.BootService{EnabledState: 32769}, nil)
	management.EXPECT().GetCIMBootSourceSetting().Return(cimBoot.Response{
		Body: cimBoot.Body{
			PullResponse: cimBoot.PullResponse{
				BootSourceSettingItems: []cimBoot.BootSourceSetting{
					{InstanceID: "Intel(r) AMT: Force OCR UEFI HTTPS Boot"},
					{InstanceID: "Intel(r) AMT: Force OCR UEFI Boot Option"},
				},
			},
		},
	}, nil)
	management.EXPECT().GetPowerCapabilities().Return(boot.BootCapabilitiesResponse{
		ForceUEFIHTTPSBoot:    true,
		ForceWinREBoot:        false,
		ForceUEFILocalPBABoot: false,
	}, nil)
	management.EXPECT().GetBootData().Return(boot.BootSettingDataResponse{
		UEFIHTTPSBootEnabled:    true,
		WinREBootEnabled:        false,
		UEFILocalPBABootEnabled: false,
	}, nil)
}

func expectSetFeaturesSuccessSOL(t *testing.T, repo *mocks.MockDeviceManagementRepository, wsmanMock *mocks.MockWSMAN, management *mocks.MockManagement, device *entity.Device, enableSOL bool) {
	t.Helper()

	repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
	wsmanMock.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(management, nil)

	requestedState := redirection.IDERIsEnabledAndSOLIsDisabled
	if enableSOL {
		requestedState = redirection.IDERAndSOLAreEnabled
	}

	management.EXPECT().RequestAMTRedirectionServiceStateChange(true, enableSOL).Return(redirection.RequestedState(requestedState), 1, nil)
	management.EXPECT().SetKVMRedirection(false).Return(0, nil)
	management.EXPECT().GetAMTRedirectionService().Return(redirection.Response{
		Body: redirection.Body{
			GetAndPutResponse: redirection.RedirectionResponse{
				EnabledState:    32771,
				ListenerEnabled: true,
			},
		},
	}, nil)
	management.EXPECT().SetAMTRedirectionService(&redirection.RedirectionRequest{
		EnabledState:    requestedState,
		ListenerEnabled: true,
	}).Return(redirection.Response{}, nil)
	management.EXPECT().GetIPSOptInService().Return(optin.Response{
		Body: optin.Body{
			GetAndPutResponse: optin.OptInServiceResponse{
				OptInRequired: 1,
				OptInState:    0,
			},
		},
	}, nil)
	management.EXPECT().SetIPSOptInService(optin.OptInServiceRequest{
		OptInRequired: 1,
		OptInState:    0,
	}).Return(nil)
	management.EXPECT().BootServiceStateChange(32769).Return(cimBoot.BootService{}, nil)
	management.EXPECT().GetBootService().Return(cimBoot.BootService{EnabledState: 32769}, nil)
	management.EXPECT().GetCIMBootSourceSetting().Return(cimBoot.Response{
		Body: cimBoot.Body{
			PullResponse: cimBoot.PullResponse{
				BootSourceSettingItems: []cimBoot.BootSourceSetting{
					{InstanceID: "Intel(r) AMT: Force OCR UEFI HTTPS Boot"},
					{InstanceID: "Intel(r) AMT: Force OCR UEFI Boot Option"},
				},
			},
		},
	}, nil)
	management.EXPECT().GetPowerCapabilities().Return(boot.BootCapabilitiesResponse{
		ForceUEFIHTTPSBoot:    true,
		ForceWinREBoot:        false,
		ForceUEFILocalPBABoot: false,
	}, nil)
	management.EXPECT().GetBootData().Return(boot.BootSettingDataResponse{
		UEFIHTTPSBootEnabled:    true,
		WinREBootEnabled:        false,
		UEFILocalPBABootEnabled: false,
	}, nil)
}

func TestUpdateGraphicalConsoleServiceEnabledRepo(t *testing.T) {
	t.Parallel()

	errDeviceNotFound := errors.New(ErrMsgDeviceNotFound)
	errAMT := errors.New("amt refused")

	tests := []struct {
		name    string
		enabled bool
		setup   func(*testing.T, *mocks.MockDeviceManagementRepository, *mocks.MockWSMAN, *mocks.MockManagement, *mocks.MockManagement, *entity.Device)
		wantErr error
	}{
		{
			name:    "enable success",
			enabled: true,
			setup: func(
				t *testing.T,
				repo *mocks.MockDeviceManagementRepository,
				wsmanMock *mocks.MockWSMAN,
				getMgmt *mocks.MockManagement,
				setMgmt *mocks.MockManagement,
				device *entity.Device,
			) {
				t.Helper()
				expectGetFeaturesSuccess(t, repo, wsmanMock, getMgmt, device)
				expectSetFeaturesSuccess(t, repo, wsmanMock, setMgmt, device, true)
			},
		},
		{
			name:    "GetFeatures device not found",
			enabled: true,
			setup: func(
				_ *testing.T,
				repo *mocks.MockDeviceManagementRepository,
				wsmanMock *mocks.MockWSMAN,
				getMgmt *mocks.MockManagement,
				_ *mocks.MockManagement,
				device *entity.Device,
			) {
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
				wsmanMock.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(getMgmt, errDeviceNotFound)
			},
			wantErr: ErrSystemNotFound,
		},
		{
			name:    "GetFeatures generic error",
			enabled: true,
			setup: func(
				_ *testing.T,
				repo *mocks.MockDeviceManagementRepository,
				wsmanMock *mocks.MockWSMAN,
				getMgmt *mocks.MockManagement,
				_ *mocks.MockManagement,
				device *entity.Device,
			) {
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
				wsmanMock.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(getMgmt, errAMT)
			},
			wantErr: errAMT,
		},
		{
			name:    "SetFeatures device not found",
			enabled: true,
			setup: func(
				t *testing.T,
				repo *mocks.MockDeviceManagementRepository,
				wsmanMock *mocks.MockWSMAN,
				getMgmt *mocks.MockManagement,
				setMgmt *mocks.MockManagement,
				device *entity.Device,
			) {
				t.Helper()
				expectGetFeaturesSuccess(t, repo, wsmanMock, getMgmt, device)
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
				wsmanMock.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(setMgmt, errDeviceNotFound)
			},
			wantErr: ErrSystemNotFound,
		},
		{
			name:    "SetFeatures generic error",
			enabled: true,
			setup: func(
				t *testing.T,
				repo *mocks.MockDeviceManagementRepository,
				wsmanMock *mocks.MockWSMAN,
				getMgmt *mocks.MockManagement,
				setMgmt *mocks.MockManagement,
				device *entity.Device,
			) {
				t.Helper()
				expectGetFeaturesSuccess(t, repo, wsmanMock, getMgmt, device)
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
				wsmanMock.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(setMgmt, errAMT)
			},
			wantErr: errAMT,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			device := &entity.Device{GUID: "system-1", TenantID: "tenant-1"}
			repoMock := mocks.NewMockDeviceManagementRepository(ctrl)
			wsmanMock := mocks.NewMockWSMAN(ctrl)
			wsmanMock.EXPECT().Worker().Return().AnyTimes()

			getMgmt := mocks.NewMockManagement(ctrl)
			setMgmt := mocks.NewMockManagement(ctrl)

			uc := devices.New(repoMock, wsmanMock, mocks.NewMockRedirection(ctrl), logger.New("error"), mocks.MockCrypto{})
			repo := &WsmanComputerSystemRepo{usecase: uc, log: logger.New("error")}

			tt.setup(t, repoMock, wsmanMock, getMgmt, setMgmt, device)

			err := repo.UpdateGraphicalConsoleServiceEnabled(context.Background(), device.GUID, tt.enabled)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("UpdateGraphicalConsoleServiceEnabled() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUpdateSerialConsoleServiceEnabledRepo(t *testing.T) {
	t.Parallel()

	errDeviceNotFound := errors.New(ErrMsgDeviceNotFound)
	errAMT := errors.New("amt refused")

	tests := []struct {
		name    string
		enabled bool
		setup   func(*testing.T, *mocks.MockDeviceManagementRepository, *mocks.MockWSMAN, *mocks.MockManagement, *mocks.MockManagement, *entity.Device)
		wantErr error
	}{
		{
			name:    "success",
			enabled: true,
			setup: func(
				t *testing.T,
				repo *mocks.MockDeviceManagementRepository,
				wsmanMock *mocks.MockWSMAN,
				getMgmt *mocks.MockManagement,
				setMgmt *mocks.MockManagement,
				device *entity.Device,
			) {
				t.Helper()
				expectGetFeaturesSuccess(t, repo, wsmanMock, getMgmt, device)
				expectSetFeaturesSuccessSOL(t, repo, wsmanMock, setMgmt, device, true)
			},
		},
		{
			name:    "disable success",
			enabled: false,
			setup: func(
				t *testing.T,
				repo *mocks.MockDeviceManagementRepository,
				wsmanMock *mocks.MockWSMAN,
				getMgmt *mocks.MockManagement,
				setMgmt *mocks.MockManagement,
				device *entity.Device,
			) {
				t.Helper()
				expectGetFeaturesSuccess(t, repo, wsmanMock, getMgmt, device)
				expectSetFeaturesSuccessSOL(t, repo, wsmanMock, setMgmt, device, false)
			},
		},
		{
			name:    "GetFeatures device not found",
			enabled: true,
			setup: func(
				_ *testing.T,
				repo *mocks.MockDeviceManagementRepository,
				wsmanMock *mocks.MockWSMAN,
				getMgmt *mocks.MockManagement,
				_ *mocks.MockManagement,
				device *entity.Device,
			) {
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
				wsmanMock.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(getMgmt, errDeviceNotFound)
			},
			wantErr: ErrSystemNotFound,
		},
		{
			name:    "GetFeatures generic error",
			enabled: true,
			setup: func(
				_ *testing.T,
				repo *mocks.MockDeviceManagementRepository,
				wsmanMock *mocks.MockWSMAN,
				getMgmt *mocks.MockManagement,
				_ *mocks.MockManagement,
				device *entity.Device,
			) {
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
				wsmanMock.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(getMgmt, errAMT)
			},
			wantErr: errAMT,
		},
		{
			name:    "SetFeatures device not found",
			enabled: true,
			setup: func(
				t *testing.T,
				repo *mocks.MockDeviceManagementRepository,
				wsmanMock *mocks.MockWSMAN,
				getMgmt *mocks.MockManagement,
				setMgmt *mocks.MockManagement,
				device *entity.Device,
			) {
				t.Helper()
				expectGetFeaturesSuccess(t, repo, wsmanMock, getMgmt, device)
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
				wsmanMock.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(setMgmt, errDeviceNotFound)
			},
			wantErr: ErrSystemNotFound,
		},
		{
			name:    "SetFeatures generic error",
			enabled: true,
			setup: func(
				t *testing.T,
				repo *mocks.MockDeviceManagementRepository,
				wsmanMock *mocks.MockWSMAN,
				getMgmt *mocks.MockManagement,
				setMgmt *mocks.MockManagement,
				device *entity.Device,
			) {
				t.Helper()
				expectGetFeaturesSuccess(t, repo, wsmanMock, getMgmt, device)
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
				wsmanMock.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(setMgmt, errAMT)
			},
			wantErr: errAMT,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			device := &entity.Device{GUID: "system-1", TenantID: "tenant-1"}
			repoMock := mocks.NewMockDeviceManagementRepository(ctrl)
			wsmanMock := mocks.NewMockWSMAN(ctrl)
			wsmanMock.EXPECT().Worker().Return().AnyTimes()

			getMgmt := mocks.NewMockManagement(ctrl)
			setMgmt := mocks.NewMockManagement(ctrl)

			uc := devices.New(repoMock, wsmanMock, mocks.NewMockRedirection(ctrl), logger.New("error"), mocks.MockCrypto{})
			repo := &WsmanComputerSystemRepo{usecase: uc, log: logger.New("error")}

			tt.setup(t, repoMock, wsmanMock, getMgmt, setMgmt, device)

			err := repo.UpdateSerialConsoleServiceEnabled(context.Background(), device.GUID, tt.enabled)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("UpdateSerialConsoleServiceEnabled() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRequestKVMConsentRepo(t *testing.T) {
	t.Parallel()

	errDeviceNotFound := errors.New(ErrMsgDeviceNotFound)
	errAMT := errors.New("amt refused")
	device := &entity.Device{GUID: "system-1", TenantID: "tenant-1"}

	tests := []struct {
		name    string
		setup   func(*mocks.MockDeviceManagementRepository, *mocks.MockWSMAN, *mocks.MockManagement)
		wantErr error
	}{
		{
			name: "success",
			setup: func(repoMock *mocks.MockDeviceManagementRepository, wsmanMock *mocks.MockWSMAN, management *mocks.MockManagement) {
				gomock.InOrder(
					repoMock.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil),
					wsmanMock.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(management, nil),
					management.EXPECT().GetAMTVersion().Return([]software.SoftwareIdentity{}, nil),
					management.EXPECT().GetSetupAndConfiguration().Return([]setupandconfiguration.SetupAndConfigurationServiceResponse{{ProvisioningMode: setupandconfiguration.ClientControlMode}}, nil),
					repoMock.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil),
					wsmanMock.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(management, nil),
					management.EXPECT().GetUserConsentCode().Return(optin.Response{}, nil),
				)
			},
		},
		{
			name: "device not found",
			setup: func(repoMock *mocks.MockDeviceManagementRepository, _ *mocks.MockWSMAN, _ *mocks.MockManagement) {
				repoMock.EXPECT().GetByID(context.Background(), device.GUID, "").Return(nil, errDeviceNotFound).Times(2)
			},
			wantErr: ErrSystemNotFound,
		},
		{
			name: "wsman error",
			setup: func(repoMock *mocks.MockDeviceManagementRepository, wsmanMock *mocks.MockWSMAN, management *mocks.MockManagement) {
				repoMock.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil).Times(2)
				wsmanMock.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(management, errAMT).Times(2)
			},
			wantErr: errAMT,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			repoMock := mocks.NewMockDeviceManagementRepository(ctrl)
			wsmanMock := mocks.NewMockWSMAN(ctrl)
			wsmanMock.EXPECT().Worker().Return().AnyTimes()

			management := mocks.NewMockManagement(ctrl)

			uc := devices.New(repoMock, wsmanMock, mocks.NewMockRedirection(ctrl), logger.New("error"), mocks.MockCrypto{})
			repo := &WsmanComputerSystemRepo{usecase: uc, log: logger.New("error")}

			tt.setup(repoMock, wsmanMock, management)

			err := repo.RequestKVMConsent(context.Background(), device.GUID)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("RequestKVMConsent() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSubmitKVMConsentCodeRepo(t *testing.T) {
	t.Parallel()

	errDeviceNotFound := errors.New(ErrMsgDeviceNotFound)
	errAMT := errors.New("amt refused")
	device := &entity.Device{GUID: "system-1", TenantID: "tenant-1"}

	tests := []struct {
		name    string
		setup   func(*mocks.MockDeviceManagementRepository, *mocks.MockWSMAN, *mocks.MockManagement)
		wantErr error
		code    string
	}{
		{
			name: "success",
			code: "123456",
			setup: func(repoMock *mocks.MockDeviceManagementRepository, wsmanMock *mocks.MockWSMAN, management *mocks.MockManagement) {
				repoMock.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
				wsmanMock.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(management, nil)
				management.EXPECT().SendConsentCode(123456).Return(optin.Response{}, nil)
			},
		},
		{
			name: "device not found",
			code: "123456",
			setup: func(repoMock *mocks.MockDeviceManagementRepository, _ *mocks.MockWSMAN, _ *mocks.MockManagement) {
				repoMock.EXPECT().GetByID(context.Background(), device.GUID, "").Return(nil, errDeviceNotFound)
			},
			wantErr: ErrSystemNotFound,
		},
		{
			name: "wsman error",
			code: "123456",
			setup: func(repoMock *mocks.MockDeviceManagementRepository, wsmanMock *mocks.MockWSMAN, management *mocks.MockManagement) {
				repoMock.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
				wsmanMock.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(management, errAMT)
			},
			wantErr: errAMT,
		},
		{
			name: "invalid consent code",
			code: "12ab",
			setup: func(_ *mocks.MockDeviceManagementRepository, _ *mocks.MockWSMAN, _ *mocks.MockManagement) {
			},
			wantErr: ErrInvalidConsentCode,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			repoMock := mocks.NewMockDeviceManagementRepository(ctrl)
			wsmanMock := mocks.NewMockWSMAN(ctrl)
			wsmanMock.EXPECT().Worker().Return().AnyTimes()

			management := mocks.NewMockManagement(ctrl)

			uc := devices.New(repoMock, wsmanMock, mocks.NewMockRedirection(ctrl), logger.New("error"), mocks.MockCrypto{})
			repo := &WsmanComputerSystemRepo{usecase: uc, log: logger.New("error")}

			tt.setup(repoMock, wsmanMock, management)

			err := repo.SubmitKVMConsentCode(context.Background(), device.GUID, tt.code)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("SubmitKVMConsentCode() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCancelKVMConsentRepo(t *testing.T) {
	t.Parallel()

	errDeviceNotFound := errors.New(ErrMsgDeviceNotFound)
	errAMT := errors.New("amt refused")
	device := &entity.Device{GUID: "system-1", TenantID: "tenant-1"}

	tests := []struct {
		name    string
		setup   func(*mocks.MockDeviceManagementRepository, *mocks.MockWSMAN, *mocks.MockManagement)
		wantErr error
	}{
		{
			name: "success",
			setup: func(repoMock *mocks.MockDeviceManagementRepository, wsmanMock *mocks.MockWSMAN, management *mocks.MockManagement) {
				repoMock.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
				wsmanMock.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(management, nil)
				management.EXPECT().CancelUserConsentRequest().Return(optin.Response{}, nil)
			},
		},
		{
			name: "device not found",
			setup: func(repoMock *mocks.MockDeviceManagementRepository, _ *mocks.MockWSMAN, _ *mocks.MockManagement) {
				repoMock.EXPECT().GetByID(context.Background(), device.GUID, "").Return(nil, errDeviceNotFound)
			},
			wantErr: ErrSystemNotFound,
		},
		{
			name: "wsman error",
			setup: func(repoMock *mocks.MockDeviceManagementRepository, wsmanMock *mocks.MockWSMAN, management *mocks.MockManagement) {
				repoMock.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
				wsmanMock.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(management, errAMT)
			},
			wantErr: errAMT,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			repoMock := mocks.NewMockDeviceManagementRepository(ctrl)
			wsmanMock := mocks.NewMockWSMAN(ctrl)
			wsmanMock.EXPECT().Worker().Return().AnyTimes()

			management := mocks.NewMockManagement(ctrl)

			uc := devices.New(repoMock, wsmanMock, mocks.NewMockRedirection(ctrl), logger.New("error"), mocks.MockCrypto{})
			repo := &WsmanComputerSystemRepo{usecase: uc, log: logger.New("error")}

			tt.setup(repoMock, wsmanMock, management)

			err := repo.CancelKVMConsent(context.Background(), device.GUID)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("CancelKVMConsent() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsSixDigitNumeric(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		code string
		want bool
	}{
		{name: "valid six digits", code: "123456", want: true},
		{name: "too short", code: "12345", want: false},
		{name: "too long", code: "1234567", want: false},
		{name: "contains non digit", code: "12a456", want: false},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := isSixDigitNumeric(tt.code)
			if got != tt.want {
				t.Fatalf("isSixDigitNumeric(%q) = %v, want %v", tt.code, got, tt.want)
			}
		})
	}
}

func assertSerialConsoleOEM(t *testing.T, got *redfishv1.ComputerSystemHostSerialConsole, wantSOLStatus, wantControlMode, wantUserConsentStatus string) {
	t.Helper()

	if got.OEM == nil || got.OEM.Intel == nil || got.OEM.Intel.AMT == nil {
		t.Fatal("OEM.Intel.AMT is nil")
	}

	amt := got.OEM.Intel.AMT

	if amt.SOLStatus != wantSOLStatus {
		t.Errorf("SOLStatus = %q, want %q", amt.SOLStatus, wantSOLStatus)
	}

	if amt.ControlMode != wantControlMode {
		t.Errorf("ControlMode = %q, want %q", amt.ControlMode, wantControlMode)
	}

	if amt.UserConsentStatus != wantUserConsentStatus {
		t.Errorf("UserConsentStatus = %q, want %q", amt.UserConsentStatus, wantUserConsentStatus)
	}
}

func assertSerialConsole(t *testing.T, got *redfishv1.ComputerSystemHostSerialConsole, wantEnabled bool, wantURI, wantSOLStatus, wantControlMode, wantUserConsentStatus string) {
	t.Helper()

	if got == nil {
		t.Fatal("buildSerialConsole() returned nil")

		return
	}

	if got.MaxConcurrentSessions == nil || *got.MaxConcurrentSessions != 1 {
		t.Errorf("MaxConcurrentSessions = %v, want 1", got.MaxConcurrentSessions)
	}

	if got.WebSocket == nil {
		t.Fatal("WebSocket is nil")

		return
	}

	if got.WebSocket.ServiceEnabled == nil || *got.WebSocket.ServiceEnabled != wantEnabled {
		t.Errorf("ServiceEnabled = %v, want %v", got.WebSocket.ServiceEnabled, wantEnabled)
	}

	if got.WebSocket.Interactive == nil || !*got.WebSocket.Interactive {
		t.Errorf("Interactive = %v, want true", got.WebSocket.Interactive)
	}

	if wantURI == "" {
		if got.WebSocket.ConsoleURI != nil {
			t.Errorf("ConsoleURI = %v, want nil", got.WebSocket.ConsoleURI)
		}
	} else {
		if got.WebSocket.ConsoleURI == nil || *got.WebSocket.ConsoleURI != wantURI {
			t.Errorf("ConsoleURI = %v, want %q", got.WebSocket.ConsoleURI, wantURI)
		}
	}

	assertSerialConsoleOEM(t, got, wantSOLStatus, wantControlMode, wantUserConsentStatus)
}

func TestBuildSerialConsole(t *testing.T) {
	t.Parallel()

	repo := &WsmanComputerSystemRepo{log: logger.New("error")}

	tests := []struct {
		name                  string
		systemID              string
		features              dtov2.Features
		controlMode           string
		wantEnabled           bool
		wantURI               string
		wantSOLStatus         string
		wantControlMode       string
		wantUserConsentStatus string
	}{
		{
			name:                  "SOL not available - no URI",
			systemID:              "system-1",
			features:              dtov2.Features{EnableSOL: true, Redirection: false, UserConsent: "none"},
			controlMode:           controlModeACM,
			wantEnabled:           true,
			wantURI:               "",
			wantSOLStatus:         StateDisabled,
			wantControlMode:       controlModeACM,
			wantUserConsentStatus: userConsentNotRequired,
		},
		{
			name:                  "SOL available with URI",
			systemID:              "system-1",
			features:              dtov2.Features{EnableSOL: true, Redirection: true, UserConsent: "none"},
			controlMode:           controlModeACM,
			wantEnabled:           true,
			wantURI:               "/relay/webrelay.ashx?host=system-1&mode=sol",
			wantSOLStatus:         StateEnabled,
			wantControlMode:       controlModeACM,
			wantUserConsentStatus: userConsentNotRequired,
		},
		{
			name:                  "SOL disabled",
			systemID:              "system-1",
			features:              dtov2.Features{EnableSOL: false, Redirection: true, UserConsent: "none"},
			controlMode:           controlModeACM,
			wantEnabled:           false,
			wantURI:               "/relay/webrelay.ashx?host=system-1&mode=sol",
			wantSOLStatus:         StateDisabled,
			wantControlMode:       controlModeACM,
			wantUserConsentStatus: userConsentNotRequired,
		},
		{
			name:                  "consent required and in session - active",
			systemID:              "system-1",
			features:              dtov2.Features{EnableSOL: true, Redirection: true, UserConsent: "sol", OptInState: int(optin.InSession)},
			controlMode:           controlModeCCM,
			wantEnabled:           true,
			wantURI:               "/relay/webrelay.ashx?host=system-1&mode=sol",
			wantSOLStatus:         solStatusActive,
			wantControlMode:       controlModeCCM,
			wantUserConsentStatus: userConsentGranted,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := repo.buildSerialConsole(tt.systemID, tt.features, tt.controlMode)
			assertSerialConsole(t, got, tt.wantEnabled, tt.wantURI, tt.wantSOLStatus, tt.wantControlMode, tt.wantUserConsentStatus)
		})
	}
}

func TestGetByIDSkipsConsoleEnrichmentWhenTimeBudgetIsLow(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	device := &entity.Device{GUID: "system-1", TenantID: "tenant-1", Password: "encrypted"}
	repoMock := mocks.NewMockDeviceManagementRepository(ctrl)
	wsmanMock := mocks.NewMockWSMAN(ctrl)
	management := mocks.NewMockManagement(ctrl)

	wsmanMock.EXPECT().Worker().Return().AnyTimes()
	repoMock.EXPECT().GetByID(gomock.Any(), device.GUID, "").Return(device, nil).Times(3)
	wsmanMock.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(management, nil).Times(2)
	management.EXPECT().GetPowerState().Return([]cimservice.CIM_AssociatedPowerManagementService{{PowerState: cimmodels.PowerState(redfishv1.CIMPowerStateOn)}}, nil)
	management.EXPECT().GetOSPowerSavingState().Return(ipspower.OSPowerSavingState(0), nil)
	management.EXPECT().GetHardwareInfo().Return(wsmanAPI.HWResults{}, nil)

	uc := devices.New(repoMock, wsmanMock, mocks.NewMockRedirection(ctrl), logger.New("error"), mocks.MockCrypto{})
	repo := &WsmanComputerSystemRepo{usecase: uc, log: logger.New("error")}

	ctx := createContextWithDeadline(t, 1*time.Second)

	got, err := repo.GetByID(ctx, device.GUID)
	if err != nil {
		t.Fatalf("GetByID() error = %v, want nil", err)
	}

	if got == nil {
		t.Fatal("GetByID() returned nil system")
	}

	if got.PowerState != redfishv1.PowerStateOn {
		t.Fatalf("PowerState = %q, want %q", got.PowerState, redfishv1.PowerStateOn)
	}

	if got.GraphicalConsole != nil {
		t.Fatalf("GraphicalConsole = %#v, want nil when time budget is insufficient", got.GraphicalConsole)
	}

	if got.SerialConsole != nil {
		t.Fatalf("SerialConsole = %#v, want nil when time budget is insufficient", got.SerialConsole)
	}
}

func TestGetByIDFallsBackToCachedConsoleData(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	device := &entity.Device{
		GUID:       "system-1",
		TenantID:   "tenant-1",
		Password:   "encrypted",
		UseTLS:     true,
		DeviceInfo: `{"currentMode":"Admin Control Mode","features":"{\"userConsent\":\"none\",\"enableSOL\":true,\"enableIDER\":false,\"enableKVM\":true,\"redirection\":true,\"optInState\":0,\"kvmAvailable\":true,\"httpBoot\":false,\"httpBootSupported\":false,\"winREBootSupported\":false,\"localPBABootSupported\":false}"}`,
	}

	repoMock := mocks.NewMockDeviceManagementRepository(ctrl)
	wsmanMock := mocks.NewMockWSMAN(ctrl)
	phaseOneManagement := mocks.NewMockManagement(ctrl)
	phaseTwoManagement := mocks.NewMockManagement(ctrl)

	wsmanMock.EXPECT().Worker().Return().AnyTimes()
	repoMock.EXPECT().GetByID(gomock.Any(), device.GUID, "").Return(device, nil).Times(5)

	var setupCallCount atomic.Int32

	setupWsmanClientCall := wsmanMock.EXPECT().
		SetupWsmanClient(gomock.Any(), gomock.Any(), false, true)

	setupWsmanClientCall.DoAndReturn(
		func(_ context.Context, _ entity.Device, _, _ bool) (wsmanAPI.Management, error) {
			if setupCallCount.Add(1) <= 2 {
				return phaseOneManagement, nil
			}

			return phaseTwoManagement, nil
		},
	).Times(4)

	phaseOneManagement.EXPECT().GetPowerState().Return([]cimservice.CIM_AssociatedPowerManagementService{{PowerState: cimmodels.PowerState(redfishv1.CIMPowerStateOn)}}, nil)
	phaseOneManagement.EXPECT().GetOSPowerSavingState().Return(ipspower.OSPowerSavingState(0), nil)
	phaseOneManagement.EXPECT().GetHardwareInfo().Return(wsmanAPI.HWResults{}, errors.New("hardware unavailable"))

	phaseTwoManagement.EXPECT().GetAMTRedirectionService().Return(redirection.Response{}, errors.New("features unavailable"))
	phaseTwoManagement.EXPECT().GetAMTVersion().Return(nil, errors.New("version unavailable"))

	uc := devices.New(repoMock, wsmanMock, mocks.NewMockRedirection(ctrl), logger.New("error"), mocks.MockCrypto{})
	repo := &WsmanComputerSystemRepo{usecase: uc, log: logger.New("error")}

	got, err := repo.GetByID(context.Background(), device.GUID)
	if err != nil {
		t.Fatalf("GetByID() error = %v, want nil", err)
	}

	if got == nil {
		t.Fatal("GetByID() returned nil system")
	}

	assertGraphicalConsole(t, got.GraphicalConsole, true, []string{kvmConnectTypeKVMIP}, 16995, StateEnabled, userConsentNotRequired)
	assertSerialConsole(t, got.SerialConsole, true, "/relay/webrelay.ashx?host=system-1&mode=sol", StateEnabled, controlModeACM, userConsentNotRequired)
}

func TestGetByIDUsesSafeDefaultsWhenConsoleEnrichmentFailsWithoutCache(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	device := &entity.Device{GUID: "system-1", TenantID: "tenant-1", Password: "encrypted"}
	repoMock := mocks.NewMockDeviceManagementRepository(ctrl)
	wsmanMock := mocks.NewMockWSMAN(ctrl)
	phaseOneManagement := mocks.NewMockManagement(ctrl)
	phaseTwoManagement := mocks.NewMockManagement(ctrl)

	wsmanMock.EXPECT().Worker().Return().AnyTimes()
	repoMock.EXPECT().GetByID(gomock.Any(), device.GUID, "").Return(device, nil).Times(5)

	var setupCallCount atomic.Int32

	setupWsmanClientCall := wsmanMock.EXPECT().
		SetupWsmanClient(gomock.Any(), gomock.Any(), false, true)

	setupWsmanClientCall.DoAndReturn(
		func(_ context.Context, _ entity.Device, _, _ bool) (wsmanAPI.Management, error) {
			if setupCallCount.Add(1) <= 2 {
				return phaseOneManagement, nil
			}

			return phaseTwoManagement, nil
		},
	).Times(4)

	phaseOneManagement.EXPECT().GetPowerState().Return(nil, errors.New("power unavailable"))
	phaseOneManagement.EXPECT().GetHardwareInfo().Return(wsmanAPI.HWResults{}, nil)

	phaseTwoManagement.EXPECT().GetAMTRedirectionService().Return(redirection.Response{}, errors.New("features unavailable"))
	phaseTwoManagement.EXPECT().GetAMTVersion().Return([]software.SoftwareIdentity{}, nil)
	phaseTwoManagement.EXPECT().GetSetupAndConfiguration().Return([]setupandconfiguration.SetupAndConfigurationServiceResponse{{ProvisioningMode: setupandconfiguration.AdminControlMode}}, nil)

	uc := devices.New(repoMock, wsmanMock, mocks.NewMockRedirection(ctrl), logger.New("error"), mocks.MockCrypto{})
	repo := &WsmanComputerSystemRepo{usecase: uc, log: logger.New("error")}

	got, err := repo.GetByID(context.Background(), device.GUID)
	if err != nil {
		t.Fatalf("GetByID() error = %v, want nil", err)
	}

	if got == nil {
		t.Fatal("GetByID() returned nil system")
	}

	if got.PowerState != redfishv1.PowerStateOff {
		t.Fatalf("PowerState = %q, want %q", got.PowerState, redfishv1.PowerStateOff)
	}

	assertGraphicalConsole(t, got.GraphicalConsole, false, nil, 0, StateDisabled, userConsentNotRequired)
	assertSerialConsole(t, got.SerialConsole, false, "", StateDisabled, controlModeACM, userConsentNotRequired)
}

func TestIsContextTimeoutOrCancelError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "context deadline", err: context.DeadlineExceeded, want: true},
		{name: "context cancel", err: context.Canceled, want: true},
		{name: "other error", err: errors.New("other"), want: false},
		{name: "nil error", err: nil, want: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := isContextTimeoutOrCancelError(tt.err)
			if got != tt.want {
				t.Errorf("isContextTimeoutOrCancelError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHasSufficientTimeBudget(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		ctx    context.Context
		budget time.Duration
		want   bool
	}{
		{name: "background context", ctx: context.Background(), budget: 1 * time.Second, want: true},
		{name: "context with sufficient deadline", ctx: createContextWithDeadline(t, 5*time.Second), budget: 1 * time.Second, want: true},
		{name: "context with insufficient deadline", ctx: createContextWithDeadline(t, 100*time.Millisecond), budget: 1 * time.Second, want: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := hasSufficientTimeBudget(tt.ctx, tt.budget)
			if got != tt.want {
				t.Errorf("hasSufficientTimeBudget() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetCachedFeaturesFromDevice(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		device *dto.Device
		want   bool
		wantV2 dtov2.Features
	}{
		{name: "nil device", device: nil, want: false},
		{name: "device without info", device: &dto.Device{}, want: false},
		{name: "blank features", device: &dto.Device{DeviceInfo: &dto.DeviceInfo{Features: "  "}}, want: false},
		{name: "invalid features", device: &dto.Device{DeviceInfo: &dto.DeviceInfo{Features: "not-json"}}, want: false},
		{
			name:   "v2 features json",
			device: &dto.Device{DeviceInfo: &dto.DeviceInfo{Features: `{"userConsent":"none","enableSOL":true,"enableIDER":false,"enableKVM":true,"redirection":true,"optInState":0,"kvmAvailable":true,"httpBoot":false,"httpBootSupported":true,"winREBootSupported":false,"localPBABootSupported":true}`}},
			want:   true,
			wantV2: dtov2.Features{UserConsent: "none", EnableSOL: true, EnableKVM: true, Redirection: true, KVMAvailable: true, HTTPSBootSupported: true, LocalPBABootSupported: true},
		},
		{
			name:   "v1 features json is mapped to v2",
			device: &dto.Device{DeviceInfo: &dto.DeviceInfo{Features: `{"userConsent":"kvm","enableSOL":true,"enableIDER":true,"enableKVM":false,"redirection":true,"optInState":1,"kvmAvailable":false,"ocr":true,"httpsBootSupported":true,"winREBootSupported":true,"localPBABootSupported":false}`}},
			want:   true,
			wantV2: dtov2.Features{UserConsent: "kvm", EnableSOL: true, EnableIDER: true, Redirection: true, OptInState: 1, OCR: true, HTTPSBootSupported: true, WinREBootSupported: true},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := getCachedFeaturesFromDevice(tt.device)
			if ok != tt.want {
				t.Errorf("getCachedFeaturesFromDevice() ok = %v, want %v", ok, tt.want)
			}

			if ok && got != tt.wantV2 {
				t.Errorf("getCachedFeaturesFromDevice() = %#v, want %#v", got, tt.wantV2)
			}
		})
	}
}

func TestGetCachedControlModeFromDevice(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		device   *dto.Device
		wantMode string
	}{
		{name: "nil device", device: nil, wantMode: ""},
		{name: "device without info", device: &dto.Device{}, wantMode: ""},
		{name: "admin mode string", device: &dto.Device{DeviceInfo: &dto.DeviceInfo{CurrentMode: "Admin Control Mode"}}, wantMode: controlModeACM},
		{name: "client mode string", device: &dto.Device{DeviceInfo: &dto.DeviceInfo{CurrentMode: "client control mode"}}, wantMode: controlModeCCM},
		{name: "short acm mode", device: &dto.Device{DeviceInfo: &dto.DeviceInfo{CurrentMode: "acm"}}, wantMode: controlModeACM},
		{name: "unknown mode", device: &dto.Device{DeviceInfo: &dto.DeviceInfo{CurrentMode: "unknown"}}, wantMode: ""},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := getCachedControlModeFromDevice(tt.device)
			if got != tt.wantMode {
				t.Errorf("getCachedControlModeFromDevice() = %q, want %q", got, tt.wantMode)
			}
		})
	}
}

// ============================================================================
// SOL Consent Repository Tests
// ============================================================================

func TestRequestSolConsentRepo_ACMReturnsNotRequiredError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repoMock := mocks.NewMockDeviceManagementRepository(ctrl)
	wsmanMock := mocks.NewMockWSMAN(ctrl)
	wsmanMock.EXPECT().Worker().Return().AnyTimes()

	device := &entity.Device{
		GUID:     "system-1",
		TenantID: "tenant-1",
	}

	management := mocks.NewMockManagement(ctrl)

	repoMock.EXPECT().GetByID(gomock.Any(), device.GUID, "").Return(device, nil).AnyTimes()
	wsmanMock.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(management, nil).AnyTimes()
	management.EXPECT().GetAMTVersion().Return([]software.SoftwareIdentity{}, nil).AnyTimes()
	management.EXPECT().GetSetupAndConfiguration().Return([]setupandconfiguration.SetupAndConfigurationServiceResponse{{ProvisioningMode: setupandconfiguration.AdminControlMode}}, nil).AnyTimes()

	uc := devices.New(repoMock, wsmanMock, mocks.NewMockRedirection(ctrl), logger.New("error"), mocks.MockCrypto{})
	repo := &WsmanComputerSystemRepo{usecase: uc, log: logger.New("error")}

	err := repo.RequestSolConsent(context.Background(), device.GUID)
	if !errors.Is(err, ErrSOLConsentNotRequiredInACM) {
		t.Fatalf("RequestSolConsent() error = %v, wantErr %v", err, ErrSOLConsentNotRequiredInACM)
	}
}

func TestSubmitSolConsentCodeRepo(t *testing.T) {
	t.Parallel()

	errDeviceNotFound := errors.New(ErrMsgDeviceNotFound)
	errAMT := errors.New("amt refused")
	device := &entity.Device{GUID: "system-1", TenantID: "tenant-1"}

	tests := []struct {
		name    string
		code    string
		setup   func(*mocks.MockDeviceManagementRepository, *mocks.MockWSMAN, *mocks.MockManagement)
		wantErr error
	}{
		{
			name: "success",
			code: "123456",
			setup: func(repoMock *mocks.MockDeviceManagementRepository, wsmanMock *mocks.MockWSMAN, management *mocks.MockManagement) {
				gomock.InOrder(
					repoMock.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil),
					wsmanMock.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(management, nil),
					management.EXPECT().SendConsentCode(gomock.Any()).Return(optin.Response{}, nil),
				)
			},
		},
		{
			name:    "invalid consent code",
			code:    "12345",
			wantErr: ErrInvalidConsentCode,
		},
		{
			name: "device not found",
			code: "123456",
			setup: func(repoMock *mocks.MockDeviceManagementRepository, _ *mocks.MockWSMAN, _ *mocks.MockManagement) {
				repoMock.EXPECT().GetByID(context.Background(), device.GUID, "").Return(nil, errDeviceNotFound)
			},
			wantErr: ErrSystemNotFound,
		},
		{
			name: "wsman error",
			code: "123456",
			setup: func(repoMock *mocks.MockDeviceManagementRepository, wsmanMock *mocks.MockWSMAN, _ *mocks.MockManagement) {
				gomock.InOrder(
					repoMock.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil),
					wsmanMock.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(nil, errAMT),
				)
			},
			wantErr: errAMT,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			repoMock := mocks.NewMockDeviceManagementRepository(ctrl)
			wsmanMock := mocks.NewMockWSMAN(ctrl)
			wsmanMock.EXPECT().Worker().Return().AnyTimes()

			management := mocks.NewMockManagement(ctrl)

			uc := devices.New(repoMock, wsmanMock, mocks.NewMockRedirection(ctrl), logger.New("error"), mocks.MockCrypto{})
			repo := &WsmanComputerSystemRepo{usecase: uc, log: logger.New("error")}

			if tt.setup != nil {
				tt.setup(repoMock, wsmanMock, management)
			}

			err := repo.SubmitSolConsentCode(context.Background(), device.GUID, tt.code)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("SubmitSolConsentCode() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCancelSolConsentRepo(t *testing.T) {
	t.Parallel()

	errDeviceNotFound := errors.New(ErrMsgDeviceNotFound)
	errAMT := errors.New("amt refused")
	device := &entity.Device{GUID: "system-1", TenantID: "tenant-1"}

	tests := []struct {
		name    string
		setup   func(*mocks.MockDeviceManagementRepository, *mocks.MockWSMAN, *mocks.MockManagement)
		wantErr error
	}{
		{
			name: "success",
			setup: func(repoMock *mocks.MockDeviceManagementRepository, wsmanMock *mocks.MockWSMAN, management *mocks.MockManagement) {
				gomock.InOrder(
					repoMock.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil),
					wsmanMock.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(management, nil),
					management.EXPECT().CancelUserConsentRequest().Return(optin.Response{}, nil),
				)
			},
		},
		{
			name: "device not found",
			setup: func(repoMock *mocks.MockDeviceManagementRepository, _ *mocks.MockWSMAN, _ *mocks.MockManagement) {
				repoMock.EXPECT().GetByID(context.Background(), device.GUID, "").Return(nil, errDeviceNotFound)
			},
			wantErr: ErrSystemNotFound,
		},
		{
			name: "wsman error",
			setup: func(repoMock *mocks.MockDeviceManagementRepository, wsmanMock *mocks.MockWSMAN, _ *mocks.MockManagement) {
				gomock.InOrder(
					repoMock.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil),
					wsmanMock.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(nil, errAMT),
				)
			},
			wantErr: errAMT,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			repoMock := mocks.NewMockDeviceManagementRepository(ctrl)
			wsmanMock := mocks.NewMockWSMAN(ctrl)
			wsmanMock.EXPECT().Worker().Return().AnyTimes()

			management := mocks.NewMockManagement(ctrl)

			uc := devices.New(repoMock, wsmanMock, mocks.NewMockRedirection(ctrl), logger.New("error"), mocks.MockCrypto{})
			repo := &WsmanComputerSystemRepo{usecase: uc, log: logger.New("error")}

			tt.setup(repoMock, wsmanMock, management)

			err := repo.CancelSolConsent(context.Background(), device.GUID)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("CancelSolConsent() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRequestSolConsentRepo_ReturnValueFailure(t *testing.T) {
	t.Parallel()

	device := &entity.Device{GUID: "system-1", TenantID: "tenant-1"}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repoMock := mocks.NewMockDeviceManagementRepository(ctrl)
	wsmanMock := mocks.NewMockWSMAN(ctrl)
	wsmanMock.EXPECT().Worker().Return().AnyTimes()

	management := mocks.NewMockManagement(ctrl)

	uc := devices.New(repoMock, wsmanMock, mocks.NewMockRedirection(ctrl), logger.New("error"), mocks.MockCrypto{})
	repo := &WsmanComputerSystemRepo{usecase: uc, log: logger.New("error")}

	gomock.InOrder(
		repoMock.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil),
		wsmanMock.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(management, nil),
		management.EXPECT().GetAMTVersion().Return([]software.SoftwareIdentity{}, nil),
		management.EXPECT().GetSetupAndConfiguration().Return([]setupandconfiguration.SetupAndConfigurationServiceResponse{{ProvisioningMode: setupandconfiguration.ClientControlMode}}, nil),
		repoMock.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil),
		wsmanMock.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(management, nil),
		management.EXPECT().GetUserConsentCode().Return(optin.Response{
			Body: optin.Body{StartOptInResponse: optin.StartOptIn_OUTPUT{ReturnValue: 5}},
		}, nil),
	)

	err := repo.RequestSolConsent(context.Background(), device.GUID)

	var consentErr *ConsentFailedError
	if !errors.As(err, &consentErr) {
		t.Fatalf("RequestSolConsent() error type = %T, want *ConsentFailedError", err)
	}
}

func TestSubmitSolConsentCodeRepo_ReturnValueFailure(t *testing.T) {
	t.Parallel()

	device := &entity.Device{GUID: "system-1", TenantID: "tenant-1"}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repoMock := mocks.NewMockDeviceManagementRepository(ctrl)
	wsmanMock := mocks.NewMockWSMAN(ctrl)
	wsmanMock.EXPECT().Worker().Return().AnyTimes()

	management := mocks.NewMockManagement(ctrl)

	uc := devices.New(repoMock, wsmanMock, mocks.NewMockRedirection(ctrl), logger.New("error"), mocks.MockCrypto{})
	repo := &WsmanComputerSystemRepo{usecase: uc, log: logger.New("error")}

	gomock.InOrder(
		repoMock.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil),
		wsmanMock.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(management, nil),
		management.EXPECT().SendConsentCode(gomock.Any()).Return(optin.Response{
			Body: optin.Body{SendOptInCodeResponse: optin.SendOptInCode_OUTPUT{ReturnValue: 2066}},
		}, nil),
	)

	err := repo.SubmitSolConsentCode(context.Background(), device.GUID, "123456")

	var consentErr *ConsentFailedError
	if !errors.As(err, &consentErr) {
		t.Fatalf("SubmitSolConsentCode() error type = %T, want *ConsentFailedError", err)
	}
}

func TestCancelSolConsentRepo_ReturnValueFailure(t *testing.T) {
	t.Parallel()

	device := &entity.Device{GUID: "system-1", TenantID: "tenant-1"}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repoMock := mocks.NewMockDeviceManagementRepository(ctrl)
	wsmanMock := mocks.NewMockWSMAN(ctrl)
	wsmanMock.EXPECT().Worker().Return().AnyTimes()

	management := mocks.NewMockManagement(ctrl)

	uc := devices.New(repoMock, wsmanMock, mocks.NewMockRedirection(ctrl), logger.New("error"), mocks.MockCrypto{})
	repo := &WsmanComputerSystemRepo{usecase: uc, log: logger.New("error")}

	gomock.InOrder(
		repoMock.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil),
		wsmanMock.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(management, nil),
		management.EXPECT().CancelUserConsentRequest().Return(optin.Response{
			Body: optin.Body{CancelOptInResponse: optin.CancelOptIn_OUTPUT{ReturnValue: 2}},
		}, nil),
	)

	err := repo.CancelSolConsent(context.Background(), device.GUID)

	var consentErr *ConsentFailedError
	if !errors.As(err, &consentErr) {
		t.Fatalf("CancelSolConsent() error type = %T, want *ConsentFailedError", err)
	}
}

// Helper functions for tests.
func createContextWithDeadline(t *testing.T, d time.Duration) context.Context {
	t.Helper()

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(d))

	t.Cleanup(cancel)

	return ctx
}
