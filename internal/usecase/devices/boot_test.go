package devices_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/boot"

	"github.com/device-management-toolkit/console/internal/entity"
	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/mocks"
	devices "github.com/device-management-toolkit/console/internal/usecase/devices"
	devicewsman "github.com/device-management-toolkit/console/internal/usecase/devices/wsman"
	"github.com/device-management-toolkit/console/pkg/consoleerrors"
)

func TestGetBootCapabilities(t *testing.T) {
	t.Parallel()

	device := &entity.Device{
		GUID:     "device-guid-123",
		TenantID: "tenant-id-456",
	}

	fullCapabilities := boot.BootCapabilitiesResponse{
		PlatformErase: 1,
	}

	expectedDTO := dto.BootCapabilities{
		PlatformErase: 1,
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
					GetBootCapabilities().
					Return(fullCapabilities, nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
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
					GetByID(context.Background(), device.GUID, "").
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
					GetByID(context.Background(), device.GUID, "").
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
					GetByID(context.Background(), device.GUID, "").
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
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: dto.BootCapabilities{},
			err: ErrGeneral,
		},
		{
			name: "GetBootCapabilities wsman call returns error",
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).
					Return(man2, nil)
				man2.EXPECT().
					GetBootCapabilities().
					Return(boot.BootCapabilitiesResponse{}, ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
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

			res, err := useCase.GetBootCapabilities(context.Background(), device.GUID)
			require.Equal(t, tc.err, err)
			require.Equal(t, tc.res, res)
		})
	}
}

func TestSetRPEEnabled(t *testing.T) {
	t.Parallel()

	device := &entity.Device{
		GUID:     "device-guid-123",
		TenantID: "tenant-id-456",
	}

	tests := []struct {
		name     string
		enabled  bool
		manMock  func(*mocks.MockWSMAN, *mocks.MockManagement)
		repoMock func(*mocks.MockDeviceManagementRepository)
		err      error
	}{
		{
			name:    "success - enable RPE on supported device",
			enabled: true,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).
					Return(man2, nil)
				man2.EXPECT().
					GetBootCapabilities().
					Return(boot.BootCapabilitiesResponse{PlatformErase: 1}, nil)
				man2.EXPECT().
					SetRPEEnabled(true).
					Return(nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			err: nil,
		},
		{
			name:    "success - disable RPE on supported device",
			enabled: false,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).
					Return(man2, nil)
				man2.EXPECT().
					GetBootCapabilities().
					Return(boot.BootCapabilitiesResponse{PlatformErase: 1}, nil)
				man2.EXPECT().
					SetRPEEnabled(false).
					Return(nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			err: nil,
		},
		{
			name:    "device does not support RPE - PlatformErase is 0",
			enabled: true,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).
					Return(man2, nil)
				man2.EXPECT().
					GetBootCapabilities().
					Return(boot.BootCapabilitiesResponse{PlatformErase: 0}, nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			err: devices.ValidationError{}.Wrap("SetRPEEnabled", "check boot capabilities", "device does not support Remote Platform Erase"),
		},
		{
			name:    "GetByID returns error",
			enabled: true,
			manMock: func(_ *mocks.MockWSMAN, _ *mocks.MockManagement) {},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(nil, ErrGeneral)
			},
			err: ErrGeneral,
		},
		{
			name:    "GetByID returns nil device",
			enabled: true,
			manMock: func(_ *mocks.MockWSMAN, _ *mocks.MockManagement) {},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(nil, nil)
			},
			err: devices.ErrNotFound,
		},
		{
			name:    "GetByID returns device with empty GUID",
			enabled: true,
			manMock: func(_ *mocks.MockWSMAN, _ *mocks.MockManagement) {},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(&entity.Device{GUID: "", TenantID: "tenant-id-456"}, nil)
			},
			err: devices.ErrNotFound,
		},
		{
			name:    "SetupWsmanClient returns error",
			enabled: true,
			manMock: func(man *mocks.MockWSMAN, _ *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).
					Return(nil, ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			err: ErrGeneral,
		},
		{
			name:    "GetBootCapabilities returns error",
			enabled: true,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).
					Return(man2, nil)
				man2.EXPECT().
					GetBootCapabilities().
					Return(boot.BootCapabilitiesResponse{}, ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			err: ErrGeneral,
		},
		{
			name:    "SetRPEEnabled wsman call returns error",
			enabled: true,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).
					Return(man2, nil)
				man2.EXPECT().
					GetBootCapabilities().
					Return(boot.BootCapabilitiesResponse{PlatformErase: 1}, nil)
				man2.EXPECT().
					SetRPEEnabled(true).
					Return(ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
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

			err := useCase.SetRPEEnabled(context.Background(), device.GUID, tc.enabled)
			require.Equal(t, tc.err, err)
		})
	}
}

