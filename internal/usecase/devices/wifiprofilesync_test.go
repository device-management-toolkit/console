package devices_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/boot"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/wifiportconfiguration"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/wifi"

	"github.com/device-management-toolkit/console/internal/entity"
	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/mocks"
	"github.com/device-management-toolkit/console/internal/usecase/devices"
	"github.com/device-management-toolkit/console/pkg/logger"
)

var errWiFiProfileSync = errors.New("wifi profile sync failure")

func testBoolPtr(b bool) *bool { return &b }

func initWiFiProfileSyncTest(t *testing.T) (*devices.UseCase, *mocks.MockWSMAN, *mocks.MockManagement, *mocks.MockDeviceManagementRepository) {
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

func TestWiFiProfileSync(t *testing.T) {
	t.Parallel()

	const guid = "device-guid-123"

	tests := []struct {
		name      string
		setupMock func(t *testing.T, repo *mocks.MockDeviceManagementRepository, wsmanMock *mocks.MockWSMAN, management *mocks.MockManagement)
		invoke    func(useCase *devices.UseCase) (dto.WirelessProfileSyncResponse, error)
		want      dto.WirelessProfileSyncResponse
		wantErr   bool
		wantErrIs error
	}{
		{
			name: "get wireless profile sync - enabled and supported",
			setupMock: func(_ *testing.T, repo *mocks.MockDeviceManagementRepository, wsmanMock *mocks.MockWSMAN, management *mocks.MockManagement) {
				expectSetupSuccess(repo, wsmanMock, management)
				management.EXPECT().GetWiFiPorts().Return([]wifi.WiFiPort{{}}, nil)
				management.EXPECT().GetWiFiPortConfigurationService().Return(wifiportconfiguration.WiFiPortConfigurationServiceResponse{
					LocalProfileSynchronizationEnabled: wifiportconfiguration.LocalUserProfileSynchronizationEnabled,
					UEFIWiFiProfileShareEnabled:        true,
				}, nil)
				management.EXPECT().GetPowerCapabilities().Return(boot.BootCapabilitiesResponse{UEFIWiFiCoExistenceAndProfileShare: true}, nil)
			},
			invoke: func(useCase *devices.UseCase) (dto.WirelessProfileSyncResponse, error) {
				return useCase.GetWirelessProfileSync(context.Background(), guid)
			},
			want: dto.WirelessProfileSyncResponse{LocalProfileSync: true, UEFIProfileSync: true, UEFIProfileSyncSupported: true},
		},
		{
			name: "get wireless profile sync - disabled and unsupported",
			setupMock: func(_ *testing.T, repo *mocks.MockDeviceManagementRepository, wsmanMock *mocks.MockWSMAN, management *mocks.MockManagement) {
				expectSetupSuccess(repo, wsmanMock, management)
				management.EXPECT().GetWiFiPorts().Return([]wifi.WiFiPort{{}}, nil)
				management.EXPECT().GetWiFiPortConfigurationService().Return(wifiportconfiguration.WiFiPortConfigurationServiceResponse{
					LocalProfileSynchronizationEnabled: wifiportconfiguration.LocalSyncDisabled,
					UEFIWiFiProfileShareEnabled:        false,
				}, nil)
				management.EXPECT().GetPowerCapabilities().Return(boot.BootCapabilitiesResponse{UEFIWiFiCoExistenceAndProfileShare: false}, nil)
			},
			invoke: func(useCase *devices.UseCase) (dto.WirelessProfileSyncResponse, error) {
				return useCase.GetWirelessProfileSync(context.Background(), guid)
			},
			want: dto.WirelessProfileSyncResponse{LocalProfileSync: false, UEFIProfileSync: false, UEFIProfileSyncSupported: false},
		},
		{
			name: "get wireless profile sync - setup error",
			setupMock: func(_ *testing.T, repo *mocks.MockDeviceManagementRepository, _ *mocks.MockWSMAN, _ *mocks.MockManagement) {
				repo.EXPECT().GetByID(context.Background(), guid, "").Return(nil, errWiFiProfileSync)
			},
			invoke: func(useCase *devices.UseCase) (dto.WirelessProfileSyncResponse, error) {
				return useCase.GetWirelessProfileSync(context.Background(), guid)
			},
			wantErr: true,
		},
		{
			name: "get wireless profile sync - get wifi ports error",
			setupMock: func(_ *testing.T, repo *mocks.MockDeviceManagementRepository, wsmanMock *mocks.MockWSMAN, management *mocks.MockManagement) {
				expectSetupSuccess(repo, wsmanMock, management)
				management.EXPECT().GetWiFiPorts().Return(nil, errWiFiProfileSync)
			},
			invoke: func(useCase *devices.UseCase) (dto.WirelessProfileSyncResponse, error) {
				return useCase.GetWirelessProfileSync(context.Background(), guid)
			},
			wantErr: true,
		},
		{
			name: "get wireless profile sync - get config error",
			setupMock: func(_ *testing.T, repo *mocks.MockDeviceManagementRepository, wsmanMock *mocks.MockWSMAN, management *mocks.MockManagement) {
				expectSetupSuccess(repo, wsmanMock, management)
				management.EXPECT().GetWiFiPorts().Return([]wifi.WiFiPort{{}}, nil)
				management.EXPECT().GetWiFiPortConfigurationService().Return(wifiportconfiguration.WiFiPortConfigurationServiceResponse{}, errWiFiProfileSync)
			},
			invoke: func(useCase *devices.UseCase) (dto.WirelessProfileSyncResponse, error) {
				return useCase.GetWirelessProfileSync(context.Background(), guid)
			},
			wantErr: true,
		},
		{
			name: "get wireless profile sync - power capabilities error",
			setupMock: func(_ *testing.T, repo *mocks.MockDeviceManagementRepository, wsmanMock *mocks.MockWSMAN, management *mocks.MockManagement) {
				expectSetupSuccess(repo, wsmanMock, management)
				management.EXPECT().GetWiFiPorts().Return([]wifi.WiFiPort{{}}, nil)
				management.EXPECT().GetWiFiPortConfigurationService().Return(wifiportconfiguration.WiFiPortConfigurationServiceResponse{}, nil)
				management.EXPECT().GetPowerCapabilities().Return(boot.BootCapabilitiesResponse{}, errWiFiProfileSync)
			},
			invoke: func(useCase *devices.UseCase) (dto.WirelessProfileSyncResponse, error) {
				return useCase.GetWirelessProfileSync(context.Background(), guid)
			},
			wantErr: true,
		},
		{
			name: "set wireless profile sync - enable local (PUT issued)",
			setupMock: func(t *testing.T, repo *mocks.MockDeviceManagementRepository, wsmanMock *mocks.MockWSMAN, management *mocks.MockManagement) {
				t.Helper()

				expectSetupSuccess(repo, wsmanMock, management)
				management.EXPECT().GetWiFiPorts().Return([]wifi.WiFiPort{{}}, nil)
				management.EXPECT().GetWiFiPortConfigurationService().Return(wifiportconfiguration.WiFiPortConfigurationServiceResponse{
					LocalProfileSynchronizationEnabled: wifiportconfiguration.LocalSyncDisabled,
					UEFIWiFiProfileShareEnabled:        false,
				}, nil)
				management.EXPECT().GetPowerCapabilities().Return(boot.BootCapabilitiesResponse{UEFIWiFiCoExistenceAndProfileShare: true}, nil)
				management.EXPECT().PutWiFiPortConfigurationService(gomock.Any()).DoAndReturn(func(req wifiportconfiguration.WiFiPortConfigurationServiceRequest) (wifiportconfiguration.WiFiPortConfigurationServiceResponse, error) {
					assert.Equal(t, wifiportconfiguration.LocalUserProfileSynchronizationEnabled, req.LocalProfileSynchronizationEnabled)
					assert.False(t, req.UEFIWiFiProfileShareEnabled)

					return wifiportconfiguration.WiFiPortConfigurationServiceResponse{
						LocalProfileSynchronizationEnabled: wifiportconfiguration.LocalUserProfileSynchronizationEnabled,
						UEFIWiFiProfileShareEnabled:        false,
					}, nil
				})
			},
			invoke: func(useCase *devices.UseCase) (dto.WirelessProfileSyncResponse, error) {
				return useCase.SetWirelessProfileSync(context.Background(), guid, dto.WirelessProfileSyncRequest{LocalProfileSync: testBoolPtr(true)})
			},
			want: dto.WirelessProfileSyncResponse{LocalProfileSync: true, UEFIProfileSync: false, UEFIProfileSyncSupported: true},
		},
		{
			name: "set wireless profile sync - enable uefi supported (PUT issued)",
			setupMock: func(t *testing.T, repo *mocks.MockDeviceManagementRepository, wsmanMock *mocks.MockWSMAN, management *mocks.MockManagement) {
				t.Helper()

				expectSetupSuccess(repo, wsmanMock, management)
				management.EXPECT().GetWiFiPorts().Return([]wifi.WiFiPort{{}}, nil)
				management.EXPECT().GetWiFiPortConfigurationService().Return(wifiportconfiguration.WiFiPortConfigurationServiceResponse{
					LocalProfileSynchronizationEnabled: wifiportconfiguration.LocalSyncDisabled,
					UEFIWiFiProfileShareEnabled:        false,
				}, nil)
				management.EXPECT().GetPowerCapabilities().Return(boot.BootCapabilitiesResponse{UEFIWiFiCoExistenceAndProfileShare: true}, nil)
				management.EXPECT().PutWiFiPortConfigurationService(gomock.Any()).DoAndReturn(func(req wifiportconfiguration.WiFiPortConfigurationServiceRequest) (wifiportconfiguration.WiFiPortConfigurationServiceResponse, error) {
					assert.True(t, req.UEFIWiFiProfileShareEnabled)

					return wifiportconfiguration.WiFiPortConfigurationServiceResponse{
						LocalProfileSynchronizationEnabled: wifiportconfiguration.LocalSyncDisabled,
						UEFIWiFiProfileShareEnabled:        true,
					}, nil
				})
			},
			invoke: func(useCase *devices.UseCase) (dto.WirelessProfileSyncResponse, error) {
				return useCase.SetWirelessProfileSync(context.Background(), guid, dto.WirelessProfileSyncRequest{UEFIProfileSync: testBoolPtr(true)})
			},
			want: dto.WirelessProfileSyncResponse{LocalProfileSync: false, UEFIProfileSync: true, UEFIProfileSyncSupported: true},
		},
		{
			name: "set wireless profile sync - enable uefi unsupported rejected (no PUT)",
			setupMock: func(_ *testing.T, repo *mocks.MockDeviceManagementRepository, wsmanMock *mocks.MockWSMAN, management *mocks.MockManagement) {
				expectSetupSuccess(repo, wsmanMock, management)
				management.EXPECT().GetWiFiPorts().Return([]wifi.WiFiPort{{}}, nil)
				management.EXPECT().GetWiFiPortConfigurationService().Return(wifiportconfiguration.WiFiPortConfigurationServiceResponse{
					LocalProfileSynchronizationEnabled: wifiportconfiguration.LocalSyncDisabled,
					UEFIWiFiProfileShareEnabled:        false,
				}, nil)
				management.EXPECT().GetPowerCapabilities().Return(boot.BootCapabilitiesResponse{UEFIWiFiCoExistenceAndProfileShare: false}, nil)
			},
			invoke: func(useCase *devices.UseCase) (dto.WirelessProfileSyncResponse, error) {
				return useCase.SetWirelessProfileSync(context.Background(), guid, dto.WirelessProfileSyncRequest{LocalProfileSync: testBoolPtr(true), UEFIProfileSync: testBoolPtr(true)})
			},
			wantErr:   true,
			wantErrIs: devices.ErrUEFIProfileSyncNotSupported,
		},
		{
			name: "set wireless profile sync - no change needed",
			setupMock: func(_ *testing.T, repo *mocks.MockDeviceManagementRepository, wsmanMock *mocks.MockWSMAN, management *mocks.MockManagement) {
				expectSetupSuccess(repo, wsmanMock, management)
				management.EXPECT().GetWiFiPorts().Return([]wifi.WiFiPort{{}}, nil)
				management.EXPECT().GetWiFiPortConfigurationService().Return(wifiportconfiguration.WiFiPortConfigurationServiceResponse{
					LocalProfileSynchronizationEnabled: wifiportconfiguration.LocalUserProfileSynchronizationEnabled,
					UEFIWiFiProfileShareEnabled:        false,
				}, nil)
				management.EXPECT().GetPowerCapabilities().Return(boot.BootCapabilitiesResponse{UEFIWiFiCoExistenceAndProfileShare: true}, nil)
			},
			invoke: func(useCase *devices.UseCase) (dto.WirelessProfileSyncResponse, error) {
				return useCase.SetWirelessProfileSync(context.Background(), guid, dto.WirelessProfileSyncRequest{LocalProfileSync: testBoolPtr(true), UEFIProfileSync: testBoolPtr(false)})
			},
			want: dto.WirelessProfileSyncResponse{LocalProfileSync: true, UEFIProfileSync: false, UEFIProfileSyncSupported: true},
		},
		{
			name: "set wireless profile sync - empty request returns current state",
			setupMock: func(_ *testing.T, repo *mocks.MockDeviceManagementRepository, wsmanMock *mocks.MockWSMAN, management *mocks.MockManagement) {
				expectSetupSuccess(repo, wsmanMock, management)
				management.EXPECT().GetWiFiPorts().Return([]wifi.WiFiPort{{}}, nil)
				management.EXPECT().GetWiFiPortConfigurationService().Return(wifiportconfiguration.WiFiPortConfigurationServiceResponse{
					LocalProfileSynchronizationEnabled: wifiportconfiguration.LocalUserProfileSynchronizationEnabled,
					UEFIWiFiProfileShareEnabled:        true,
				}, nil)
				management.EXPECT().GetPowerCapabilities().Return(boot.BootCapabilitiesResponse{UEFIWiFiCoExistenceAndProfileShare: true}, nil)
			},
			invoke: func(useCase *devices.UseCase) (dto.WirelessProfileSyncResponse, error) {
				return useCase.SetWirelessProfileSync(context.Background(), guid, dto.WirelessProfileSyncRequest{})
			},
			want: dto.WirelessProfileSyncResponse{LocalProfileSync: true, UEFIProfileSync: true, UEFIProfileSyncSupported: true},
		},
		{
			name: "set wireless profile sync - setup error",
			setupMock: func(_ *testing.T, repo *mocks.MockDeviceManagementRepository, _ *mocks.MockWSMAN, _ *mocks.MockManagement) {
				repo.EXPECT().GetByID(context.Background(), guid, "").Return(nil, errWiFiProfileSync)
			},
			invoke: func(useCase *devices.UseCase) (dto.WirelessProfileSyncResponse, error) {
				return useCase.SetWirelessProfileSync(context.Background(), guid, dto.WirelessProfileSyncRequest{LocalProfileSync: testBoolPtr(true)})
			},
			wantErr: true,
		},
		{
			name: "set wireless profile sync - get wifi ports error",
			setupMock: func(_ *testing.T, repo *mocks.MockDeviceManagementRepository, wsmanMock *mocks.MockWSMAN, management *mocks.MockManagement) {
				expectSetupSuccess(repo, wsmanMock, management)
				management.EXPECT().GetWiFiPorts().Return(nil, errWiFiProfileSync)
			},
			invoke: func(useCase *devices.UseCase) (dto.WirelessProfileSyncResponse, error) {
				return useCase.SetWirelessProfileSync(context.Background(), guid, dto.WirelessProfileSyncRequest{LocalProfileSync: testBoolPtr(true)})
			},
			wantErr: true,
		},
		{
			name: "set wireless profile sync - get config error",
			setupMock: func(_ *testing.T, repo *mocks.MockDeviceManagementRepository, wsmanMock *mocks.MockWSMAN, management *mocks.MockManagement) {
				expectSetupSuccess(repo, wsmanMock, management)
				management.EXPECT().GetWiFiPorts().Return([]wifi.WiFiPort{{}}, nil)
				management.EXPECT().GetWiFiPortConfigurationService().Return(wifiportconfiguration.WiFiPortConfigurationServiceResponse{}, errWiFiProfileSync)
			},
			invoke: func(useCase *devices.UseCase) (dto.WirelessProfileSyncResponse, error) {
				return useCase.SetWirelessProfileSync(context.Background(), guid, dto.WirelessProfileSyncRequest{LocalProfileSync: testBoolPtr(true)})
			},
			wantErr: true,
		},
		{
			name: "set wireless profile sync - power capabilities error",
			setupMock: func(_ *testing.T, repo *mocks.MockDeviceManagementRepository, wsmanMock *mocks.MockWSMAN, management *mocks.MockManagement) {
				expectSetupSuccess(repo, wsmanMock, management)
				management.EXPECT().GetWiFiPorts().Return([]wifi.WiFiPort{{}}, nil)
				management.EXPECT().GetWiFiPortConfigurationService().Return(wifiportconfiguration.WiFiPortConfigurationServiceResponse{}, nil)
				management.EXPECT().GetPowerCapabilities().Return(boot.BootCapabilitiesResponse{}, errWiFiProfileSync)
			},
			invoke: func(useCase *devices.UseCase) (dto.WirelessProfileSyncResponse, error) {
				return useCase.SetWirelessProfileSync(context.Background(), guid, dto.WirelessProfileSyncRequest{LocalProfileSync: testBoolPtr(true)})
			},
			wantErr: true,
		},
		{
			name: "set wireless profile sync - put config error",
			setupMock: func(_ *testing.T, repo *mocks.MockDeviceManagementRepository, wsmanMock *mocks.MockWSMAN, management *mocks.MockManagement) {
				expectSetupSuccess(repo, wsmanMock, management)
				management.EXPECT().GetWiFiPorts().Return([]wifi.WiFiPort{{}}, nil)
				management.EXPECT().GetWiFiPortConfigurationService().Return(wifiportconfiguration.WiFiPortConfigurationServiceResponse{
					LocalProfileSynchronizationEnabled: wifiportconfiguration.LocalSyncDisabled,
				}, nil)
				management.EXPECT().GetPowerCapabilities().Return(boot.BootCapabilitiesResponse{UEFIWiFiCoExistenceAndProfileShare: true}, nil)
				management.EXPECT().PutWiFiPortConfigurationService(gomock.Any()).Return(wifiportconfiguration.WiFiPortConfigurationServiceResponse{}, errWiFiProfileSync)
			},
			invoke: func(useCase *devices.UseCase) (dto.WirelessProfileSyncResponse, error) {
				return useCase.SetWirelessProfileSync(context.Background(), guid, dto.WirelessProfileSyncRequest{LocalProfileSync: testBoolPtr(true)})
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, wsmanMock, management, repo := initWiFiProfileSyncTest(t)
			tc.setupMock(t, repo, wsmanMock, management)

			got, err := tc.invoke(useCase)

			if tc.wantErr {
				require.Error(t, err)

				if tc.wantErrIs != nil {
					require.ErrorIs(t, err, tc.wantErrIs)
				}

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func expectSetupSuccess(repo *mocks.MockDeviceManagementRepository, wsmanMock *mocks.MockWSMAN, management *mocks.MockManagement) {
	const guid = "device-guid-123"

	device := &entity.Device{GUID: guid}

	repo.EXPECT().GetByID(context.Background(), guid, "").Return(device, nil)
	wsmanMock.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(management, nil)
}
