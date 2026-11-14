package devices_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/boot"

	"github.com/device-management-toolkit/console/internal/entity"
	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	dtov2 "github.com/device-management-toolkit/console/internal/entity/dto/v2"
	"github.com/device-management-toolkit/console/internal/mocks"
	devices "github.com/device-management-toolkit/console/internal/usecase/devices"
	devicewsman "github.com/device-management-toolkit/console/internal/usecase/devices/wsman"
	"github.com/device-management-toolkit/console/pkg/consoleerrors"
)

func TestGetRemoteEraseCapabilities(t *testing.T) {
	t.Parallel()

	device := &entity.Device{
		GUID:     "device-guid-123",
		TenantID: "tenant-id-456",
	}

	fullCapabilities := boot.BootCapabilitiesResponse{
		PlatformErase:          0x01, // RPE support bit
		ConfigurationDataReset: true,
	}

	expectedDTO := dto.BootCapabilities{
		UnconfigureCSME: true,
	}

	tests := []struct {
		name     string
		manMock  func(*mocks.MockWSMAN, *mocks.MockManagement)
		repoMock func(*mocks.MockDeviceManagementRepository)
		res      dto.BootCapabilities
		err      error
	}{
		{
			name: "success",
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).
					Return(man2, nil)
				man2.EXPECT().
					GetPowerCapabilities().
					Return(fullCapabilities, nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			res: expectedDTO,
			err: nil,
		},
		{
			name:    "GetByID returns error",
			manMock: func(_ *mocks.MockWSMAN, _ *mocks.MockManagement) {},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, device.TenantID).
					Return(nil, ErrGeneral)
			},
			res: dto.BootCapabilities{},
			err: ErrGeneral,
		},
		{
			name:    "GetByID returns nil device",
			manMock: func(_ *mocks.MockWSMAN, _ *mocks.MockManagement) {},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, device.TenantID).
					Return(nil, nil)
			},
			res: dto.BootCapabilities{},
			err: devices.ErrNotFound,
		},
		{
			name:    "GetByID returns device with empty GUID",
			manMock: func(_ *mocks.MockWSMAN, _ *mocks.MockManagement) {},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, device.TenantID).
					Return(&entity.Device{GUID: "", TenantID: "tenant-id-456"}, nil)
			},
			res: dto.BootCapabilities{},
			err: devices.ErrNotFound,
		},
		{
			name: "SetupWsmanClient returns error",
			manMock: func(man *mocks.MockWSMAN, _ *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).
					Return(nil, ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			res: dto.BootCapabilities{},
			err: ErrGeneral,
		},
		{
			name: "GetPowerCapabilities wsman call returns error",
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).
					Return(man2, nil)
				man2.EXPECT().
					GetPowerCapabilities().
					Return(boot.BootCapabilitiesResponse{}, ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			res: dto.BootCapabilities{},
			err: ErrGeneral,
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, wsmanMock, management, repo := initInfoTest(t)
			tc.manMock(wsmanMock, management)
			tc.repoMock(repo)

			res, err := useCase.GetRemoteEraseCapabilities(context.Background(), device.GUID, device.TenantID)
			require.Equal(t, tc.err, err)
			require.Equal(t, tc.res, res)
		})
	}
}

func TestGetRemoteEraseStatus(t *testing.T) {
	t.Parallel()

	useCase, wsmanMock, management, repo := initInfoTest(t)
	ctx := context.Background()
	device := &entity.Device{GUID: "device-guid", TenantID: "tenant"}
	repo.EXPECT().GetByID(ctx, device.GUID, device.TenantID).Return(device, nil)
	wsmanMock.EXPECT().SetupWsmanClient(ctx, *device, false, true).Return(management, nil)
	management.EXPECT().GetBootData().Return(boot.BootSettingDataResponse{BIOSLastStatus: []uint16{0xffff, 2}}, nil)

	result, err := useCase.GetRemoteEraseStatus(ctx, device.GUID, device.TenantID)
	require.NoError(t, err)

	status, detail := uint16(0xffff), uint16(2)
	require.Equal(t, dtov2.RemoteEraseStatus{
		Status:     dtov2.RemoteEraseStateFailed,
		Message:    "BIOS erase failed: drive password/authentication failure.",
		BIOSStatus: &status,
		BIOSDetail: &detail,
	}, result)
}

