package wificonfigs_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/device-management-toolkit/console/internal/entity"
	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/mocks"
	"github.com/device-management-toolkit/console/internal/usecase/wificonfigs"
	"github.com/device-management-toolkit/console/pkg/logger"
)

type test struct {
	name        string
	top         int
	skip        int
	input       dto.WirelessConfig
	profileName string
	tenantID    string
	mock        func(*mocks.MockWiFiConfigsRepository, ...interface{})
	res         interface{}
	err         error
}
type MockIEEE8021x struct{}

func (m MockIEEE8021x) CheckProfileExists(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}

func (m MockIEEE8021x) GetCount(_ context.Context, _ string) (int, error) {
	return 0, nil
}

func (m MockIEEE8021x) Get(_ context.Context, _, _ int, _ string) ([]dto.IEEE8021xConfig, error) {
	return []dto.IEEE8021xConfig{}, nil
}

func (m MockIEEE8021x) GetByName(_ context.Context, _, _ string) (*dto.IEEE8021xConfig, error) {
	return &dto.IEEE8021xConfig{}, nil
}

func (m MockIEEE8021x) Delete(_ context.Context, _, _ string) error {
	return nil
}

func (m MockIEEE8021x) Update(_ context.Context, _ *dto.IEEE8021xConfig) (*dto.IEEE8021xConfig, error) {
	return &dto.IEEE8021xConfig{}, nil
}

func (m MockIEEE8021x) Insert(_ context.Context, _ *dto.IEEE8021xConfig) (*dto.IEEE8021xConfig, error) {
	return &dto.IEEE8021xConfig{}, nil
}

func wificonfigsTest(t *testing.T) (*wificonfigs.UseCase, *mocks.MockWiFiConfigsRepository) {
	t.Helper()

	mockCtl := gomock.NewController(t)
	defer mockCtl.Finish()

	repo := mocks.NewMockWiFiConfigsRepository(mockCtl)
	log := logger.New("error")
	ieeeMock := MockIEEE8021x{}
	cryptoMock := mocks.MockCrypto{}
	useCase := wificonfigs.New(repo, ieeeMock, log, cryptoMock)

	return useCase, repo
}

func TestCheckProfileExists(t *testing.T) {
	t.Parallel()

	tests := []test{
		{
			name:        "empty result",
			profileName: "example-wirelessconfig",
			tenantID:    "tenant-id-456",
			mock: func(repo *mocks.MockWiFiConfigsRepository, args ...interface{}) {
				repo.EXPECT().CheckProfileExists(context.Background(), args[0], args[1]).Return(false, nil)
			},
			res: false,
			err: nil,
		},
		{
			name:        "result with error",
			profileName: "nonexistent-wirelessconfig",
			tenantID:    "tenant-id-456",
			mock: func(repo *mocks.MockWiFiConfigsRepository, args ...interface{}) {
				repo.EXPECT().CheckProfileExists(context.Background(), args[0], args[1]).Return(false, wificonfigs.ErrDatabase)
			},
			res: false,
			err: wificonfigs.ErrDatabase,
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, repo := wificonfigsTest(t)

			tc.mock(repo, tc.profileName, tc.tenantID)

			res, err := useCase.CheckProfileExists(context.Background(), tc.profileName, tc.tenantID)

			require.Equal(t, tc.res, res)
			require.IsType(t, tc.err, err)
		})
	}
}

func TestGetCount(t *testing.T) {
	t.Parallel()

	tests := []test{
		{
			name: "empty result",
			mock: func(repo *mocks.MockWiFiConfigsRepository, args ...interface{}) {
				repo.EXPECT().GetCount(context.Background(), "").Return(args[0], args[1])
			},
			res: 0,
			err: nil,
		},
		{
			name: "result with error",
			mock: func(repo *mocks.MockWiFiConfigsRepository, args ...interface{}) {
				repo.EXPECT().GetCount(context.Background(), "").Return(args[0], args[1])
			},
			res: 0,
			err: wificonfigs.ErrDatabase,
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			useCase, repo := wificonfigsTest(t)
			tc.mock(repo, tc.res, tc.err)

			res, err := useCase.GetCount(context.Background(), "")

			require.Equal(t, tc.res, res)
			require.IsType(t, tc.err, err)
		})
	}
}

