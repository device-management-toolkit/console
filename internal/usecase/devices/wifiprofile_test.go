package devices_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/config"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/wifiportconfiguration"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/concrete"
	cimIEEE8021x "github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/ieee8021x"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/models"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/wifi"

	"github.com/device-management-toolkit/console/internal/entity"
	"github.com/device-management-toolkit/console/internal/mocks"
	"github.com/device-management-toolkit/console/internal/repoerrors"
	"github.com/device-management-toolkit/console/internal/usecase/devices"
	"github.com/device-management-toolkit/console/internal/usecase/devices/wsman"
	"github.com/device-management-toolkit/console/pkg/logger"
)

const (
	testWiFiEndpoint                 = "WiFi Endpoint 0"
	testUserSettingsInstanceIDPrefix = "Intel(r) AMT:WiFi Endpoint User Settings"
)

func initWiFiProfileTest(t *testing.T) (*devices.UseCase, *mocks.MockWSMAN, *mocks.MockManagement, *mocks.MockDeviceManagementRepository) {
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

func expectedWiFiRequest(profile config.WirelessProfile) wifi.WiFiEndpointSettingsRequest {
	authMethod, ok := wifi.ParseAuthenticationMethod(profile.AuthenticationMethod)
	if !ok {
		panic(fmt.Sprintf("invalid authentication method in test profile: %q", profile.AuthenticationMethod))
	}

	encryptionMethod, ok := wifi.ParseEncryptionMethod(profile.EncryptionMethod)
	if !ok {
		panic(fmt.Sprintf("invalid encryption method in test profile: %q", profile.EncryptionMethod))
	}

	return wifi.WiFiEndpointSettingsRequest{
		ElementName:          profile.ProfileName,
		InstanceID:           fmt.Sprintf("Intel(r) AMT:WiFi Endpoint Settings %s", profile.ProfileName),
		AuthenticationMethod: authMethod,
		EncryptionMethod:     encryptionMethod,
		SSID:                 profile.SSID,
		Priority:             profile.Priority,
		PSKPassPhrase:        profile.Password,
	}
}

func TestGetWirelessProfiles(t *testing.T) {
	t.Parallel()

	device := &entity.Device{GUID: "device-guid-123"}
	tests := []struct {
		name     string
		manMock  func(*mocks.MockWSMAN, *mocks.MockManagement)
		repoMock func(*mocks.MockDeviceManagementRepository)
		res      []config.WirelessProfile
		err      error
	}{
		{
			name: "success filters endpoint user settings by instance id",
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				gomock.InOrder(
					man.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(man2, nil),
					man2.EXPECT().GetWiFiSettings().Return([]wifi.WiFiEndpointSettingsResponse{
						{ElementName: "ignored-user-setting", InstanceID: testUserSettingsInstanceIDPrefix + " Profile"},
						{
							ElementName:          "Corp",
							InstanceID:           "Intel(r) AMT:WiFi Endpoint Settings Corp",
							SSID:                 "CorpSSID",
							AuthenticationMethod: wifi.AuthenticationMethodWPA2PSK,
							EncryptionMethod:     wifi.EncryptionMethodCCMP,
							Priority:             2,
						},
					}, nil),
					man2.EXPECT().GetCIMIEEE8021xSettings().Return(cimIEEE8021x.Response{}, nil),
					man2.EXPECT().GetConcreteDependencies().Return([]concrete.ConcreteDependency{}, nil),
				)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
			},
			res: []config.WirelessProfile{{
				ProfileName:          "Corp",
				SSID:                 "CorpSSID",
				AuthenticationMethod: "WPA2PSK",
				EncryptionMethod:     "CCMP",
				Priority:             2,
			}},
			err: nil,
		},
		{
			name: "success maps associated ieee8021x",
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				gomock.InOrder(
					man.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(man2, nil),
					man2.EXPECT().GetWiFiSettings().Return([]wifi.WiFiEndpointSettingsResponse{
						{
							ElementName:          "Corp",
							InstanceID:           "Intel(r) AMT:WiFi Endpoint Settings Corp",
							SSID:                 "CorpSSID",
							AuthenticationMethod: wifi.AuthenticationMethodWPA2IEEE8021x,
							EncryptionMethod:     wifi.EncryptionMethodCCMP,
							Priority:             1,
						},
					}, nil),
					man2.EXPECT().GetCIMIEEE8021xSettings().Return(cimIEEE8021x.Response{Body: cimIEEE8021x.Body{PullResponse: cimIEEE8021x.PullResponse{IEEE8021xSettingsItems: []cimIEEE8021x.IEEE8021xSettingsResponse{{
						InstanceID:             "Intel(r) AMT:IEEE 802.1x Settings Corp",
						AuthenticationProtocol: 2,
						Username:               "corp-user",
						Password:               "corp-pass",
					}}}}}, nil),
					man2.EXPECT().GetConcreteDependencies().Return([]concrete.ConcreteDependency{{
						Antecedent: models.AssociationReference{ReferenceParameters: models.ReferenceParametersNoNamespace{ResourceURI: "http://schemas.dmtf.org/wbem/wscim/1/cim-schema/2/CIM_WiFiEndpointSettings", SelectorSet: models.SelectorNoNamespace{Selectors: []models.SelectorResponse{{Name: "InstanceID", Text: "Intel(r) AMT:WiFi Endpoint Settings Corp"}}}}},
						Dependent:  models.AssociationReference{ReferenceParameters: models.ReferenceParametersNoNamespace{ResourceURI: "http://schemas.dmtf.org/wbem/wscim/1/cim-schema/2/CIM_IEEE8021xSettings", SelectorSet: models.SelectorNoNamespace{Selectors: []models.SelectorResponse{{Name: "InstanceID", Text: "Intel(r) AMT:IEEE 802.1x Settings Corp"}}}}},
					}}, nil),
				)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
			},
			res: []config.WirelessProfile{{
				ProfileName:          "Corp",
				SSID:                 "CorpSSID",
				AuthenticationMethod: "WPA2IEEE8021x",
				EncryptionMethod:     "CCMP",
				Priority:             1,
				IEEE8021x: &config.IEEE8021x{
					AuthenticationProtocol: 2,
					Username:               "corp-user",
					Password:               "corp-pass",
				},
			}},
			err: nil,
		},
		{
			name: "success maps ieee8021x by profile name fallback",
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				gomock.InOrder(
					man.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(man2, nil),
					man2.EXPECT().GetWiFiSettings().Return([]wifi.WiFiEndpointSettingsResponse{
						{
							ElementName:          "CorpEAP2",
							InstanceID:           "Intel(r) AMT:WiFi Endpoint Settings CorpEAP2",
							SSID:                 "CorpNet2",
							AuthenticationMethod: wifi.AuthenticationMethodWPA2IEEE8021x,
							EncryptionMethod:     wifi.EncryptionMethodCCMP,
							Priority:             4,
						},
					}, nil),
					man2.EXPECT().GetCIMIEEE8021xSettings().Return(cimIEEE8021x.Response{Body: cimIEEE8021x.Body{PullResponse: cimIEEE8021x.PullResponse{IEEE8021xSettingsItems: []cimIEEE8021x.IEEE8021xSettingsResponse{{
						ElementName:            "CorpEAP2",
						InstanceID:             "Intel(r) AMT:IEEE 802.1x Settings CorpEAP2",
						AuthenticationProtocol: 2,
						Username:               "corp-user",
						Password:               "corp-pass",
					}}}}}, nil),
					man2.EXPECT().GetConcreteDependencies().Return([]concrete.ConcreteDependency{}, nil),
				)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
			},
			res: []config.WirelessProfile{{
				ProfileName:          "CorpEAP2",
				SSID:                 "CorpNet2",
				AuthenticationMethod: "WPA2IEEE8021x",
				EncryptionMethod:     "CCMP",
				Priority:             4,
				IEEE8021x: &config.IEEE8021x{
					AuthenticationProtocol: 2,
					Username:               "corp-user",
					Password:               "corp-pass",
				},
			}},
			err: nil,
		},
		{
			name:    "GetByID fails",
			manMock: nil,
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(nil, ErrGeneral)
			},
			res: nil,
			err: devices.ErrGeneral,
		},
		{
			name:    "device not found",
			manMock: nil,
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(nil, nil)
			},
			res: nil,
			err: devices.ErrNotFound,
		},
		{
			name:    "device GUID empty",
			manMock: nil,
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(&entity.Device{}, nil)
			},
			res: nil,
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
			res: nil,
			err: devices.ErrGeneral,
		},
		{
			name: "GetWiFiSettings fails",
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(man2, nil)
				man2.EXPECT().GetWiFiSettings().Return(nil, ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
			},
			res: nil,
			err: devices.ErrGeneral,
		},
		{
			name: "GetCIMIEEE8021xSettings fails",
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				gomock.InOrder(
					man.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(man2, nil),
					man2.EXPECT().GetWiFiSettings().Return([]wifi.WiFiEndpointSettingsResponse{}, nil),
					man2.EXPECT().GetCIMIEEE8021xSettings().Return(cimIEEE8021x.Response{}, ErrGeneral),
				)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
			},
			res: nil,
			err: devices.ErrGeneral,
		},
		{
			name: "GetConcreteDependencies fails",
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				gomock.InOrder(
					man.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(man2, nil),
					man2.EXPECT().GetWiFiSettings().Return([]wifi.WiFiEndpointSettingsResponse{}, nil),
					man2.EXPECT().GetCIMIEEE8021xSettings().Return(cimIEEE8021x.Response{}, nil),
					man2.EXPECT().GetConcreteDependencies().Return(nil, ErrGeneral),
				)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
			},
			res: nil,
			err: devices.ErrGeneral,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, wsmanMock, management, repo := initWiFiProfileTest(t)

			if tc.manMock != nil {
				tc.manMock(wsmanMock, management)
			}

			if tc.repoMock != nil {
				tc.repoMock(repo)
			}

			res, err := useCase.GetWirelessProfiles(context.Background(), device.GUID)
			require.Equal(t, tc.res, res)

			if tc.err != nil {
				require.IsType(t, tc.err, err)

				return
			}

			require.NoError(t, err)
		})
	}
}

