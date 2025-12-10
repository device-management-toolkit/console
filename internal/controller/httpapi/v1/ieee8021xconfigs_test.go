package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/mocks"
	"github.com/device-management-toolkit/console/internal/usecase/ieee8021xconfigs"
	"github.com/device-management-toolkit/console/pkg/logger"
)

func ieee8021xconfigsTest(t *testing.T) (*mocks.MockIEEE8021xConfigsFeature, *gin.Engine) {
	t.Helper()

	mockCtl := gomock.NewController(t)
	defer mockCtl.Finish()

	log := logger.New("error")
	mockIEEE8021xConfigs := mocks.NewMockIEEE8021xConfigsFeature(mockCtl)

	engine := gin.New()
	handler := engine.Group("/api/v1/admin")

	NewIEEE8021xConfigRoutes(handler, mockIEEE8021xConfigs, log)

	return mockIEEE8021xConfigs, engine
}

type testIEEE8021xConfigs struct {
	name         string
	method       string
	url          string
	mock         func(repo *mocks.MockIEEE8021xConfigsFeature)
	response     interface{}
	requestBody  dto.IEEE8021xConfig
	expectedCode int
}

var pxeTime = 120

var ieee8021xconfigTest = dto.IEEE8021xConfig{
	ProfileName:            "newprofile",
	AuthenticationProtocol: 2,
	PXETimeout:             &pxeTime,
	TenantID:               "tenant1",
	Version:                "1.0",
	WiredInterface:         false,
}