func TestGetRemoteEraseStatusErrors(t *testing.T) {
	t.Parallel()

	device := &entity.Device{GUID: "device-guid", TenantID: "tenant"}
	tests := []struct {
		name     string
		item     *entity.Device
		repoErr  error
		setupErr error
		bootErr  error
		wantErr  error
	}{
		{name: "repository error", repoErr: ErrGeneral, wantErr: ErrGeneral},
		{name: "nil device", wantErr: devices.ErrNotFound},
		{name: "empty GUID", item: &entity.Device{TenantID: device.TenantID}, wantErr: devices.ErrNotFound},
		{name: "client setup error", item: device, setupErr: ErrGeneral, wantErr: ErrGeneral},
		{name: "boot data error", item: device, bootErr: ErrGeneral, wantErr: ErrGeneral},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, wsmanMock, management, repo := initInfoTest(t)
			ctx := context.Background()
			repo.EXPECT().GetByID(ctx, device.GUID, device.TenantID).Return(tc.item, tc.repoErr)

			if tc.repoErr == nil && tc.item != nil && tc.item.GUID != "" {
				wsmanMock.EXPECT().SetupWsmanClient(ctx, *tc.item, false, true).Return(management, tc.setupErr)

				if tc.setupErr == nil {
					management.EXPECT().GetBootData().Return(boot.BootSettingDataResponse{}, tc.bootErr)
				}
			}

			result, err := useCase.GetRemoteEraseStatus(ctx, device.GUID, device.TenantID)
			require.ErrorIs(t, err, tc.wantErr)
			require.Equal(t, dtov2.RemoteEraseStatus{}, result)
		})
	}
}