func TestAddWirelessProfile(t *testing.T) { //nolint:gocognit // table-driven coverage for create flow branches
	t.Parallel()

	device := &entity.Device{GUID: "device-guid-123"}
	profile := config.WirelessProfile{
		ProfileName:          "Home",
		SSID:                 "HomeSSID",
		Priority:             1,
		Password:             "password",
		AuthenticationMethod: "WPA2PSK",
		EncryptionMethod:     "CCMP",
	}

	tests := []struct {
		name      string
		ctx       context.Context
		profile   config.WirelessProfile
		manMock   func(*mocks.MockWSMAN, *mocks.MockManagement)
		repoMock  func(*mocks.MockDeviceManagementRepository)
		err       error
		errString bool
	}{
		{
			name:    "success",
			profile: profile,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				gomock.InOrder(
					man.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(man2, nil),
					man2.EXPECT().GetWiFiSettings().Return([]wifi.WiFiEndpointSettingsResponse{}, nil),
					man2.EXPECT().AddWiFiSettings(expectedWiFiRequest(profile), models.IEEE8021xSettings{}, testWiFiEndpoint, "", "").Return(wifiportconfiguration.Response{}, nil),
				)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
			},
		},
		{
			name:    "duplicate profile name",
			profile: profile,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				gomock.InOrder(
					man.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(man2, nil),
					man2.EXPECT().GetWiFiSettings().Return([]wifi.WiFiEndpointSettingsResponse{{InstanceID: "Intel(r) AMT:WiFi Endpoint Settings Home", ElementName: "Home"}}, nil),
				)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
			},
			err: repoerrors.NotUniqueError{},
		},
		{
			name:    "duplicate profile priority",
			profile: profile,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				gomock.InOrder(
					man.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(man2, nil),
					man2.EXPECT().GetWiFiSettings().Return([]wifi.WiFiEndpointSettingsResponse{{
						InstanceID:  "Intel(r) AMT:WiFi Endpoint Settings Office",
						ElementName: "Office",
						Priority:    1,
					}}, nil),
				)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
			},
			err: repoerrors.NotUniqueError{},
		},
		{
			name:    "setup fails",
			profile: profile,
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(nil, ErrGeneral)
			},
			err: devices.ErrGeneral,
		},
		{
			name:    "read wifi settings fails",
			profile: profile,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(man2, nil)
				man2.EXPECT().GetWiFiSettings().Return(nil, ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
			},
			err: devices.ErrGeneral,
		},
		{
			name:    "add wifi settings fails",
			profile: profile,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				gomock.InOrder(
					man.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(man2, nil),
					man2.EXPECT().GetWiFiSettings().Return([]wifi.WiFiEndpointSettingsResponse{}, nil),
					man2.EXPECT().AddWiFiSettings(expectedWiFiRequest(profile), models.IEEE8021xSettings{}, testWiFiEndpoint, "", "").Return(wifiportconfiguration.Response{}, ErrGeneral),
				)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
			},
			err: devices.ErrGeneral,
		},
		{
			name: "invalid authentication method",
			profile: config.WirelessProfile{
				ProfileName:          "Home",
				SSID:                 "HomeSSID",
				Priority:             1,
				Password:             "password",
				AuthenticationMethod: "INVALID",
				EncryptionMethod:     "CCMP",
			},
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				gomock.InOrder(
					man.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(man2, nil),
					man2.EXPECT().GetWiFiSettings().Return([]wifi.WiFiEndpointSettingsResponse{}, nil),
				)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
			},
			err:       fmt.Errorf("invalid authentication method %q for profile %q", "INVALID", "Home"),
			errString: true,
		},
		{
			name: "canceled context returns before apply delay completes",
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()

				return ctx
			}(),
			profile: config.WirelessProfile{
				ProfileName:          "Corp",
				SSID:                 "CorpSSID",
				Priority:             1,
				AuthenticationMethod: "WPA2IEEE8021x",
				EncryptionMethod:     "CCMP",
				IEEE8021x: &config.IEEE8021x{
					PrivateKey: "new-private",
				},
			},
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				gomock.InOrder(
					man.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(man2, nil),
					man2.EXPECT().GetWiFiSettings().Return([]wifi.WiFiEndpointSettingsResponse{}, nil),
					man2.EXPECT().GetCertificates().Return(wsman.Certificates{}, nil),
					man2.EXPECT().AddPrivateKey("new-private").Return("private-handle", nil),
				)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().GetByID(gomock.Any(), device.GUID, "").Return(device, nil)
			},
			err: context.Canceled,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, wsmanMock, management, repo := initWiFiProfileTest(t)

			if tc.manMock != nil {
				tc.manMock(wsmanMock, management)
			}

			if tc.repoMock != nil {
				tc.repoMock(repo)
			}

			ctx := tc.ctx
			if ctx == nil {
				ctx = context.Background()
			}

			err := useCase.AddWirelessProfile(ctx, device.GUID, tc.profile)
			if tc.err != nil {
				require.Error(t, err)

				if tc.errString {
					require.Equal(t, tc.err.Error(), err.Error())

					return
				}

				require.IsType(t, tc.err, err)

				return
			}

			require.NoError(t, err)
		})
	}
}

