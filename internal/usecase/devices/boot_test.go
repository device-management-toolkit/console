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

func TestGetRemoteEraseCapabilities(t *testing.T) {
	t.Parallel()

	device := &entity.Device{
		GUID:     "device-guid-123",
		TenantID: "tenant-id-456",
	}

	fullCapabilities := boot.BootCapabilitiesResponse{
		PlatformErase: 1,
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

			res, err := useCase.GetRemoteEraseCapabilities(context.Background(), device.GUID)
			require.Equal(t, tc.err, err)
			require.Equal(t, tc.res, res)
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
			name: "success - eraseMask 0 erases all",
			req:  dto.RemoteEraseRequest{},
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).
					Return(man2, nil)
				man2.EXPECT().
					GetBootCapabilities().
					Return(boot.BootCapabilitiesResponse{PlatformErase: 3}, nil)
				man2.EXPECT().
					SetRemoteEraseOptions(0).
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
			name: "success - specific supported eraseMask",
			req:  dto.RemoteEraseRequest{SecureEraseAllSSDs: true, TPMClear: true},
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).
					Return(man2, nil)
				man2.EXPECT().
					GetBootCapabilities().
					Return(boot.BootCapabilitiesResponse{PlatformErase: 3}, nil)
				man2.EXPECT().
					SetRemoteEraseOptions(0x44).
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
			name: "device does not support RPE - PlatformErase is 0",
			req:  dto.RemoteEraseRequest{},
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
			err: devices.ValidationError{}.Wrap("SetRemoteEraseOptions", "check boot capabilities", "device does not support Remote Platform Erase"),
		},
		{
			name: "eraseMask with PlatformErase nonzero succeeds regardless of specific bits",
			req:  dto.RemoteEraseRequest{SecureEraseAllSSDs: true},
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).
					Return(man2, nil)
				man2.EXPECT().
					GetBootCapabilities().
					Return(boot.BootCapabilitiesResponse{PlatformErase: 3}, nil)
				man2.EXPECT().
					SetRemoteEraseOptions(4).
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
			name:    "GetByID returns error",
			req:     dto.RemoteEraseRequest{},
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
			req:     dto.RemoteEraseRequest{},
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
			req:     dto.RemoteEraseRequest{},
			manMock: func(_ *mocks.MockWSMAN, _ *mocks.MockManagement) {},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
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
					GetByID(context.Background(), device.GUID, "").
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
			name: "SetRemoteEraseOptions wsman call returns error",
			req:  dto.RemoteEraseRequest{},
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).
					Return(man2, nil)
				man2.EXPECT().
					GetBootCapabilities().
					Return(boot.BootCapabilitiesResponse{PlatformErase: 3}, nil)
				man2.EXPECT().
					SetRemoteEraseOptions(0).
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
			name: "SetRemoteEraseOptions returns ErrRPENotEnabled - converted to NotSupportedError",
			req:  dto.RemoteEraseRequest{},
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).
					Return(man2, nil)
				man2.EXPECT().
					GetBootCapabilities().
					Return(boot.BootCapabilitiesResponse{PlatformErase: 3}, nil)
				man2.EXPECT().
					SetRemoteEraseOptions(0).
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

			err := useCase.SetRemoteEraseOptions(context.Background(), device.GUID, tc.req)
			require.Equal(t, tc.err, err)
		})
	}
}
