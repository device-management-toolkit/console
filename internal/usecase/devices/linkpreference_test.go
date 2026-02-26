package devices_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"

	"github.com/device-management-toolkit/console/internal/cache"
	"github.com/device-management-toolkit/console/internal/entity"
	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/mocks"
	devices "github.com/device-management-toolkit/console/internal/usecase/devices"
	"github.com/device-management-toolkit/console/internal/usecase/devices/wsman"
	"github.com/device-management-toolkit/console/pkg/logger"
)

func initLinkPreferenceTest(t *testing.T) (*devices.UseCase, *mocks.MockWSMAN, *mocks.MockManagement, *mocks.MockDeviceManagementRepository) {
	t.Helper()

	mockCtl := gomock.NewController(t)
	defer mockCtl.Finish()

	repo := mocks.NewMockDeviceManagementRepository(mockCtl)
	wsmanMock := mocks.NewMockWSMAN(mockCtl)
	wsmanMock.EXPECT().Worker().Return().AnyTimes()

	management := mocks.NewMockManagement(mockCtl)
	log := logger.New("error")
	u := devices.New(repo, wsmanMock, mocks.NewMockRedirection(mockCtl), log, mocks.MockCrypto{}, cache.New(30*time.Second, 5*time.Second))

	return u, wsmanMock, management, repo
}

func TestSetLinkPreference(t *testing.T) {
	t.Parallel()

	device := &entity.Device{
		GUID:     "device-guid-123",
		TenantID: "tenant-id-456",
	}

	request := dto.LinkPreferenceRequest{
		LinkPreference: 1, // ME
		Timeout:        300,
	}

	tests := []struct {
		name     string
		request  dto.LinkPreferenceRequest
		manMock  func(*mocks.MockWSMAN, *mocks.MockManagement)
		repoMock func(*mocks.MockDeviceManagementRepository)
		res      dto.LinkPreferenceResponse
		err      error
	}{
		{
			name:    "success - set to ME",
			request: request,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(wsman.Management(man2), nil)
				man2.EXPECT().
					SetLinkPreference(uint32(1), uint32(300)).
					Return(0, nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: dto.LinkPreferenceResponse{ReturnValue: 0},
			err: nil,
		},
		{
			name: "success - set to HOST",
			request: dto.LinkPreferenceRequest{
				LinkPreference: 2, // HOST
				Timeout:        60,
			},
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(wsman.Management(man2), nil)
				man2.EXPECT().
					SetLinkPreference(uint32(2), uint32(60)).
					Return(0, nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: dto.LinkPreferenceResponse{ReturnValue: 0},
			err: nil,
		},
		{
			name:    "GetById fails - device not found",
			request: request,
			manMock: func(_ *mocks.MockWSMAN, _ *mocks.MockManagement) {},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(nil, ErrGeneral)
			},
			res: dto.LinkPreferenceResponse{},
			err: devices.ErrGeneral,
		},
		{
			name:    "no WiFi port found",
			request: request,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(man2, nil)
				man2.EXPECT().
					SetLinkPreference(uint32(1), uint32(300)).
					Return(-1, wsman.ErrNoWiFiPort)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: dto.LinkPreferenceResponse{ReturnValue: -1},
			err: wsman.ErrNoWiFiPort,
		},
		{
			name:    "SetLinkPreference fails with AMT error",
			request: request,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(man2, nil)
				man2.EXPECT().
					SetLinkPreference(uint32(1), uint32(300)).
					Return(5, errors.New("invalid parameter"))
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: dto.LinkPreferenceResponse{ReturnValue: 5},
			err: errors.New("invalid parameter"),
		},
		{
			name:    "SetLinkPreference fails with general error",
			request: request,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(man2, nil)
				man2.EXPECT().
					SetLinkPreference(uint32(1), uint32(300)).
					Return(0, ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: dto.LinkPreferenceResponse{ReturnValue: 0},
			err: ErrGeneral,
		},
		{
			name: "failure - incorrect device type",
			request: dto.LinkPreferenceRequest{
				LinkPreference: 3, // Invalid
				Timeout:        300,
			},
			manMock: func(_ *mocks.MockWSMAN, _ *mocks.MockManagement) {},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: dto.LinkPreferenceResponse{},
			err: devices.ErrValidationUseCase.Wrap("SetLinkPreference", "validate link preference", "linkPreference must be 1 (ME) or 2 (Host)"),
		},
		{
			name: "failure - timeout beyond range",
			request: dto.LinkPreferenceRequest{
				LinkPreference: 1,
				Timeout:        65536, // Beyond maxTimeout (65535)
			},
			manMock: func(_ *mocks.MockWSMAN, _ *mocks.MockManagement) {},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: dto.LinkPreferenceResponse{},
			err: devices.ErrValidationUseCase.Wrap("SetLinkPreference", "validate timeout", "timeout max value is 65535"),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, wsmanMock, management, repo := initLinkPreferenceTest(t)

			if tc.manMock != nil {
				tc.manMock(wsmanMock, management)
			}

			tc.repoMock(repo)

			res, err := useCase.SetLinkPreference(context.Background(), device.GUID, tc.request)

			require.Equal(t, tc.res, res)

			if tc.err != nil {
				assert.Equal(t, err.Error(), tc.err.Error())
			}
		})
	}
}
