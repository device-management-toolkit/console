package v1

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/mocks"
	"github.com/device-management-toolkit/console/internal/usecase/devices"
	"github.com/device-management-toolkit/console/internal/usecase/devices/wsman"
	"github.com/device-management-toolkit/console/pkg/logger"
)

var errProfileSync = errors.New("profile sync failure")

func TestWiFiProfileSyncHandlers(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	const path = "/api/v1/amt/networkSettings/wireless/profileSync/my-guid"

	tests := []struct {
		name       string
		method     string
		body       string
		setupMock  func(devMock *mocks.MockDeviceManagementFeature)
		wantStatus int
		wantBody   string
	}{
		{
			name:   "get wireless profile sync",
			method: http.MethodGet,
			setupMock: func(devMock *mocks.MockDeviceManagementFeature) {
				devMock.EXPECT().
					GetWirelessProfileSync(gomock.Any(), "my-guid").
					Return(dto.WirelessProfileSyncResponse{LocalProfileSync: true, UEFIProfileSync: false, UEFIProfileSyncSupported: true}, nil)
			},
			wantStatus: http.StatusOK,
			wantBody:   `{"localProfileSync":true,"uefiProfileSync":false,"uefiProfileSyncSupported":true}`,
		},
		{
			name:   "get wireless profile sync - usecase error",
			method: http.MethodGet,
			setupMock: func(devMock *mocks.MockDeviceManagementFeature) {
				devMock.EXPECT().
					GetWirelessProfileSync(gomock.Any(), "my-guid").
					Return(dto.WirelessProfileSyncResponse{}, errProfileSync)
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:   "get wireless profile sync - no wifi port",
			method: http.MethodGet,
			setupMock: func(devMock *mocks.MockDeviceManagementFeature) {
				devMock.EXPECT().
					GetWirelessProfileSync(gomock.Any(), "my-guid").
					Return(dto.WirelessProfileSyncResponse{}, wsman.ErrNoWiFiPort)
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:   "set wireless profile sync",
			method: http.MethodPost,
			body:   `{"localProfileSync":false,"uefiProfileSync":true}`,
			setupMock: func(devMock *mocks.MockDeviceManagementFeature) {
				devMock.EXPECT().
					SetWirelessProfileSync(gomock.Any(), "my-guid", gomock.Any()).
					Return(dto.WirelessProfileSyncResponse{LocalProfileSync: false, UEFIProfileSync: true, UEFIProfileSyncSupported: true}, nil)
			},
			wantStatus: http.StatusOK,
			wantBody:   `{"localProfileSync":false,"uefiProfileSync":true,"uefiProfileSyncSupported":true}`,
		},
		{
			name:       "set wireless profile sync - invalid payload",
			method:     http.MethodPost,
			body:       `{"localProfileSync":"not-a-bool"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:   "set wireless profile sync - usecase error",
			method: http.MethodPost,
			body:   `{"localProfileSync":true}`,
			setupMock: func(devMock *mocks.MockDeviceManagementFeature) {
				devMock.EXPECT().
					SetWirelessProfileSync(gomock.Any(), "my-guid", gomock.Any()).
					Return(dto.WirelessProfileSyncResponse{}, errProfileSync)
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:   "set wireless profile sync - no wifi port",
			method: http.MethodPost,
			body:   `{"localProfileSync":true}`,
			setupMock: func(devMock *mocks.MockDeviceManagementFeature) {
				devMock.EXPECT().
					SetWirelessProfileSync(gomock.Any(), "my-guid", gomock.Any()).
					Return(dto.WirelessProfileSyncResponse{}, wsman.ErrNoWiFiPort)
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:   "set wireless profile sync - uefi unsupported conflict",
			method: http.MethodPost,
			body:   `{"uefiProfileSync":true}`,
			setupMock: func(devMock *mocks.MockDeviceManagementFeature) {
				devMock.EXPECT().
					SetWirelessProfileSync(gomock.Any(), "my-guid", gomock.Any()).
					Return(dto.WirelessProfileSyncResponse{}, devices.ErrUEFIProfileSyncNotSupported)
			},
			wantStatus: http.StatusConflict,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockCtl := gomock.NewController(t)
			defer mockCtl.Finish()

			devMock := mocks.NewMockDeviceManagementFeature(mockCtl)
			if tc.setupMock != nil {
				tc.setupMock(devMock)
			}

			engine := gin.New()
			handler := engine.Group("/api/v1")
			NewAmtRoutes(handler, devMock, nil, nil, logger.New("error"))

			reqBody := io.Reader(http.NoBody)
			if tc.body != "" {
				reqBody = bytes.NewBufferString(tc.body)
			}

			req := httptest.NewRequest(tc.method, path, reqBody)
			if tc.method == http.MethodPost {
				req.Header.Set("Content-Type", "application/json")
			}

			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)

			require.Equal(t, tc.wantStatus, w.Code)

			if tc.wantBody != "" {
				require.JSONEq(t, tc.wantBody, w.Body.String())
			}
		})
	}
}
