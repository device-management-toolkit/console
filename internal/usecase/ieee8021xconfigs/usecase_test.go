package ieee8021xconfigs_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/device-management-toolkit/console/internal/entity"
	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/mocks"
	"github.com/device-management-toolkit/console/internal/usecase/ieee8021xconfigs"
	"github.com/device-management-toolkit/console/pkg/logger"
)

type test struct {
	name        string
	top         int
	skip        int
	tenantID    string
	input       entity.IEEE8021xConfig
	profileName string
	mock        func(*mocks.MockIEEE8021xConfigsRepository)
	res         interface{}
	err         error
}

func ieee8021xconfigsTest(t *testing.T) (*ieee8021xconfigs.UseCase, *mocks.MockIEEE8021xConfigsRepository) {
	t.Helper()

	mockCtl := gomock.NewController(t)
	defer mockCtl.Finish()

	log := logger.New("error")

	repo := mocks.NewMockIEEE8021xConfigsRepository(mockCtl)

	useCase := ieee8021xconfigs.New(repo, log)

	return useCase, repo
}

func TestCheckProfileExists(t *testing.T) {
	t.Parallel()

	tests := []test{
		{
			name:        "empty result",
			profileName: "example-ieee8021xconfig",
			tenantID:    "tenant-id-456",
			mock: func(repo *mocks.MockIEEE8021xConfigsRepository) {
				repo.EXPECT().CheckProfileExists(context.Background(), "example-ieee8021xconfig", "tenant-id-456").Return(false, nil)
			},
			res: false,
			err: nil,
		},
		{
			name:        "result with error",
			profileName: "nonexistent-ieee8021xconfig",
			tenantID:    "tenant-id-456",
			mock: func(repo *mocks.MockIEEE8021xConfigsRepository) {
				repo.EXPECT().CheckProfileExists(context.Background(), "nonexistent-ieee8021xconfig", "tenant-id-456").Return(false, ieee8021xconfigs.ErrDatabase)
			},
			res: false,
			err: ieee8021xconfigs.ErrDatabase,
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			useCase, repo := ieee8021xconfigsTest(t)

			tc.mock(repo)

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
			mock: func(repo *mocks.MockIEEE8021xConfigsRepository) {
				repo.EXPECT().GetCount(context.Background(), "").Return(0, nil)
			},
			res: 0,
			err: nil,
		},
		{
			name: "result with error",
			mock: func(repo *mocks.MockIEEE8021xConfigsRepository) {
				repo.EXPECT().GetCount(context.Background(), "").Return(0, ieee8021xconfigs.ErrDatabase)
			},
			res: 0,
			err: ieee8021xconfigs.ErrDatabase,
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			useCase, repo := ieee8021xconfigsTest(t)

			tc.mock(repo)

			res, err := useCase.GetCount(context.Background(), "")

			require.Equal(t, tc.res, res)
			require.IsType(t, tc.err, err)
		})
	}
}