func TestDeleteWirelessProfile(t *testing.T) {
	t.Parallel()

	device := &entity.Device{GUID: "device-guid-123"}

	tests := []struct {
		name     string
		manMock  func(*mocks.MockWSMAN, *mocks.MockManagement)
		repoMock func(*mocks.MockDeviceManagementRepository)
		err      error
	}{
		{
			name: "success",
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				gomock.InOrder(
					man.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(man2, nil),
					man2.EXPECT().GetWiFiSettings().Return([]wifi.WiFiEndpointSettingsResponse{{InstanceID: "Intel(r) AMT:WiFi Endpoint Settings Home", ElementName: "Home"}}, nil),
					man2.EXPECT().DeleteWiFiSetting("Intel(r) AMT:WiFi Endpoint Settings Home").Return(nil),
				)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
			},
		},
		{
			name: "profile not found",
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(man2, nil)
				man2.EXPECT().GetWiFiSettings().Return([]wifi.WiFiEndpointSettingsResponse{}, nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
			},
			err: devices.ErrNotFound,
		},
		{
			name: "read wifi settings fails",
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(man2, nil)
				man2.EXPECT().GetWiFiSettings().Return(nil, ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
			},
			err: devices.ErrGeneral,
		},
		{
			name: "delete fails",
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				gomock.InOrder(
					man.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(man2, nil),
					man2.EXPECT().GetWiFiSettings().Return([]wifi.WiFiEndpointSettingsResponse{{InstanceID: "Intel(r) AMT:WiFi Endpoint Settings Home", ElementName: "Home"}}, nil),
					man2.EXPECT().DeleteWiFiSetting("Intel(r) AMT:WiFi Endpoint Settings Home").Return(ErrGeneral),
				)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
			},
			err: devices.ErrGeneral,
		},
		{
			name: "setup fails",
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(nil, ErrGeneral)
			},
			err: devices.ErrGeneral,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, wsmanMock, management, repo := initWiFiProfileTest(t)

			if tc.manMock != nil {
				tc.manMock(wsmanMock, management)
			}

			if tc.repoMock != nil {
				tc.repoMock(repo)
			}

			err := useCase.DeleteWirelessProfile(context.Background(), device.GUID, "Home")
			if tc.err != nil {
				require.Error(t, err)
				require.IsType(t, tc.err, err)

				return
			}

			require.NoError(t, err)
		})
	}
}

