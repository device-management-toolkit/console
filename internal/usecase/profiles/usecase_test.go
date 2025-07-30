package profiles_test

import (
	"context"
	"testing"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/config"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	local "github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/internal/entity"
	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/mocks"
	"github.com/device-management-toolkit/console/internal/usecase/profiles"
	"github.com/device-management-toolkit/console/pkg/logger"
)

type test struct {
	name        string
	top         int
	skip        int
	tenantID    string
	profileName string
	input       entity.Profile
	mock        func(*mocks.MockProfilesRepository, *mocks.MockWiFiConfigsRepository, *mocks.MockProfileWiFiConfigsFeature)
	res         interface{}
	err         error
}

func profilesTest(t *testing.T) (*profiles.UseCase, *mocks.MockProfilesRepository, *mocks.MockWiFiConfigsRepository, *mocks.MockProfileWiFiConfigsFeature) {
	t.Helper()

	mockCtl := gomock.NewController(t)
	defer mockCtl.Finish()

	repo := mocks.NewMockProfilesRepository(mockCtl)
	wificonfigs := mocks.NewMockWiFiConfigsRepository(mockCtl)
	profilewificonfigs := mocks.NewMockProfileWiFiConfigsFeature(mockCtl)
	ieeeMock := mocks.NewMockIEEE8021xConfigsFeature(mockCtl)
	domains := mocks.NewMockDomainsRepository(mockCtl)
	cira := mocks.NewMockCIRAConfigsRepository(mockCtl)
	security := mocks.MockCrypto{}
	log := logger.New("error")
	useCase := profiles.New(repo, wificonfigs, profilewificonfigs, ieeeMock, log, domains, cira, security)

	return useCase, repo, wificonfigs, profilewificonfigs
}

func TestGetCount(t *testing.T) {
	t.Parallel()

	tests := []test{
		{
			name: "empty result",
			mock: func(repo *mocks.MockProfilesRepository, _ *mocks.MockWiFiConfigsRepository, _ *mocks.MockProfileWiFiConfigsFeature) {
				repo.EXPECT().GetCount(context.Background(), "").Return(0, nil)
			},
			res: 0,
			err: nil,
		},
		{
			name: "result with error",
			mock: func(repo *mocks.MockProfilesRepository, _ *mocks.MockWiFiConfigsRepository, _ *mocks.MockProfileWiFiConfigsFeature) {
				repo.EXPECT().GetCount(context.Background(), "").Return(0, profiles.ErrDatabase)
			},
			res: 0,
			err: profiles.ErrDatabase,
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			useCase, repo, wifiFeat, pwfFeat := profilesTest(t)

			tc.mock(repo, wifiFeat, pwfFeat)

			res, err := useCase.GetCount(context.Background(), "")

			require.Equal(t, tc.res, res)
			require.IsType(t, tc.err, err)
		})
	}
}

