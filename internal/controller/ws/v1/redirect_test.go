package v1

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
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
	mockFeature := mocks.NewMockFeature(ctrl)
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
		mockFeature := mocks.NewMockFeature(ctrl)
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
		mockFeature := mocks.NewMockFeature(ctrl)
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
		mockFeature := mocks.NewMockFeature(ctrl)
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
		mockFeature := mocks.NewMockFeature(ctrl)
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
		mockFeature := mocks.NewMockFeature(ctrl)
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
			mockFeature := mocks.NewMockFeature(ctrl)
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
