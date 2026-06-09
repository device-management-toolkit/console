package devices_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/wifi"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/common"

	"github.com/device-management-toolkit/console/internal/entity"
	"github.com/device-management-toolkit/console/internal/mocks"
	"github.com/device-management-toolkit/console/internal/usecase/devices"
	"github.com/device-management-toolkit/console/internal/usecase/devices/wsman"
	"github.com/device-management-toolkit/console/pkg/logger"
)

func initWiFiStateTest(t *testing.T) (*devices.UseCase, *mocks.MockWSMAN, *mocks.MockManagement, *mocks.MockDeviceManagementRepository) {
	t.Helper()

	mockCtl := gomock.NewController(t)

	repo := mocks.NewMockDeviceManagementRepository(mockCtl)
	wsmanMock := mocks.NewMockWSMAN(mockCtl)
	wsmanMock.EXPECT().Worker().Return().AnyTimes()

	management := mocks.NewMockManagement(mockCtl)
	log := logger.New("error")
	u := devices.New(repo, wsmanMock, mocks.NewMockRedirection(mockCtl), log, mocks.MockCrypto{})

	return u, wsmanMock, management, repo
}

func TestRequestWirelessStateChange(t *testing.T) {
	t.Parallel()

	device := &entity.Device{GUID: "device-guid-123"}

	tests := []struct {
		name      string
		request   wifi.RequestedState
		manMock   func(*mocks.MockWSMAN, *mocks.MockManagement)
		repoMock  func(*mocks.MockDeviceManagementRepository)
		res       wifi.RequestedState
		err       error
		errString bool
	}{
		{
			name:    "success - enable S0 + Sx/AC",
			request: wifi.RequestedStateWifiEnabledS0SxAC,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(man2, nil)
				man2.EXPECT().WiFiRequestStateChange(wifi.RequestedStateWifiEnabledS0SxAC).Return(nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
			},
			res: wifi.RequestedStateWifiEnabledS0SxAC,
			err: nil,
		},
		{
			name:    "success - disable",
			request: wifi.RequestedStateWifiDisabled,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(man2, nil)
				man2.EXPECT().WiFiRequestStateChange(wifi.RequestedStateWifiDisabled).Return(nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
			},
			res: wifi.RequestedStateWifiDisabled,
			err: nil,
		},
		{
			name:    "success - enable S0",
			request: wifi.RequestedStateWifiEnabledS0,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(man2, nil)
				man2.EXPECT().WiFiRequestStateChange(wifi.RequestedStateWifiEnabledS0).Return(nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
			},
			res: wifi.RequestedStateWifiEnabledS0,
			err: nil,
		},
		{
			name:      "failure - invalid requested state",
			request:   wifi.RequestedState(2),
			manMock:   nil,
			repoMock:  nil,
			res:       0,
			err:       devices.ErrValidationUseCase.Wrap("RequestWirelessStateChange", "validate requested state", "state must be one of 3, 32768, 32769"),
			errString: true,
		},
		{
			name:    "GetByID fails",
			request: wifi.RequestedStateWifiDisabled,
			manMock: nil,
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(nil, ErrGeneral)
			},
			res: 0,
			err: devices.ErrGeneral,
		},
		{
			name:    "device not found",
			request: wifi.RequestedStateWifiDisabled,
			manMock: nil,
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(nil, nil)
			},
			res: 0,
			err: devices.ErrNotFound,
		},
		{
			name:    "SetupWsmanClient fails",
			request: wifi.RequestedStateWifiDisabled,
			manMock: func(man *mocks.MockWSMAN, _ *mocks.MockManagement) {
				man.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(nil, ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
			},
			res: 0,
			err: devices.ErrGeneral,
		},
		{
			name:    "WiFiRequestStateChange fails",
			request: wifi.RequestedStateWifiDisabled,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(wsman.Management(man2), nil)
				man2.EXPECT().WiFiRequestStateChange(wifi.RequestedStateWifiDisabled).Return(ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
			},
			res: 0,
			err: devices.ErrGeneral,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, wsmanMock, management, repo := initWiFiStateTest(t)

			if tc.manMock != nil {
				tc.manMock(wsmanMock, management)
			}

			if tc.repoMock != nil {
				tc.repoMock(repo)
			}

			res, err := useCase.RequestWirelessStateChange(context.Background(), device.GUID, tc.request)

			require.Equal(t, tc.res, res)

			if tc.err != nil {
				require.IsType(t, tc.err, err)

				if tc.errString {
					assert.Equal(t, tc.err.Error(), err.Error())
				}

				return
			}

			require.NoError(t, err)
		})
	}
}

