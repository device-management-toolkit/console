package devices_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/boot"
	cimBoot "github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/boot"

	"github.com/device-management-toolkit/console/internal/entity"
	"github.com/device-management-toolkit/console/internal/mocks"
	devices "github.com/device-management-toolkit/console/internal/usecase/devices"
	"github.com/device-management-toolkit/console/pkg/logger"
)

type bootTest struct {
	name     string
	manMock  func(*mocks.MockWSMAN, *mocks.MockManagement)
	repoMock func(*mocks.MockDeviceManagementRepository)
	res      any
	err      error
}

func initBootTest(t *testing.T) (*devices.UseCase, *mocks.MockWSMAN, *mocks.MockManagement, *mocks.MockDeviceManagementRepository) {
	t.Helper()

	mockCtl := gomock.NewController(t)
	defer mockCtl.Finish()

	repo := mocks.NewMockDeviceManagementRepository(mockCtl)
	wsmanMock := mocks.NewMockWSMAN(mockCtl)
	wsmanMock.EXPECT().Worker().Return().AnyTimes()

	managementMock := mocks.NewMockManagement(mockCtl)
	log := logger.New("error")
	u := devices.New(repo, wsmanMock, mocks.NewMockRedirection(mockCtl), log, mocks.MockCrypto{})

	return u, wsmanMock, managementMock, repo
}

