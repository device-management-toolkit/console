package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/mocks"
	"github.com/device-management-toolkit/console/internal/usecase/devices"
	"github.com/device-management-toolkit/console/pkg/logger"
)

var testConfigOnce sync.Once

func setupTestConfig() {
	testConfigOnce.Do(func() {
		if config.ConsoleConfig == nil {
			config.ConsoleConfig = &config.Config{
				Auth: config.Auth{
					JWTKey:                   "test-key",
					JWTExpiration:            24 * time.Hour,
					RedirectionJWTExpiration: 5 * time.Minute,
				},
			}
		}
	})
}

func devicesTest(t *testing.T) (*mocks.MockDeviceManagementFeature, *gin.Engine) {
	t.Helper()
	setupTestConfig()

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

	requestDeviceFields = map[string]bool{
		"connectionstatus": true,
		"mpsinstance":      true,
		"hostname":         true,
		"guid":             true,
		"mpsusername":      true,
		"tags":             true,
		"tenantid":         true,
		"friendlyname":     true,
		"dnssuffix":        true,
		"lastconnected":    true,
		"lastseen":         true,
		"lastdisconnected": true,
		"username":         true,
		"password":         true,
		"mpspassword":      true,
		"mebxpassword":     true,
		"usetls":           true,
		"allowselfsigned":  true,
		"certhash":         true,
	}
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
				device.EXPECT().Update(context.Background(), deviceTest, requestDeviceFields).Return(deviceTest, nil)
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
				device.EXPECT().Update(context.Background(), deviceTest, requestDeviceFields).Return(nil, devices.ErrDatabase)
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

// TestDevicesUpdatePartialPatch verifies that the v1 update controller
// translates the JSON keys actually present in the body into the fields map
// passed to Update, and forwards a DTO containing only those provided values.
// The merge against the existing record is exercised in the usecase tests.
const testDeviceGUID = "4c4c4544-0046-3510-8050-c2c04f365033"

func TestDevicesUpdatePartialPatch(t *testing.T) {
	t.Parallel()

	guid := testDeviceGUID

	incoming := &dto.Device{
		GUID:     guid,
		Hostname: "test-device-renamed",
	}

	expectedFields := map[string]bool{"guid": true, "hostname": true}

	response := &dto.Device{
		GUID:         guid,
		Hostname:     "test-device-renamed",
		Tags:         []string{"lab", "floor-2"},
		MPSUsername:  "admin",
		MPSPassword:  "P@ssw0rd!",
		Username:     "admin",
		Password:     "AmtP@ss123",
		MEBXPassword: "mebxsecret",
	}

	devicesFeature, engine := devicesTest(t)

	devicesFeature.EXPECT().
		Update(context.Background(), incoming, expectedFields).
		Return(response, nil)

	body := []byte(`{"guid":"` + guid + `","hostname":"test-device-renamed"}`)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPatch, "/api/v1/devices", bytes.NewBuffer(body))
	require.NoError(t, err)

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	expected, _ := json.Marshal(response)
	require.Equal(t, string(expected), w.Body.String())
}