func TestUpdateWirelessProfile(t *testing.T) { //nolint:gocognit // table-driven coverage for update flow branches
	t.Parallel()

	device := &entity.Device{GUID: "device-guid-123"}
	baseSettings := []wifi.WiFiEndpointSettingsResponse{{InstanceID: "Intel(r) AMT:WiFi Endpoint Settings Home", ElementName: "Home", Priority: 1}}

	tests := []struct {
		name        string
		ctx         context.Context
		profileName string
		request     config.WirelessProfile
		manMock     func(*mocks.MockWSMAN, *mocks.MockManagement)
		repoMock    func(*mocks.MockDeviceManagementRepository)
		err         error
		errString   bool
	}{
		{
			name:        "success",
			profileName: "Home",
			request: config.WirelessProfile{
				ProfileName:          "Home",
				SSID:                 "HomeSSID2",
				Priority:             2,
				Password:             "password",
				AuthenticationMethod: "WPA2PSK",
				EncryptionMethod:     "CCMP",
			},
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				req := config.WirelessProfile{ProfileName: "Home", SSID: "HomeSSID2", Priority: 2, Password: "password", AuthenticationMethod: "WPA2PSK", EncryptionMethod: "CCMP"}
				gomock.InOrder(
					man.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(man2, nil),
					man2.EXPECT().GetWiFiSettings().Return(baseSettings, nil),
					man2.EXPECT().UpdateWiFiSettings(expectedWiFiRequest(req), models.IEEE8021xSettings{}, "", "").Return(wifiportconfiguration.Response{}, nil),
				)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
			},
		},
		{
			name:        "success when matching priority belongs to current profile",
			profileName: "Home",
			request: config.WirelessProfile{
				ProfileName:          "Home",
				SSID:                 "HomeSSID2",
				Priority:             1,
				Password:             "password",
				AuthenticationMethod: "WPA2PSK",
				EncryptionMethod:     "CCMP",
			},
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				req := config.WirelessProfile{ProfileName: "Home", SSID: "HomeSSID2", Priority: 1, Password: "password", AuthenticationMethod: "WPA2PSK", EncryptionMethod: "CCMP"}
				gomock.InOrder(
					man.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(man2, nil),
					man2.EXPECT().GetWiFiSettings().Return(baseSettings, nil),
					man2.EXPECT().UpdateWiFiSettings(expectedWiFiRequest(req), models.IEEE8021xSettings{}, "", "").Return(wifiportconfiguration.Response{}, nil),
				)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
			},
		},
		{
			name:        "profile name differs only by case",
			profileName: "home",
			request: config.WirelessProfile{
				ProfileName:          "home",
				SSID:                 "HomeSSID2",
				Priority:             1,
				Password:             "password",
				AuthenticationMethod: "WPA2PSK",
				EncryptionMethod:     "CCMP",
			},
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				gomock.InOrder(
					man.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(man2, nil),
					man2.EXPECT().GetWiFiSettings().Return(baseSettings, nil),
				)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
			},
			err: devices.ErrNotFound,
		},
		{
			name:        "success when context pause before update completes",
			profileName: "Home",
			request: config.WirelessProfile{
				ProfileName:          "Home",
				SSID:                 "HomeSSID2",
				Priority:             2,
				AuthenticationMethod: "WPA2IEEE8021x",
				EncryptionMethod:     "CCMP",
				IEEE8021x: &config.IEEE8021x{
					PrivateKey: "new-private",
				},
			},
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				req := config.WirelessProfile{ProfileName: "Home", SSID: "HomeSSID2", Priority: 2, AuthenticationMethod: "WPA2IEEE8021x", EncryptionMethod: "CCMP", IEEE8021x: &config.IEEE8021x{PrivateKey: "new-private"}}

				gomock.InOrder(
					man.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(man2, nil),
					man2.EXPECT().GetWiFiSettings().Return(baseSettings, nil),
					man2.EXPECT().GetCertificates().Return(wsman.Certificates{}, nil),
					man2.EXPECT().AddPrivateKey("new-private").Return("private-handle", nil),
					man2.EXPECT().UpdateWiFiSettings(expectedWiFiRequest(req), models.IEEE8021xSettings{ElementName: "Home", InstanceID: "Intel(r) AMT:IEEE 802.1x Settings Home"}, "", "").Return(wifiportconfiguration.Response{}, nil),
				)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
			},
		},
		{
			name:        "duplicate profile priority",
			profileName: "Home",
			request: config.WirelessProfile{
				ProfileName:          "Home",
				SSID:                 "HomeSSID2",
				Priority:             2,
				Password:             "password",
				AuthenticationMethod: "WPA2PSK",
				EncryptionMethod:     "CCMP",
			},
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				settings := []wifi.WiFiEndpointSettingsResponse{
					{InstanceID: "Intel(r) AMT:WiFi Endpoint Settings Home", ElementName: "Home", Priority: 1},
					{InstanceID: "Intel(r) AMT:WiFi Endpoint Settings Office", ElementName: "Office", Priority: 2},
				}

				gomock.InOrder(
					man.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(man2, nil),
					man2.EXPECT().GetWiFiSettings().Return(settings, nil),
				)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
			},
			err: repoerrors.NotUniqueError{},
		},
		{
			name:        "setup fails",
			profileName: "Home",
			request: config.WirelessProfile{
				ProfileName:          "Home",
				SSID:                 "HomeSSID2",
				Priority:             2,
				Password:             "password",
				AuthenticationMethod: "WPA2PSK",
				EncryptionMethod:     "CCMP",
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(nil, ErrGeneral)
			},
			err: devices.ErrGeneral,
		},
		{
			name:        "profile not found",
			profileName: "Home",
			request: config.WirelessProfile{
				ProfileName:          "Home",
				SSID:                 "HomeSSID2",
				Priority:             2,
				Password:             "password",
				AuthenticationMethod: "WPA2PSK",
				EncryptionMethod:     "CCMP",
			},
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(man2, nil)
				man2.EXPECT().GetWiFiSettings().Return([]wifi.WiFiEndpointSettingsResponse{}, nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
			},
			err: devices.ErrNotFound,
		},
		{
			name:        "read wifi settings fails",
			profileName: "Home",
			request: config.WirelessProfile{
				ProfileName:          "Home",
				SSID:                 "HomeSSID2",
				Priority:             2,
				Password:             "password",
				AuthenticationMethod: "WPA2PSK",
				EncryptionMethod:     "CCMP",
			},
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(man2, nil)
				man2.EXPECT().GetWiFiSettings().Return(nil, ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
			},
			err: devices.ErrGeneral,
		},
		{
			name:        "update wifi settings fails",
			profileName: "Home",
			request: config.WirelessProfile{
				ProfileName:          "Home",
				SSID:                 "HomeSSID2",
				Priority:             2,
				Password:             "password",
				AuthenticationMethod: "WPA2PSK",
				EncryptionMethod:     "CCMP",
			},
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				req := config.WirelessProfile{ProfileName: "Home", SSID: "HomeSSID2", Priority: 2, Password: "password", AuthenticationMethod: "WPA2PSK", EncryptionMethod: "CCMP"}
				gomock.InOrder(
					man.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(man2, nil),
					man2.EXPECT().GetWiFiSettings().Return(baseSettings, nil),
					man2.EXPECT().UpdateWiFiSettings(expectedWiFiRequest(req), models.IEEE8021xSettings{}, "", "").Return(wifiportconfiguration.Response{}, ErrGeneral),
				)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
			},
			err: devices.ErrGeneral,
		},
		{
			name:        "invalid authentication method",
			profileName: "Home",
			request: config.WirelessProfile{
				ProfileName:          "Home",
				SSID:                 "HomeSSID2",
				Priority:             2,
				Password:             "password",
				AuthenticationMethod: "INVALID",
				EncryptionMethod:     "CCMP",
			},
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(man2, nil)
				man2.EXPECT().GetWiFiSettings().Return(baseSettings, nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
			},
			err:       fmt.Errorf("invalid authentication method %q for profile %q", "INVALID", "Home"),
			errString: true,
		},
		{
			name: "canceled context returns before update delay completes",
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()

				return ctx
			}(),
			profileName: "Home",
			request: config.WirelessProfile{
				ProfileName:          "Home",
				SSID:                 "HomeSSID2",
				Priority:             2,
				AuthenticationMethod: "WPA2IEEE8021x",
				EncryptionMethod:     "CCMP",
				IEEE8021x: &config.IEEE8021x{
					PrivateKey: "new-private",
				},
			},
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				gomock.InOrder(
					man.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(man2, nil),
					man2.EXPECT().GetWiFiSettings().Return(baseSettings, nil),
					man2.EXPECT().GetCertificates().Return(wsman.Certificates{}, nil),
					man2.EXPECT().AddPrivateKey("new-private").Return("private-handle", nil),
				)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().GetByID(gomock.Any(), device.GUID, "").Return(device, nil)
			},
			err: context.Canceled,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, wsmanMock, management, repo := initWiFiProfileTest(t)

			if tc.manMock != nil {
				tc.manMock(wsmanMock, management)
			}

			if tc.repoMock != nil {
				tc.repoMock(repo)
			}

			ctx := tc.ctx
			if ctx == nil {
				ctx = context.Background()
			}

			err := useCase.UpdateWirelessProfile(ctx, device.GUID, tc.request)
			if tc.err != nil {
				require.Error(t, err)

				if tc.errString {
					require.Equal(t, tc.err.Error(), err.Error())

					return
				}

				require.IsType(t, tc.err, err)

				return
			}

			require.NoError(t, err)
		})
	}
}

