package usecase

import (
	"context"
	"errors"
	"testing"

	gomock "go.uber.org/mock/gomock"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/boot"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/redirection"
	cimBoot "github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/boot"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/kvm"
	optin "github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/ips/optin"

	"github.com/device-management-toolkit/console/internal/entity"
	dtov2 "github.com/device-management-toolkit/console/internal/entity/dto/v2"
	"github.com/device-management-toolkit/console/internal/mocks"
	"github.com/device-management-toolkit/console/internal/usecase/devices"
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

func boolToListener(enabled bool) int {
	if enabled {
		return 1
	}

	return 0
}

func expectGetFeaturesSuccess(t *testing.T, repo *mocks.MockDeviceManagementRepository, wsmanMock *mocks.MockWSMAN, management *mocks.MockManagement, device *entity.Device, kvmEnabled bool) {
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

	kvmState := kvm.EnabledState(0)
	if kvmEnabled {
		kvmState = kvm.EnabledState(redirection.Enabled)
	}

	management.EXPECT().GetKVMRedirection().Return(kvm.Response{
		Body: kvm.Body{
			GetResponse: kvm.KVMRedirectionSAP{EnabledState: kvmState},
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
				expectGetFeaturesSuccess(t, repo, wsmanMock, getMgmt, device, false)
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
				expectGetFeaturesSuccess(t, repo, wsmanMock, getMgmt, device, false)
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
				expectGetFeaturesSuccess(t, repo, wsmanMock, getMgmt, device, false)
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