// encoding/json unmarshals case-insensitively; the merge must see the field as
// provided regardless of the casing the client used.
func TestDevicesUpdatePartialPatchMixedCaseKeys(t *testing.T) {
	t.Parallel()

	guid := testDeviceGUID

	incoming := &dto.Device{
		GUID:     guid,
		Hostname: "test-device-renamed",
	}

	expectedFields := map[string]bool{"guid": true, "hostname": true}

	devicesFeature, engine := devicesTest(t)

	devicesFeature.EXPECT().
		Update(context.Background(), incoming, expectedFields).
		Return(incoming, nil)

	body := []byte(`{"GUID":"` + guid + `","Hostname":"test-device-renamed"}`)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPatch, "/api/v1/devices", bytes.NewBuffer(body))
	require.NoError(t, err)

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestDevicesUpdatePartialPatchTracksDeviceInfoSubfields(t *testing.T) {
	t.Parallel()

	guid := testDeviceGUID

	incoming := &dto.Device{
		GUID: guid,
		DeviceInfo: &dto.DeviceInfo{
			FWVersion: "16.1.30",
		},
	}

	expectedFields := map[string]bool{
		"guid":                 true,
		"deviceinfo":           true,
		"deviceinfo.fwversion": true,
	}

	devicesFeature, engine := devicesTest(t)

	devicesFeature.EXPECT().
		Update(context.Background(), incoming, expectedFields).
		Return(incoming, nil)

	body := []byte(`{"guid":"` + guid + `","deviceInfo":{"fwVersion":"16.1.30"}}`)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPatch, "/api/v1/devices", bytes.NewBuffer(body))
	require.NoError(t, err)

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestDevicesInsertAcceptsFullDeviceInfo(t *testing.T) {
	t.Parallel()

	lmsInstalled := false
	amtEnabledInBIOS := true
	dhcpEnabled := true
	ethernetAdapterCount := 2
	monitorConnected := true
	ieee8021xEnabled := false

	incoming := &dto.Device{
		GUID:     testDeviceGUID,
		Hostname: "test-device",
		DeviceInfo: &dto.DeviceInfo{
			FWVersion:   "16.1.30",
			FWBuild:     "3400",
			FWSku:       "11",
			CurrentMode: "Admin",
			Features:    "SOL,IDER,KVM",
			IPAddress:   "10.0.0.12",
			LastUpdated: &timeNow,
			TLSMode:     "TLS 1.2",
			UPID: map[string]json.RawMessage{
				"oemPlatformIdType": json.RawMessage(`"Not Set (0)"`),
				"oemId":             json.RawMessage(`""`),
				"csmeId":            json.RawMessage(`"4A45A39C5ED9462082510000"`),
			},
			AMTEnabledInBIOS:     &amtEnabledInBIOS,
			MEInterfaceVersion:   "16.1.25.2124",
			DHCPEnabled:          &dhcpEnabled,
			CertHashes:           []string{"a1b2c3", "d4e5f6"},
			LMSInstalled:         &lmsInstalled,
			LMSVersion:           "2410.5.0.0",
			OSName:               "linux",
			OSVersion:            "6.8.0-51-generic",
			OSDistro:             "Ubuntu 24.04 LTS",
			CPUModel:             "Intel(R) Core(TM) Ultra 7 165H",
			OSIPAddress:          "10.49.76.163",
			EthernetAdapterCount: &ethernetAdapterCount,
			MonitorConnected:     &monitorConnected,
			IEEE8021XEnabled:     &ieee8021xEnabled,
		},
	}

	devicesFeature, engine := devicesTest(t)

	devicesFeature.EXPECT().
		Insert(context.Background(), incoming).
		Return(incoming, nil)

	body := []byte(`{
		"guid":"` + testDeviceGUID + `",
		"hostname":"test-device",
		"deviceInfo":{
			"fwVersion":"16.1.30",
			"fwBuild":"3400",
			"fwSku":"11",
			"currentMode":"Admin",
			"features":"SOL,IDER,KVM",
			"ipAddress":"10.0.0.12",
			"lastUpdated":"` + timeNow.Format(time.RFC3339Nano) + `",
			"tlsMode":"TLS 1.2",
			"upid":{
				"oemPlatformIdType":"Not Set (0)",
				"oemId":"",
				"csmeId":"4A45A39C5ED9462082510000"
			},
			"amtEnabledInBIOS":true,
			"meInterfaceVersion":"16.1.25.2124",
			"dhcpEnabled":true,
			"certHashes":["a1b2c3","d4e5f6"],
			"lmsInstalled":false,
			"lmsVersion":"2410.5.0.0",
			"osName":"linux",
			"osVersion":"6.8.0-51-generic",
			"osDistro":"Ubuntu 24.04 LTS",
			"cpuModel":"Intel(R) Core(TM) Ultra 7 165H",
			"osIpAddress":"10.49.76.163",
			"ethernetAdapterCount":2,
			"monitorConnected":true,
			"ieee8021xEnabled":false
		}
	}`)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/devices", bytes.NewBuffer(body))
	require.NoError(t, err)

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)

	expected, _ := json.Marshal(incoming)
	require.Equal(t, string(expected), w.Body.String())
}

