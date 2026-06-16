package devices_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/ethernetport"
	cimieee8021x "github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/ieee8021x"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/wifi"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/ips/ieee8021x"

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
					SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).
					Return(man2, nil)
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
					SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).
					Return(man2, nil)
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

func TestGetWiredNetworkSettings(t *testing.T) {
	t.Parallel()

	device := &entity.Device{
		GUID:     "device-guid-123",
		TenantID: "tenant-id-456",
	}

	wiredResult := wsman.NetworkResults{
		EthernetPortSettingsResult: []ethernetport.SettingsResponse{
			{
				ElementName: "Intel(r) AMT Ethernet Port Settings",
				InstanceID:  "Intel(r) AMT Ethernet Port Settings 0",
				DHCPEnabled: true,
				IPAddress:   "192.168.1.10",
			},
		},
		IPSIEEE8021xSettingsResult: ieee8021x.IEEE8021xSettingsResponse{Enabled: 3},
	}

	wirelessOnlyResult := wsman.NetworkResults{
		EthernetPortSettingsResult: []ethernetport.SettingsResponse{
			{
				ElementName: "Intel(r) AMT Ethernet Port Settings",
				InstanceID:  "Intel(r) AMT Ethernet Port Settings 1",
			},
		},
	}

	tests := []struct {
		name     string
		manMock  func(*mocks.MockWSMAN, *mocks.MockManagement)
		repoMock func(*mocks.MockDeviceManagementRepository)
		res      dto.WiredNetworkInfo
		err      error
	}{
		{
			name: "success",
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).
					Return(man2, nil)
				man2.EXPECT().
					GetNetworkSettings().
					Return(wiredResult, nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: dto.WiredNetworkInfo{
				IEEE8021x: dto.IEEE8021x{Enabled: "Disabled"},
				NetworkInfo: dto.NetworkInfo{
					ElementName:            "Intel(r) AMT Ethernet Port Settings",
					InstanceID:             "Intel(r) AMT Ethernet Port Settings 0",
					DHCPEnabled:            true,
					IPAddress:              "192.168.1.10",
					PhysicalConnectionType: "Integrated LAN NIC",
					PhysicalNicMedium:      "SMBUS",
				},
			},
			err: nil,
		},
		{
			name: "no wired interface returns not found",
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).
					Return(man2, nil)
				man2.EXPECT().
					GetNetworkSettings().
					Return(wirelessOnlyResult, nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: dto.WiredNetworkInfo{},
			err: devices.ErrNotFound,
		},
		{
			name: "GetById fails",
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(nil, ErrGeneral)
			},
			res: dto.WiredNetworkInfo{},
			err: devices.ErrGeneral,
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

			res, err := useCase.GetWiredNetworkSettings(context.Background(), device.GUID)

			require.Equal(t, tc.res, res)
			require.IsType(t, tc.err, err)
		})
	}
}

