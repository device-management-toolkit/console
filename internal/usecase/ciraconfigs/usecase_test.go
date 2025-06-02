package ciraconfigs_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/device-management-toolkit/console/internal/entity"
	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/mocks"
	"github.com/device-management-toolkit/console/internal/usecase/ciraconfigs"
	"github.com/device-management-toolkit/console/pkg/logger"
)

type test struct {
	name       string
	top        int
	skip       int
	configName string
	tenantID   string
	input      dto.CIRAConfig
	mock       func(*mocks.MockCIRAConfigsRepository)
	res        interface{}
	err        error
}

func ciraconfigsTest(t *testing.T) (*ciraconfigs.UseCase, *mocks.MockCIRAConfigsRepository) {
	t.Helper()

	mockCtl := gomock.NewController(t)
	defer mockCtl.Finish()

	repo := mocks.NewMockCIRAConfigsRepository(mockCtl)
	crypto := mocks.MockCrypto{}
	log := logger.New("error")
	useCase := ciraconfigs.New(repo, log, crypto)

	return useCase, repo
}

func TestGetCount(t *testing.T) {
	t.Parallel()

	tests := []test{
		{
			name: "empty result",
			mock: func(repo *mocks.MockCIRAConfigsRepository) {
				repo.EXPECT().GetCount(context.Background(), "").Return(0, nil)
			},
			res: 0,
			err: nil,
		},
		{
			name: "result with error",
			mock: func(repo *mocks.MockCIRAConfigsRepository) {
				repo.EXPECT().GetCount(context.Background(), "").Return(0, ciraconfigs.ErrDatabase.Wrap("", "", nil))
			},
			res: 0,
			err: ciraconfigs.ErrDatabase.Wrap("", "", nil),
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, repo := ciraconfigsTest(t)

			tc.mock(repo)

			res, err := useCase.GetCount(context.Background(), "")

			require.Equal(t, tc.res, res)
			require.IsType(t, tc.err, err)
		})
	}
}