func TestSetRemoteEraseOptions(t *testing.T) {
	t.Parallel()

	device := &entity.Device{
		GUID:     "device-guid-123",
		TenantID: "tenant-id-456",
	}

	tests := []struct {
		name     string
		req      dto.RemoteEraseRequest
		manMock  func(*mocks.MockWSMAN, *mocks.MockManagement)
		repoMock func(*mocks.MockDeviceManagementRepository)
		err      error
	}{
		{
			name: "no options selected returns ValidationError before reaching WSMAN",
			req:  dto.RemoteEraseRequest{},
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).
					Return(man2, nil)
				man2.EXPECT().
					GetPowerCapabilities().
					Return(boot.BootCapabilitiesResponse{PlatformErase: 3}, nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			err: devices.ValidationError{}.Wrap("SetRemoteEraseOptions", "check erase options", "at least one erase option must be selected"),
		},
		{
			name: "success - specific supported eraseMask",
			req:  dto.RemoteEraseRequest{SecureEraseAllSSDs: true, TPMClear: true},
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).
					Return(man2, nil)
				man2.EXPECT().
					GetPowerCapabilities().
					Return(boot.BootCapabilitiesResponse{PlatformErase: 3}, nil)
				man2.EXPECT().
					SetRemoteEraseOptions(0x44, devicewsman.RPECredentials{}).
					Return(nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			err: nil,
		},
		{
			name: "success - RestoreBIOSToEOM sets platformEraseBIOSReload bit",
			req:  dto.RemoteEraseRequest{RestoreBIOSToEOM: true},
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).
					Return(man2, nil)
				man2.EXPECT().
					GetPowerCapabilities().
					Return(boot.BootCapabilitiesResponse{PlatformErase: 3}, nil)
				man2.EXPECT().
					SetRemoteEraseOptions(0x4000000, devicewsman.RPECredentials{}).
					Return(nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			err: nil,
		},
		{
			name: "device does not support RPE - PlatformErase is 0 and ConfigurationDataReset false",
			req:  dto.RemoteEraseRequest{},
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).
					Return(man2, nil)
				man2.EXPECT().
					GetPowerCapabilities().
					Return(boot.BootCapabilitiesResponse{PlatformErase: 0, ConfigurationDataReset: false}, nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			err: devices.ValidationError{}.Wrap("SetRemoteEraseOptions", "check boot capabilities", "device does not support Remote Platform Erase"),
		},
		{
			name: "CSME-only device - PlatformErase 0 but ConfigurationDataReset true - UnconfigureCSME accepted",
			req:  dto.RemoteEraseRequest{UnconfigureCSME: true},
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).
					Return(man2, nil)
				man2.EXPECT().
					GetPowerCapabilities().
					Return(boot.BootCapabilitiesResponse{PlatformErase: 0, ConfigurationDataReset: true}, nil)
				man2.EXPECT().
					SetRemoteEraseOptions(0x10000, devicewsman.RPECredentials{}).
					Return(nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			err: nil,
		},
		{
			name: "eraseMask with PlatformErase nonzero succeeds regardless of specific bits",
			req:  dto.RemoteEraseRequest{SecureEraseAllSSDs: true},
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).
					Return(man2, nil)
				man2.EXPECT().
					GetPowerCapabilities().
					Return(boot.BootCapabilitiesResponse{PlatformErase: 3}, nil)
				man2.EXPECT().
					SetRemoteEraseOptions(4, devicewsman.RPECredentials{}).
					Return(nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			err: nil,
		},
		{
			name:    "GetByID returns error",
			req:     dto.RemoteEraseRequest{},
			manMock: func(_ *mocks.MockWSMAN, _ *mocks.MockManagement) {},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, device.TenantID).
					Return(nil, ErrGeneral)
			},
			err: ErrGeneral,
		},
		{
			name:    "GetByID returns nil device",
			req:     dto.RemoteEraseRequest{},
			manMock: func(_ *mocks.MockWSMAN, _ *mocks.MockManagement) {},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, device.TenantID).
					Return(nil, nil)
			},
			err: devices.ErrNotFound,
		},
		{
			name:    "GetByID returns device with empty GUID",
			req:     dto.RemoteEraseRequest{},
			manMock: func(_ *mocks.MockWSMAN, _ *mocks.MockManagement) {},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, device.TenantID).
					Return(&entity.Device{GUID: "", TenantID: "tenant-id-456"}, nil)
			},
			err: devices.ErrNotFound,
		},
		{
			name: "SetupWsmanClient returns error",
			req:  dto.RemoteEraseRequest{},
			manMock: func(man *mocks.MockWSMAN, _ *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).
					Return(nil, ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			err: ErrGeneral,
		},
		{
			name: "GetBootCapabilities returns error",
			req:  dto.RemoteEraseRequest{},
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).
					Return(man2, nil)
				man2.EXPECT().
					GetPowerCapabilities().
					Return(boot.BootCapabilitiesResponse{}, ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			err: ErrGeneral,
		},
		{
			name: "SetRemoteEraseOptions wsman call returns error",
			req:  dto.RemoteEraseRequest{SecureEraseAllSSDs: true},
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).
					Return(man2, nil)
				man2.EXPECT().
					GetPowerCapabilities().
					Return(boot.BootCapabilitiesResponse{PlatformErase: 3}, nil)
				man2.EXPECT().
					SetRemoteEraseOptions(0x04, devicewsman.RPECredentials{}). // platformEraseSecureErase
					Return(ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			err: ErrGeneral,
		},
		{
			name: "success - UnconfigureCSME only",
			req:  dto.RemoteEraseRequest{UnconfigureCSME: true},
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).
					Return(man2, nil)
				man2.EXPECT().
					GetPowerCapabilities().
					Return(boot.BootCapabilitiesResponse{PlatformErase: 0x01}, nil)
				man2.EXPECT().
					SetRemoteEraseOptions(0x10000, devicewsman.RPECredentials{}).
					Return(nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			err: nil,
		},
		{
			name: "UnconfigureCSME allowed when RPE is supported",
			req:  dto.RemoteEraseRequest{UnconfigureCSME: true},
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).
					Return(man2, nil)
				man2.EXPECT().
					GetPowerCapabilities().
					Return(boot.BootCapabilitiesResponse{PlatformErase: 3}, nil)
				man2.EXPECT().
					SetRemoteEraseOptions(0x10000, devicewsman.RPECredentials{}).
					Return(nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			err: nil,
		},
		{
			name: "no erase options selected returns ValidationError",
			req:  dto.RemoteEraseRequest{},
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).
					Return(man2, nil)
				man2.EXPECT().
					GetPowerCapabilities().
					Return(boot.BootCapabilitiesResponse{PlatformErase: 3}, nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			err: devices.ValidationError{}.Wrap("SetRemoteEraseOptions", "check erase options", "at least one erase option must be selected"),
		},
		{
			name: "SetRemoteEraseOptions returns ErrRPENotEnabled - converted to NotSupportedError",
			req:  dto.RemoteEraseRequest{SecureEraseAllSSDs: true},
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).
					Return(man2, nil)
				man2.EXPECT().
					GetPowerCapabilities().
					Return(boot.BootCapabilitiesResponse{PlatformErase: 3}, nil)
				man2.EXPECT().
					SetRemoteEraseOptions(0x04, devicewsman.RPECredentials{}).
					Return(devicewsman.ErrRPENotEnabled)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			err: devices.NotSupportedError{Console: consoleerrors.CreateConsoleError("Remote Platform Erase is not enabled by the BIOS on this device")},
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, wsmanMock, management, repo := initInfoTest(t)
			tc.manMock(wsmanMock, management)
			tc.repoMock(repo)

			err := useCase.SetRemoteEraseOptions(context.Background(), device.GUID, device.TenantID, tc.req)
			require.Equal(t, tc.err, err)
		})
	}
}

