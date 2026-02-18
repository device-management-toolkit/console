package v1

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
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
	config.ConsoleConfig.WSCompression = false // Disable compression for predictable test behavior
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
			// Expect initial websocket request log
			mockLogger.EXPECT().Info("Websocket connection request: host=%s, mode=%s, client=%s", "someHost", "someMode", gomock.Any())

			if tc.upgraderError != nil {
				mockUpgrader.EXPECT().
					Upgrade(gomock.Any(), gomock.Any(), nil).
					Return(nil, tc.upgraderError)
				mockLogger.EXPECT().Debug("failed to cast Upgrader to *websocket.Upgrader")
				mockLogger.EXPECT().Error(tc.upgraderError, "Websocket upgrade failed (host=%s, mode=%s)", "someHost", "someMode")
			} else {
				mockUpgrader.EXPECT().
					Upgrade(gomock.Any(), gomock.Any(), nil).
					Return(&websocket.Conn{}, nil)

				mockLogger.EXPECT().Debug("failed to cast Upgrader to *websocket.Upgrader")
				mockLogger.EXPECT().Debug("Websocket compression disabled (host=%s, mode=%s)", "someHost", "someMode")
				mockLogger.EXPECT().Info("Websocket connection opened successfully (host=%s, mode=%s)", "someHost", "someMode")

				if tc.redirectError != nil {
					mockLogger.EXPECT().Error(tc.redirectError, "Redirect failed (host=%s, mode=%s)", "someHost", "someMode")
				} else {
					mockLogger.EXPECT().Info("Websocket connection closed normally (host=%s, mode=%s)", "someHost", "someMode")
				}

				mockFeature.EXPECT().
					Redirect(gomock.Any(), gomock.Any(), "someHost", "someMode").
					Return(tc.redirectError)
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