func TestGet(t *testing.T) {
	t.Parallel()

	ieeeProfileName := "test-8021x-profile-1"

	testProfiles := []entity.Profile{
		{
			ProfileName:            "test-profile-1",
			TenantID:               "tenant-id-456",
			IEEE8021xProfileName:   &ieeeProfileName,
			AuthenticationProtocol: &[]int{1}[0],
			WiredInterface:         &[]bool{true}[0],
		},
		{
			ProfileName: "test-profile-2",
			TenantID:    "tenant-id-456",
		},
	}

	testProfileDTOs := []dto.Profile{
		{
			ProfileName:          "test-profile-1",
			TenantID:             "tenant-id-456",
			Tags:                 []string{""},
			IEEE8021xProfileName: &ieeeProfileName,
			IEEE8021xProfile: &dto.IEEE8021xConfig{
				ProfileName:            ieeeProfileName,
				TenantID:               "tenant-id-456",
				WiredInterface:         true,
				AuthenticationProtocol: 1,
			},
		},
		{
			ProfileName: "test-profile-2",
			TenantID:    "tenant-id-456",
			Tags:        []string{""},
		},
	}

	tests := []test{
		{
			name:     "successful retrieval",
			top:      10,
			skip:     0,
			tenantID: "tenant-id-456",
			mock: func(repo *mocks.MockProfilesRepository, _ *mocks.MockWiFiConfigsRepository, profileWifiRepo *mocks.MockProfileWiFiConfigsFeature) {
				repo.EXPECT().
					Get(context.Background(), 10, 0, "tenant-id-456").
					Return(testProfiles, nil)
				profileWifiRepo.EXPECT().
					GetByProfileName(context.Background(), "test-profile-1", "tenant-id-456").
					Return([]dto.ProfileWiFiConfigs{}, nil)
				profileWifiRepo.EXPECT().
					GetByProfileName(context.Background(), "test-profile-2", "tenant-id-456").
					Return([]dto.ProfileWiFiConfigs{}, nil)
			},
			res: testProfileDTOs,
			err: nil,
		},
		{
			name:     "database error",
			top:      5,
			skip:     0,
			tenantID: "tenant-id-456",
			mock: func(repo *mocks.MockProfilesRepository, _ *mocks.MockWiFiConfigsRepository, _ *mocks.MockProfileWiFiConfigsFeature) {
				repo.EXPECT().
					Get(context.Background(), 5, 0, "tenant-id-456").
					Return(nil, profiles.ErrDatabase)
			},
			res: []dto.Profile(nil),
			err: profiles.ErrDatabase,
		},
		{
			name:     "zero results",
			top:      10,
			skip:     20,
			tenantID: "tenant-id-456",
			mock: func(repo *mocks.MockProfilesRepository, _ *mocks.MockWiFiConfigsRepository, _ *mocks.MockProfileWiFiConfigsFeature) {
				repo.EXPECT().
					Get(context.Background(), 10, 20, "tenant-id-456").
					Return([]entity.Profile{}, nil)
			},
			res: []dto.Profile{},
			err: nil,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			useCase, repo, wifiFeat, pwfFeat := profilesTest(t)

			tc.mock(repo, wifiFeat, pwfFeat)

			results, err := useCase.Get(context.Background(), tc.top, tc.skip, tc.tenantID)

			require.Equal(t, tc.res, results)

			if tc.err != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGetByName(t *testing.T) {
	t.Parallel()

	profile := &entity.Profile{
		ProfileName: "test-profile",
		TenantID:    "tenant-id-456",
		Version:     "1.0.0",
	}

	profileDTO := &dto.Profile{
		ProfileName: "test-profile",
		TenantID:    "tenant-id-456",
		Version:     "1.0.0",
		Tags:        []string{""},
	}

	tests := []test{
		{
			name: "successful retrieval",
			input: entity.Profile{
				ProfileName: "test-profile",
				TenantID:    "tenant-id-456",
			},
			mock: func(repo *mocks.MockProfilesRepository, _ *mocks.MockWiFiConfigsRepository, profilewificonfigfeat *mocks.MockProfileWiFiConfigsFeature) {
				repo.EXPECT().
					GetByName(context.Background(), "test-profile", "tenant-id-456").
					Return(profile, nil)
				profilewificonfigfeat.EXPECT().
					GetByProfileName(context.Background(), "test-profile", "tenant-id-456").
					Return([]dto.ProfileWiFiConfigs{}, nil)
			},
			res: profileDTO,
			err: nil,
		},
		{
			name: "profile not found",
			input: entity.Profile{
				ProfileName: "unknown-profile",
				TenantID:    "tenant-id-456",
			},
			mock: func(repo *mocks.MockProfilesRepository, _ *mocks.MockWiFiConfigsRepository, _ *mocks.MockProfileWiFiConfigsFeature) {
				repo.EXPECT().
					GetByName(context.Background(), "unknown-profile", "tenant-id-456").
					Return(nil, nil)
			},
			res: (*dto.Profile)(nil),
			err: profiles.ErrNotFound,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			useCase, repo, wifiFeat, pwfFeat := profilesTest(t)

			tc.mock(repo, wifiFeat, pwfFeat)

			res, err := useCase.GetByName(context.Background(), tc.input.ProfileName, tc.input.TenantID)

			require.Equal(t, tc.res, res)

			if tc.err != nil {
				require.Contains(t, err.Error(), tc.err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	t.Parallel()

	tests := []test{
		{
			name:        "successful deletion",
			profileName: "example-profile",
			tenantID:    "tenant-id-456",
			mock: func(repo *mocks.MockProfilesRepository, _ *mocks.MockWiFiConfigsRepository, profilewificonfigfeat *mocks.MockProfileWiFiConfigsFeature) {
				profilewificonfigfeat.EXPECT().
					DeleteByProfileName(context.Background(), "example-profile", "tenant-id-456").
					Return(nil)
				repo.EXPECT().
					Delete(context.Background(), "example-profile", "tenant-id-456").
					Return(true, nil)
			},
			err: nil,
		},
		{
			name:        "deletion fails - profile not found",
			profileName: "nonexistent-profile",
			tenantID:    "tenant-id-456",
			mock: func(repo *mocks.MockProfilesRepository, _ *mocks.MockWiFiConfigsRepository, profilewificonfigfeat *mocks.MockProfileWiFiConfigsFeature) {
				profilewificonfigfeat.EXPECT().
					DeleteByProfileName(context.Background(), "nonexistent-profile", "tenant-id-456").
					Return(nil)
				repo.EXPECT().
					Delete(context.Background(), "nonexistent-profile", "tenant-id-456").
					Return(false, nil)
			},
			err: profiles.ErrNotFound,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			useCase, repo, wifiFeat, pwfFeat := profilesTest(t)

			tc.mock(repo, wifiFeat, pwfFeat)

			err := useCase.Delete(context.Background(), tc.profileName, tc.tenantID)

			if tc.err != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestUpdate(t *testing.T) {
	t.Parallel()

	profile := &entity.Profile{
		ProfileName:  "example-profile",
		TenantID:     "tenant-id-456",
		Version:      "1.0.0",
		AMTPassword:  "encrypted",
		MEBXPassword: "encrypted",
	}

	profileDTO := &dto.Profile{
		ProfileName: "example-profile",
		TenantID:    "tenant-id-456",
		Version:     "1.0.0",
		Tags:        []string{""},
		WiFiConfigs: []dto.ProfileWiFiConfigs{
			{
				ProfileName:         "example-profile",
				WirelessProfileName: "wireless-profile-1",
			},
		},
	}

	tests := []test{
		{
			name: "successful update",
			mock: func(repo *mocks.MockProfilesRepository, wifiConfig *mocks.MockWiFiConfigsRepository, profilewificonfigfeat *mocks.MockProfileWiFiConfigsFeature) {
				repo.EXPECT().
					Update(context.Background(), profile).
					Return(true, nil)
				profilewificonfigfeat.EXPECT().
					DeleteByProfileName(context.Background(), profile.ProfileName, profile.TenantID).
					Return(nil)
				repo.EXPECT().
					GetByName(context.Background(), profile.ProfileName, profile.TenantID).
					Return(profile, nil)
				wifiConfig.EXPECT().
					CheckProfileExists(context.Background(), profileDTO.WiFiConfigs[0].WirelessProfileName, profileDTO.TenantID).
					Return(true, nil)
			},
			res: profileDTO,
			err: nil,
		},
		{
			name: "update fails - not found",
			mock: func(repo *mocks.MockProfilesRepository, wifiConfig *mocks.MockWiFiConfigsRepository, _ *mocks.MockProfileWiFiConfigsFeature) {
				repo.EXPECT().
					Update(context.Background(), profile).
					Return(false, profiles.ErrNotFound)
				wifiConfig.EXPECT().
					CheckProfileExists(context.Background(), profileDTO.WiFiConfigs[0].WirelessProfileName, profileDTO.TenantID).
					Return(true, nil)
			},
			res: (*dto.Profile)(nil),
			err: profiles.ErrDatabase,
		},
		{
			name: "update fails - database error",
			mock: func(repo *mocks.MockProfilesRepository, wifiConfig *mocks.MockWiFiConfigsRepository, _ *mocks.MockProfileWiFiConfigsFeature) {
				repo.EXPECT().
					Update(context.Background(), profile).
					Return(false, profiles.ErrDatabase)
				wifiConfig.EXPECT().
					CheckProfileExists(context.Background(), profileDTO.WiFiConfigs[0].WirelessProfileName, profileDTO.TenantID).
					Return(true, nil)
			},
			res: (*dto.Profile)(nil),
			err: profiles.ErrDatabase,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			useCase, repo, wifiFeat, pwfFeat := profilesTest(t)

			tc.mock(repo, wifiFeat, pwfFeat)

			result, err := useCase.Update(context.Background(), profileDTO)

			require.Equal(t, tc.res, result)
			require.IsType(t, err, tc.err)
		})
	}
}

func TestInsert(t *testing.T) {
	t.Parallel()

	profile := &entity.Profile{
		ProfileName:  "new-profile",
		TenantID:     "tenant-id-789",
		Version:      "1.0.0",
		Tags:         "",
		DHCPEnabled:  true,
		AMTPassword:  "encrypted",
		MEBXPassword: "encrypted",
	}

	profileDTO := &dto.Profile{
		ProfileName: "new-profile",
		TenantID:    "tenant-id-789",
		Version:     "1.0.0",
		Tags:        []string{""},
		DHCPEnabled: true,
		WiFiConfigs: []dto.ProfileWiFiConfigs{
			{
				ProfileName:         "new-profile",
				WirelessProfileName: "wireless-profile-1",
			},
		},
	}

	tests := []test{
		{
			name: "successful insertion",
			mock: func(repo *mocks.MockProfilesRepository, wifiRepo *mocks.MockWiFiConfigsRepository, profilewificonfigfeat *mocks.MockProfileWiFiConfigsFeature) {
				repo.EXPECT().
					Insert(context.Background(), profile).
					Return("unique-profile-id", nil)
				profilewificonfigfeat.EXPECT().
					Insert(context.Background(), &profileDTO.WiFiConfigs[0]).
					Return(nil)
				repo.EXPECT().
					GetByName(context.Background(), profile.ProfileName, profile.TenantID).
					Return(profile, nil)
				wifiRepo.EXPECT().
					CheckProfileExists(context.Background(), profileDTO.WiFiConfigs[0].WirelessProfileName, profileDTO.TenantID).
					Return(true, nil)
			},
			res: profileDTO,
			err: nil,
		},
		{
			name: "insertion fails - database error",
			mock: func(repo *mocks.MockProfilesRepository, wifiRepo *mocks.MockWiFiConfigsRepository, _ *mocks.MockProfileWiFiConfigsFeature) {
				repo.EXPECT().
					Insert(context.Background(), profile).
					Return("", profiles.ErrDatabase)
				wifiRepo.EXPECT().
					CheckProfileExists(context.Background(), profileDTO.WiFiConfigs[0].WirelessProfileName, profileDTO.TenantID).
					Return(true, nil)
			},
			res: (*dto.Profile)(nil),
			err: profiles.ErrDatabase,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			useCase, repo, wifiFeat, pwfFeat := profilesTest(t)

			tc.mock(repo, wifiFeat, pwfFeat)

			id, err := useCase.Insert(context.Background(), profileDTO)

			require.Equal(t, tc.res, id)

			if tc.err != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestHandleIEEE8021xSettings(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tenantID := "tenant-id-4"
	ieeeProfileName := "test-8021x-profile"

	tests := []struct {
		name     string
		data     *entity.Profile
		mock     func(ieeeMock *mocks.MockIEEE8021xConfigsFeature)
		expected *config.IEEE8021x
		err      error
	}{
		{
			name: "with IEEE 802.1x profile",
			data: &entity.Profile{
				IEEE8021xProfileName: &ieeeProfileName,
			},
			mock: func(ieeeMock *mocks.MockIEEE8021xConfigsFeature) {
				ieeeMock.EXPECT().
					GetByName(ctx, ieeeProfileName, tenantID).
					Return(&dto.IEEE8021xConfig{
						AuthenticationProtocol: 0,
						PXETimeout:             func(i int) *int { return &i }(30),
					}, nil)
			},
			expected: &config.IEEE8021x{
				AuthenticationProtocol: 0,
				PXETimeout:             30,
			},
			err: nil,
		},
		{
			name:     "no IEEE 802.1x profile",
			data:     &entity.Profile{},
			mock:     func(_ *mocks.MockIEEE8021xConfigsFeature) {},
			expected: nil,
			err:      nil,
		},
		{
			name: "error retrieving IEEE 802.1x profile",
			data: &entity.Profile{
				IEEE8021xProfileName: &ieeeProfileName,
			},
			mock: func(ieeeMock *mocks.MockIEEE8021xConfigsFeature) {
				ieeeMock.EXPECT().
					GetByName(ctx, ieeeProfileName, tenantID).
					Return(nil, profiles.ErrDatabase)
			},
			expected: nil,
			err:      profiles.ErrDatabase,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ieeeMock := mocks.NewMockIEEE8021xConfigsFeature(gomock.NewController(t))
			configuration := &config.Configuration{}

			tc.mock(ieeeMock)

			useCase := profiles.New(nil, nil, nil, ieeeMock, nil, nil, nil, nil)

			err := useCase.HandleIEEE8021xSettings(ctx, tc.data, configuration, tenantID)

			if tc.err != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.err.Error())
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expected, configuration.Configuration.Network.Wired.IEEE8021x)
			}
		})
	}
}

func TestGetProfileData(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tenantID := "tenant-id-5"

	tests := []struct {
		name        string
		profileName string
		mock        func(repoMock *mocks.MockProfilesRepository)
		expected    *entity.Profile
		err         error
	}{
		{
			name:        "successful retrieval",
			profileName: "test-profile",
			mock: func(repoMock *mocks.MockProfilesRepository) {
				repoMock.EXPECT().
					GetByName(ctx, "test-profile", tenantID).
					Return(&entity.Profile{
						ProfileName: "test-profile",
					}, nil)
			},
			expected: &entity.Profile{
				ProfileName: "test-profile",
			},
			err: nil,
		},
		{
			name:        "profile not found",
			profileName: "unknown-profile",
			mock: func(repoMock *mocks.MockProfilesRepository) {
				repoMock.EXPECT().
					GetByName(ctx, "unknown-profile", tenantID).
					Return(nil, profiles.ErrNotFound)
			},
			expected: nil,
			err:      profiles.ErrNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			repoMock := mocks.NewMockProfilesRepository(gomock.NewController(t))

			tc.mock(repoMock)

			useCase := profiles.New(repoMock, nil, nil, nil, nil, nil, nil, nil)

			data, err := useCase.GetProfileData(ctx, tc.profileName, tenantID)

			require.Equal(t, tc.expected, data)

			if tc.err != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGetDomainInformation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tenantID := "tenant-id-6"

	tests := []struct {
		name       string
		activation string
		domainName string
		mock       func(domainsMock *mocks.MockDomainsRepository)
		expected   *entity.Domain
		err        error
	}{
		{
			name:       "successful retrieval for acmactivate",
			activation: "acmactivate",
			domainName: "vpro",
			mock: func(domainsMock *mocks.MockDomainsRepository) {
				domainsMock.EXPECT().
					GetByName(ctx, "vpro", tenantID).
					Return(&entity.Domain{
						ProvisioningCertPassword: "encryptedCert",
					}, nil)
			},
			expected: &entity.Domain{
				ProvisioningCertPassword: "decrypted",
			},
			err: nil,
		},
		{
			name:       "no domains found",
			activation: "acmactivate",
			domainName: "domainName",
			mock: func(domainsMock *mocks.MockDomainsRepository) {
				domainsMock.EXPECT().
					GetByName(ctx, "domainName", tenantID).
					Return((*entity.Domain)(nil), nil)
			},
			expected: nil,
			err:      profiles.ErrNotFound.WrapWithMessage("Export", "uc.domains.Get", "No domains found"),
		},
		{
			name:       "on error",
			activation: "acmactivate",
			domainName: "badRequest",
			mock: func(domainsMock *mocks.MockDomainsRepository) {
				domainsMock.EXPECT().
					GetByName(ctx, "badRequest", tenantID).
					Return(nil, profiles.ErrNotFound)
			},
			expected: nil,
			err:      profiles.ErrNotFound.WrapWithMessage("Export", "uc.domains.Get", "No domains found"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			domainsMock := mocks.NewMockDomainsRepository(gomock.NewController(t))
			cryptoMock := &mocks.MockCrypto{}

			tc.mock(domainsMock)

			useCase := profiles.New(nil, nil, nil, nil, nil, domainsMock, nil, cryptoMock)

			domain, err := useCase.GetDomainInformation(ctx, tc.activation, tc.domainName, tenantID)

			require.Equal(t, tc.expected, domain)

			if tc.err != nil || domain == nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestDecryptPasswords(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		data     *entity.Profile
		expected *entity.Profile
		err      error
	}{
		{
			name: "successful decryption",
			data: &entity.Profile{
				AMTPassword:  "encryptedAMT",
				MEBXPassword: "encryptedMEBX",
			},
			expected: &entity.Profile{
				AMTPassword:  "decrypted",
				MEBXPassword: "decrypted",
			},
			err: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cryptoMock := &mocks.MockCrypto{}

			useCase := profiles.New(nil, nil, nil, nil, nil, nil, nil, cryptoMock)

			err := useCase.DecryptPasswords(tc.data)

			if tc.err != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.err.Error())
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expected.AMTPassword, tc.data.AMTPassword)
				require.Equal(t, tc.expected.MEBXPassword, tc.data.MEBXPassword)
			}
		})
	}
}

func TestBuildWirelessProfiles(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tenantID := "tenant-id-457"

	wifiConfigs := []dto.ProfileWiFiConfigs{
		{
			WirelessProfileName: "wifi-profile-1",
			Priority:            1,
		},
	}

	tests := []struct {
		name     string
		mock     func(wifiMock *mocks.MockWiFiConfigsRepository)
		expected []config.WirelessProfile
		err      error
	}{
		{
			name: "successful profile build",
			mock: func(wifiMock *mocks.MockWiFiConfigsRepository) {
				wifiMock.EXPECT().
					GetByName(ctx, "wifi-profile-1", tenantID).
					Return(&entity.WirelessConfig{
						ProfileName:          "wifi-profile-1",
						SSID:                 "wifi-ssid",
						AuthenticationMethod: 4,
						EncryptionMethod:     4,
						PSKPassphrase:        "encryptedPassphrase",
					}, nil)
			},
			expected: []config.WirelessProfile{
				{
					ProfileName:          "wifi-profile-1",
					SSID:                 "wifi-ssid",
					Priority:             1,
					Password:             "decrypted",
					AuthenticationMethod: "WPAPSK",
					EncryptionMethod:     "CCMP",
				},
			},
			err: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			wifiMock := mocks.NewMockWiFiConfigsRepository(gomock.NewController(t))
			ieeeMock := mocks.NewMockIEEE8021xConfigsFeature(gomock.NewController(t))
			cryptoMock := &mocks.MockCrypto{}

			tc.mock(wifiMock)

			useCase := profiles.New(nil, wifiMock, nil, ieeeMock, nil, nil, nil, cryptoMock)

			wifiProfiles, err := useCase.BuildWirelessProfiles(ctx, wifiConfigs, tenantID)

			require.Equal(t, tc.expected, wifiProfiles)

			if tc.err != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestBuildConfigurationObject(t *testing.T) {
	t.Parallel()

	originalConfig := local.ConsoleConfig

	t.Cleanup(func() {
		local.ConsoleConfig = originalConfig
	})

	local.ConsoleConfig = &local.Config{
		EA: local.EA{
			URL:      "http://test.com:8080",
			Username: "username",
			Password: "password",
		},
	}

	tests := []struct {
		name     string
		profile  *entity.Profile
		domain   *entity.Domain
		wifi     []config.WirelessProfile
		cira     *entity.CIRAConfig
		expected config.Configuration
	}{
		{
			name: "successful configuration build",
			profile: &entity.Profile{
				ProfileName:   "test-profile",
				DHCPEnabled:   true,
				IPSyncEnabled: true,
				Activation:    "acmactivate",
				AMTPassword:   "testAMTPassword",
				MEBXPassword:  "testMEBXPassword",
				TLSMode:       2,
				KVMEnabled:    true,
				SOLEnabled:    true,
				IDEREnabled:   true,
				UserConsent:   "None",
			},
			domain: &entity.Domain{
				ProvisioningCert:         "testCert",
				ProvisioningCertPassword: "testCertPwd",
			},
			wifi: []config.WirelessProfile{
				{
					SSID:     "wifi-ssid",
					Priority: 1,
				},
			},
			expected: config.Configuration{
				Name: "test-profile",
				Configuration: config.RemoteManagement{
					GeneralSettings: config.GeneralSettings{
						SharedFQDN:              false,
						NetworkInterfaceEnabled: 0,
						PingResponseEnabled:     false,
					},
					Network: config.Network{
						Wired: config.Wired{
							DHCPEnabled:    true,
							IPSyncEnabled:  true,
							SharedStaticIP: false,
						},
						Wireless: config.Wireless{
							Profiles: []config.WirelessProfile{
								{
									SSID:     "wifi-ssid",
									Priority: 1,
								},
							},
						},
					},
					Redirection: config.Redirection{
						Enabled: true,
						Services: config.Services{
							KVM:  true,
							SOL:  true,
							IDER: true,
						},
						UserConsent: "None",
					},
					TLS: config.TLS{
						MutualAuthentication: false,
						Enabled:              true,
						AllowNonTLS:          true,
					},
					EnterpriseAssistant: config.EnterpriseAssistant{
						URL:      "http://test.com:8080",
						Username: "username",
						Password: "password",
					},
					AMTSpecific: config.AMTSpecific{
						ControlMode:         "acmactivate",
						AdminPassword:       "testAMTPassword",
						MEBXPassword:        "testMEBXPassword",
						ProvisioningCert:    "testCert",
						ProvisioningCertPwd: "testCertPwd",
						CIRA: config.CIRA{
							EnvironmentDetection: []string{},
						},
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase := profiles.New(nil, nil, nil, nil, nil, nil, nil, nil)

			result := useCase.BuildConfigurationObject(tc.profile.ProfileName, tc.profile, tc.domain, tc.wifi, tc.cira)

			require.Equal(t, tc.expected, result)
		})
	}
}

func TestGetWiFiConfigurations(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	profileName := "test-profile"
	tenantID := "tenant-id-456"

	tests := []struct {
		name     string
		mock     func(profileWiFiMock *mocks.MockProfileWiFiConfigsFeature)
		expected []dto.ProfileWiFiConfigs
		err      error
	}{
		{
			name: "successful retrieval",
			mock: func(profileWiFiMock *mocks.MockProfileWiFiConfigsFeature) {
				profileWiFiMock.EXPECT().
					GetByProfileName(ctx, profileName, tenantID).
					Return([]dto.ProfileWiFiConfigs{
						{
							WirelessProfileName: "wifi-profile-1",
						},
					}, nil)
			},
			expected: []dto.ProfileWiFiConfigs{
				{
					WirelessProfileName: "wifi-profile-1",
				},
			},
			err: nil,
		},
		{
			name: "error during retrieval",
			mock: func(profileWiFiMock *mocks.MockProfileWiFiConfigsFeature) {
				profileWiFiMock.EXPECT().
					GetByProfileName(ctx, profileName, tenantID).
					Return(nil, profiles.ErrDatabase)
			},
			expected: nil,
			err:      profiles.ErrDatabase,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			profileWiFiMock := mocks.NewMockProfileWiFiConfigsFeature(gomock.NewController(t))

			tc.mock(profileWiFiMock)

			useCase := profiles.New(nil, nil, profileWiFiMock, nil, nil, nil, nil, nil)

			wifiConfigs, err := useCase.GetWiFiConfigurations(ctx, profileName, tenantID)

			require.Equal(t, tc.expected, wifiConfigs)

			if tc.err != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestSerializeAndEncryptYAML(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                  string
		configuration         config.Configuration
		expectedEncryptedData string
		expectedEncryptionKey string
		err                   error
	}{
		{
			name: "successful serialization and encryption",
			configuration: config.Configuration{
				Name: "test-config",
				Configuration: config.RemoteManagement{
					GeneralSettings: config.GeneralSettings{
						SharedFQDN: true,
					},
				},
			},
			expectedEncryptedData: "encrypted",
			expectedEncryptionKey: "key",
			err:                   nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cryptoMock := &mocks.MockCrypto{}

			useCase := profiles.New(nil, nil, nil, nil, nil, nil, nil, cryptoMock)

			encryptedData, encryptionKey, err := useCase.SerializeAndEncryptYAML(tc.configuration)

			require.Equal(t, tc.expectedEncryptedData, encryptedData)
			require.Equal(t, tc.expectedEncryptionKey, encryptionKey)

			if tc.err != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}
