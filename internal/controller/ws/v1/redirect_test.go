package v1

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/internal/mocks"
)

var (
	ErrUpgrade  = errors.New("upgrade error")
	ErrRedirect = errors.New("redirection error")
)

func TestWebSocketHandler(t *testing.T) { //nolint:paralleltest // logging library is not thread-safe for tests
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	_, _ = config.NewConfig()

	config.ConsoleConfig.Disabled = true
	mockFeature := mocks.NewMockDeviceManagementFeature(ctrl)
	mockUpgrader := mocks.NewMockUpgrader(ctrl)
	mockLogger := mocks.NewMockLogger(ctrl)

	tests := []struct {
		name           string
		upgraderError  error
		redirectError  error
		expectedStatus int
	}{
		{
			name:           "Success case",
			upgraderError:  nil,
			redirectError:  nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Upgrade error",
			upgraderError:  ErrUpgrade,
			redirectError:  nil,
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "Redirect error",
			upgraderError:  nil,
			redirectError:  ErrRedirect,
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests { //nolint:paralleltest // logging library is not thread-safe for tests
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if tc.upgraderError != nil {
				mockUpgrader.EXPECT().
					Upgrade(gomock.Any(), gomock.Any(), nil).
					Return(nil, tc.upgraderError)
				mockLogger.EXPECT().Debug("failed to cast Upgrader to *websocket.Upgrader")
				mockLogger.EXPECT().Debug("KVM_TIMING: WebSocket upgrade", "duration_ms", gomock.Any())
			} else {
				mockUpgrader.EXPECT().
					Upgrade(gomock.Any(), gomock.Any(), nil).
					Return(&websocket.Conn{}, nil)

				mockLogger.EXPECT().Debug("failed to cast Upgrader to *websocket.Upgrader")
				mockLogger.EXPECT().Debug("KVM_TIMING: WebSocket upgrade", "duration_ms", gomock.Any())
				mockLogger.EXPECT().Info("Websocket connection opened")

				mockFeature.EXPECT().
					Redirect(gomock.Any(), gomock.Any(), "someHost", "someMode").
					Return(tc.redirectError)

				// Total connection time is always logged after Redirect completes
				mockLogger.EXPECT().Debug("KVM_TIMING: Total connection time", "duration_ms", gomock.Any(), "mode", "someMode")

				if tc.redirectError != nil {
					mockLogger.EXPECT().Error(tc.redirectError, "http - devices - v1 - redirect")
				}
			}

			r := gin.Default()
			RegisterRoutes(r, mockLogger, mockFeature, mockUpgrader)

			req := httptest.NewRequest(http.MethodGet, "/relay/webrelay.ashx?host=someHost&mode=someMode", http.NoBody)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)
		})
	}
}

