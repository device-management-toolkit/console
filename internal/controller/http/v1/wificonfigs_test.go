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
	"github.com/device-management-toolkit/console/internal/usecase/wificonfigs"
	"github.com/device-management-toolkit/console/pkg/logger"
)

func wifiTest(t *testing.T) (*mocks.MockWiFiConfigsFeature, *gin.Engine) {
	t.Helper()

	mockCtl := gomock.NewController(t)
	defer mockCtl.Finish()

	log := logger.New("error")
	wificonfig := mocks.NewMockWiFiConfigsFeature(mockCtl)

	engine := gin.New()
	handler := engine.Group("/api/v1/admin")

	NewWirelessConfigRoutes(handler, wificonfig, log)

	return wificonfig, engine
}

type wifiConfigTest struct {
	name         string
	method       string
	url          string
	mock         func(repo *mocks.MockWiFiConfigsFeature)
	response     interface{}
	requestBody  dto.WirelessConfig
	expectedCode int
}

var (
	requestWiFiConfig  = dto.WirelessConfig{AuthenticationMethod: 4, EncryptionMethod: 3, SSID: "exampleSSID", PSKValue: 12345, PSKPassphrase: "examplepassphrase", ProfileName: "newprofile", LinkPolicy: []int{1, 2, 3}, TenantID: "tenant1", Version: "1.0"}
	responseWiFiConfig = dto.WirelessConfig{AuthenticationMethod: 4, EncryptionMethod: 3, SSID: "exampleSSID", PSKValue: 12345, PSKPassphrase: "examplepassphrase", ProfileName: "newprofile", LinkPolicy: []int{1, 2, 3}, TenantID: "tenant1", Version: "1.0"}
)

