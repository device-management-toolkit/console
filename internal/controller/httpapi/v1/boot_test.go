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

func TestGetRemoteEraseCapabilities(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		mock         func(m *mocks.MockDeviceManagementFeature)
		expectedCode int
		response     interface{}
	}{
		{
			name: "getRemoteEraseCapabilities - successful retrieval",
			mock: func(m *mocks.MockDeviceManagementFeature) {
				m.EXPECT().GetRemoteEraseCapabilities(context.Background(), "valid-guid").
					Return(dto.BootCapabilities{SecureEraseAllSSDs: true}, nil)
			},
			expectedCode: http.StatusOK,
			response:     dto.BootCapabilities{SecureEraseAllSSDs: true},
		},
		{
			name: "getRemoteEraseCapabilities - service failure",
			mock: func(m *mocks.MockDeviceManagementFeature) {
				m.EXPECT().GetRemoteEraseCapabilities(context.Background(), "valid-guid").
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

			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/amt/boot/remoteErase/valid-guid", http.NoBody)
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

func TestSetRemoteEraseOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		requestBody  interface{}
		mock         func(m *mocks.MockDeviceManagementFeature)
		expectedCode int
	}{
		{
			name:        "setRemoteEraseOptions - successful",
			requestBody: dto.RemoteEraseRequest{SecureEraseAllSSDs: true, TPMClear: true},
			mock: func(m *mocks.MockDeviceManagementFeature) {
				m.EXPECT().SetRemoteEraseOptions(context.Background(), "valid-guid", dto.RemoteEraseRequest{SecureEraseAllSSDs: true, TPMClear: true}).
					Return(nil)
			},
			expectedCode: http.StatusOK,
		},
		{
			name:        "setRemoteEraseOptions - invalid JSON payload",
			requestBody: "invalid-json",
			mock: func(_ *mocks.MockDeviceManagementFeature) {
			},
			expectedCode: http.StatusInternalServerError,
		},
		{
			name:        "setRemoteEraseOptions - service failure",
			requestBody: dto.RemoteEraseRequest{UnconfigureCSME: true},
			mock: func(m *mocks.MockDeviceManagementFeature) {
				m.EXPECT().SetRemoteEraseOptions(context.Background(), "valid-guid", dto.RemoteEraseRequest{UnconfigureCSME: true}).
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
			req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/amt/boot/remoteErase/valid-guid", bytes.NewBuffer(reqBody))
			require.NoError(t, err)

			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)

			require.Equal(t, tc.expectedCode, w.Code)
		})
	}
}
