package devices_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/software"

	"github.com/device-management-toolkit/console/internal/entity"
	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/usecase/devices"
)

func TestAdopt(t *testing.T) {
	t.Parallel()

	discovered := true
	stored := func() *entity.Device {
		return &entity.Device{
			GUID: "device-guid-123", TenantID: "tenant", Username: "old", Password: "encrypted", Discovered: &discovered,
			DeviceInfo: `{"discovered":true,"fwSku":"16392","currentMode":"Admin Control Mode"}`,
		}
	}

	t.Run("saves the credentials and marks the device managed after they verify", func(t *testing.T) {
		t.Parallel()

		useCase, wsmanMock, management, repo := initInfoTest(t)

		repo.EXPECT().GetByID(gomock.Any(), "device-guid-123", "tenant").Return(stored(), nil).AnyTimes()
		wsmanMock.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(management, nil)
		management.EXPECT().GetAMTVersion().Return([]software.SoftwareIdentity{}, nil)
		wsmanMock.EXPECT().DestroyWsmanClient(gomock.Any()).AnyTimes()
		repo.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, d *entity.Device) (bool, error) {
			require.Equal(t, "admin", d.Username)
			require.Equal(t, "encrypted", d.Password)
			require.NotNil(t, d.Discovered)
			require.False(t, *d.Discovered)

			return true, nil
		})

		_, err := useCase.Adopt(context.Background(), "device-guid-123", "tenant", "admin", "P@ssw0rd")
		require.NoError(t, err)
	})

	t.Run("refuses a device that is not eligible without contacting it", func(t *testing.T) {
		t.Parallel()

		useCase, _, _, repo := initInfoTest(t)

		notActivated := &entity.Device{
			GUID: "device-guid-123", TenantID: "tenant", Discovered: &discovered,
			DeviceInfo: `{"discovered":true,"fwSku":"16392","currentMode":"Not Activated"}`,
		}
		repo.EXPECT().GetByID(gomock.Any(), "device-guid-123", "tenant").Return(notActivated, nil)

		_, err := useCase.Adopt(context.Background(), "device-guid-123", "tenant", "admin", "P@ssw0rd")
		require.ErrorIs(t, err, devices.ErrAdoptNotEligible)
	})

	t.Run("rejects wrong credentials without saving them", func(t *testing.T) {
		t.Parallel()

		useCase, wsmanMock, management, repo := initInfoTest(t)

		repo.EXPECT().GetByID(gomock.Any(), "device-guid-123", "tenant").Return(stored(), nil)
		wsmanMock.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(management, nil)
		management.EXPECT().GetAMTVersion().Return(nil, errors.New("wsman.Client post received: 401 Unauthorized"))
		wsmanMock.EXPECT().DestroyWsmanClient(dto.Device{GUID: "device-guid-123", TenantID: "tenant"})

		_, err := useCase.Adopt(context.Background(), "device-guid-123", "tenant", "admin", "wrong")
		require.ErrorIs(t, err, devices.ErrInvalidCredentials)
	})

	t.Run("passes an unreachable device through as a different error", func(t *testing.T) {
		t.Parallel()

		useCase, wsmanMock, _, repo := initInfoTest(t)
		unreachable := errors.New("dial tcp: connection refused")

		repo.EXPECT().GetByID(gomock.Any(), "device-guid-123", "tenant").Return(stored(), nil)
		wsmanMock.EXPECT().SetupWsmanClient(gomock.Any(), gomock.Any(), false, true).Return(nil, unreachable)
		wsmanMock.EXPECT().DestroyWsmanClient(gomock.Any())

		_, err := useCase.Adopt(context.Background(), "device-guid-123", "tenant", "admin", "pw")
		require.ErrorIs(t, err, unreachable)
		require.NotErrorIs(t, err, devices.ErrInvalidCredentials)
	})

	t.Run("returns not found for an unknown device", func(t *testing.T) {
		t.Parallel()

		useCase, _, _, repo := initInfoTest(t)

		repo.EXPECT().GetByID(gomock.Any(), "missing", "tenant").Return(nil, nil)

		_, err := useCase.Adopt(context.Background(), "missing", "tenant", "admin", "pw")
		require.ErrorIs(t, err, devices.ErrNotFound)
	})
}