func TestAddWirelessProfileIEEE8021xCertificateHandling(t *testing.T) {
	t.Parallel()

	device := &entity.Device{GUID: "device-guid-123"}

	successProfile := config.WirelessProfile{
		ProfileName:          "Corp",
		SSID:                 "CorpSSID",
		Priority:             1,
		AuthenticationMethod: "WPA2IEEE8021x",
		EncryptionMethod:     "CCMP",
		IEEE8021x: &config.IEEE8021x{
			AuthenticationProtocol: 2,
			Username:               "corp-user",
			Password:               "corp-pass",
			PrivateKey:             "new-private",
			ClientCert:             "new-client",
			CACert:                 "new-ca",
		},
	}

	ieeeRequest := models.IEEE8021xSettings{
		ElementName:            "Corp",
		InstanceID:             "Intel(r) AMT:IEEE 802.1x Settings Corp",
		AuthenticationProtocol: 2,
		Username:               "corp-user",
		Password:               "corp-pass",
	}

	tests := []struct {
		name     string
		profile  config.WirelessProfile
		manMock  func(*mocks.MockWSMAN, *mocks.MockManagement)
		repoMock func(*mocks.MockDeviceManagementRepository)
		err      error
	}{
		{
			name:    "success with all ieee8021x credentials",
			profile: successProfile,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				gomock.InOrder(
					man.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(man2, nil),
					man2.EXPECT().GetWiFiSettings().Return([]wifi.WiFiEndpointSettingsResponse{}, nil),
					man2.EXPECT().GetCertificates().Return(wsman.Certificates{}, nil),
					man2.EXPECT().AddPrivateKey("new-private").Return("private-handle", nil),
					man2.EXPECT().AddClientCert("new-client").Return("client-handle", nil),
					man2.EXPECT().AddTrustedRootCert("new-ca").Return("root-handle", nil),
					man2.EXPECT().AddWiFiSettings(expectedWiFiRequest(successProfile), ieeeRequest, testWiFiEndpoint, "client-handle", "root-handle").Return(wifiportconfiguration.Response{}, nil),
				)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
			},
		},
		{
			name: "configure ieee8021x certificates - get certificates fails",
			profile: config.WirelessProfile{
				ProfileName:          "Corp",
				SSID:                 "CorpSSID",
				Priority:             1,
				AuthenticationMethod: "WPA2IEEE8021x",
				EncryptionMethod:     "CCMP",
				IEEE8021x: &config.IEEE8021x{
					PrivateKey: "new-private",
				},
			},
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				gomock.InOrder(
					man.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(man2, nil),
					man2.EXPECT().GetWiFiSettings().Return([]wifi.WiFiEndpointSettingsResponse{}, nil),
					man2.EXPECT().GetCertificates().Return(wsman.Certificates{}, ErrGeneral),
				)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
			},
			err: ErrGeneral,
		},
		{
			name: "configure ieee8021x certificates - add private key fails",
			profile: config.WirelessProfile{
				ProfileName:          "Corp",
				SSID:                 "CorpSSID",
				Priority:             1,
				AuthenticationMethod: "WPA2IEEE8021x",
				EncryptionMethod:     "CCMP",
				IEEE8021x: &config.IEEE8021x{
					PrivateKey: "new-private",
				},
			},
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				gomock.InOrder(
					man.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(man2, nil),
					man2.EXPECT().GetWiFiSettings().Return([]wifi.WiFiEndpointSettingsResponse{}, nil),
					man2.EXPECT().GetCertificates().Return(wsman.Certificates{}, nil),
					man2.EXPECT().AddPrivateKey("new-private").Return("", ErrGeneral),
				)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
			},
			err: ErrGeneral,
		},
		{
			name: "configure ieee8021x certificates - add client cert fails",
			profile: config.WirelessProfile{
				ProfileName:          "Corp",
				SSID:                 "CorpSSID",
				Priority:             1,
				AuthenticationMethod: "WPA2IEEE8021x",
				EncryptionMethod:     "CCMP",
				IEEE8021x: &config.IEEE8021x{
					ClientCert: "new-client",
				},
			},
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				gomock.InOrder(
					man.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(man2, nil),
					man2.EXPECT().GetWiFiSettings().Return([]wifi.WiFiEndpointSettingsResponse{}, nil),
					man2.EXPECT().GetCertificates().Return(wsman.Certificates{}, nil),
					man2.EXPECT().AddClientCert("new-client").Return("", ErrGeneral),
				)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
			},
			err: ErrGeneral,
		},
		{
			name: "configure ieee8021x certificates - add trusted root cert fails",
			profile: config.WirelessProfile{
				ProfileName:          "Corp",
				SSID:                 "CorpSSID",
				Priority:             1,
				AuthenticationMethod: "WPA2IEEE8021x",
				EncryptionMethod:     "CCMP",
				IEEE8021x: &config.IEEE8021x{
					CACert: "new-ca",
				},
			},
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				gomock.InOrder(
					man.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(man2, nil),
					man2.EXPECT().GetWiFiSettings().Return([]wifi.WiFiEndpointSettingsResponse{}, nil),
					man2.EXPECT().GetCertificates().Return(wsman.Certificates{}, nil),
					man2.EXPECT().AddTrustedRootCert("new-ca").Return("", ErrGeneral),
				)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
			},
			err: ErrGeneral,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, wsmanMock, management, repo := initWiFiProfileTest(t)

			if tc.manMock != nil {
				tc.manMock(wsmanMock, management)
			}

			if tc.repoMock != nil {
				tc.repoMock(repo)
			}

			err := useCase.AddWirelessProfile(context.Background(), device.GUID, tc.profile)
			if tc.err != nil {
				require.Error(t, err)
				require.IsType(t, tc.err, err)

				return
			}

			require.NoError(t, err)
		})
	}
}
