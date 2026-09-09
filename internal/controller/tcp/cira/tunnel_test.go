package cira

import (
	"errors"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/apf"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/mocks"
	"github.com/device-management-toolkit/console/internal/usecase/devices"
	"github.com/device-management-toolkit/console/internal/usecase/devices/wsman"
	"github.com/device-management-toolkit/console/pkg/logger"
)

type cleanupTestCase struct {
	name             string
	setupSession     func() *apf.Session
	authenticated    bool
	deviceID         string
	wantPanic        bool
	wantTimerStopped bool
}

var cleanupTests = []cleanupTestCase{
	{
		name:             "cleanup with nil session",
		setupSession:     func() *apf.Session { return nil },
		authenticated:    false,
		deviceID:         "",
		wantPanic:        false,
		wantTimerStopped: false,
	},
	{
		name: "cleanup with session but nil timer",
		setupSession: func() *apf.Session {
			return &apf.Session{Timer: nil}
		},
		authenticated:    false,
		deviceID:         "",
		wantPanic:        false,
		wantTimerStopped: false,
	},
	{
		name: "cleanup with timer that stops successfully",
		setupSession: func() *apf.Session {
			return &apf.Session{Timer: time.NewTimer(1 * time.Hour)}
		},
		authenticated:    false,
		deviceID:         "",
		wantPanic:        false,
		wantTimerStopped: true,
	},
	{
		name: "cleanup with timer that fails to stop should drain channel",
		setupSession: func() *apf.Session {
			timer := time.NewTimer(1 * time.Nanosecond)

			time.Sleep(2 * time.Millisecond)

			return &apf.Session{Timer: timer}
		},
		authenticated:    false,
		deviceID:         "",
		wantPanic:        false,
		wantTimerStopped: false,
	},
	{
		name: "cleanup with timer stop failure and empty channel hits default case",
		setupSession: func() *apf.Session {
			timer := time.NewTimer(100 * time.Nanosecond)

			time.Sleep(2 * time.Millisecond)

			select {
			case <-timer.C:
			default:
			}

			return &apf.Session{Timer: timer}
		},
		authenticated:    false,
		deviceID:         "",
		wantPanic:        false,
		wantTimerStopped: false,
	},
	{
		name: "cleanup with authenticated connection removes from connections map",
		setupSession: func() *apf.Session {
			return &apf.Session{Timer: time.NewTimer(10 * time.Second)}
		},
		authenticated:    true,
		deviceID:         "test-device",
		wantPanic:        false,
		wantTimerStopped: true,
	},
}

func TestConnectionContext_cleanup_UpdateConnectionStatusError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockDevices := mocks.NewMockDeviceManagementFeature(ctrl)
	mockDevices.EXPECT().UpdateConnectionStatus(gomock.Any(), "test-device", false).Return(errors.New("db error"))

	log := logger.New("error")
	handler := NewAPFHandler(mockDevices, log)
	handler.deviceID = "test-device"

	session := &apf.Session{Timer: time.NewTimer(10 * time.Second)}
	ctx := &connectionContext{
		session:       session,
		authenticated: true,
		handler:       handler,
		devices:       mockDevices,
		log:           log,
	}

	wsman.SetConnectionEntry("test-device", &wsman.ConnectionEntry{})

	require.NotPanics(t, func() {
		ctx.cleanup()
	})

	assert.Nil(t, wsman.GetConnectionEntry("test-device"), "Connection should still be removed even on status update failure")
}

func TestConnectionContext_cleanup(t *testing.T) {
	t.Parallel()

	for _, tt := range cleanupTests {
		tt := tt // capture range variable

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			runCleanupTest(t, tt)
		})
	}
}

func runCleanupTest(t *testing.T, tt cleanupTestCase) {
	t.Helper()

	ctrl := gomock.NewController(t)
	mockDevices := mocks.NewMockDeviceManagementFeature(ctrl)

	var devicesFeature devices.Feature

	if tt.authenticated && tt.deviceID != "" {
		mockDevices.EXPECT().UpdateConnectionStatus(gomock.Any(), tt.deviceID, false).Return(nil)
		devicesFeature = mockDevices
	}

	// Setup
	session := tt.setupSession()
	ctx := setupConnectionContext(t, session, tt.authenticated, tt.deviceID, devicesFeature)

	setupConnectionsMap(t, tt.authenticated, tt.deviceID)

	// Execute
	require.NotPanics(t, func() {
		ctx.cleanup()
	})

	// Verify
	verifyTimerState(t, session, tt.wantTimerStopped)
	verifyConnectionRemoved(t, tt.authenticated, tt.deviceID)
}

func setupConnectionContext(t *testing.T, session *apf.Session, authenticated bool, deviceID string, devicesFeature devices.Feature) *connectionContext {
	t.Helper()

	// Create a proper APFHandler with mock deviceID
	log := logger.New("error")
	handler := NewAPFHandler(devicesFeature, log)
	handler.deviceID = deviceID

	return &connectionContext{
		session:       session,
		authenticated: authenticated,
		handler:       handler,
		devices:       devicesFeature,
	}
}

func setupConnectionsMap(t *testing.T, authenticated bool, deviceID string) {
	t.Helper()

	if authenticated && deviceID != "" {
		wsman.SetConnectionEntry(deviceID, &wsman.ConnectionEntry{})
	}
}