func TestGet(t *testing.T) {
	t.Parallel()

	testCIRAConfigs := []entity.CIRAConfig{
		{
			ConfigName: "test-config-1",
			TenantID:   "tenant-id-456",
		},
		{
			ConfigName: "test-config-2",
			TenantID:   "tenant-id-456",
		},
	}

	testCIRAConfigDTOs := []dto.CIRAConfig{
		{
			ConfigName: "test-config-1",
			TenantID:   "tenant-id-456",
		},
		{
			ConfigName: "test-config-2",
			TenantID:   "tenant-id-456",
		},
	}

	tests := []test{
		{
			name:     "successful retrieval",
			top:      10,
			skip:     0,
			tenantID: "tenant-id-456",
			mock: func(repo *mocks.MockCIRAConfigsRepository) {
				repo.EXPECT().
					Get(context.Background(), 10, 0, "tenant-id-456").
					Return(testCIRAConfigs, nil)
			},
			res: testCIRAConfigDTOs,
			err: nil,
		},
		{
			name:     "database error",
			top:      5,
			skip:     0,
			tenantID: "tenant-id-456",
			mock: func(repo *mocks.MockCIRAConfigsRepository) {
				repo.EXPECT().
					Get(context.Background(), 5, 0, "tenant-id-456").
					Return(nil, ciraconfigs.ErrDatabase)
			},
			res: []dto.CIRAConfig(nil),
			err: ciraconfigs.ErrDatabase,
		},
		{
			name:     "zero results",
			top:      10,
			skip:     20,
			tenantID: "tenant-id-456",
			mock: func(repo *mocks.MockCIRAConfigsRepository) {
				repo.EXPECT().
					Get(context.Background(), 10, 20, "tenant-id-456").
					Return([]entity.CIRAConfig{}, nil)
			},
			res: []dto.CIRAConfig{},
			err: nil,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, repo := ciraconfigsTest(t)

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

	ciraconfig := &entity.CIRAConfig{
		ConfigName: "test-config",
		TenantID:   "tenant-id-456",
		Version:    "1.0.0",
	}

	ciraconfigDTO := &dto.CIRAConfig{
		ConfigName: "test-config",
		TenantID:   "tenant-id-456",
		Version:    "1.0.0",
	}

	tests := []test{
		{
			name: "successful retrieval",
			input: dto.CIRAConfig{
				ConfigName: "test-config",
				TenantID:   "tenant-id-456",
			},
			mock: func(repo *mocks.MockCIRAConfigsRepository) {
				repo.EXPECT().
					GetByName(context.Background(), "test-config", "tenant-id-456").
					Return(ciraconfig, nil)
			},
			res: ciraconfigDTO,
			err: nil,
		},
		{
			name: "ciraconfig not found",
			input: dto.CIRAConfig{
				ConfigName: "unknown-ciraconfig",
				TenantID:   "tenant-id-456",
			},
			mock: func(repo *mocks.MockCIRAConfigsRepository) {
				repo.EXPECT().
					GetByName(context.Background(), "unknown-ciraconfig", "tenant-id-456").
					Return(nil, nil)
			},
			res: (*dto.CIRAConfig)(nil),
			err: ciraconfigs.ErrNotFound,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, repo := ciraconfigsTest(t)

			tc.mock(repo)

			res, err := useCase.GetByName(context.Background(), tc.input.ConfigName, tc.input.TenantID)

			require.Equal(t, tc.res, res)

			if tc.err != nil {
				require.Contains(t, err.Error(), tc.err.Error())
			} else {
				require.Equal(t, tc.res, res)
				require.NoError(t, err)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	t.Parallel()

	tests := []test{
		{
			name:       "successful deletion",
			configName: "example-ciraconfig",
			tenantID:   "tenant-id-456",
			mock: func(repo *mocks.MockCIRAConfigsRepository) {
				repo.EXPECT().
					Delete(context.Background(), "example-ciraconfig", "tenant-id-456").
					Return(true, nil)
			},
			err: nil,
		},
		{
			name:       "deletion fails - ciraconfig not found",
			configName: "nonexistent-ciraconfig",
			tenantID:   "tenant-id-456",
			mock: func(repo *mocks.MockCIRAConfigsRepository) {
				repo.EXPECT().
					Delete(context.Background(), "nonexistent-ciraconfig", "tenant-id-456").
					Return(false, nil)
			},
			err: ciraconfigs.ErrNotFound,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, repo := ciraconfigsTest(t)

			tc.mock(repo)

			err := useCase.Delete(context.Background(), tc.configName, tc.tenantID)

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

	ciraconfig := &entity.CIRAConfig{
		ConfigName: "test-config",
		TenantID:   "tenant-id-456",
		Password:   "encrypted",
		Version:    "1.0.0",
	}

	ciraconfigDTO := &dto.CIRAConfig{
		ConfigName: "test-config",
		TenantID:   "tenant-id-456",
		Version:    "1.0.0",
	}

	tests := []test{
		{
			name: "successful update",
			mock: func(repo *mocks.MockCIRAConfigsRepository) {
				repo.EXPECT().
					Update(context.Background(), ciraconfig).
					Return(true, nil)
				repo.EXPECT().
					GetByName(context.Background(), "test-config", "tenant-id-456").
					Return(ciraconfig, nil)
			},
			res: ciraconfigDTO,
			err: nil,
		},
		{
			name: "update fails - not found",
			mock: func(repo *mocks.MockCIRAConfigsRepository) {
				repo.EXPECT().
					Update(context.Background(), ciraconfig).
					Return(false, nil)
			},
			res: (*dto.CIRAConfig)(nil),
			err: ciraconfigs.ErrNotFound,
		},
		{
			name: "update fails - database error",
			mock: func(repo *mocks.MockCIRAConfigsRepository) {
				repo.EXPECT().
					Update(context.Background(), ciraconfig).
					Return(false, ciraconfigs.ErrDatabase)
			},
			res: (*dto.CIRAConfig)(nil),
			err: ciraconfigs.ErrDatabase,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, repo := ciraconfigsTest(t)

			tc.mock(repo)

			result, err := useCase.Update(context.Background(), ciraconfigDTO)

			require.Equal(t, tc.res, result)
			require.IsType(t, tc.err, err)
		})
	}
}

func TestInsert(t *testing.T) {
	t.Parallel()

	ciraconfig := &entity.CIRAConfig{
		ConfigName: "test-config",
		TenantID:   "tenant-id-456",
		Password:   "encrypted",
		Version:    "1.0.0",
	}

	ciraconfigDTO := &dto.CIRAConfig{
		ConfigName: "test-config",
		TenantID:   "tenant-id-456",
		Version:    "1.0.0",
	}

	tests := []test{
		{
			name: "successful insertion",
			mock: func(repo *mocks.MockCIRAConfigsRepository) {
				repo.EXPECT().
					Insert(context.Background(), ciraconfig).
					Return("unique-ciraconfig-id", nil)
				repo.EXPECT().
					GetByName(context.Background(), "test-config", "tenant-id-456").
					Return(ciraconfig, nil)
			},
			res: ciraconfigDTO,
			err: nil,
		},
		{
			name: "insertion fails - database error",
			mock: func(repo *mocks.MockCIRAConfigsRepository) {
				repo.EXPECT().
					Insert(context.Background(), ciraconfig).
					Return("", ciraconfigs.ErrDatabase)
			},
			res: (*dto.CIRAConfig)(nil),
			err: ciraconfigs.ErrDatabase,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, repo := ciraconfigsTest(t)

			tc.mock(repo)

			config, err := useCase.Insert(context.Background(), ciraconfigDTO)

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
