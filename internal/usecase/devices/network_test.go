package devices_test

import (
	"context"
	"testing"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/ethernetport"
	cimieee8021x "github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/ieee8021x"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/wifi"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/ips/ieee8021x"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"

	"github.com/device-management-toolkit/console/internal/entity"
	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/mocks"
	devices "github.com/device-management-toolkit/console/internal/usecase/devices"
	"github.com/device-management-toolkit/console/internal/usecase/devices/wsman"
	"github.com/device-management-toolkit/console/pkg/logger"
)

func initNetworkTest(t *testing.T) (*devices.UseCase, *mocks.MockWSMAN, *mocks.MockManagement, *mocks.MockDeviceManagementRepository) {
	t.Helper()

	mockCtl := gomock.NewController(t)
	defer mockCtl.Finish()

	repo := mocks.NewMockDeviceManagementRepository(mockCtl)
	wsmanMock := mocks.NewMockWSMAN(mockCtl)
	wsmanMock.EXPECT().Worker().Return().AnyTimes()

	management := mocks.NewMockManagement(mockCtl)
	log := logger.New("error")
	u := devices.New(repo, wsmanMock, mocks.NewMockRedirection(mockCtl), log, mocks.MockCrypto{})

	return u, wsmanMock, management, repo
}

func TestGetNetworkSettings(t *testing.T) {
	t.Parallel()

	device := &entity.Device{
		GUID:     "device-guid-123",
		TenantID: "tenant-id-456",
	}

	tests := []test{
		{
			name:   "success",
			action: 0,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(man2)
				man2.EXPECT().
					GetNetworkSettings().
					Return(wsman.NetworkResults{
						EthernetPortSettingsResult: []ethernetport.SettingsResponse{
							{
								ElementName:            "Intel(r) AMT Ethernet Port Settings",
								InstanceID:             "Intel(r) AMT Ethernet Port Settings 0",
								LinkPolicy:             []ethernetport.LinkPolicy{14, 16},
								PhysicalConnectionType: 0,
								PhysicalNicMedium:      0,
							}, {
								ElementName:             "Intel(r) AMT Ethernet Port Settings",
								InstanceID:              "Intel(r) AMT Ethernet Port Settings 1",
								LinkPolicy:              []ethernetport.LinkPolicy{14, 16},
								LinkPreference:          1,
								LinkControl:             1,
								WLANLinkProtectionLevel: 1,
								PhysicalConnectionType:  3,
								PhysicalNicMedium:       1,
							},
						},
						IPSIEEE8021xSettingsResult: ieee8021x.IEEE8021xSettingsResponse{
							Enabled:       3,
							AvailableInS0: false,
							PxeTimeout:    0,
						},
						WiFiSettingsResult: []wifi.WiFiEndpointSettingsResponse{{
							ElementName:          "test-ssid",
							SSID:                 "test-ssid",
							AuthenticationMethod: 6,
							EncryptionMethod:     3,
							Priority:             1,
							BSSType:              2,
						}},
						CIMIEEE8021xSettingsResult: cimieee8021x.PullResponse{
							IEEE8021xSettingsItems: []cimieee8021x.IEEE8021xSettingsResponse{{}},
						},
					}, nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: dto.NetworkSettings{
				Wired: &dto.WiredNetworkInfo{
					IEEE8021x: dto.IEEE8021x{
						Enabled:       "Disabled",
						AvailableInS0: false,
						PxeTimeout:    0,
					},
					NetworkInfo: dto.NetworkInfo{
						ElementName:            "Intel(r) AMT Ethernet Port Settings",
						InstanceID:             "Intel(r) AMT Ethernet Port Settings 0",
						LinkPolicy:             []string{"Sx AC", "S0 DC"},
						PhysicalConnectionType: "Integrated LAN NIC",
						PhysicalNicMedium:      "SMBUS",
					},
				},
				Wireless: &dto.WirelessNetworkInfo{
					WiFiNetworks: []dto.WiFiNetwork{{
						ElementName:          "test-ssid",
						SSID:                 "test-ssid",
						AuthenticationMethod: "WPA2PSK",
						EncryptionMethod:     "TKIP",
						Priority:             1,
						BSSType:              "Independent",
					}},
					IEEE8021xSettings: []dto.IEEE8021xSettings{{}},
					NetworkInfo: dto.NetworkInfo{
						ElementName:             "Intel(r) AMT Ethernet Port Settings",
						InstanceID:              "Intel(r) AMT Ethernet Port Settings 1",
						LinkPolicy:              []string{"Sx AC", "S0 DC"},
						LinkPreference:          "Management Engine",
						LinkControl:             "Management Engine",
						WLANLinkProtectionLevel: "None",
						PhysicalConnectionType:  "Wireless LAN",
						PhysicalNicMedium:       "PCIe",
					},
				},
			},
			err: nil,
		},
		{
			name:   "GetById fails",
			action: 0,
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(nil, ErrGeneral)
			},
			res: dto.NetworkSettings{},
			err: devices.ErrGeneral,
		},
		{
			name:   "GetNetworkSettings fails",
			action: 0,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(man2)
				man2.EXPECT().
					GetNetworkSettings().
					Return(wsman.NetworkResults{}, ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: dto.NetworkSettings{},
			err: ErrGeneral,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, wsmanMock, management, repo := initNetworkTest(t)

			if tc.manMock != nil {
				tc.manMock(wsmanMock, management)
			}

			tc.repoMock(repo)

			res, err := useCase.GetNetworkSettings(context.Background(), device.GUID)

			require.Equal(t, tc.res, res)
			require.IsType(t, tc.err, err)
		})
	}
}
