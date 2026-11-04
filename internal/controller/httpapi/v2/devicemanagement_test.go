package v2

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	dtov2 "github.com/device-management-toolkit/console/internal/entity/dto/v2"
	"github.com/device-management-toolkit/console/internal/mocks"
	"github.com/device-management-toolkit/console/pkg/logger"
)

var ErrGeneral = errors.New("general error")

func deviceManagementTest(t *testing.T) (*mocks.MockDeviceManagementFeature, *gin.Engine) {
	t.Helper()

	mockCtl := gomock.NewController(t)
	defer mockCtl.Finish()

	log := logger.New("error")
	deviceManagement := mocks.NewMockDeviceManagementFeature(mockCtl)

	engine := gin.New()
	handler := engine.Group("/api/v2")

	NewAmtRoutes(handler, deviceManagement, log)

	return deviceManagement, engine
}

func TestDeviceManagement(t *testing.T) {
	t.Parallel()

	featuresInput := dto.Features{
		UserConsent: "kvm",
		EnableSOL:   true,
		EnableIDER:  true,
		EnableKVM:   true,
		Redirection: true,
	}

	featuresOutput := dtov2.Features{
		UserConsent:  "kvm",
		EnableSOL:    true,
		EnableIDER:   true,
		EnableKVM:    true,
		Redirection:  true,
		KVMAvailable: true,
	}

	tests := []struct {
		name         string
		url          string
		method       string
		mock         func(m *mocks.MockDeviceManagementFeature)
		expectedCode int
		requestBody  interface{}
		response     interface{}
	}{
		{
			name:   "getRemoteEraseStatus - BIOS authentication failure",
			url:    "/api/v2/amt/boot/remoteErase/valid-guid/status",
			method: http.MethodGet,
			mock: func(m *mocks.MockDeviceManagementFeature) {
				m.EXPECT().GetRemoteEraseStatus(context.Background(), "valid-guid", "").
					Return(dtov2.RemoteEraseStatus{Status: "failed", Message: "Drive password/authentication failure"}, nil)
			},
			expectedCode: http.StatusOK,
			response:     dtov2.RemoteEraseStatus{Status: "failed", Message: "Drive password/authentication failure"},
		},
		{
			name:   "getRemoteEraseStatus - unavailable",
			url:    "/api/v2/amt/boot/remoteErase/valid-guid/status",
			method: http.MethodGet,
			mock: func(m *mocks.MockDeviceManagementFeature) {
				m.EXPECT().GetRemoteEraseStatus(context.Background(), "valid-guid", "").
					Return(dtov2.RemoteEraseStatus{}, ErrGeneral)
			},
			expectedCode: http.StatusInternalServerError,
		},
		{
			name:   "getVersion - successful retrieval",
			url:    "/api/v2/amt/version/valid-guid",
			method: http.MethodGet,
			mock: func(m *mocks.MockDeviceManagementFeature) {
				m.EXPECT().GetVersion(context.Background(), "valid-guid", "").
					Return(dto.Version{}, dtov2.Version{}, nil)
			},
			expectedCode: http.StatusOK,
			response:     dtov2.Version{},
		},
		{
			name:   "getVersion - retrieval failure",
			url:    "/api/v2/amt/version/valid-guid",
			method: http.MethodGet,
			mock: func(m *mocks.MockDeviceManagementFeature) {
				m.EXPECT().GetVersion(context.Background(), "valid-guid", "").
					Return(dto.Version{}, dtov2.Version{}, ErrGeneral)
			},
			expectedCode: http.StatusInternalServerError,
			response:     gin.H{"error": "general error", "message": "general error"},
		},
		{
			name:   "getFeatures - successful retrieval",
			url:    "/api/v2/amt/features/valid-guid",
			method: http.MethodGet,
			mock: func(m *mocks.MockDeviceManagementFeature) {
				m.EXPECT().GetFeatures(context.Background(), "valid-guid", "").
					Return(dto.Features{}, dtov2.Features{}, nil)
			},
			expectedCode: http.StatusOK,
			response:     dtov2.Features{},
		},
		{
			name:   "getFeatures - retrieval failure",
			url:    "/api/v2/amt/features/valid-guid",
			method: http.MethodGet,
			mock: func(m *mocks.MockDeviceManagementFeature) {
				m.EXPECT().GetFeatures(context.Background(), "valid-guid", "").
					Return(dto.Features{}, dtov2.Features{}, ErrGeneral)
			},
			expectedCode: http.StatusInternalServerError,
			response:     gin.H{"error": "general error", "message": "general error"},
		},
		{
			name:         "setFeatures - invalid request body",
			url:          "/api/v2/amt/features/valid-guid",
			method:       http.MethodPost,
			mock:         func(_ *mocks.MockDeviceManagementFeature) {},
			expectedCode: http.StatusBadRequest,
			requestBody:  []string{"invalid"},
			response: gin.H{
				"error":   "json: cannot unmarshal array into Go value of type dto.Features",
				"message": "json: cannot unmarshal array into Go value of type dto.Features",
			},
		},
		{
			name:   "setFeatures - update failure",
			url:    "/api/v2/amt/features/valid-guid",
			method: http.MethodPost,
			mock: func(m *mocks.MockDeviceManagementFeature) {
				m.EXPECT().SetFeatures(context.Background(), "valid-guid", "", featuresInput).
					Return(dto.Features{}, dtov2.Features{}, ErrGeneral)
			},
			expectedCode: http.StatusInternalServerError,
			requestBody:  featuresInput,
			response:     gin.H{"error": "general error", "message": "general error"},
		},
		{
			name:   "setFeatures - successful post",
			url:    "/api/v2/amt/features/valid-guid",
			method: http.MethodPost,
			mock: func(m *mocks.MockDeviceManagementFeature) {
				m.EXPECT().SetFeatures(context.Background(), "valid-guid", "", featuresInput).
					Return(dto.Features{}, featuresOutput, nil)
			},
			expectedCode: http.StatusOK,
			requestBody:  featuresInput,
			response:     featuresOutput,
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			deviceManagement, engine := deviceManagementTest(t)

			tc.mock(deviceManagement)

			var req *http.Request

			var err error

			if tc.method == http.MethodPost || tc.method == http.MethodPatch || tc.method == http.MethodDelete {
				reqBody, _ := json.Marshal(tc.requestBody)
				req, err = http.NewRequestWithContext(context.Background(), tc.method, tc.url, bytes.NewBuffer(reqBody))
			} else {
				req, err = http.NewRequestWithContext(context.Background(), tc.method, tc.url, http.NoBody)
			}

			if err != nil {
				t.Fatalf("Couldn't create request: %v\n", err)
			}

			w := httptest.NewRecorder()

			engine.ServeHTTP(w, req)

			require.Equal(t, tc.expectedCode, w.Code)

			if tc.response != nil {
				jsonBytes, _ := json.Marshal(tc.response)
				require.Equal(t, string(jsonBytes), w.Body.String())
			}
		})
	}
}