func TestIEEE8021xConfigsRoutes(t *testing.T) {
	t.Parallel()

	tests := []testIEEE8021xConfigs{
		{
			name:   "get all ieee8021xconfigs",
			method: http.MethodGet,
			url:    "/api/v1/admin/ieee8021xconfigs",
			mock: func(ieeeConfig *mocks.MockIEEE8021xConfigsFeature) {
				ieeeConfig.EXPECT().Get(context.Background(), 25, 0, "").Return([]dto.IEEE8021xConfig{{
					ProfileName: "profile",
				}}, nil)
			},
			response:     []dto.IEEE8021xConfig{{ProfileName: "profile"}},
			expectedCode: http.StatusOK,
		},
		{
			name:   "get all ieee8021xconfigs - with count",
			method: http.MethodGet,
			url:    "/api/v1/admin/ieee8021xconfigs?$top=10&$skip=1&$count=true",
			mock: func(ieeeConfig *mocks.MockIEEE8021xConfigsFeature) {
				ieeeConfig.EXPECT().Get(context.Background(), 10, 1, "").Return([]dto.IEEE8021xConfig{{
					ProfileName: "profile",
				}}, nil)
				ieeeConfig.EXPECT().GetCount(context.Background(), "").Return(1, nil)
			},
			response:     dto.IEEE8021xConfigCountResponse{Count: 1, Data: []dto.IEEE8021xConfig{{ProfileName: "profile"}}},
			expectedCode: http.StatusOK,
		},
		{
			name:   "get all ieee8021xconfigs - failed",
			method: http.MethodGet,
			url:    "/api/v1/admin/ieee8021xconfigs",
			mock: func(ieeeConfig *mocks.MockIEEE8021xConfigsFeature) {
				ieeeConfig.EXPECT().Get(context.Background(), 25, 0, "").Return(nil, ieee8021xconfigs.ErrDatabase)
			},
			response:     ieee8021xconfigs.ErrDatabase,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:   "get ieee8021xconfig by name",
			method: http.MethodGet,
			url:    "/api/v1/admin/ieee8021xconfigs/profile",
			mock: func(ieeeConfig *mocks.MockIEEE8021xConfigsFeature) {
				ieeeConfig.EXPECT().GetByName(context.Background(), "profile", "").Return(&dto.IEEE8021xConfig{
					ProfileName: "profile",
				}, nil)
			},
			response:     dto.IEEE8021xConfig{ProfileName: "profile"},
			expectedCode: http.StatusOK,
		},
		{
			name:   "get ieee8021xconfig by name - failed",
			method: http.MethodGet,
			url:    "/api/v1/admin/ieee8021xconfigs/profile",
			mock: func(ieeeConfig *mocks.MockIEEE8021xConfigsFeature) {
				ieeeConfig.EXPECT().GetByName(context.Background(), "profile", "").Return(nil, ieee8021xconfigs.ErrDatabase)
			},
			response:     ieee8021xconfigs.ErrDatabase,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:   "insert ieee8021xconfig",
			method: http.MethodPost,
			url:    "/api/v1/admin/ieee8021xconfigs",
			mock: func(ieeeConfig *mocks.MockIEEE8021xConfigsFeature) {
				ieeeConfig.EXPECT().Insert(context.Background(), &ieee8021xconfigTest).Return(&ieee8021xconfigTest, nil)
			},
			response:     ieee8021xconfigTest,
			requestBody:  ieee8021xconfigTest,
			expectedCode: http.StatusCreated,
		},
		{
			name:   "insert ieee8021xconfig - failed",
			method: http.MethodPost,
			url:    "/api/v1/admin/ieee8021xconfigs",
			mock: func(ieeeConfig *mocks.MockIEEE8021xConfigsFeature) {
				ieeeConfig.EXPECT().Insert(context.Background(), &ieee8021xconfigTest).Return(nil, ieee8021xconfigs.ErrDatabase)
			},
			response:     ieee8021xconfigs.ErrDatabase,
			requestBody:  ieee8021xconfigTest,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:   "delete ieee8021xconfig",
			method: http.MethodDelete,
			url:    "/api/v1/admin/ieee8021xconfigs/profile",
			mock: func(ieeeConfig *mocks.MockIEEE8021xConfigsFeature) {
				ieeeConfig.EXPECT().Delete(context.Background(), "profile", "").Return(nil)
			},
			response:     nil,
			expectedCode: http.StatusNoContent,
		},
		{
			name:   "delete ieee8021xconfig - failed",
			method: http.MethodDelete,
			url:    "/api/v1/admin/ieee8021xconfigs/profile",
			mock: func(ieeeConfig *mocks.MockIEEE8021xConfigsFeature) {
				ieeeConfig.EXPECT().Delete(context.Background(), "profile", "").Return(ieee8021xconfigs.ErrDatabase)
			},
			response:     ieee8021xconfigs.ErrDatabase,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:   "update ieee8021xconfig",
			method: http.MethodPatch,
			url:    "/api/v1/admin/ieee8021xconfigs",
			mock: func(ieeeConfig *mocks.MockIEEE8021xConfigsFeature) {
				ieeeConfig.EXPECT().Update(context.Background(), &ieee8021xconfigTest).Return(&ieee8021xconfigTest, nil)
			},
			response:     ieee8021xconfigTest,
			requestBody:  ieee8021xconfigTest,
			expectedCode: http.StatusOK,
		},
		{
			name:   "update ieee8021xconfig - failed",
			method: http.MethodPatch,
			url:    "/api/v1/admin/ieee8021xconfigs",
			mock: func(ieeeConfig *mocks.MockIEEE8021xConfigsFeature) {
				ieeeConfig.EXPECT().Update(context.Background(), &ieee8021xconfigTest).Return(nil, ieee8021xconfigs.ErrDatabase)
			},
			response:     ieee8021xconfigs.ErrDatabase,
			requestBody:  ieee8021xconfigTest,
			expectedCode: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ieee8021xconfigsFeature, engine := ieee8021xconfigsTest(t)

			tc.mock(ieee8021xconfigsFeature)

			var req *http.Request

			var err error

			if tc.requestBody.ProfileName != "" {
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

			if tc.expectedCode == http.StatusOK || tc.expectedCode == http.StatusCreated {
				jsonBytes, _ := json.Marshal(tc.response)
				require.Equal(t, string(jsonBytes), w.Body.String())
			}
		})
	}
}
