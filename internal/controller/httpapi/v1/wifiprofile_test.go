package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/config"

	"github.com/device-management-toolkit/console/internal/mocks"
)

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

	expectedProfiles := []config.WirelessProfile{{
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
				feature.EXPECT().GetWirelessProfiles(context.Background(), "device-guid").Return(expectedProfiles, nil)
			},
			response:     expectedProfiles,
			expectedCode: http.StatusOK,
		},
		{
			name:   "get wireless profiles - service failure",
			method: http.MethodGet,
			url:    "/api/v1/amt/networkSettings/wireless/profile/device-guid",
			mock: func(feature *mocks.MockDeviceManagementFeature) {
				feature.EXPECT().GetWirelessProfiles(context.Background(), "device-guid").Return(nil, ErrGeneral)
			},
			expectedCode: http.StatusInternalServerError,
		},
		{
			name:        "add wireless profile",
			method:      http.MethodPost,
			url:         "/api/v1/amt/networkSettings/wireless/profile/device-guid",
			requestBody: request,
			mock: func(feature *mocks.MockDeviceManagementFeature) {
				feature.EXPECT().AddWirelessProfile(context.Background(), "device-guid", request).Return(nil)
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
				feature.EXPECT().AddWirelessProfile(context.Background(), "device-guid", request).Return(ErrGeneral)
			},
			expectedCode: http.StatusInternalServerError,
		},
		{
			name:        "update wireless profile",
			method:      http.MethodPatch,
			url:         "/api/v1/amt/networkSettings/wireless/profile/device-guid",
			requestBody: request,
			mock: func(feature *mocks.MockDeviceManagementFeature) {
				feature.EXPECT().UpdateWirelessProfile(context.Background(), "device-guid", request).Return(nil)
			},
			expectedCode: http.StatusNoContent,
		},
		{
			name:        "update wireless profile - uses body profile name",
			method:      http.MethodPatch,
			url:         "/api/v1/amt/networkSettings/wireless/profile/device-guid",
			requestBody: config.WirelessProfile{ProfileName: "guest", SSID: "CorpNet", Priority: 1, AuthenticationMethod: "WPA2PSK", EncryptionMethod: "CCMP", Password: "password123"},
			mock: func(feature *mocks.MockDeviceManagementFeature) {
				feature.EXPECT().UpdateWirelessProfile(context.Background(), "device-guid", config.WirelessProfile{ProfileName: "guest", SSID: "CorpNet", Priority: 1, AuthenticationMethod: "WPA2PSK", EncryptionMethod: "CCMP", Password: "password123"}).Return(nil)
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
				feature.EXPECT().UpdateWirelessProfile(context.Background(), "device-guid", request).Return(ErrGeneral)
			},
			expectedCode: http.StatusInternalServerError,
		},
		{
			name:   "delete wireless profile",
			method: http.MethodDelete,
			url:    "/api/v1/amt/networkSettings/wireless/profile/device-guid/office",
			mock: func(feature *mocks.MockDeviceManagementFeature) {
				feature.EXPECT().DeleteWirelessProfile(context.Background(), "device-guid", "office").Return(nil)
			},
			expectedCode: http.StatusNoContent,
		},
		{
			name:   "delete wireless profile - service failure",
			method: http.MethodDelete,
			url:    "/api/v1/amt/networkSettings/wireless/profile/device-guid/office",
			mock: func(feature *mocks.MockDeviceManagementFeature) {
				feature.EXPECT().DeleteWirelessProfile(context.Background(), "device-guid", "office").Return(ErrGeneral)
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
