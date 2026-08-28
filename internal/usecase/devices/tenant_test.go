package devices_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/power"

	"github.com/device-management-toolkit/console/internal/entity"
	"github.com/device-management-toolkit/console/internal/mocks"
	"github.com/device-management-toolkit/console/internal/tenant"
	devices "github.com/device-management-toolkit/console/internal/usecase/devices"
	"github.com/device-management-toolkit/console/pkg/logger"
)

func TestSendPowerActionUsesRequestTenant(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	repo := mocks.NewMockDeviceManagementRepository(ctrl)
	wsman := mocks.NewMockWSMAN(ctrl)
	management := mocks.NewMockManagement(ctrl)

	wsman.EXPECT().Worker().AnyTimes()
	uc := devices.New(repo, wsman, mocks.NewMockRedirection(ctrl), logger.New("error"), mocks.MockCrypto{})
	ctx := tenant.WithContext(context.Background(), "tenant-a")
	device := &entity.Device{GUID: "device-guid", TenantID: "tenant-a", Password: "encrypted"}

	repo.EXPECT().GetByID(ctx, device.GUID, "tenant-a").Return(device, nil)
	wsman.EXPECT().SetupWsmanClient(ctx, gomock.Any(), false, true).Return(management, nil)
	management.EXPECT().SendPowerAction(0).Return(power.PowerActionResponse{}, nil)

	_, err := uc.SendPowerAction(ctx, device.GUID, 0)
	require.NoError(t, err)
}
