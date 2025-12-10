package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/mocks"
	"github.com/device-management-toolkit/console/internal/usecase/devices"
	"github.com/device-management-toolkit/console/pkg/logger"
)

func devicesTest(t *testing.T) (*mocks.MockDeviceManagementFeature, *gin.Engine) {
	t.Helper()

	mockCtl := gomock.NewController(t)
	defer mockCtl.Finish()

	log := logger.New("error")
	device := mocks.NewMockDeviceManagementFeature(mockCtl)

	engine := gin.New()
	handler := engine.Group("/api/v1")

	NewDeviceRoutes(handler, device, log)

	return device, engine
}

type deviceTest struct {
	name         string
	method       string
	url          string
	mock         func(repo *mocks.MockDeviceManagementFeature)
	response     interface{}
	requestBody  dto.Device
	expectedCode int
}

var (
	timeNow        = time.Now().UTC()
	requestDevice  = dto.Device{ConnectionStatus: true, MPSInstance: "mpsInstance", Hostname: "hostname", GUID: "guid", MPSUsername: "mpsusername", Tags: []string{"tag1", "tag2"}, TenantID: "tenantId", FriendlyName: "friendlyName", DNSSuffix: "dnsSuffix", Username: "admin", Password: "password", UseTLS: true, AllowSelfSigned: true, LastConnected: &timeNow, LastSeen: &timeNow, LastDisconnected: &timeNow}
	responseDevice = dto.Device{ConnectionStatus: true, MPSInstance: "mpsInstance", Hostname: "hostname", GUID: "guid", MPSUsername: "mpsusername", Tags: []string{"tag1", "tag2"}, TenantID: "tenantId", FriendlyName: "friendlyName", DNSSuffix: "dnsSuffix", Username: "admin", Password: "password", UseTLS: true, AllowSelfSigned: true, LastConnected: &timeNow, LastSeen: &timeNow, LastDisconnected: &timeNow}
)