func TestGet(t *testing.T) {
	t.Parallel()

	IEEE8021xConfigs := []entity.IEEE8021xConfig{
		{
			ProfileName: "test-IEEE8021xConfig-1",
			TenantID:    "tenant-id-456",
		},
		{
			ProfileName: "test-IEEE8021xConfig-2",
			TenantID:    "tenant-id-456",
		},
	}

	IEEE8021xConfigDTOs := []dto.IEEE8021xConfig{
		{
			ProfileName: "test-IEEE8021xConfig-1",
			TenantID:    "tenant-id-456",
		},
		{
			ProfileName: "test-IEEE8021xConfig-2",
			TenantID:    "tenant-id-456",
		},
	}

	tests := []test{
		{
			name:     "successful retrieval",
			top:      10,
			skip:     0,
			tenantID: "tenant-id-456",
			mock: func(repo *mocks.MockIEEE8021xConfigsRepository) {
				repo.EXPECT().
					Get(context.Background(), 10, 0, "tenant-id-456").
					Return(IEEE8021xConfigs, nil)
			},
			res: IEEE8021xConfigDTOs,
			err: nil,
		},
		{
			name:     "database error",
			top:      5,
			skip:     0,
			tenantID: "tenant-id-456",
			mock: func(repo *mocks.MockIEEE8021xConfigsRepository) {
				repo.EXPECT().
					Get(context.Background(), 5, 0, "tenant-id-456").
					Return(nil, ieee8021xconfigs.ErrDatabase)
			},
			res: []dto.IEEE8021xConfig(nil),
			err: ieee8021xconfigs.ErrDatabase,
		},
		{
			name:     "zero results",
			top:      10,
			skip:     20,
			tenantID: "tenant-id-456",
			mock: func(repo *mocks.MockIEEE8021xConfigsRepository) {
				repo.EXPECT().
					Get(context.Background(), 10, 20, "tenant-id-456").
					Return([]entity.IEEE8021xConfig{}, nil)
			},
			res: []dto.IEEE8021xConfig{},
			err: nil,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			useCase, repo := ieee8021xconfigsTest(t)

			tc.mock(repo)

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

	ieee8021xconfig := &entity.IEEE8021xConfig{
		ProfileName: "test-profile",
		TenantID:    "tenant-id-456",
		Version:     "1.0.0",
	}

	ieee8021xconfigDTO := &dto.IEEE8021xConfig{
		ProfileName: "test-profile",
		TenantID:    "tenant-id-456",
		Version:     "1.0.0",
	}

	tests := []test{
		{
			name: "successful retrieval",
			input: entity.IEEE8021xConfig{
				ProfileName: "test-ieee8021xconfig",
				TenantID:    "tenant-id-456",
			},
			mock: func(repo *mocks.MockIEEE8021xConfigsRepository) {
				repo.EXPECT().
					GetByName(context.Background(), "test-ieee8021xconfig", "tenant-id-456").
					Return(ieee8021xconfig, nil)
			},
			res: ieee8021xconfigDTO,
			err: nil,
		},
		{
			name: "ieee8021xconfig not found",
			input: entity.IEEE8021xConfig{
				ProfileName: "unknown-ieee8021xconfig",
				TenantID:    "tenant-id-456",
			},
			mock: func(repo *mocks.MockIEEE8021xConfigsRepository) {
				repo.EXPECT().
					GetByName(context.Background(), "unknown-ieee8021xconfig", "tenant-id-456").
					Return(nil, nil)
			},
			res: (*dto.IEEE8021xConfig)(nil),
			err: ieee8021xconfigs.ErrNotFound,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			useCase, repo := ieee8021xconfigsTest(t)

			tc.mock(repo)

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
			profileName: "example-ieee8021xconfig",
			tenantID:    "tenant-id-456",
			mock: func(repo *mocks.MockIEEE8021xConfigsRepository) {
				repo.EXPECT().
					Delete(context.Background(), "example-ieee8021xconfig", "tenant-id-456").
					Return(true, nil)
			},
			err: nil,
		},
		{
			name:        "deletion fails - ieee8021xconfig not found",
			profileName: "nonexistent-ieee8021xconfig",
			tenantID:    "tenant-id-456",
			mock: func(repo *mocks.MockIEEE8021xConfigsRepository) {
				repo.EXPECT().
					Delete(context.Background(), "nonexistent-ieee8021xconfig", "tenant-id-456").
					Return(false, nil)
			},
			err: ieee8021xconfigs.ErrNotFound,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			useCase, repo := ieee8021xconfigsTest(t)

			tc.mock(repo)

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

	ieee8021xconfig := &entity.IEEE8021xConfig{
		ProfileName: "example-ieee8021xconfig",
		TenantID:    "tenant-id-456",
		Version:     "1.0.0",
	}

	ieee8021xconfigDTO := &dto.IEEE8021xConfig{
		ProfileName: "example-ieee8021xconfig",
		TenantID:    "tenant-id-456",
		Version:     "1.0.0",
	}

	tests := []test{
		{
			name: "successful update",
			mock: func(repo *mocks.MockIEEE8021xConfigsRepository) {
				repo.EXPECT().
					Update(context.Background(), ieee8021xconfig).
					Return(true, nil)
				repo.EXPECT().
					GetByName(context.Background(), ieee8021xconfig.ProfileName, ieee8021xconfig.TenantID).
					Return(ieee8021xconfig, nil)
			},
			res: ieee8021xconfigDTO,
			err: nil,
		},
		{
			name: "update fails - not found",
			mock: func(repo *mocks.MockIEEE8021xConfigsRepository) {
				repo.EXPECT().
					Update(context.Background(), ieee8021xconfig).
					Return(false, ieee8021xconfigs.ErrNotFound)
			},
			res: (*dto.IEEE8021xConfig)(nil),
			err: ieee8021xconfigs.ErrDatabase,
		},
		{
			name: "update fails - database error",
			mock: func(repo *mocks.MockIEEE8021xConfigsRepository) {
				repo.EXPECT().
					Update(context.Background(), ieee8021xconfig).
					Return(false, ieee8021xconfigs.ErrDatabase)
			},
			res: (*dto.IEEE8021xConfig)(nil),
			err: ieee8021xconfigs.ErrDatabase,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			useCase, repo := ieee8021xconfigsTest(t)

			tc.mock(repo)

			result, err := useCase.Update(context.Background(), ieee8021xconfigDTO)

			require.Equal(t, tc.res, result)
			require.IsType(t, tc.err, err)
		})
	}
}

func TestInsert(t *testing.T) {
	t.Parallel()

	ieee8021xconfig := &entity.IEEE8021xConfig{
		ProfileName: "new-ieee8021xconfig",
		TenantID:    "tenant-id-789",
		Version:     "1.0.0",
	}

	ieee8021xconfigDTO := &dto.IEEE8021xConfig{
		ProfileName: "new-ieee8021xconfig",
		TenantID:    "tenant-id-789",
		Version:     "1.0.0",
	}

	tests := []test{
		{
			name: "successful insertion",
			mock: func(repo *mocks.MockIEEE8021xConfigsRepository) {
				repo.EXPECT().
					Insert(context.Background(), ieee8021xconfig).
					Return("unique-ieee8021xconfig", nil)
				repo.EXPECT().
					GetByName(context.Background(), ieee8021xconfig.ProfileName, ieee8021xconfig.TenantID).
					Return(ieee8021xconfig, nil)
			},
			res: ieee8021xconfigDTO,
			err: nil,
		},
		{
			name: "insertion fails - database error",
			mock: func(repo *mocks.MockIEEE8021xConfigsRepository) {
				repo.EXPECT().
					Insert(context.Background(), ieee8021xconfig).
					Return("", ieee8021xconfigs.ErrDatabase)
			},
			res: (*dto.IEEE8021xConfig)(nil),
			err: ieee8021xconfigs.ErrDatabase,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			useCase, repo := ieee8021xconfigsTest(t)

			tc.mock(repo)

			config, err := useCase.Insert(context.Background(), ieee8021xconfigDTO)

			require.Equal(t, tc.res, config)

			if tc.err != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}