// TestLoginRedirection verifies the device redirection token endpoint
func TestLoginRedirection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		deviceID     string
		mock         func(devFeature *mocks.MockDeviceManagementFeature)
		expectedCode int
		expectedErr  bool
	}{
		{
			name:     "login redirection - success",
			deviceID: "test-device-guid",
			mock: func(devFeature *mocks.MockDeviceManagementFeature) {
				devFeature.EXPECT().GetByID(context.Background(), "test-device-guid", "", false).
					Return(&dto.Device{GUID: "test-device-guid", Hostname: "test-host"}, nil)
			},
			expectedCode: http.StatusOK,
			expectedErr:  false,
		},
		{
			name:     "login redirection - device not found",
			deviceID: "invalid-guid",
			mock: func(devFeature *mocks.MockDeviceManagementFeature) {
				devFeature.EXPECT().GetByID(context.Background(), "invalid-guid", "", false).
					Return(nil, devices.ErrNotFound)
			},
			expectedCode: http.StatusNotFound,
			expectedErr:  true,
		},
		{
			name:     "login redirection - database error",
			deviceID: "test-device-guid",
			mock: func(devFeature *mocks.MockDeviceManagementFeature) {
				devFeature.EXPECT().GetByID(context.Background(), "test-device-guid", "", false).
					Return(nil, devices.ErrDatabase)
			},
			expectedCode: http.StatusBadRequest,
			expectedErr:  true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			devicesFeature, engine := devicesTest(t)
			tc.mock(devicesFeature)

			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet,
				"/api/v1/authorize/redirection/"+tc.deviceID, http.NoBody)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)

			require.Equal(t, tc.expectedCode, w.Code)

			if !tc.expectedErr && tc.expectedCode == http.StatusOK {
				// Parse response
				var response map[string]string

				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				tokenString, ok := response["token"]
				require.True(t, ok, "token field not found in response")
				require.NotEmpty(t, tokenString)

				// Decode and verify token expiration
				verifyRedirectionTokenExpiration(t, tokenString)
			}
		})
	}
}

// verifyRedirectionTokenExpiration decodes JWT token and verifies 5-minute expiration
func verifyRedirectionTokenExpiration(t *testing.T, tokenString string) {
	t.Helper()

	// Parse JWT claims without verification (we just need to check the structure)
	claims := jwt.RegisteredClaims{}
	_, err := jwt.ParseWithClaims(tokenString, &claims, func(_ *jwt.Token) (interface{}, error) {
		// Return the key for verification (we're using a test key)
		return []byte(config.ConsoleConfig.JWTKey), nil
	})
	require.NoError(t, err, "token should be parseable")

	// Verify ExpiresAt is set
	require.NotNil(t, claims.ExpiresAt, "token should have expiration time")

	// Calculate expected expiration window
	now := time.Now()
	expirationTime := claims.ExpiresAt.Time
	timeDiff := expirationTime.Sub(now)

	// Should be approximately 5 minutes (with some tolerance for test execution time)
	expectedDuration := config.ConsoleConfig.RedirectionJWTExpiration
	tolerance := 10 * time.Second

	// Verify expiration is close to configured RedirectionJWTExpiration (5 minutes by default)
	require.True(t, timeDiff > expectedDuration-tolerance && timeDiff < expectedDuration+tolerance,
		"token expiration should be ~5 minutes, got %v", timeDiff)

	// Specifically verify it's NOT 24 hours (the bug)
	maxWrongExpiration := 24 * time.Hour
	require.True(t, timeDiff < maxWrongExpiration-time.Hour,
		"token expiration time %v is suspiciously close to 24 hours (the bug)", timeDiff)
}