func TestPatchWiredNetworkSettings(t *testing.T) {
	t.Parallel()

	device := &entity.Device{
		GUID:     "device-guid-123",
		TenantID: "tenant-id-456",
	}

	currentSettings := []ethernetport.SettingsResponse{
		{
			ElementName: "Intel(r) AMT Ethernet Port Settings",
			InstanceID:  "Intel(r) AMT Ethernet Port Settings 0",
			DHCPEnabled: false,
			IPAddress:   "192.168.1.10",
		},
	}

	dhcpTrue := true
	dhcpFalse := false

	tests := []struct {
		name     string
		req      dto.WiredNetworkConfigRequest
		manMock  func(*testing.T, *mocks.MockWSMAN, *mocks.MockManagement)
		repoMock func(*mocks.MockDeviceManagementRepository)
		err      error
	}{
		{
			name: "success switch to dhcp",
			req:  dto.WiredNetworkConfigRequest{DHCPEnabled: &dhcpTrue},
			manMock: func(t *testing.T, man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				t.Helper()

				man.EXPECT().
					SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).
					Return(man2, nil)
				man2.EXPECT().
					GetEthernetPortSettings().
					Return(currentSettings, nil)
				man2.EXPECT().
					PutEthernetPortSettings(gomock.Any(), "Intel(r) AMT Ethernet Port Settings 0").
					DoAndReturn(func(req ethernetport.SettingsRequest, _ string) (ethernetport.Response, error) {
						// DHCP mode: AMT acquires its IPv4 config, host sync is forced
						// on, and any explicit IP fields must be cleared.
						require.True(t, req.DHCPEnabled)
						require.True(t, req.IpSyncEnabled)
						require.False(t, req.SharedStaticIp)
						require.Empty(t, req.IPAddress)
						require.Empty(t, req.SubnetMask)
						require.Empty(t, req.DefaultGateway)
						require.Empty(t, req.PrimaryDNS)
						require.Empty(t, req.SecondaryDNS)

						return ethernetport.Response{}, nil
					})
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			err: nil,
		},
		{
			name:     "validation error dhcp with static ip",
			req:      dto.WiredNetworkConfigRequest{DHCPEnabled: &dhcpTrue, IPAddress: "192.168.1.5"},
			manMock:  func(_ *testing.T, _ *mocks.MockWSMAN, _ *mocks.MockManagement) {},
			repoMock: func(_ *mocks.MockDeviceManagementRepository) {},
			err:      devices.ErrValidationUseCase,
		},
		{
			name:     "validation error ip sync with static ip",
			req:      dto.WiredNetworkConfigRequest{DHCPEnabled: &dhcpFalse, IPSyncEnabled: &dhcpTrue, IPAddress: "192.168.1.5"},
			manMock:  func(_ *testing.T, _ *mocks.MockWSMAN, _ *mocks.MockManagement) {},
			repoMock: func(_ *mocks.MockDeviceManagementRepository) {},
			err:      devices.ErrValidationUseCase,
		},
		{
			name: "not supported ieee8021x provided",
			req: dto.WiredNetworkConfigRequest{
				DHCPEnabled: &dhcpTrue,
				IEEE8021x:   &dto.WiredIEEE8021xConfig{ProfileName: "wired-eap"},
			},
			manMock:  func(_ *testing.T, _ *mocks.MockWSMAN, _ *mocks.MockManagement) {},
			repoMock: func(_ *mocks.MockDeviceManagementRepository) {},
			err:      devices.ErrNotSupportedUseCase,
		},
		{
			name:     "validation error nothing requested",
			req:      dto.WiredNetworkConfigRequest{DHCPEnabled: &dhcpFalse},
			manMock:  func(_ *testing.T, _ *mocks.MockWSMAN, _ *mocks.MockManagement) {},
			repoMock: func(_ *mocks.MockDeviceManagementRepository) {},
			err:      devices.ErrValidationUseCase,
		},
		{
			name:     "validation error static missing subnet",
			req:      dto.WiredNetworkConfigRequest{DHCPEnabled: &dhcpFalse, IPAddress: "192.168.1.5"},
			manMock:  func(_ *testing.T, _ *mocks.MockWSMAN, _ *mocks.MockManagement) {},
			repoMock: func(_ *mocks.MockDeviceManagementRepository) {},
			err:      devices.ErrValidationUseCase,
		},
		{
			name: "validation error static missing gateway",
			req: dto.WiredNetworkConfigRequest{
				DHCPEnabled: &dhcpFalse,
				IPAddress:   "192.168.1.5",
				SubnetMask:  "255.255.255.0",
			},
			manMock:  func(_ *testing.T, _ *mocks.MockWSMAN, _ *mocks.MockManagement) {},
			repoMock: func(_ *mocks.MockDeviceManagementRepository) {},
			err:      devices.ErrValidationUseCase,
		},
		{
			name: "validation error static missing primary dns",
			req: dto.WiredNetworkConfigRequest{
				DHCPEnabled:    &dhcpFalse,
				IPAddress:      "192.168.1.5",
				SubnetMask:     "255.255.255.0",
				DefaultGateway: "192.168.1.1",
			},
			manMock:  func(_ *testing.T, _ *mocks.MockWSMAN, _ *mocks.MockManagement) {},
			repoMock: func(_ *mocks.MockDeviceManagementRepository) {},
			err:      devices.ErrValidationUseCase,
		},
		{
			name: "success static ip",
			req: dto.WiredNetworkConfigRequest{
				DHCPEnabled:    &dhcpFalse,
				IPAddress:      "192.168.1.5",
				SubnetMask:     "255.255.255.0",
				DefaultGateway: "192.168.1.1",
				PrimaryDNS:     "192.168.1.1",
			},
			manMock: func(t *testing.T, man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				t.Helper()

				man.EXPECT().
					SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).
					Return(man2, nil)
				man2.EXPECT().
					GetEthernetPortSettings().
					Return(currentSettings, nil)
				man2.EXPECT().
					PutEthernetPortSettings(gomock.Any(), "Intel(r) AMT Ethernet Port Settings 0").
					DoAndReturn(func(req ethernetport.SettingsRequest, _ string) (ethernetport.Response, error) {
						// Manual static IP: ip sync inherited from current (false),
						// so the supplied IP fields are carried through unchanged.
						require.False(t, req.DHCPEnabled)
						require.False(t, req.IpSyncEnabled)
						require.False(t, req.SharedStaticIp)
						require.Equal(t, "192.168.1.5", req.IPAddress)
						require.Equal(t, "255.255.255.0", req.SubnetMask)
						require.Equal(t, "192.168.1.1", req.DefaultGateway)
						require.Equal(t, "192.168.1.1", req.PrimaryDNS)

						return ethernetport.Response{}, nil
					})
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			err: nil,
		},
		{
			name: "GetById fails",
			req:  dto.WiredNetworkConfigRequest{DHCPEnabled: &dhcpTrue},
			manMock: func(_ *testing.T, _ *mocks.MockWSMAN, _ *mocks.MockManagement) {
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(nil, ErrGeneral)
			},
			err: devices.ErrGeneral,
		},
		{
			name: "no wired interface",
			req:  dto.WiredNetworkConfigRequest{DHCPEnabled: &dhcpTrue},
			manMock: func(_ *testing.T, man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).
					Return(man2, nil)
				man2.EXPECT().
					GetEthernetPortSettings().
					Return([]ethernetport.SettingsResponse{}, nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			err: devices.ErrNotFound,
		},
		{
			name: "device not found nil item",
			req:  dto.WiredNetworkConfigRequest{DHCPEnabled: &dhcpTrue},
			manMock: func(_ *testing.T, _ *mocks.MockWSMAN, _ *mocks.MockManagement) {
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(nil, nil)
			},
			err: devices.ErrNotFound,
		},
		{
			name: "SetupWsmanClient fails",
			req:  dto.WiredNetworkConfigRequest{DHCPEnabled: &dhcpTrue},
			manMock: func(_ *testing.T, man *mocks.MockWSMAN, _ *mocks.MockManagement) {
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
			name: "GetEthernetPortSettings fails",
			req:  dto.WiredNetworkConfigRequest{DHCPEnabled: &dhcpTrue},
			manMock: func(_ *testing.T, man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).
					Return(man2, nil)
				man2.EXPECT().
					GetEthernetPortSettings().
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
			name: "PutEthernetPortSettings fails",
			req:  dto.WiredNetworkConfigRequest{DHCPEnabled: &dhcpTrue},
			manMock: func(_ *testing.T, man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).
					Return(man2, nil)
				man2.EXPECT().
					GetEthernetPortSettings().
					Return(currentSettings, nil)
				man2.EXPECT().
					PutEthernetPortSettings(gomock.Any(), "Intel(r) AMT Ethernet Port Settings 0").
					Return(ethernetport.Response{}, ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			err: ErrGeneral,
		},
		{
			name: "success static ip with host sync enabled",
			req: dto.WiredNetworkConfigRequest{
				DHCPEnabled:   &dhcpFalse,
				IPSyncEnabled: &dhcpTrue,
			},
			manMock: func(t *testing.T, man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				t.Helper()

				man.EXPECT().
					SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).
					Return(man2, nil)
				man2.EXPECT().
					GetEthernetPortSettings().
					Return(currentSettings, nil)
				man2.EXPECT().
					PutEthernetPortSettings(gomock.Any(), "Intel(r) AMT Ethernet Port Settings 0").
					DoAndReturn(func(req ethernetport.SettingsRequest, _ string) (ethernetport.Response, error) {
						// Host-synced static IP: SharedStaticIp follows IpSyncEnabled
						// and the explicit IP fields are cleared because they are
						// supplied by the host OS.
						require.False(t, req.DHCPEnabled)
						require.True(t, req.IpSyncEnabled)
						require.True(t, req.SharedStaticIp)
						require.Empty(t, req.IPAddress)
						require.Empty(t, req.SubnetMask)
						require.Empty(t, req.DefaultGateway)
						require.Empty(t, req.PrimaryDNS)

						return ethernetport.Response{}, nil
					})
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			err: nil,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, wsmanMock, management, repo := initNetworkTest(t)

			tc.manMock(t, wsmanMock, management)
			tc.repoMock(repo)

			err := useCase.PatchWiredNetworkSettings(context.Background(), device.GUID, tc.req)

			require.IsType(t, tc.err, err)
		})
	}
}