func TestGetWirelessState(t *testing.T) {
	t.Parallel()

	device := &entity.Device{GUID: "device-guid-123"}

	tests := []struct {
		name     string
		manMock  func(*mocks.MockWSMAN, *mocks.MockManagement)
		repoMock func(*mocks.MockDeviceManagementRepository)
		res      wifi.EnabledState
		err      error
	}{
		{
			name: "success",
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(man2, nil)
				man2.EXPECT().EnumerateWiFiPort().Return(wifi.Response{
					Body: wifi.Body{
						EnumerateResponse: common.EnumerateResponse{EnumerationContext: "test-context"},
					},
				}, nil)
				man2.EXPECT().PullWiFiPort("test-context").Return(wifi.Response{
					Body: wifi.Body{
						PullResponse: wifi.PullResponse{
							WiFiPortItems: []wifi.WiFiPort{{EnabledState: 32769}},
						},
					},
				}, nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
			},
			res: wifi.EnabledStateWifiEnabledS0SxAC,
			err: nil,
		},
		{
			name:    "GetByID fails",
			manMock: nil,
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(nil, ErrGeneral)
			},
			res: 0,
			err: devices.ErrGeneral,
		},
		{
			name:    "device not found",
			manMock: nil,
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(nil, nil)
			},
			res: 0,
			err: devices.ErrNotFound,
		},
		{
			name: "SetupWsmanClient fails",
			manMock: func(man *mocks.MockWSMAN, _ *mocks.MockManagement) {
				man.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(nil, ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
			},
			res: 0,
			err: devices.ErrGeneral,
		},
		{
			name: "EnumerateWiFiPort fails",
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(man2, nil)
				man2.EXPECT().EnumerateWiFiPort().Return(wifi.Response{}, ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
			},
			res: 0,
			err: devices.ErrGeneral,
		},
		{
			name: "PullWiFiPort fails",
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(man2, nil)
				man2.EXPECT().EnumerateWiFiPort().Return(wifi.Response{
					Body: wifi.Body{
						EnumerateResponse: common.EnumerateResponse{EnumerationContext: "test-context"},
					},
				}, nil)
				man2.EXPECT().PullWiFiPort("test-context").Return(wifi.Response{}, ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
			},
			res: 0,
			err: devices.ErrGeneral,
		},
		{
			name: "no wifi ports found",
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(man2, nil)
				man2.EXPECT().EnumerateWiFiPort().Return(wifi.Response{
					Body: wifi.Body{
						EnumerateResponse: common.EnumerateResponse{EnumerationContext: "test-context"},
					},
				}, nil)
				man2.EXPECT().PullWiFiPort("test-context").Return(wifi.Response{
					Body: wifi.Body{
						PullResponse: wifi.PullResponse{
							WiFiPortItems: []wifi.WiFiPort{},
						},
					},
				}, nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
			},
			res: 0,
			err: wsman.ErrNoWiFiPort,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, wsmanMock, management, repo := initWiFiStateTest(t)

			if tc.manMock != nil {
				tc.manMock(wsmanMock, management)
			}

			if tc.repoMock != nil {
				tc.repoMock(repo)
			}

			res, err := useCase.GetWirelessState(context.Background(), device.GUID)

			require.Equal(t, tc.res, res)

			if tc.err != nil {
				require.IsType(t, tc.err, err)

				return
			}

			require.NoError(t, err)
		})
	}
}
