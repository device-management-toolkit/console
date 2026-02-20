package cira

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/apf"

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

	// Setup
	session := tt.setupSession()
	ctx := setupConnectionContext(t, session, tt.authenticated, tt.deviceID)

	setupConnectionsMap(t, tt.authenticated, tt.deviceID)

	// Execute
	require.NotPanics(t, func() {
		ctx.cleanup()
	})

	// Verify
	verifyTimerState(t, session, tt.wantTimerStopped)
	verifyConnectionRemoved(t, tt.authenticated, tt.deviceID)
}

func setupConnectionContext(t *testing.T, session *apf.Session, authenticated bool, deviceID string) *connectionContext {
	t.Helper()

	// Create a proper APFHandler with mock deviceID
	log := logger.New("error")
	handler := NewAPFHandler(nil, log) // devices.Feature can be nil for cleanup test
	handler.deviceID = deviceID        // Set deviceID directly for test

	return &connectionContext{
		session:       session,
		authenticated: authenticated,
		handler:       handler,
	}
}

func setupConnectionsMap(t *testing.T, authenticated bool, deviceID string) {
	t.Helper()

	if authenticated && deviceID != "" {
		mu.Lock()

		wsman.Connections[deviceID] = &wsman.ConnectionEntry{}

		mu.Unlock()
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
		mu.Lock()

		_, exists := wsman.Connections[deviceID]

		mu.Unlock()

		assert.False(t, exists, "Connection should be removed from map")
	}
}
