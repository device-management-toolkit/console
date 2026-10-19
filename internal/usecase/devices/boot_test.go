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

	tests := []struct {
		name    string
		status  []uint16
		want    string
		message string
	}{
		{name: "missing", want: "unknown", message: "unconfirmed"},
		{name: "truncated", status: []uint16{0}, want: "unknown", message: "unconfirmed"},
		{name: "success", status: []uint16{0, 0}, want: "succeeded", message: "completed successfully"},
		{name: "in progress", status: []uint16{1, 0}, want: "pending", message: "Waiting"},
		{name: "not updated", status: []uint16{2, 0}, want: "pending", message: "Waiting"},
		{name: "password failure", status: []uint16{0xffff, 2}, want: "failed", message: "password/authentication failure"},
		{name: "drive failure", status: []uint16{0xffff, 1}, want: "failed", message: "general drive failure"},
		{name: "unsupported", status: []uint16{0xffff, 3}, want: "failed", message: "not supported"},
		{name: "other failure", status: []uint16{0xffff, 99}, want: "failed", message: "erase failed"},
		{name: "unknown", status: []uint16{4, 0}, want: "unknown", message: "unconfirmed"},
		{name: "inconsistent success", status: []uint16{0, 2}, want: "unknown", message: "unconfirmed"},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			useCase, wsmanMock, management, repo := initInfoTest(t)
			device := &entity.Device{GUID: "device-guid", TenantID: "tenant"}
			repo.EXPECT().GetByID(gomock.Any(), device.GUID, device.TenantID).Return(device, nil)
			wsmanMock.EXPECT().SetupWsmanClient(gomock.Any(), *device, false, true).Return(management, nil)
			management.EXPECT().GetBootData().Return(boot.BootSettingDataResponse{BIOSLastStatus: tc.status}, nil)

			result, err := useCase.GetRemoteEraseStatus(context.Background(), device.GUID, device.TenantID)
			require.NoError(t, err)
			require.Equal(t, tc.want, result.Status)
			require.Contains(t, result.Message, tc.message)
		})
	}
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
			name:     "oversized SSD password rejected before reaching WSMAN",
			req:      dto.RemoteEraseRequest{SecureEraseAllSSDs: true, SSDPassword: strings.Repeat("p", 65)},
			manMock:  func(_ *mocks.MockWSMAN, _ *mocks.MockManagement) {},
			repoMock: func(_ *mocks.MockDeviceManagementRepository) {},
			err:      devices.ValidationError{}.Wrap("SetRemoteEraseOptions", "check RPE credentials", devicewsman.ErrRPECredentialTooLong.Error()),
		},
		{
			name:     "oversized PSID rejected before reaching WSMAN",
			req:      dto.RemoteEraseRequest{SecureEraseAllSSDs: true, PSID: string(make([]byte, 65))},
			manMock:  func(_ *mocks.MockWSMAN, _ *mocks.MockManagement) {},
			repoMock: func(_ *mocks.MockDeviceManagementRepository) {},
			err:      devices.ValidationError{}.Wrap("SetRemoteEraseOptions", "check RPE credentials", devicewsman.ErrRPECredentialTooLong.Error()),
		},
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
					SetRemoteEraseOptions(0x44, "", "").
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
					SetRemoteEraseOptions(0x4000000, "", "").
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
					SetRemoteEraseOptions(0x10000, "", "").
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
					SetRemoteEraseOptions(4, "", "").
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
					SetRemoteEraseOptions(0x04, "", ""). // platformEraseSecureErase
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
					SetRemoteEraseOptions(0x10000, "", "").
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
					SetRemoteEraseOptions(0x10000, "", "").
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
			name: "success - 64-byte multibyte SSD password and PSID forwarded to wsman",
			req:  dto.RemoteEraseRequest{SecureEraseAllSSDs: true, SSDPassword: strings.Repeat("\xc3\xa9", 32), PSID: "drive-psid"},
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).
					Return(man2, nil)
				man2.EXPECT().
					GetPowerCapabilities().
					Return(boot.BootCapabilitiesResponse{PlatformErase: 3}, nil)
				man2.EXPECT().
					SetRemoteEraseOptions(4, strings.Repeat("\xc3\xa9", 32), "drive-psid").
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
					SetRemoteEraseOptions(0x04, "", "").
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

func TestSetRemoteEraseOptionsPasswordByteLimit(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name     string
		password string
		valid    bool
	}{
		{"33 ASCII bytes", strings.Repeat("p", 33), true},
		{"64 ASCII bytes", strings.Repeat("p", 64), true},
		{"64 mixed UTF-8 bytes", strings.Repeat("\xc3\xa9", 16) + strings.Repeat("p", 32), true},
		{"64 multibyte UTF-8 bytes", strings.Repeat("\xc3\xa9", 32), true},
		{"65 ASCII bytes", strings.Repeat("p", 65), false},
		{"65 mixed UTF-8 bytes", strings.Repeat("\xc3\xa9", 32) + "p", false},
		{"66 multibyte UTF-8 bytes", strings.Repeat("\xc3\xa9", 33), false},
	} {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, wsmanMock, management, repo := initInfoTest(t)
			ctx := context.Background()
			device := entity.Device{GUID: "device-guid", TenantID: "tenant-id"}
			req := dto.RemoteEraseRequest{SecureEraseAllSSDs: true, SSDPassword: tc.password, PSID: strings.Repeat("s", 64)}

			if tc.valid {
				repo.EXPECT().GetByID(ctx, device.GUID, device.TenantID).Return(&device, nil)
				wsmanMock.EXPECT().SetupWsmanClient(ctx, device, false, true).Return(management, nil)
				management.EXPECT().GetPowerCapabilities().Return(boot.BootCapabilitiesResponse{PlatformErase: 4}, nil)
				management.EXPECT().SetRemoteEraseOptions(4, tc.password, req.PSID).Return(nil)
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