func TestSendRemoteErase(t *testing.T) {
	t.Parallel()

	device := &entity.Device{
		GUID:     "device-guid-123",
		TenantID: "tenant-id-456",
	}

	tests := []struct {
		name      string
		eraseMask int
		manMock   func(*mocks.MockWSMAN, *mocks.MockManagement)
		repoMock  func(*mocks.MockDeviceManagementRepository)
		err       error
	}{
		{
			name:      "success - eraseMask 0 erases all",
			eraseMask: 0,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).
					Return(man2, nil)
				man2.EXPECT().
					GetBootCapabilities().
					Return(boot.BootCapabilitiesResponse{PlatformErase: 3}, nil)
				man2.EXPECT().
					SendRemoteErase(0).
					Return(nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			err: nil,
		},
		{
			name:      "success - specific supported eraseMask",
			eraseMask: 2,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).
					Return(man2, nil)
				man2.EXPECT().
					GetBootCapabilities().
					Return(boot.BootCapabilitiesResponse{PlatformErase: 3}, nil)
				man2.EXPECT().
					SendRemoteErase(2).
					Return(nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			err: nil,
		},
		{
			name:      "device does not support RPE - PlatformErase is 0",
			eraseMask: 0,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).
					Return(man2, nil)
				man2.EXPECT().
					GetBootCapabilities().
					Return(boot.BootCapabilitiesResponse{PlatformErase: 0}, nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			err: devices.ValidationError{}.Wrap("SendRemoteErase", "check boot capabilities", "device does not support Remote Platform Erase"),
		},
		{
			name:      "eraseMask with PlatformErase nonzero succeeds regardless of specific bits",
			eraseMask: 4,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).
					Return(man2, nil)
				man2.EXPECT().
					GetBootCapabilities().
					Return(boot.BootCapabilitiesResponse{PlatformErase: 3}, nil)
				man2.EXPECT().
					SendRemoteErase(4).
					Return(nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			err: nil,
		},
		{
			name:      "GetByID returns error",
			eraseMask: 0,
			manMock:   func(_ *mocks.MockWSMAN, _ *mocks.MockManagement) {},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(nil, ErrGeneral)
			},
			err: ErrGeneral,
		},
		{
			name:      "GetByID returns nil device",
			eraseMask: 0,
			manMock:   func(_ *mocks.MockWSMAN, _ *mocks.MockManagement) {},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(nil, nil)
			},
			err: devices.ErrNotFound,
		},
		{
			name:      "GetByID returns device with empty GUID",
			eraseMask: 0,
			manMock:   func(_ *mocks.MockWSMAN, _ *mocks.MockManagement) {},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(&entity.Device{GUID: "", TenantID: "tenant-id-456"}, nil)
			},
			err: devices.ErrNotFound,
		},
		{
			name:      "SetupWsmanClient returns error",
			eraseMask: 0,
			manMock: func(man *mocks.MockWSMAN, _ *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).
					Return(nil, ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			err: ErrGeneral,
		},
		{
			name:      "GetBootCapabilities returns error",
			eraseMask: 0,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).
					Return(man2, nil)
				man2.EXPECT().
					GetBootCapabilities().
					Return(boot.BootCapabilitiesResponse{}, ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			err: ErrGeneral,
		},
		{
			name:      "SendRemoteErase wsman call returns error",
			eraseMask: 0,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).
					Return(man2, nil)
				man2.EXPECT().
					GetBootCapabilities().
					Return(boot.BootCapabilitiesResponse{PlatformErase: 3}, nil)
				man2.EXPECT().
					SendRemoteErase(0).
					Return(ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			err: ErrGeneral,
		},
		{
			name:      "SendRemoteErase returns ErrRPENotEnabled - converted to NotSupportedError",
			eraseMask: 0,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).
					Return(man2, nil)
				man2.EXPECT().
					GetBootCapabilities().
					Return(boot.BootCapabilitiesResponse{PlatformErase: 3}, nil)
				man2.EXPECT().
					SendRemoteErase(0).
					Return(devicewsman.ErrRPENotEnabled)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
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

			err := useCase.SendRemoteErase(context.Background(), device.GUID, tc.eraseMask)
			require.Equal(t, tc.err, err)
		})
	}
}