func TestGetBootData(t *testing.T) {
	t.Parallel()

	device := &entity.Device{
		GUID:     "device-guid-123",
		Password: "encrypted",
		TenantID: "tenant-id-456",
	}

	bootDataResponse := boot.BootSettingDataResponse{
		InstanceID: "Intel(r) AMT: Boot Configuration 0",
	}

	tests := []bootTest{
		{
			name: "success",
			manMock: func(man *mocks.MockWSMAN, hmm *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(hmm, nil)
				hmm.EXPECT().
					GetBootData().
					Return(bootDataResponse, nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: bootDataResponse,
			err: nil,
		},
		{
			name: "failed to get device from repo",
			manMock: func(_ *mocks.MockWSMAN, _ *mocks.MockManagement) {
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(nil, ErrGeneral)
			},
			res: boot.BootSettingDataResponse{},
			err: ErrGeneral,
		},
		{
			name: "device not found",
			manMock: func(_ *mocks.MockWSMAN, _ *mocks.MockManagement) {
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(nil, nil)
			},
			res: boot.BootSettingDataResponse{},
			err: devices.ErrNotFound,
		},
		{
			name: "failed to setup wsman client",
			manMock: func(man *mocks.MockWSMAN, _ *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(nil, ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: boot.BootSettingDataResponse{},
			err: ErrGeneral,
		},
		{
			name: "failed to get boot data",
			manMock: func(man *mocks.MockWSMAN, hmm *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(hmm, nil)
				hmm.EXPECT().
					GetBootData().
					Return(boot.BootSettingDataResponse{}, ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: boot.BootSettingDataResponse{},
			err: ErrGeneral,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, wsmanMock, managementMock, repo := initBootTest(t)

			tc.repoMock(repo)
			tc.manMock(wsmanMock, managementMock)

			res, err := useCase.GetBootData(context.Background(), device.GUID)

			require.Equal(t, tc.err, err)
			assert.Equal(t, tc.res, res)
		})
	}
}

func TestSetBootData(t *testing.T) {
	t.Parallel()

	device := &entity.Device{
		GUID:     "device-guid-123",
		Password: "encrypted",
		TenantID: "tenant-id-456",
	}

	bootDataRequest := boot.BootSettingDataRequest{
		InstanceID: "Intel(r) AMT: Boot Configuration 0",
	}

	bootDataResponse := boot.BootSettingDataResponse{
		InstanceID: "Intel(r) AMT: Boot Configuration 0",
	}

	changeBootOrderResponse := cimBoot.ChangeBootOrder_OUTPUT{
		ReturnValue: 0,
	}

	tests := []bootTest{
		{
			name: "success",
			manMock: func(man *mocks.MockWSMAN, hmm *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(hmm, nil)
				hmm.EXPECT().
					ChangeBootOrder("").
					Return(changeBootOrderResponse, nil)
				hmm.EXPECT().
					SetBootData(bootDataRequest).
					Return(bootDataResponse, nil)
				hmm.EXPECT().
					SetBootConfigRole(1).
					Return(bootDataResponse, nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			err: nil,
		},
		{
			name: "failed to get device from repo",
			manMock: func(_ *mocks.MockWSMAN, _ *mocks.MockManagement) {
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(nil, ErrGeneral)
			},
			err: ErrGeneral,
		},
		{
			name: "device not found",
			manMock: func(_ *mocks.MockWSMAN, _ *mocks.MockManagement) {
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(nil, nil)
			},
			err: devices.ErrNotFound,
		},
		{
			name: "failed to setup wsman client",
			manMock: func(man *mocks.MockWSMAN, _ *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
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
			name: "failed to clear boot order",
			manMock: func(man *mocks.MockWSMAN, hmm *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(hmm, nil)
				hmm.EXPECT().
					ChangeBootOrder("").
					Return(cimBoot.ChangeBootOrder_OUTPUT{}, ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			err: ErrGeneral,
		},
		{
			name: "failed to set boot data",
			manMock: func(man *mocks.MockWSMAN, hmm *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(hmm, nil)
				hmm.EXPECT().
					ChangeBootOrder("").
					Return(changeBootOrderResponse, nil)
				hmm.EXPECT().
					SetBootData(bootDataRequest).
					Return(boot.BootSettingDataResponse{}, ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			err: ErrGeneral,
		},
		{
			name: "failed to set boot config role",
			manMock: func(man *mocks.MockWSMAN, hmm *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(hmm, nil)
				hmm.EXPECT().
					ChangeBootOrder("").
					Return(changeBootOrderResponse, nil)
				hmm.EXPECT().
					SetBootData(bootDataRequest).
					Return(bootDataResponse, nil)
				hmm.EXPECT().
					SetBootConfigRole(1).
					Return(boot.BootSettingDataResponse{}, ErrGeneral)
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

			useCase, wsmanMock, managementMock, repo := initBootTest(t)

			tc.repoMock(repo)
			tc.manMock(wsmanMock, managementMock)

			err := useCase.SetBootData(context.Background(), device.GUID, bootDataRequest)

			require.Equal(t, tc.err, err)
		})
	}
}

func TestChangeBootOrder(t *testing.T) {
	t.Parallel()

	device := &entity.Device{
		GUID:     "device-guid-123",
		Password: "encrypted",
		TenantID: "tenant-id-456",
	}

	bootSource := "pxe"

	changeBootOrderResponse := cimBoot.ChangeBootOrder_OUTPUT{
		ReturnValue: 0,
	}

	tests := []bootTest{
		{
			name: "success",
			manMock: func(man *mocks.MockWSMAN, hmm *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(hmm, nil)
				hmm.EXPECT().
					ChangeBootOrder(bootSource).
					Return(changeBootOrderResponse, nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			err: nil,
		},
		{
			name: "failed to get device from repo",
			manMock: func(_ *mocks.MockWSMAN, _ *mocks.MockManagement) {
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(nil, ErrGeneral)
			},
			err: ErrGeneral,
		},
		{
			name: "device not found",
			manMock: func(_ *mocks.MockWSMAN, _ *mocks.MockManagement) {
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(nil, nil)
			},
			err: devices.ErrNotFound,
		},
		{
			name: "failed to setup wsman client",
			manMock: func(man *mocks.MockWSMAN, _ *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
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
			name: "failed to change boot order",
			manMock: func(man *mocks.MockWSMAN, hmm *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(hmm, nil)
				hmm.EXPECT().
					ChangeBootOrder(bootSource).
					Return(cimBoot.ChangeBootOrder_OUTPUT{}, ErrGeneral)
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

			useCase, wsmanMock, managementMock, repo := initBootTest(t)

			tc.repoMock(repo)
			tc.manMock(wsmanMock, managementMock)

			err := useCase.ChangeBootOrder(context.Background(), device.GUID, bootSource)

			require.Equal(t, tc.err, err)
		})
	}
}