// TestWebSocketHandlerDeviceBinding: WS accepts only a token whose deviceId matches host.
func TestWebSocketHandlerDeviceBinding(t *testing.T) { //nolint:paralleltest // logging library is not thread-safe for tests
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	_, _ = config.NewConfig()

	config.ConsoleConfig.Disabled = false
	config.ConsoleConfig.JWTKey = "test-jwt-key"

	// deviceID == "" mimics a login token (no deviceId claim).
	tokenFor := func(deviceID string) string {
		claims := jwt.MapClaims{
			"exp": time.Now().Add(5 * time.Minute).Unix(),
		}
		if deviceID != "" {
			claims["deviceId"] = deviceID
		}

		s, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(config.ConsoleConfig.JWTKey))

		return s
	}

	t.Run("rejects token whose deviceId does not match host", func(t *testing.T) { //nolint:paralleltest // shared logger
		mockFeature := mocks.NewMockDeviceManagementFeature(ctrl)
		mockUpgrader := mocks.NewMockUpgrader(ctrl)
		mockLogger := mocks.NewMockLogger(ctrl)
		mockLogger.EXPECT().Warn("redirection token not authorized for requested device", "host", "deviceB")

		r := gin.Default()
		RegisterRoutes(r, mockLogger, mockFeature, mockUpgrader)

		req := httptest.NewRequest(http.MethodGet, "/relay/webrelay.ashx?host=deviceB&mode=kvm", http.NoBody)
		req.Header.Set("Sec-Websocket-Protocol", tokenFor("deviceA"))

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("rejects login token with no deviceId", func(t *testing.T) { //nolint:paralleltest // shared logger
		mockFeature := mocks.NewMockDeviceManagementFeature(ctrl)
		mockUpgrader := mocks.NewMockUpgrader(ctrl)
		mockLogger := mocks.NewMockLogger(ctrl)
		mockLogger.EXPECT().Warn("redirection token not authorized for requested device", "host", "deviceA")

		r := gin.Default()
		RegisterRoutes(r, mockLogger, mockFeature, mockUpgrader)

		req := httptest.NewRequest(http.MethodGet, "/relay/webrelay.ashx?host=deviceA&mode=kvm", http.NoBody)
		req.Header.Set("Sec-Websocket-Protocol", tokenFor("")) // no deviceId == login token

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("rejects login token when host is also empty", func(t *testing.T) { //nolint:paralleltest // shared logger
		mockFeature := mocks.NewMockDeviceManagementFeature(ctrl)
		mockUpgrader := mocks.NewMockUpgrader(ctrl)
		mockLogger := mocks.NewMockLogger(ctrl)
		mockLogger.EXPECT().Warn("redirection token not authorized for requested device", "host", "")

		r := gin.Default()
		RegisterRoutes(r, mockLogger, mockFeature, mockUpgrader)

		req := httptest.NewRequest(http.MethodGet, "/relay/webrelay.ashx?mode=kvm", http.NoBody)
		req.Header.Set("Sec-Websocket-Protocol", tokenFor("")) // no deviceId, no host

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("accepts token whose deviceId matches host", func(t *testing.T) { //nolint:paralleltest // shared logger
		mockFeature := mocks.NewMockDeviceManagementFeature(ctrl)
		mockUpgrader := mocks.NewMockUpgrader(ctrl)
		mockLogger := mocks.NewMockLogger(ctrl)

		mockUpgrader.EXPECT().Upgrade(gomock.Any(), gomock.Any(), nil).Return(&websocket.Conn{}, nil)
		mockLogger.EXPECT().Debug("failed to cast Upgrader to *websocket.Upgrader")
		mockLogger.EXPECT().Debug("KVM_TIMING: WebSocket upgrade", "duration_ms", gomock.Any())
		mockLogger.EXPECT().Info("Websocket connection opened")
		mockFeature.EXPECT().Redirect(gomock.Any(), gomock.Any(), "deviceA", "kvm").Return(nil)
		mockLogger.EXPECT().Debug("KVM_TIMING: Total connection time", "duration_ms", gomock.Any(), "mode", "kvm")

		r := gin.Default()
		RegisterRoutes(r, mockLogger, mockFeature, mockUpgrader)

		req := httptest.NewRequest(http.MethodGet, "/relay/webrelay.ashx?host=deviceA&mode=kvm", http.NoBody)
		req.Header.Set("Sec-Websocket-Protocol", tokenFor("deviceA"))

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("accepts token whose deviceId matches host in different case", func(t *testing.T) { //nolint:paralleltest // shared logger
		mockFeature := mocks.NewMockDeviceManagementFeature(ctrl)
		mockUpgrader := mocks.NewMockUpgrader(ctrl)
		mockLogger := mocks.NewMockLogger(ctrl)

		mockUpgrader.EXPECT().Upgrade(gomock.Any(), gomock.Any(), nil).Return(&websocket.Conn{}, nil)
		mockLogger.EXPECT().Debug("failed to cast Upgrader to *websocket.Upgrader")
		mockLogger.EXPECT().Debug("KVM_TIMING: WebSocket upgrade", "duration_ms", gomock.Any())
		mockLogger.EXPECT().Info("Websocket connection opened")
		mockFeature.EXPECT().Redirect(gomock.Any(), gomock.Any(), "DeviceA", "kvm").Return(nil)
		mockLogger.EXPECT().Debug("KVM_TIMING: Total connection time", "duration_ms", gomock.Any(), "mode", "kvm")

		r := gin.Default()
		RegisterRoutes(r, mockLogger, mockFeature, mockUpgrader)

		req := httptest.NewRequest(http.MethodGet, "/relay/webrelay.ashx?host=DeviceA&mode=kvm", http.NoBody)
		req.Header.Set("Sec-Websocket-Protocol", tokenFor("devicea"))

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// TestWebSocketHandlerRealUpgrader exercises the *websocket.Upgrader branch the
// mock-based tests above never reach, over a real TCP handshake. A single
// upgrader is shared by every relay request, so the handler must set
// Subprotocols on a per-request copy: mutating the shared one in place is a
// data race (caught by -race in the concurrent subtest) and lets one handshake
// negotiate against another's token, which yields no subprotocol at all and is
// rejected by browsers.
func TestWebSocketHandlerRealUpgrader(t *testing.T) { //nolint:paralleltest // shared config and logger
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	_, _ = config.NewConfig()

	config.ConsoleConfig.Disabled = false
	config.ConsoleConfig.JWTKey = "test-jwt-key"

	tokenFor := func(deviceID string) string {
		claims := jwt.MapClaims{
			"exp":      time.Now().Add(5 * time.Minute).Unix(),
			"deviceId": deviceID,
		}

		s, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(config.ConsoleConfig.JWTKey))

		return s
	}

	mockLogger := mocks.NewMockLogger(ctrl)
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()

	// The handshake response is already flushed by the time Redirect runs, so
	// closing here just releases the hijacked connection.
	mockFeature := mocks.NewMockDeviceManagementFeature(ctrl)
	mockFeature.EXPECT().
		Redirect(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ *gin.Context, conn *websocket.Conn, _, _ string) error {
			return conn.Close()
		}).
		AnyTimes()

	// Shared by every handshake, exactly as app.setupHTTPHandler wires it.
	upgrader := &websocket.Upgrader{ReadBufferSize: 1024, WriteBufferSize: 1024}

	r := gin.New()
	RegisterRoutes(r, mockLogger, mockFeature, upgrader)

	srv := httptest.NewUnstartedServer(r)
	// Writing the gin response onto a hijacked connection is expected and noisy.
	srv.Config.ErrorLog = log.New(io.Discard, "", 0)
	srv.Start()

	t.Cleanup(srv.Close)

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")

	// dial performs a real handshake as deviceID does, returning the subprotocol
	// the server negotiated and the token the client offered.
	dial := func(deviceID string) (negotiated, offered string, err error) {
		offered = tokenFor(deviceID)
		dialer := &websocket.Dialer{Subprotocols: []string{offered}}

		conn, resp, err := dialer.Dial(wsURL+"/relay/webrelay.ashx?host="+deviceID+"&mode=kvm", nil)
		if err != nil {
			return "", offered, err
		}

		defer conn.Close()

		if resp != nil {
			defer resp.Body.Close()
		}

		return conn.Subprotocol(), offered, nil
	}

	t.Run("negotiates the caller's token as the subprotocol", func(t *testing.T) { //nolint:paralleltest // shared server
		negotiated, offered, err := dial("deviceA")

		require.NoError(t, err)
		assert.Equal(t, offered, negotiated)
	})

	t.Run("leaves the shared upgrader untouched", func(t *testing.T) { //nolint:paralleltest // shared server
		_, _, err := dial("deviceB")

		require.NoError(t, err)
		assert.Nil(t, upgrader.Subprotocols, "handshake must not mutate the shared upgrader")
	})

	// The regression case: run under -race, concurrent in-place mutation of the
	// shared upgrader is reported, and handshakes negotiate each other's tokens.
	t.Run("concurrent handshakes negotiate their own subprotocol", func(t *testing.T) { //nolint:paralleltest // shared server
		const handshakes = 16

		var wg sync.WaitGroup

		results := make([]struct {
			negotiated, offered string
			err                 error
		}, handshakes)

		for i := range results {
			wg.Add(1)

			go func(i int) {
				defer wg.Done()

				results[i].negotiated, results[i].offered, results[i].err = dial(fmt.Sprintf("device-%d", i))
			}(i)
		}

		wg.Wait()

		for i, res := range results {
			require.NoErrorf(t, res.err, "handshake %d failed", i)
			assert.Equalf(t, res.offered, res.negotiated, "handshake %d negotiated another caller's subprotocol", i)
		}
	})
}

// TestWebSocketHandlerTokenValidation: WS rejects missing and unverifiable tokens.
func TestWebSocketHandlerTokenValidation(t *testing.T) { //nolint:paralleltest // logging library is not thread-safe for tests
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	_, _ = config.NewConfig()

	config.ConsoleConfig.Disabled = false
	config.ConsoleConfig.JWTKey = "test-jwt-key"

	signedWith := func(key string, expiry time.Time) string {
		claims := jwt.MapClaims{
			"exp":      expiry.Unix(),
			"deviceId": "deviceA",
		}

		s, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(key))

		return s
	}

	tests := []struct {
		name  string
		token string
	}{
		{
			name:  "rejects missing token",
			token: "",
		},
		{
			name:  "rejects malformed token",
			token: "not-a-jwt",
		},
		{
			name:  "rejects token signed with the wrong key",
			token: signedWith("wrong-jwt-key", time.Now().Add(5*time.Minute)),
		},
		{
			name:  "rejects expired token",
			token: signedWith(config.ConsoleConfig.JWTKey, time.Now().Add(-1*time.Minute)),
		},
	}

	for _, tc := range tests { //nolint:paralleltest // logging library is not thread-safe for tests
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			mockFeature := mocks.NewMockDeviceManagementFeature(ctrl)
			mockUpgrader := mocks.NewMockUpgrader(ctrl)
			mockLogger := mocks.NewMockLogger(ctrl)

			r := gin.Default()
			RegisterRoutes(r, mockLogger, mockFeature, mockUpgrader)

			req := httptest.NewRequest(http.MethodGet, "/relay/webrelay.ashx?host=deviceA&mode=kvm", http.NoBody)
			if tc.token != "" {
				req.Header.Set("Sec-Websocket-Protocol", tc.token)
			}

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}
}