func TestWiFiConfigRoutes(t *testing.T) {
	t.Parallel()

	tests := []wifiConfigTest{
		{
			name:   "get all wificonfigs",
			method: http.MethodGet,
			url:    "/api/v1/admin/wirelessconfigs",
			mock: func(wificonfig *mocks.MockWiFiConfigsFeature) {
				wificonfig.EXPECT().Get(context.Background(), 25, 0, "").Return([]dto.WirelessConfig{{
					ProfileName: "profile",
				}}, nil)
			},
			response:     []dto.WirelessConfig{{ProfileName: "profile"}},
			expectedCode: http.StatusOK,
		},
		{
			name:   "get all wificonfigs - with count",
			method: http.MethodGet,
			url:    "/api/v1/admin/wirelessconfigs?$top=10&$skip=1&$count=true",
			mock: func(wificonfig *mocks.MockWiFiConfigsFeature) {
				wificonfig.EXPECT().Get(context.Background(), 10, 1, "").Return([]dto.WirelessConfig{{
					ProfileName: "profile",
				}}, nil)
				wificonfig.EXPECT().GetCount(context.Background(), "").Return(1, nil)
			},
			response:     dto.WirelessConfigCountResponse{Count: 1, Data: []dto.WirelessConfig{{ProfileName: "profile"}}},
			expectedCode: http.StatusOK,
		},
		{
			name:   "get all wifi - failed",
			method: http.MethodGet,
			url:    "/api/v1/admin/wirelessconfigs",
			mock: func(wificonfig *mocks.MockWiFiConfigsFeature) {
				wificonfig.EXPECT().Get(context.Background(), 25, 0, "").Return(nil, wificonfigs.ErrDatabase)
			},
			response:     wificonfigs.ErrDatabase,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:   "get wificonfig by name",
			method: http.MethodGet,
			url:    "/api/v1/admin/wirelessconfigs/profile",
			mock: func(wificonfig *mocks.MockWiFiConfigsFeature) {
				wificonfig.EXPECT().GetByName(context.Background(), "profile", "").Return(&dto.WirelessConfig{
					ProfileName: "profile",
				}, nil)
			},
			response:     dto.WirelessConfig{ProfileName: "profile"},
			expectedCode: http.StatusOK,
		},
		{
			name:   "get wificonfig by name - failed",
			method: http.MethodGet,
			url:    "/api/v1/admin/wirelessconfigs/profile",
			mock: func(wificonfig *mocks.MockWiFiConfigsFeature) {
				wificonfig.EXPECT().GetByName(context.Background(), "profile", "").Return(nil, wificonfigs.ErrDatabase)
			},
			response:     wificonfigs.ErrDatabase,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:   "insert wificonfig",
			method: http.MethodPost,
			url:    "/api/v1/admin/wirelessconfigs",
			mock: func(wificonfig *mocks.MockWiFiConfigsFeature) {
				wificonfigTest := &dto.WirelessConfig{
					AuthenticationMethod: 4,
					EncryptionMethod:     3,
					SSID:                 "exampleSSID",
					PSKValue:             12345,
					PSKPassphrase:        "examplepassphrase",
					ProfileName:          "newprofile",
					LinkPolicy:           []int{1, 2, 3},
					TenantID:             "tenant1",
					Version:              "1.0",
				}
				wificonfig.EXPECT().Insert(context.Background(), wificonfigTest).Return(wificonfigTest, nil)
			},
			response:     responseWiFiConfig,
			requestBody:  requestWiFiConfig,
			expectedCode: http.StatusCreated,
		},
		{
			name:   "insert wificonfig - failed",
			method: http.MethodPost,
			url:    "/api/v1/admin/wirelessconfigs",
			mock: func(wificonfig *mocks.MockWiFiConfigsFeature) {
				wificonfigTest := &dto.WirelessConfig{
					AuthenticationMethod: 4,
					EncryptionMethod:     3,
					SSID:                 "exampleSSID",
					PSKValue:             12345,
					PSKPassphrase:        "examplepassphrase",
					ProfileName:          "newprofile",
					LinkPolicy:           []int{1, 2, 3},
					TenantID:             "tenant1",
					Version:              "1.0",
				}
				wificonfig.EXPECT().Insert(context.Background(), wificonfigTest).Return(nil, wificonfigs.ErrDatabase)
			},
			response:     wificonfigs.ErrDatabase,
			requestBody:  requestWiFiConfig,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:   "insert wificonfig validation - failed",
			method: http.MethodPost,
			url:    "/api/v1/admin/wirelessconfigs",
			mock: func(wificonfig *mocks.MockWiFiConfigsFeature) {
				wificonfigTest := &dto.WirelessConfig{
					AuthenticationMethod: 4,
					EncryptionMethod:     3,
					SSID:                 "exampleSSID",
					PSKValue:             12345,
					PSKPassphrase:        "examplepassphrase",
					ProfileName:          "newprofile",
					LinkPolicy:           []int{1, 2, 3},
					TenantID:             "tenant1",
					Version:              "1.0",
				}
				wificonfig.EXPECT().Insert(context.Background(), wificonfigTest).Return(nil, wificonfigs.ErrDatabase)
			},
			response:     wificonfigs.ErrDatabase,
			requestBody:  dto.WirelessConfig{AuthenticationMethod: 4, EncryptionMethod: 3, SSID: "exampleSSID", PSKValue: 12345, PSKPassphrase: "examplepassphrase", ProfileName: "newprofile", LinkPolicy: []int{1, 2, 3}, TenantID: "tenant1", Version: "1.0"},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:   "delete wificonfig",
			method: http.MethodDelete,
			url:    "/api/v1/admin/wirelessconfigs/profile",
			mock: func(wificonfig *mocks.MockWiFiConfigsFeature) {
				wificonfig.EXPECT().Delete(context.Background(), "profile", "").Return(nil)
			},
			response:     nil,
			expectedCode: http.StatusNoContent,
		},
		{
			name:   "delete wificonfig - failed",
			method: http.MethodDelete,
			url:    "/api/v1/admin/wirelessconfigs/profile",
			mock: func(wificonfig *mocks.MockWiFiConfigsFeature) {
				wificonfig.EXPECT().Delete(context.Background(), "profile", "").Return(wificonfigs.ErrDatabase)
			},
			response:     wificonfigs.ErrDatabase,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:   "update wificonfig",
			method: http.MethodPatch,
			url:    "/api/v1/admin/wirelessconfigs",
			mock: func(wificonfig *mocks.MockWiFiConfigsFeature) {
				wificonfigTest := &dto.WirelessConfig{
					AuthenticationMethod: 4,
					EncryptionMethod:     3,
					SSID:                 "exampleSSID",
					PSKValue:             12345,
					PSKPassphrase:        "examplepassphrase",
					ProfileName:          "newprofile",
					LinkPolicy:           []int{1, 2, 3},
					TenantID:             "tenant1",
					Version:              "1.0",
				}
				wificonfig.EXPECT().Update(context.Background(), wificonfigTest).Return(wificonfigTest, nil)
			},
			response:     responseWiFiConfig,
			requestBody:  requestWiFiConfig,
			expectedCode: http.StatusOK,
		},
		{
			name:   "update wificonfig - failed",
			method: http.MethodPatch,
			url:    "/api/v1/admin/wirelessconfigs",
			mock: func(wificonfig *mocks.MockWiFiConfigsFeature) {
				wificonfigTest := &dto.WirelessConfig{
					AuthenticationMethod: 4,
					EncryptionMethod:     3,
					SSID:                 "exampleSSID",
					PSKValue:             12345,
					PSKPassphrase:        "examplepassphrase",
					ProfileName:          "newprofile",
					LinkPolicy:           []int{1, 2, 3},
					TenantID:             "tenant1",
					Version:              "1.0",
				}
				wificonfig.EXPECT().Update(context.Background(), wificonfigTest).Return(nil, wificonfigs.ErrDatabase)
			},
			response:     wificonfigs.ErrDatabase,
			requestBody:  requestWiFiConfig,
			expectedCode: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			wifiConfigsFeature, engine := wifiTest(t)

			tc.mock(wifiConfigsFeature)

			var req *http.Request

			var err error

			if tc.method == http.MethodPost || tc.method == http.MethodPatch {
				reqBody, _ := json.Marshal(tc.requestBody)
				req, err = http.NewRequest(tc.method, tc.url, bytes.NewBuffer(reqBody))
			} else {
				req, err = http.NewRequest(tc.method, tc.url, http.NoBody)
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