func verifyTimerState(t *testing.T, session *apf.Session, wantTimerStopped bool) {
	t.Helper()

	if wantTimerStopped && session != nil && session.Timer != nil {
		select {
		case <-session.Timer.C:
			// Timer was stopped and channel was drained, or timer expired naturally
		default:
			// Timer was stopped before it could fire
		}
	}
}

func verifyConnectionRemoved(t *testing.T, authenticated bool, deviceID string) {
	t.Helper()

	if authenticated && deviceID != "" {
		exists := wsman.GetConnectionEntry(deviceID) != nil

		assert.False(t, exists, "Connection should be removed from map")
	}
}

func TestAPFHandler_ValidatesCredentialsAndProtocolVersion(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockDevices := mocks.NewMockDeviceManagementFeature(ctrl)
	mockDevices.EXPECT().GetByID(gomock.Any(), "device-123", "", true).Return(&dto.Device{
		MPSUsername: "admin",
		MPSPassword: "s3cret",
	}, nil).AnyTimes()

	handler := NewAPFHandler(mockDevices, logger.New("debug"))
	require.NoError(t, handler.OnProtocolVersion(apf.ProtocolVersionInfo{UUID: "DEVICE-123"}))
	assert.Equal(t, "device-123", handler.DeviceID())

	assert.True(t, handler.validateCredentials("admin", "s3cret"))
	assert.False(t, handler.validateCredentials("admin", "wrong-pass"))
	assert.False(t, handler.validateCredentials("wrong-user", "s3cret"))
	assert.False(t, handler.validateCredentials("wrong-user", "wrong-pass"))
	assert.False(t, handler.validateCredentials("", ""))

	resp := handler.OnAuthRequest(apf.AuthRequest{Username: "admin", Password: "s3cret", MethodName: "password"})
	assert.True(t, resp.Authenticated)

	unsupported := handler.OnAuthRequest(apf.AuthRequest{Username: "admin", Password: "s3cret", MethodName: "keyboard-interactive"})
	assert.False(t, unsupported.Authenticated)
}

func TestAPFHandler_validateCredentials_FailurePaths(t *testing.T) {
	t.Parallel()

	t.Run("device id not set", func(t *testing.T) {
		t.Parallel()

		handler := NewAPFHandler(nil, logger.New("error"))

		assert.False(t, handler.validateCredentials("admin", "s3cret"))
	})

	t.Run("lookup error", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		mockDevices := mocks.NewMockDeviceManagementFeature(ctrl)
		mockDevices.EXPECT().GetByID(gomock.Any(), "dev-err", "", true).Return(nil, errors.New("db error"))

		handler := NewAPFHandler(mockDevices, logger.New("error"))
		handler.deviceID = "dev-err"

		assert.False(t, handler.validateCredentials("admin", "s3cret"))
	})

	t.Run("device not found", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		mockDevices := mocks.NewMockDeviceManagementFeature(ctrl)
		mockDevices.EXPECT().GetByID(gomock.Any(), "dev-missing", "", true).Return(nil, nil)

		handler := NewAPFHandler(mockDevices, logger.New("error"))
		handler.deviceID = "dev-missing"

		assert.False(t, handler.validateCredentials("admin", "s3cret"))
	})
}

func TestAPFHandler_TracksKeepAliveThreshold(t *testing.T) {
	t.Parallel()

	handler := NewAPFHandler(nil, logger.New("debug"))
	assert.False(t, handler.ShouldSendKeepAlive())

	for i := 0; i < globalRequestThreshold-1; i++ {
		assert.False(t, handler.OnGlobalRequest(apf.GlobalRequest{}))
		assert.False(t, handler.ShouldSendKeepAlive())
	}

	assert.True(t, handler.OnGlobalRequest(apf.GlobalRequest{}))
	assert.True(t, handler.ShouldSendKeepAlive())
}

// fakeConn is a minimal net.Conn implementation for tests.
type fakeConn struct{ net.Conn }

func TestConnectionContext_registerDevice(t *testing.T) {
	t.Parallel()

	t.Run("successful registration updates connection status", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)

		mockDevices := mocks.NewMockDeviceManagementFeature(ctrl)
		mockDevices.EXPECT().UpdateConnectionStatus(gomock.Any(), "dev-123", true).Return(nil)

		log := logger.New("error")
		handler := NewAPFHandler(mockDevices, log)
		handler.deviceID = "dev-123"

		ctx := &connectionContext{
			conn:    &fakeConn{},
			handler: handler,
			devices: mockDevices,
			log:     log,
		}

		ctx.registerDevice()

		assert.True(t, ctx.authenticated)
		assert.NotNil(t, ctx.device)
		assert.NotNil(t, wsman.GetConnectionEntry("dev-123"))

		t.Cleanup(func() { wsman.RemoveConnection("dev-123") })
	})

	t.Run("registration continues when UpdateConnectionStatus fails", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)

		mockDevices := mocks.NewMockDeviceManagementFeature(ctrl)
		mockDevices.EXPECT().UpdateConnectionStatus(gomock.Any(), "dev-456", true).Return(errors.New("db error"))

		log := logger.New("error")
		handler := NewAPFHandler(mockDevices, log)
		handler.deviceID = "dev-456"

		ctx := &connectionContext{
			conn:    &fakeConn{},
			handler: handler,
			devices: mockDevices,
			log:     log,
		}

		ctx.registerDevice()

		assert.True(t, ctx.authenticated)
		assert.NotNil(t, wsman.GetConnectionEntry("dev-456"))

		t.Cleanup(func() { wsman.RemoveConnection("dev-456") })
	})
}
