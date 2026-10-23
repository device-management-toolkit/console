package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/config"

	dto "github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/mocks"
	"github.com/device-management-toolkit/console/internal/usecase/devices/wsman"
)

// ginBindValidator keeps gin's default validator. Package variables are set
// before init() runs, so this is read before domains_test.go disables it.
var ginBindValidator = binding.Validator

type wiFiProfileRouteTest struct {
	name         string
	method       string
	url          string
	mock         func(*mocks.MockDeviceManagementFeature)
	requestBody  interface{}
	rawBody      string
	response     interface{}
	expectedCode int
}

func TestWiFiProfileRoutes(t *testing.T) { //nolint:gocognit // table-driven HTTP route coverage with many scenarios
	t.Parallel()

	request := config.WirelessProfile{
		ProfileName:          "office",
		SSID:                 "CorpNet",
		Priority:             1,
		AuthenticationMethod: "WPA2PSK",
		EncryptionMethod:     "CCMP",
		Password:             "password123",
	}

	expectedProfiles := []dto.WirelessProfileResponse{{
		ProfileName:          "office",
		SSID:                 "CorpNet",
		Priority:             1,
		AuthenticationMethod: "WPA2PSK",
		EncryptionMethod:     "CCMP",
	}}

	tests := []wiFiProfileRouteTest{
		{
			name:   "get wireless profiles",
			method: http.MethodGet,
			url:    "/api/v1/amt/networkSettings/wireless/profile/device-guid",
			mock: func(feature *mocks.MockDeviceManagementFeature) {
				feature.EXPECT().GetWirelessProfiles(context.Background(), "device-guid", "").Return(expectedProfiles, nil)
			},
			response:     expectedProfiles,
			expectedCode: http.StatusOK,
		},
		{
			name:   "get wireless profiles - service failure",
			method: http.MethodGet,
			url:    "/api/v1/amt/networkSettings/wireless/profile/device-guid",
			mock: func(feature *mocks.MockDeviceManagementFeature) {
				feature.EXPECT().GetWirelessProfiles(context.Background(), "device-guid", "").Return(nil, ErrGeneral)
			},
			expectedCode: http.StatusInternalServerError,
		},
		{
			name:        "add wireless profile",
			method:      http.MethodPost,
			url:         "/api/v1/amt/networkSettings/wireless/profile/device-guid",
			requestBody: request,
			mock: func(feature *mocks.MockDeviceManagementFeature) {
				feature.EXPECT().AddWirelessProfile(context.Background(), "device-guid", "", request).Return(nil)
			},
			expectedCode: http.StatusNoContent,
		},
		{
			name:         "add wireless profile - bind failure",
			method:       http.MethodPost,
			url:          "/api/v1/amt/networkSettings/wireless/profile/device-guid",
			rawBody:      `{"profileName":`,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:        "add wireless profile - service failure",
			method:      http.MethodPost,
			url:         "/api/v1/amt/networkSettings/wireless/profile/device-guid",
			requestBody: request,
			mock: func(feature *mocks.MockDeviceManagementFeature) {
				feature.EXPECT().AddWirelessProfile(context.Background(), "device-guid", "", request).Return(ErrGeneral)
			},
			expectedCode: http.StatusInternalServerError,
		},
		{
			name:        "add wireless profile - no wifi port",
			method:      http.MethodPost,
			url:         "/api/v1/amt/networkSettings/wireless/profile/device-guid",
			requestBody: request,
			mock: func(feature *mocks.MockDeviceManagementFeature) {
				feature.EXPECT().AddWirelessProfile(context.Background(), "device-guid", "", request).Return(wsman.ErrNoWiFiPort)
			},
			expectedCode: http.StatusNotFound,
		},
		{
			name:        "update wireless profile",
			method:      http.MethodPatch,
			url:         "/api/v1/amt/networkSettings/wireless/profile/device-guid",
			requestBody: request,
			mock: func(feature *mocks.MockDeviceManagementFeature) {
				feature.EXPECT().UpdateWirelessProfile(context.Background(), "device-guid", "", request).Return(nil)
			},
			expectedCode: http.StatusNoContent,
		},
		{
			name:        "update wireless profile - uses body profile name",
			method:      http.MethodPatch,
			url:         "/api/v1/amt/networkSettings/wireless/profile/device-guid",
			requestBody: config.WirelessProfile{ProfileName: "guest", SSID: "CorpNet", Priority: 1, AuthenticationMethod: "WPA2PSK", EncryptionMethod: "CCMP", Password: "password123"},
			mock: func(feature *mocks.MockDeviceManagementFeature) {
				feature.EXPECT().UpdateWirelessProfile(context.Background(), "device-guid", "", config.WirelessProfile{ProfileName: "guest", SSID: "CorpNet", Priority: 1, AuthenticationMethod: "WPA2PSK", EncryptionMethod: "CCMP", Password: "password123"}).Return(nil)
			},
			expectedCode: http.StatusNoContent,
		},
		{
			name:         "update wireless profile - bind failure",
			method:       http.MethodPatch,
			url:          "/api/v1/amt/networkSettings/wireless/profile/device-guid",
			rawBody:      `{"profileName":`,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:        "update wireless profile - service failure",
			method:      http.MethodPatch,
			url:         "/api/v1/amt/networkSettings/wireless/profile/device-guid",
			requestBody: request,
			mock: func(feature *mocks.MockDeviceManagementFeature) {
				feature.EXPECT().UpdateWirelessProfile(context.Background(), "device-guid", "", request).Return(ErrGeneral)
			},
			expectedCode: http.StatusInternalServerError,
		},
		{
			name:        "update wireless profile - no wifi port",
			method:      http.MethodPatch,
			url:         "/api/v1/amt/networkSettings/wireless/profile/device-guid",
			requestBody: request,
			mock: func(feature *mocks.MockDeviceManagementFeature) {
				feature.EXPECT().UpdateWirelessProfile(context.Background(), "device-guid", "", request).Return(wsman.ErrNoWiFiPort)
			},
			expectedCode: http.StatusNotFound,
		},
		{
			name:   "delete wireless profile",
			method: http.MethodDelete,
			url:    "/api/v1/amt/networkSettings/wireless/profile/device-guid/office",
			mock: func(feature *mocks.MockDeviceManagementFeature) {
				feature.EXPECT().DeleteWirelessProfile(context.Background(), "device-guid", "office", "").Return(nil)
			},
			expectedCode: http.StatusNoContent,
		},
		{
			name:   "delete wireless profile - service failure",
			method: http.MethodDelete,
			url:    "/api/v1/amt/networkSettings/wireless/profile/device-guid/office",
			mock: func(feature *mocks.MockDeviceManagementFeature) {
				feature.EXPECT().DeleteWirelessProfile(context.Background(), "device-guid", "office", "").Return(ErrGeneral)
			},
			expectedCode: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			feature, engine := deviceManagementTest(t)

			if tc.mock != nil {
				tc.mock(feature)
			}

			var req *http.Request

			var err error

			switch tc.method {
			case http.MethodPost, http.MethodPatch:
				if tc.rawBody != "" {
					req, err = http.NewRequestWithContext(context.Background(), tc.method, tc.url, bytes.NewBufferString(tc.rawBody))
				} else {
					payload, marshalErr := json.Marshal(tc.requestBody)
					require.NoError(t, marshalErr)

					req, err = http.NewRequestWithContext(context.Background(), tc.method, tc.url, bytes.NewBuffer(payload))
				}

				req.Header.Set("Content-Type", "application/json")
			default:
				req, err = http.NewRequestWithContext(context.Background(), tc.method, tc.url, http.NoBody)
			}

			require.NoError(t, err)

			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)

			require.Equal(t, tc.expectedCode, w.Code)

			if tc.expectedCode == http.StatusOK {
				jsonBytes, marshalErr := json.Marshal(tc.response)
				require.NoError(t, marshalErr)
				require.Equal(t, string(jsonBytes), w.Body.String())

				return
			}

			if tc.expectedCode == http.StatusNoContent {
				require.Empty(t, w.Body.String())
			}
		})
	}
}

