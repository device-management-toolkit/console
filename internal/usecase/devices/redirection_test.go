package devices_test

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/security"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman"

	"github.com/device-management-toolkit/console/internal/entity"
	"github.com/device-management-toolkit/console/internal/mocks"
	devices "github.com/device-management-toolkit/console/internal/usecase/devices"
	wsmanAPI "github.com/device-management-toolkit/console/internal/usecase/devices/wsman"
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
	name string
	res  any
	err  error
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
			res:  wsman.Messages{},
			err:  nil,
		},
	}

	for _, tc := range tests {
		tc := tc // Necessary for proper parallel execution
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			redirector, _, _ := initRedirectionTest(t)

			redirector.SafeRequirements = mocks.MockCrypto{}

			res, err := redirector.SetupWsmanClient(*device, true, true)

			require.IsType(t, tc.res, res)
			require.Equal(t, tc.err, err)
		})
	}
}

func TestSetupWsmanClient_CIRARedirection(t *testing.T) {
	t.Parallel()

	t.Run("returns error when CIRA device not connected", func(t *testing.T) {
		t.Parallel()

		device := entity.Device{
			GUID:        "cira-device-not-connected",
			MPSUsername: "admin",
		}

		redirector := &devices.Redirector{SafeRequirements: mocks.MockCrypto{}}

		_, err := redirector.SetupWsmanClient(device, true, false)
		require.ErrorIs(t, err, wsmanAPI.ErrCIRADeviceNotConnected)
	})

	t.Run("returns messages when CIRA device is connected", func(t *testing.T) {
		t.Parallel()

		guid := "cira-device-connected"

		// Set up a connection entry so the lookup succeeds
		server, client := net.Pipe()
		defer server.Close()
		defer client.Close()

		wsmanAPI.SetConnectionEntry(guid, &wsmanAPI.ConnectionEntry{
			IsCIRA: true,
			Conny:  client,
		})
		t.Cleanup(func() { wsmanAPI.RemoveConnection(guid) })

		device := entity.Device{
			GUID:        guid,
			MPSUsername: "admin",
		}

		redirector := &devices.Redirector{SafeRequirements: mocks.MockCrypto{}}

		msgs, err := redirector.SetupWsmanClient(device, true, false)
		require.NoError(t, err)
		require.NotNil(t, msgs.Client)
	})

	t.Run("non-CIRA device skips CIRA path", func(t *testing.T) {
		t.Parallel()

		device := entity.Device{
			GUID:     "normal-device",
			Hostname: "192.168.1.1",
			Username: "admin",
			Password: "encrypted",
		}

		redirector := &devices.Redirector{SafeRequirements: mocks.MockCrypto{}}

		msgs, err := redirector.SetupWsmanClient(device, true, false)
		require.NoError(t, err)
		require.NotNil(t, msgs)
	})

	t.Run("CIRA device with isRedirection false skips CIRA path", func(t *testing.T) {
		t.Parallel()

		device := entity.Device{
			GUID:        "cira-device-no-redirect",
			MPSUsername: "admin",
			Hostname:    "192.168.1.1",
			Username:    "admin",
			Password:    "encrypted",
		}

		redirector := &devices.Redirector{SafeRequirements: mocks.MockCrypto{}}

		msgs, err := redirector.SetupWsmanClient(device, false, false)
		require.NoError(t, err)
		require.NotNil(t, msgs)
	})
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
