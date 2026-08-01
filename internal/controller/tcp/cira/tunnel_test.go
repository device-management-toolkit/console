package cira

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/apf"

	"github.com/device-management-toolkit/console/internal/mocks"
	"github.com/device-management-toolkit/console/internal/usecase/devices"
	"github.com/device-management-toolkit/console/internal/usecase/devices/wsman"
	"github.com/device-management-toolkit/console/pkg/logger"
)

func writeTestCertPair(t *testing.T) (certFile, keyFile string) {
	t.Helper()

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "cira-test"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	require.NoError(t, err)

	keyBytes, err := x509.MarshalECPrivateKey(priv)
	require.NoError(t, err)

	dir := t.TempDir()
	certFile = filepath.Join(dir, "cert.pem")
	keyFile = filepath.Join(dir, "key.pem")

	require.NoError(t, os.WriteFile(certFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600))
	require.NoError(t, os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes}), 0o600))

	return certFile, keyFile
}

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

// fakeConn is a minimal net.Conn implementation for tests.
type fakeConn struct{ net.Conn }

func TestServer_cipherSuites(t *testing.T) {
	t.Parallel()

	secureIDs := make([]uint16, 0, len(tls.CipherSuites()))
	for _, suite := range tls.CipherSuites() {
		secureIDs = append(secureIDs, suite.ID)
	}

	t.Run("insecure suites are not advertised by default", func(t *testing.T) {
		t.Parallel()

		s := &Server{log: logger.New("error")}

		suites := s.cipherSuites()

		assert.Equal(t, secureIDs, suites)

		for _, insecure := range insecureCipherSuites {
			assert.NotContains(t, suites, insecure, "insecure suite must be opt-in")
		}
	})

	t.Run("insecure suites are advertised when explicitly allowed", func(t *testing.T) {
		t.Parallel()

		s := &Server{log: logger.New("error"), allowInsecureCiphers: true}

		suites := s.cipherSuites()

		want := make([]uint16, 0, len(secureIDs)+len(insecureCipherSuites))
		want = append(want, secureIDs...)
		want = append(want, insecureCipherSuites...)

		assert.Equal(t, want, suites)
	})

	t.Run("every opt-in suite is one Go considers insecure", func(t *testing.T) {
		t.Parallel()

		goInsecure := make([]uint16, 0, len(tls.InsecureCipherSuites()))
		for _, suite := range tls.InsecureCipherSuites() {
			goInsecure = append(goInsecure, suite.ID)
		}

		for _, suite := range insecureCipherSuites {
			assert.Contains(t, goInsecure, suite)
		}
	})
}

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

func TestNewServer(t *testing.T) {
	t.Parallel()

	t.Run("returns an error for an invalid cert path", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		mockDevices := mocks.NewMockDeviceManagementFeature(ctrl)

		var devicesFeature devices.Feature = mockDevices

		s, err := NewServer("does-not-exist-cert.pem", "does-not-exist-key.pem", false, devicesFeature, logger.New("error"))

		require.Error(t, err)
		assert.Nil(t, s)
	})

	//nolint:paralleltest // binds the real :4433 port; must not race other subtests for it
	t.Run("starts listening and shuts down cleanly", func(t *testing.T) {
		certFile, keyFile := writeTestCertPair(t)

		ctrl := gomock.NewController(t)
		mockDevices := mocks.NewMockDeviceManagementFeature(ctrl)

		var devicesFeature devices.Feature = mockDevices

		s, err := NewServer(certFile, keyFile, true, devicesFeature, logger.New("error"))

		require.NoError(t, err)
		require.NotNil(t, s)

		require.Eventually(t, func() bool {
			conn, dialErr := net.Dial("tcp", "127.0.0.1:"+port)
			if dialErr != nil {
				return false
			}

			_ = conn.Close()

			return true
		}, time.Second, 5*time.Millisecond, "CIRA server never started listening on port %s", port)

		require.NoError(t, s.Shutdown())
	})
}