func TestSetRemoteEraseOptionsCredentialsRequireSSDErase(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		req  dto.RemoteEraseRequest
	}{
		{"TPM only with password", dto.RemoteEraseRequest{TPMClear: true, SSDPassword: "password"}},
		{"TPM only with PSID", dto.RemoteEraseRequest{TPMClear: true, PSID: "drive-psid"}},
		{"CSME only with password", dto.RemoteEraseRequest{UnconfigureCSME: true, SSDPassword: "password"}},
		{"CSME only with PSID", dto.RemoteEraseRequest{UnconfigureCSME: true, PSID: "drive-psid"}},
		{"BIOS only with credentials", dto.RemoteEraseRequest{RestoreBIOSToEOM: true, SSDPassword: "password", PSID: "drive-psid"}},
		{"no targets with credentials", dto.RemoteEraseRequest{SSDPassword: "password", PSID: "drive-psid"}},
	} {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, _, _, _ := initInfoTest(t)
			err := useCase.SetRemoteEraseOptions(context.Background(), "device-guid", "tenant-id", tc.req)
			require.Equal(t, devices.ValidationError{}.Wrap(
				"SetRemoteEraseOptions", "check RPE credentials",
				"ssdPassword and psid require secureEraseAllSSDs to be selected",
			), err)
		})
	}
}

func TestSetRemoteEraseOptionsCredentialByteLimits(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name        string
		credentials devicewsman.RPECredentials
		valid       bool
	}{
		{"33 ASCII password bytes", devicewsman.RPECredentials{SSDPassword: strings.Repeat("p", 33)}, true},
		{"64-byte password and PSID", devicewsman.RPECredentials{SSDPassword: strings.Repeat("p", 64), PSID: strings.Repeat("s", 64)}, true},
		{"64 multibyte password bytes", devicewsman.RPECredentials{SSDPassword: strings.Repeat("\xc3\xa9", 32), PSID: "drive-psid"}, true},
		{"65 ASCII password bytes", devicewsman.RPECredentials{SSDPassword: strings.Repeat("p", 65)}, false},
		{"66 multibyte password bytes", devicewsman.RPECredentials{SSDPassword: strings.Repeat("\xc3\xa9", 33)}, false},
		{"65 PSID bytes", devicewsman.RPECredentials{PSID: strings.Repeat("s", 65)}, false},
	} {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, wsmanMock, management, repo := initInfoTest(t)
			ctx := context.Background()
			device := entity.Device{GUID: "device-guid", TenantID: "tenant-id"}
			req := dto.RemoteEraseRequest{SecureEraseAllSSDs: true, SSDPassword: tc.credentials.SSDPassword, PSID: tc.credentials.PSID}

			if tc.valid {
				repo.EXPECT().GetByID(ctx, device.GUID, device.TenantID).Return(&device, nil)
				wsmanMock.EXPECT().SetupWsmanClient(ctx, device, false, true).Return(management, nil)
				management.EXPECT().GetPowerCapabilities().Return(boot.BootCapabilitiesResponse{PlatformErase: 4}, nil)
				management.EXPECT().SetRemoteEraseOptions(4, tc.credentials).Return(nil)
			}

			err := useCase.SetRemoteEraseOptions(ctx, device.GUID, device.TenantID, req)
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Equal(t, devices.ValidationError{}.Wrap("SetRemoteEraseOptions", "check RPE credentials", devicewsman.ErrRPECredentialTooLong.Error()), err)
			}
		})
	}
}