func TestGet(t *testing.T) {
	t.Parallel()

	linkPolicy := "1,2"
	ieeeProfileName := "test-8021x"
	testWifiConfigsEntity := []entity.WirelessConfig{
		{
			ProfileName:            "test-wirelessconfig-1",
			TenantID:               "tenant-id-456",
			LinkPolicy:             &linkPolicy,
			IEEE8021xProfileName:   &ieeeProfileName,
			AuthenticationProtocol: &[]int{0}[0],
			WiredInterface:         &[]bool{false}[0],
		},
		{
			ProfileName: "test-wirelessconfig-2",
			TenantID:    "tenant-id-456",
		},
	}

	testWifiConfigDTOs := []dto.WirelessConfig{
		{
			ProfileName:          "test-wirelessconfig-1",
			TenantID:             "tenant-id-456",
			LinkPolicy:           []int{1, 2},
			IEEE8021xProfileName: &ieeeProfileName,
			IEEE8021xProfileObject: &dto.IEEE8021xConfig{
				AuthenticationProtocol: 0,
				WiredInterface:         false,
			},
		},
		{
			ProfileName: "test-wirelessconfig-2",
			TenantID:    "tenant-id-456",
			LinkPolicy:  []int{},
		},
	}

	tests := []test{
		{
			name:     "successful retrieval",
			top:      10,
			skip:     0,
			tenantID: "tenant-id-456",
			mock: func(repo *mocks.MockWiFiConfigsRepository, args ...interface{}) {
				repo.EXPECT().
					Get(context.Background(), args[0], args[1], args[2]).
					Return(testWifiConfigsEntity, nil)
			},
			res: testWifiConfigDTOs,
			err: nil,
		},
		{
			name:     "database error",
			top:      5,
			skip:     0,
			tenantID: "tenant-id-456",
			mock: func(repo *mocks.MockWiFiConfigsRepository, args ...interface{}) {
				repo.EXPECT().
					Get(context.Background(), args[0], args[1], args[2]).
					Return(nil, wificonfigs.ErrDatabase)
			},
			res: []dto.WirelessConfig(nil),
			err: wificonfigs.ErrDatabase,
		},
		{
			name:     "zero results",
			top:      10,
			skip:     20,
			tenantID: "tenant-id-456",
			mock: func(repo *mocks.MockWiFiConfigsRepository, args ...interface{}) {
				repo.EXPECT().
					Get(context.Background(), args[0], args[1], args[2]).
					Return([]entity.WirelessConfig{}, nil)
			},
			res: []dto.WirelessConfig{},
			err: nil,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			useCase, repo := wificonfigsTest(t)

			tc.mock(repo, tc.top, tc.skip, tc.tenantID)

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

	WirelessConfigEntity := &entity.WirelessConfig{
		ProfileName: "test-WirelessConfig",
		TenantID:    "tenant-id-456",
		Version:     "123",
	}

	wirelessConfigDTO := &dto.WirelessConfig{
		ProfileName: "test-WirelessConfig",
		TenantID:    "tenant-id-456",
		Version:     "123",
		LinkPolicy:  []int{},
	}

	tests := []test{
		{
			name: "successful retrieval",
			input: dto.WirelessConfig{
				ProfileName: "test-wirelessConfig",
				TenantID:    "tenant-id-456",
			},
			mock: func(repo *mocks.MockWiFiConfigsRepository, args ...interface{}) {
				repo.EXPECT().
					GetByName(context.Background(), args[0], args[1]).
					Return(WirelessConfigEntity, nil)
			},
			res: wirelessConfigDTO,
			err: nil,
		},
		{
			name: "WirelessConfig not found",
			input: dto.WirelessConfig{
				ProfileName: "unknown-WirelessConfig",
				TenantID:    "tenant-id-456",
			},
			mock: func(repo *mocks.MockWiFiConfigsRepository, args ...interface{}) {
				repo.EXPECT().
					GetByName(context.Background(), args[0], args[1]).
					Return(nil, nil)
			},
			res: (*dto.WirelessConfig)(nil),
			err: wificonfigs.ErrNotFound,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			useCase, repo := wificonfigsTest(t)

			tc.mock(repo, tc.input.ProfileName, tc.input.TenantID)

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
			profileName: "example-wirelessconfig",
			tenantID:    "tenant-id-456",
			mock: func(repo *mocks.MockWiFiConfigsRepository, args ...interface{}) {
				repo.EXPECT().
					Delete(context.Background(), args[0], args[1]).
					Return(true, nil)
			},
			err: nil,
		},
		{
			name:        "deletion fails - wirelessconfig not found",
			profileName: "nonexistent-wirelessconfig",
			tenantID:    "tenant-id-456",
			mock: func(repo *mocks.MockWiFiConfigsRepository, args ...interface{}) {
				repo.EXPECT().
					Delete(context.Background(), args[0], args[1]).
					Return(false, nil)
			},
			err: wificonfigs.ErrNotFound,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			useCase, repo := wificonfigsTest(t)

			tc.mock(repo, tc.profileName, tc.tenantID)

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

	wirelessConfig := &entity.WirelessConfig{
		ProfileName:   "test-WirelessConfig",
		TenantID:      "tenant-id-456",
		Version:       "123",
		PSKPassphrase: "encrypted",
		LinkPolicy:    new(string),
	}

	wirelessConfigDTO := &dto.WirelessConfig{
		ProfileName: "test-WirelessConfig",
		TenantID:    "tenant-id-456",
		Version:     "123",
		LinkPolicy:  []int{},
	}

	tests := []test{
		{
			name: "successful update",
			mock: func(repo *mocks.MockWiFiConfigsRepository, _ ...interface{}) {
				repo.EXPECT().
					Update(context.Background(), wirelessConfig).
					Return(true, nil)
				repo.EXPECT().
					GetByName(context.Background(), wirelessConfigDTO.ProfileName, wirelessConfigDTO.TenantID).
					Return(wirelessConfig, nil)
			},
			res: wirelessConfigDTO,
			err: nil,
		},
		{
			name: "update fails - not found",
			mock: func(repo *mocks.MockWiFiConfigsRepository, _ ...interface{}) {
				repo.EXPECT().
					Update(context.Background(), wirelessConfig).
					Return(false, wificonfigs.ErrNotFound)
			},
			res: (*dto.WirelessConfig)(nil),
			err: wificonfigs.ErrDatabase,
		},
		{
			name: "update fails - database error",
			mock: func(repo *mocks.MockWiFiConfigsRepository, _ ...interface{}) {
				repo.EXPECT().
					Update(context.Background(), wirelessConfig).
					Return(false, wificonfigs.ErrDatabase)
			},
			res: (*dto.WirelessConfig)(nil),
			err: wificonfigs.ErrDatabase,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, repo := wificonfigsTest(t)

			tc.mock(repo)

			result, err := useCase.Update(context.Background(), wirelessConfigDTO)

			require.Equal(t, tc.res, result)
			require.IsType(t, err, tc.err)
		})
	}
}

func TestInsert(t *testing.T) {
	t.Parallel()

	wirelessConfig := &entity.WirelessConfig{
		ProfileName:   "test-WirelessConfig",
		TenantID:      "tenant-id-456",
		Version:       "123",
		PSKPassphrase: "encrypted",
		LinkPolicy:    new(string),
	}

	wirelessConfigDTO := &dto.WirelessConfig{
		ProfileName: "test-WirelessConfig",
		TenantID:    "tenant-id-456",
		Version:     "123",
		LinkPolicy:  []int{},
	}

	tests := []test{
		{
			name: "successful insertion",
			mock: func(repo *mocks.MockWiFiConfigsRepository, _ ...interface{}) {
				repo.EXPECT().
					Insert(context.Background(), wirelessConfig).
					Return("unique-wirelessconfig-id", nil)
				repo.EXPECT().
					GetByName(context.Background(), wirelessConfigDTO.ProfileName, wirelessConfigDTO.TenantID).
					Return(wirelessConfig, nil)
			},
			res: wirelessConfigDTO,
			err: nil,
		},
		{
			name: "insertion fails - database error",
			mock: func(repo *mocks.MockWiFiConfigsRepository, _ ...interface{}) {
				repo.EXPECT().
					Insert(context.Background(), wirelessConfig).
					Return("", wificonfigs.ErrDatabase)
			},
			res: (*dto.WirelessConfig)(nil),
			err: wificonfigs.ErrDatabase,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, repo := wificonfigsTest(t)

			tc.mock(repo)

			id, err := useCase.Insert(context.Background(), wirelessConfigDTO)

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
