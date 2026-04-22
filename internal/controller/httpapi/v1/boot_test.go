package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/mocks"
)

func TestGetBootCapabilities(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		mock         func(m *mocks.MockDeviceManagementFeature)
		expectedCode int
		response     interface{}
	}{
		{
			name: "getBootCapabilities - successful retrieval",
			mock: func(m *mocks.MockDeviceManagementFeature) {
				m.EXPECT().GetBootCapabilities(context.Background(), "valid-guid").
					Return(dto.BootCapabilities{PlatformErase: 1}, nil)
			},
			expectedCode: http.StatusOK,
			response:     dto.BootCapabilities{PlatformErase: 1},
		},
		{
			name: "getBootCapabilities - service failure",
			mock: func(m *mocks.MockDeviceManagementFeature) {
				m.EXPECT().GetBootCapabilities(context.Background(), "valid-guid").
					Return(dto.BootCapabilities{}, ErrGeneral)
			},
			expectedCode: http.StatusInternalServerError,
			response:     nil,
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			deviceManagement, engine := deviceManagementTest(t)
			tc.mock(deviceManagement)

			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/amt/boot/capabilities/valid-guid", http.NoBody)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)

			require.Equal(t, tc.expectedCode, w.Code)

			if tc.expectedCode == http.StatusOK {
				jsonBytes, _ := json.Marshal(tc.response)
				require.Equal(t, string(jsonBytes), w.Body.String())
			}
		})
	}
}

func TestSetRPEEnabled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		requestBody  interface{}
		mock         func(m *mocks.MockDeviceManagementFeature)
		expectedCode int
	}{
		{
			name:        "setRPEEnabled - successful (enabled=true)",
			requestBody: dto.RPERequest{Enabled: true},
			mock: func(m *mocks.MockDeviceManagementFeature) {
				m.EXPECT().SetRPEEnabled(context.Background(), "valid-guid", true).
					Return(nil)
			},
			expectedCode: http.StatusOK,
		},
		{
			name:        "setRPEEnabled - successful (enabled=false)",
			requestBody: dto.RPERequest{Enabled: false},
			mock: func(m *mocks.MockDeviceManagementFeature) {
				m.EXPECT().SetRPEEnabled(context.Background(), "valid-guid", false).
					Return(nil)
			},
			expectedCode: http.StatusOK,
		},
		{
			name:        "setRPEEnabled - invalid JSON payload",
			requestBody: "invalid-json",
			mock: func(_ *mocks.MockDeviceManagementFeature) {
			},
			expectedCode: http.StatusInternalServerError,
		},
		{
			name:        "setRPEEnabled - service failure",
			requestBody: dto.RPERequest{Enabled: true},
			mock: func(m *mocks.MockDeviceManagementFeature) {
				m.EXPECT().SetRPEEnabled(context.Background(), "valid-guid", true).
					Return(ErrGeneral)
			},
			expectedCode: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			deviceManagement, engine := deviceManagementTest(t)
			tc.mock(deviceManagement)

			reqBody, _ := json.Marshal(tc.requestBody)
			req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/amt/boot/rpe/valid-guid", bytes.NewBuffer(reqBody))
			require.NoError(t, err)

			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)

			require.Equal(t, tc.expectedCode, w.Code)
		})
	}
}

func TestSendRemoteErase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		requestBody  interface{}
		mock         func(m *mocks.MockDeviceManagementFeature)
		expectedCode int
	}{
		{
			name:        "sendRemoteErase - successful",
			requestBody: dto.RemoteEraseRequest{EraseMask: 3},
			mock: func(m *mocks.MockDeviceManagementFeature) {
				m.EXPECT().SendRemoteErase(context.Background(), "valid-guid", 3).
					Return(nil)
			},
			expectedCode: http.StatusOK,
		},
		{
			name:        "sendRemoteErase - invalid JSON payload",
			requestBody: "invalid-json",
			mock: func(_ *mocks.MockDeviceManagementFeature) {
			},
			expectedCode: http.StatusInternalServerError,
		},
		{
			name:        "sendRemoteErase - service failure",
			requestBody: dto.RemoteEraseRequest{EraseMask: 1},
			mock: func(m *mocks.MockDeviceManagementFeature) {
				m.EXPECT().SendRemoteErase(context.Background(), "valid-guid", 1).
					Return(ErrGeneral)
			},
			expectedCode: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			deviceManagement, engine := deviceManagementTest(t)
			tc.mock(deviceManagement)

			reqBody, _ := json.Marshal(tc.requestBody)
			req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/amt/remoteErase/valid-guid", bytes.NewBuffer(reqBody))
			require.NoError(t, err)

			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)

			require.Equal(t, tc.expectedCode, w.Code)
		})
	}
}
