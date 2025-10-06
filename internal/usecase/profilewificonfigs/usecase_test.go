package profilewificonfigs_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/device-management-toolkit/console/internal/entity"
	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/mocks"
	"github.com/device-management-toolkit/console/internal/usecase/profilewificonfigs"
	"github.com/device-management-toolkit/console/pkg/logger"
)

type test struct {
	name        string
	tenantID    string
	profileName string
	mock        func(*mocks.MockProfileWiFiConfigsRepository)
	res         interface{}
	err         error
}

func profilewificonfigsTest(t *testing.T) (*profilewificonfigs.UseCase, *mocks.MockProfileWiFiConfigsRepository) {
	t.Helper()

	mockCtl := gomock.NewController(t)
	defer mockCtl.Finish()

	log := logger.New("error")

	repo := mocks.NewMockProfileWiFiConfigsRepository(mockCtl)

	useCase := profilewificonfigs.New(repo, log)

	return useCase, repo
}

func TestGetByProfileName(t *testing.T) {
	t.Parallel()

	tests := []test{
		{
			name:        "success",
			profileName: "profile1",
			tenantID:    "tenant1",
			mock: func(m *mocks.MockProfileWiFiConfigsRepository) {
				m.EXPECT().GetByProfileName(context.Background(), "profile1", "tenant1").Return([]entity.ProfileWiFiConfigs{
					{
						ProfileName: "profile1",
						TenantID:    "tenant1",
						Priority:    1,
					},
				}, nil)
			},
			res: []dto.ProfileWiFiConfigs{
				{
					ProfileName: "profile1",
					TenantID:    "tenant1",
					Priority:    1,
				},
			},
			err: nil,
		},

		{
			name:        "error",
			profileName: "profile1",
			tenantID:    "tenant1",
			mock: func(m *mocks.MockProfileWiFiConfigsRepository) {
				m.EXPECT().GetByProfileName(context.Background(), "profile1", "tenant1").Return(nil, profilewificonfigs.ErrDatabase)
			},
			err: profilewificonfigs.ErrDatabase.Wrap("Get", "uc.repo.Get", profilewificonfigs.ErrDatabase),
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, repo := profilewificonfigsTest(t)
			tc.mock(repo)
			_, err := useCase.GetByProfileName(context.Background(), tc.profileName, tc.tenantID)
			assert.Equal(t, tc.err, err)
		})
	}
}

func TestDeleteByProfileName(t *testing.T) {
	t.Parallel()

	tests := []test{
		{
			name:        "success",
			profileName: "profile1",
			tenantID:    "tenant1",
			mock: func(m *mocks.MockProfileWiFiConfigsRepository) {
				m.EXPECT().DeleteByProfileName(context.Background(), "profile1", "tenant1").Return(true, nil)
			},
		},
		{
			name:        "error",
			profileName: "profile1",
			tenantID:    "tenant1",
			mock: func(m *mocks.MockProfileWiFiConfigsRepository) {
				m.EXPECT().DeleteByProfileName(context.Background(), "profile1", "tenant1").Return(false, profilewificonfigs.ErrDatabase)
			},
			err: profilewificonfigs.ErrDatabase.Wrap("Delete", "uc.repo.Delete", profilewificonfigs.ErrDatabase),
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, repo := profilewificonfigsTest(t)
			tc.mock(repo)
			err := useCase.DeleteByProfileName(context.Background(), tc.profileName, tc.tenantID)
			assert.Equal(t, tc.err, err)
		})
	}
}
