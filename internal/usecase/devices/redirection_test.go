package devices_test

import (
	"testing"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/security"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"

	"github.com/device-management-toolkit/console/internal/entity"
	"github.com/device-management-toolkit/console/internal/mocks"
	devices "github.com/device-management-toolkit/console/internal/usecase/devices"
)

func initRedirectionTest(t *testing.T) (*devices.Redirector, *mocks.MockRedirection, *mocks.MockDeviceManagementRepository) {
	t.Helper()

	mockCtl := gomock.NewController(t)
	defer mockCtl.Finish()

	repo := mocks.NewMockDeviceManagementRepository(mockCtl)
	redirect := mocks.NewMockRedirection(mockCtl)
	u := &devices.Redirector{}

	return u, redirect, repo
}

type redTest struct {
	name    string
	redMock func(*mocks.MockRedirection)
	res     any
}

func TestSetupWsmanClient(t *testing.T) {
	t.Parallel()

	device := &entity.Device{
		GUID:     "device-guid-123",
		TenantID: "tenant-id-456",
	}

	tests := []redTest{
		{
			name: "success",
			redMock: func(redirect *mocks.MockRedirection) {
				redirect.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(wsman.Messages{})
			},
			res: wsman.Messages{},
		},
		{
			name: "fail",
			redMock: func(redirect *mocks.MockRedirection) {
				redirect.EXPECT().
					SetupWsmanClient(gomock.Any(), true, true).
					Return(wsman.Messages{})
			},
			res: wsman.Messages{},
		},
	}

	for _, tc := range tests {
		tc := tc // Necessary for proper parallel execution
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			redirector, redirect, _ := initRedirectionTest(t)

			tc.redMock(redirect)

			redirector.SafeRequirements = security.Crypto{
				EncryptionKey: "test",
			}

			res := redirector.SetupWsmanClient(*device, true, true)

			require.IsType(t, tc.res, res)
		})
	}
}

func TestNewRedirector(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{
			name: "success",
		},
	}

	for _, tc := range tests {
		tc := tc // Necessary for proper parallel execution
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			safeRequirements := security.Crypto{
				EncryptionKey: "test",
			}
			// Call the function under test
			redirector := devices.NewRedirector(safeRequirements)

			// Assert that the returned redirector is not nil
			require.NotNil(t, redirector)
		})
	}
}