// The other v1 route tests run with gin bind validation disabled. This test
// turns it on for its own run, so it must not run in parallel with them.
//
//nolint:paralleltest // mutates the global gin binding.Validator
func TestWiFiProfileRoutesValidation(t *testing.T) {
	v, ok := ginBindValidator.Engine().(*validator.Validate)
	require.True(t, ok)
	require.NoError(t, v.RegisterValidation("wirelessprofile", dto.ValidateWirelessProfile))

	binding.Validator = ginBindValidator

	t.Cleanup(func() { binding.Validator = nil })

	url := "/api/v1/amt/networkSettings/wireless/profile/device-guid"
	peapMissingUsername := config.WirelessProfile{
		ProfileName:          "OfficeEAP",
		SSID:                 "CorpNet",
		Priority:             1,
		AuthenticationMethod: "WPA2IEEE8021x",
		EncryptionMethod:     "CCMP",
		IEEE8021x:            &config.IEEE8021x{AuthenticationProtocol: 2, Password: "password123"},
	}

	peapMissingPassword := peapMissingUsername
	peapMissingPassword.IEEE8021x = &config.IEEE8021x{AuthenticationProtocol: 2, Username: "user"}

	peapWithUsername := peapMissingUsername
	peapWithUsername.IEEE8021x = &config.IEEE8021x{AuthenticationProtocol: 2, Username: "user", Password: "password123"}

	tests := []wiFiProfileRouteTest{
		{
			name:         "add wireless profile - ieee8021x peap missing password",
			method:       http.MethodPost,
			url:          url,
			requestBody:  peapMissingPassword,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "update wireless profile - ieee8021x peap missing password",
			method:       http.MethodPatch,
			url:          url,
			requestBody:  peapMissingPassword,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "add wireless profile - ieee8021x peap missing username",
			method:       http.MethodPost,
			url:          url,
			requestBody:  peapMissingUsername,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "update wireless profile - ieee8021x peap missing username",
			method:       http.MethodPatch,
			url:          url,
			requestBody:  peapMissingUsername,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:        "add wireless profile - ieee8021x peap with username",
			method:      http.MethodPost,
			url:         url,
			requestBody: peapWithUsername,
			mock: func(feature *mocks.MockDeviceManagementFeature) {
				feature.EXPECT().AddWirelessProfile(gomock.Any(), "device-guid", "", peapWithUsername).Return(nil)
			},
			expectedCode: http.StatusNoContent,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			feature, engine := deviceManagementTest(t)

			if tc.mock != nil {
				tc.mock(feature)
			}

			payload, err := json.Marshal(tc.requestBody)
			require.NoError(t, err)

			req, err := http.NewRequestWithContext(context.Background(), tc.method, tc.url, bytes.NewBuffer(payload))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)

			require.Equal(t, tc.expectedCode, w.Code, w.Body.String())
		})
	}
}