func TestDevicesRoutes(t *testing.T) {
	t.Parallel()

	tests := []deviceTest{
		{
			name:   "get all devices",
			method: http.MethodGet,
			url:    "/api/v1/devices",
			mock: func(device *mocks.MockDeviceManagementFeature) {
				device.EXPECT().Get(context.Background(), 25, 0, "").Return([]dto.Device{{
					GUID: "guid", MPSUsername: "mpsusername", Username: "admin", Password: "password", ConnectionStatus: true, Hostname: "hostname",
				}}, nil)
			},
			response:     []dto.Device{{GUID: "guid", MPSUsername: "mpsusername", Username: "admin", Password: "password", ConnectionStatus: true, Hostname: "hostname"}},
			expectedCode: http.StatusOK,
		},
		{
			name:   "get all devices - with count",
			method: http.MethodGet,
			url:    "/api/v1/devices?$top=10&$skip=1&$count=true",
			mock: func(device *mocks.MockDeviceManagementFeature) {
				device.EXPECT().Get(context.Background(), 10, 1, "").Return([]dto.Device{{
					GUID: "guid", MPSUsername: "mpsusername", Username: "admin", Password: "password", ConnectionStatus: true, Hostname: "hostname",
				}}, nil)
				device.EXPECT().GetCount(context.Background(), "").Return(1, nil)
			},
			response:     dto.DeviceCountResponse{Count: 1, Data: []dto.Device{{GUID: "guid", MPSUsername: "mpsusername", Username: "admin", Password: "password", ConnectionStatus: true, Hostname: "hostname"}}},
			expectedCode: http.StatusOK,
		},
		{
			name:   "get device by id",
			method: http.MethodGet,
			url:    "/api/v1/devices/123e4567-e89b-12d3-a456-426614174000",
			mock: func(device *mocks.MockDeviceManagementFeature) {
				device.EXPECT().GetByID(context.Background(), "123e4567-e89b-12d3-a456-426614174000", "", false).Return(&dto.Device{
					GUID: "123e4567-e89b-12d3-a456-426614174000", MPSUsername: "mpsusername", Username: "admin", Password: "password", ConnectionStatus: true, Hostname: "hostname",
				}, nil)
			},
			response:     &dto.Device{GUID: "123e4567-e89b-12d3-a456-426614174000", MPSUsername: "mpsusername", Username: "admin", Password: "password", ConnectionStatus: true, Hostname: "hostname"},
			expectedCode: http.StatusOK,
		},
		{
			name:   "get device by id - failed",
			method: http.MethodGet,
			url:    "/api/v1/devices/123e4567-e89b-12d3-a456-426614174000",
			mock: func(device *mocks.MockDeviceManagementFeature) {
				device.EXPECT().GetByID(context.Background(), "123e4567-e89b-12d3-a456-426614174000", "", false).Return(nil, devices.ErrDatabase)
			},
			response:     devices.ErrDatabase,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:   "get all devices - failed",
			method: http.MethodGet,
			url:    "/api/v1/devices",
			mock: func(device *mocks.MockDeviceManagementFeature) {
				device.EXPECT().Get(context.Background(), 25, 0, "").Return(nil, devices.ErrDatabase)
			},
			response:     devices.ErrDatabase,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:   "insert device",
			method: http.MethodPost,
			url:    "/api/v1/devices",
			mock: func(device *mocks.MockDeviceManagementFeature) {
				deviceTest := &dto.Device{
					ConnectionStatus: true,
					MPSInstance:      "mpsInstance",
					Hostname:         "hostname",
					GUID:             "guid",
					MPSUsername:      "mpsusername",
					Tags:             []string{"tag1", "tag2"},
					TenantID:         "tenantId",
					FriendlyName:     "friendlyName",
					DNSSuffix:        "dnsSuffix",
					Username:         "admin",
					Password:         "password",
					UseTLS:           true,
					AllowSelfSigned:  true,
					LastConnected:    &timeNow,
					LastSeen:         &timeNow,
					LastDisconnected: &timeNow,
				}
				device.EXPECT().Insert(context.Background(), deviceTest).Return(deviceTest, nil)
			},
			response:     responseDevice,
			requestBody:  requestDevice,
			expectedCode: http.StatusCreated,
		},
		{
			name:   "insert device - failed",
			method: http.MethodPost,
			url:    "/api/v1/devices",
			mock: func(device *mocks.MockDeviceManagementFeature) {
				deviceTest := &dto.Device{
					ConnectionStatus: true,
					MPSInstance:      "mpsInstance",
					Hostname:         "hostname",
					GUID:             "guid",
					MPSUsername:      "mpsusername",
					Tags:             []string{"tag1", "tag2"},
					TenantID:         "tenantId",
					FriendlyName:     "friendlyName",
					DNSSuffix:        "dnsSuffix",
					Username:         "admin",
					Password:         "password",
					UseTLS:           true,
					AllowSelfSigned:  true,
					LastConnected:    &timeNow,
					LastSeen:         &timeNow,
					LastDisconnected: &timeNow,
				}
				device.EXPECT().Insert(context.Background(), deviceTest).Return(nil, devices.ErrDatabase)
			},
			response:     devices.ErrDatabase,
			requestBody:  requestDevice,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:   "delete device",
			method: http.MethodDelete,
			url:    "/api/v1/devices/profile",
			mock: func(device *mocks.MockDeviceManagementFeature) {
				device.EXPECT().Delete(context.Background(), "profile", "").Return(nil)
			},
			response:     nil,
			expectedCode: http.StatusNoContent,
		},
		{
			name:   "delete device - failed",
			method: http.MethodDelete,
			url:    "/api/v1/devices/profile",
			mock: func(device *mocks.MockDeviceManagementFeature) {
				device.EXPECT().Delete(context.Background(), "profile", "").Return(devices.ErrDatabase)
			},
			response:     devices.ErrDatabase,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:   "update device",
			method: http.MethodPatch,
			url:    "/api/v1/devices",
			mock: func(device *mocks.MockDeviceManagementFeature) {
				deviceTest := &dto.Device{
					ConnectionStatus: true,
					MPSInstance:      "mpsInstance",
					Hostname:         "hostname",
					GUID:             "guid",
					MPSUsername:      "mpsusername",
					Tags:             []string{"tag1", "tag2"},
					TenantID:         "tenantId",
					FriendlyName:     "friendlyName",
					DNSSuffix:        "dnsSuffix",
					Username:         "admin",
					Password:         "password",
					UseTLS:           true,
					AllowSelfSigned:  true,
					LastConnected:    &timeNow,
					LastSeen:         &timeNow,
					LastDisconnected: &timeNow,
				}
				device.EXPECT().Update(context.Background(), deviceTest).Return(deviceTest, nil)
			},
			response:     responseDevice,
			requestBody:  requestDevice,
			expectedCode: http.StatusOK,
		},
		{
			name:   "update device - failed",
			method: http.MethodPatch,
			url:    "/api/v1/devices",
			mock: func(device *mocks.MockDeviceManagementFeature) {
				deviceTest := &dto.Device{
					ConnectionStatus: true,
					MPSInstance:      "mpsInstance",
					Hostname:         "hostname",
					GUID:             "guid",
					MPSUsername:      "mpsusername",
					Tags:             []string{"tag1", "tag2"},
					TenantID:         "tenantId",
					FriendlyName:     "friendlyName",
					DNSSuffix:        "dnsSuffix",
					Username:         "admin",
					Password:         "password",
					UseTLS:           true,
					AllowSelfSigned:  true,
					LastConnected:    &timeNow,
					LastSeen:         &timeNow,
					LastDisconnected: &timeNow,
				}
				device.EXPECT().Update(context.Background(), deviceTest).Return(nil, devices.ErrDatabase)
			},
			response:     devices.ErrDatabase,
			requestBody:  requestDevice,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:   "tags of a device",
			method: http.MethodGet,
			url:    "/api/v1/devices/tags",
			mock: func(device *mocks.MockDeviceManagementFeature) {
				device.EXPECT().GetDistinctTags(context.Background(), "").Return([]string{"tag1", "tag2"}, nil)
			},
			response:     []string{"tag1", "tag2"},
			expectedCode: http.StatusOK,
		},
		{
			name:   "tags of a device - failed",
			method: http.MethodGet,
			url:    "/api/v1/devices/tags",
			mock: func(device *mocks.MockDeviceManagementFeature) {
				device.EXPECT().GetDistinctTags(context.Background(), "").Return(nil, devices.ErrDatabase)
			},
			response:     devices.ErrDatabase,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:   "get devices stats",
			method: http.MethodGet,
			url:    "/api/v1/devices/stats",
			mock: func(device *mocks.MockDeviceManagementFeature) {
				device.EXPECT().GetCount(context.Background(), "").Return(5, nil)
			},
			response:     dto.DeviceStatResponse{TotalCount: 5},
			expectedCode: http.StatusOK,
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			devicesFeature, engine := devicesTest(t)

			tc.mock(devicesFeature)

			var req *http.Request

			var err error

			if tc.method == http.MethodPost || tc.method == http.MethodPatch {
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
