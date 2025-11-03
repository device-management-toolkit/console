package v1

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

	power "github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/power"

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
	amtExplorerMock := mocks.NewMockAMTExplorerFeature(mockCtl)
	exporterMock := mocks.NewMockExporter(mockCtl)
	engine := gin.New()
	handler := engine.Group("/api/v1")

	NewAmtRoutes(handler, deviceManagement, amtExplorerMock, exporterMock, log)

	return deviceManagement, engine
}

func TestDeviceManagement(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		url          string
		mock         func(m *mocks.MockDeviceManagementFeature)
		method       string
		requestBody  interface{}
		expectedCode int
		response     interface{}
		responseV2   interface{}
	}{
		{
			name:   "getVersion - successful retrieval",
			url:    "/api/v1/amt/version/valid-guid",
			method: http.MethodGet,
			mock: func(m *mocks.MockDeviceManagementFeature) {
				m.EXPECT().GetVersion(context.Background(), "valid-guid").
					Return(dto.Version{}, dtov2.Version{}, nil)
			},
			expectedCode: http.StatusOK,
			response:     dto.Version{},
		},
		{
			name:   "getVersion - service failure",
			url:    "/api/v1/amt/version/valid-guid",
			method: http.MethodGet,
			mock: func(m *mocks.MockDeviceManagementFeature) {
				m.EXPECT().GetVersion(context.Background(), "valid-guid").
					Return(dto.Version{}, dtov2.Version{}, ErrGeneral)
			},
			expectedCode: http.StatusInternalServerError,
			response:     nil,
		},
		{
			name:   "getBootSources - successful retrieval",
			url:    "/api/v1/amt/power/bootSources/valid-guid",
			method: http.MethodGet,
			mock: func(m *mocks.MockDeviceManagementFeature) {
				m.EXPECT().GetBootSourceSetting(context.Background(), "valid-guid").
					Return([]dto.BootSources{{
						InstanceID:           "Hard-drive Boot",
						BootString:           "\\OemPba.efi",
						BIOSBootString:       "Boot from Hard Drive",
						ElementName:          "",
						FailThroughSupported: 0,
						StructuredBootString: "",
					}}, nil)
			},
			expectedCode: http.StatusOK,
			response: []dto.BootSources{{
				InstanceID:           "Hard-drive Boot",
				BootString:           "\\OemPba.efi",
				BIOSBootString:       "Boot from Hard Drive",
				ElementName:          "",
				FailThroughSupported: 0,
				StructuredBootString: "",
			}},
		},
		{
			name:   "getBootSources - service failure",
			url:    "/api/v1/amt/power/bootSources/valid-guid",
			method: http.MethodGet,
			mock: func(m *mocks.MockDeviceManagementFeature) {
				m.EXPECT().GetBootSourceSetting(context.Background(), "valid-guid").
					Return(nil, ErrGeneral)
			},
			expectedCode: http.StatusInternalServerError,
			response:     nil,
		},
		{
			name:   "getFeatures - successful retrieval",
			url:    "/api/v1/amt/features/valid-guid",
			method: http.MethodGet,
			mock: func(m *mocks.MockDeviceManagementFeature) {
				m.EXPECT().GetFeatures(context.Background(), "valid-guid").
					Return(dto.Features{}, dtov2.Features{}, nil)
			},
			expectedCode: http.StatusOK,
			response: map[string]interface{}{
				"IDER":                  false,
				"KVM":                   false,
				"SOL":                   false,
				"httpsBootSupported":    false,
				"kvmAvailable":          false,
				"localPBABootSupported": false,
				"ocr":                   false,
				"optInState":            0,
				"redirection":           false,
				"remoteErase":           false,
				"userConsent":           "",
				"winREBootSupported":    false,
			},
		},
		{
			name:   "getFeatures - service failure",
			url:    "/api/v1/amt/features/valid-guid",
			method: http.MethodGet,
			mock: func(m *mocks.MockDeviceManagementFeature) {
				m.EXPECT().GetFeatures(context.Background(), "valid-guid").
					Return(dto.Features{}, dtov2.Features{}, ErrGeneral)
			},
			expectedCode: http.StatusInternalServerError,
			response:     nil,
		},
		{
			name:        "setFeatures - successful setting",
			url:         "/api/v1/amt/features/valid-guid",
			method:      http.MethodPost,
			requestBody: dto.Features{},
			mock: func(m *mocks.MockDeviceManagementFeature) {
				m.EXPECT().SetFeatures(context.Background(), "valid-guid", dto.Features{}).Return(dto.Features{}, dtov2.Features{}, nil)
			},
			expectedCode: http.StatusOK,
			response:     dto.Features{},
		},
		{
			name:        "setFeatures - invalid JSON payload",
			url:         "/api/v1/amt/features/valid-guid",
			method:      http.MethodPost,
			requestBody: "invalid-json",
			mock: func(_ *mocks.MockDeviceManagementFeature) {
			},
			expectedCode: http.StatusInternalServerError,
			response:     nil,
		},
		{
			name:        "setFeatures - service failure",
			url:         "/api/v1/amt/features/valid-guid",
			method:      http.MethodPost,
			requestBody: dto.Features{},
			mock: func(m *mocks.MockDeviceManagementFeature) {
				m.EXPECT().SetFeatures(context.Background(), "valid-guid", dto.Features{}).
					Return(dto.Features{}, dtov2.Features{}, ErrGeneral)
			},
			expectedCode: http.StatusInternalServerError,
			response:     nil,
		},
		{
			name:   "getAlarmOccurrences - successful retrieval",
			url:    "/api/v1/amt/alarmOccurrences/valid-guid",
			method: http.MethodGet,
			mock: func(m *mocks.MockDeviceManagementFeature) {
				m.EXPECT().GetAlarmOccurrences(context.Background(), "valid-guid").
					Return([]dto.AlarmClockOccurrence{}, nil)
			},
			expectedCode: http.StatusOK,
			response:     []dto.AlarmClockOccurrence{},
		},
		{
			name:   "deleteAlarmOccurrences - successful deletion",
			url:    "/api/v1/amt/alarmOccurrences/valid-guid",
			method: http.MethodDelete,
			requestBody: dto.DeleteAlarmOccurrenceRequest{
				Name: "instanceID",
			},
			mock: func(m *mocks.MockDeviceManagementFeature) {
				m.EXPECT().DeleteAlarmOccurrences(context.Background(), "valid-guid", "instanceID").Return(nil)
			},
			expectedCode: http.StatusNoContent,
		},
		{
			name:   "getHardwareInfo - successful retrieval",
			url:    "/api/v1/amt/hardwareInfo/valid-guid",
			method: http.MethodGet,
			mock: func(m *mocks.MockDeviceManagementFeature) {
				m.EXPECT().GetHardwareInfo(context.Background(), "valid-guid").
					Return(dto.HardwareInfo{}, nil)
			},
			expectedCode: http.StatusOK,
			response:     dto.HardwareInfo{},
		},
		{
			name:   "getDiskInfo - successful retrieval",
			url:    "/api/v1/amt/diskInfo/valid-guid",
			method: http.MethodGet,
			mock: func(m *mocks.MockDeviceManagementFeature) {
				m.EXPECT().GetDiskInfo(context.Background(), "valid-guid").
					Return(dto.DiskInfo{CIMMediaAccessDevice: dto.CIMResponse{Response: map[string]interface{}{"disk": "info"}}}, nil)
			},
			expectedCode: http.StatusOK,
			response:     dto.DiskInfo{CIMMediaAccessDevice: dto.CIMResponse{Response: map[string]interface{}{"disk": "info"}}},
		},
		{
			name:   "getPowerState - successful retrieval",
			url:    "/api/v1/amt/power/state/valid-guid",
			method: http.MethodGet,
			mock: func(m *mocks.MockDeviceManagementFeature) {
				m.EXPECT().GetPowerState(context.Background(), "valid-guid").
					Return(dto.PowerState{PowerState: 2}, nil)
			},
			expectedCode: http.StatusOK,
			response:     dto.PowerState{PowerState: 2},
		},
		{
			name:   "powerAction - successful action",
			url:    "/api/v1/amt/power/action/valid-guid",
			method: http.MethodPost,
			requestBody: dto.PowerAction{
				Action: 4,
			},
			mock: func(m *mocks.MockDeviceManagementFeature) {
				m.EXPECT().SendPowerAction(context.Background(), "valid-guid", 4).
					Return(power.PowerActionResponse{ReturnValue: 0}, nil)
			},
			expectedCode: http.StatusOK,
			response:     power.PowerActionResponse{ReturnValue: 0},
		},
		{
			name:   "getAuditLog - successful retrieval",
			url:    "/api/v1/amt/log/audit/valid-guid?startIndex=0",
			method: http.MethodGet,
			mock: func(m *mocks.MockDeviceManagementFeature) {
				m.EXPECT().GetAuditLog(context.Background(), 0, "valid-guid").
					Return(dto.AuditLog{}, nil)
			},
			expectedCode: http.StatusOK,
			response:     dto.AuditLog{},
		},
		{
			name:   "getEventLog - successful retrieval",
			url:    "/api/v1/amt/log/event/valid-guid?$skip=1&$top=10",
			method: http.MethodGet,
			mock: func(m *mocks.MockDeviceManagementFeature) {
				m.EXPECT().GetEventLog(context.Background(), 1, 10, "valid-guid").
					Return(dto.EventLogs{}, nil)
			},
			expectedCode: http.StatusOK,
			response:     dto.EventLogs{},
		},
		{
			name:   "setBootOptions - successful setting",
			url:    "/api/v1/amt/power/bootOptions/valid-guid",
			method: http.MethodPost,
			requestBody: dto.BootSetting{
				Action: 109,
			},
			mock: func(m *mocks.MockDeviceManagementFeature) {
				m.EXPECT().SetBootOptions(context.Background(), "valid-guid", dto.BootSetting{
					Action: 109,
				}).Return(power.PowerActionResponse{ReturnValue: 0}, nil)
			},
			expectedCode: http.StatusOK,
			response:     power.PowerActionResponse{ReturnValue: 0},
		},
		{
			name:   "successful retrieval",
			url:    "/api/v1/amt/networkSettings/valid-guid",
			method: http.MethodGet,
			mock: func(m *mocks.MockDeviceManagementFeature) {
				m.EXPECT().GetNetworkSettings(context.Background(), "valid-guid").
					Return(dto.NetworkSettings{}, nil)
			},
			expectedCode: http.StatusOK,
			response:     dto.NetworkSettings{},
		},
		{
			name:   "getCertificates - successful retrieval",
			url:    "/api/v1/amt/certificates/valid-guid",
			method: http.MethodGet,
			mock: func(m *mocks.MockDeviceManagementFeature) {
				m.EXPECT().GetCertificates(context.Background(), "valid-guid").
					Return(dto.SecuritySettings{}, nil)
			},
			expectedCode: http.StatusOK,
			response:     dto.SecuritySettings{},
		},
		{
			name:   "getCertificates - failed retrieval",
			url:    "/api/v1/amt/certificates/valid-guid",
			method: http.MethodGet,
			mock: func(m *mocks.MockDeviceManagementFeature) {
				m.EXPECT().GetCertificates(context.Background(), "valid-guid").
					Return(dto.SecuritySettings{}, ErrGeneral)
			},
			expectedCode: http.StatusInternalServerError,
			response:     dto.SecuritySettings{},
		},
		{
			name:   "getTLSsettingData - successful retrieval",
			url:    "/api/v1/amt/tls/valid-guid",
			method: http.MethodGet,
			mock: func(m *mocks.MockDeviceManagementFeature) {
				m.EXPECT().GetTLSSettingData(context.Background(), "valid-guid").
					Return([]dto.SettingDataResponse{}, nil)
			},
			expectedCode: http.StatusOK,
			response:     []dto.SettingDataResponse{},
		},
		{
			name:   "getTLSsettingData - failed retrieval",
			url:    "/api/v1/amt/tls/valid-guid",
			method: http.MethodGet,
			mock: func(m *mocks.MockDeviceManagementFeature) {
				m.EXPECT().GetTLSSettingData(context.Background(), "valid-guid").
					Return([]dto.SettingDataResponse{}, ErrGeneral)
			},
			expectedCode: http.StatusInternalServerError,
			response:     []dto.SettingDataResponse{},
		},
		{
			name:   "addCertificate - successful addition",
			url:    "/api/v1/amt/certificates/valid-guid",
			method: http.MethodPost,
			requestBody: dto.CertInfo{
				Cert:      "-----BEGIN CERTIFICATE-----\nMIIDtTM=\n-----END CERTIFICATE-----",
				IsTrusted: true,
			},
			mock: func(m *mocks.MockDeviceManagementFeature) {
				m.EXPECT().AddCertificate(
					context.Background(),
					"valid-guid",
					dto.CertInfo{
						Cert:      "-----BEGIN CERTIFICATE-----\nMIIDtTM=\n-----END CERTIFICATE-----",
						IsTrusted: true,
					},
				).Return("Intel(r) AMT Certificate: Handle: 0", nil)
			},
			expectedCode: http.StatusOK,
			response:     "Intel(r) AMT Certificate: Handle: 0",
		},
		{
			name:   "addCertificate - service failure",
			url:    "/api/v1/amt/certificates/valid-guid",
			method: http.MethodPost,
			requestBody: dto.CertInfo{
				Cert:      "-----BEGIN CERTIFICATE-----\nMIIDtTM=\n-----END CERTIFICATE-----",
				IsTrusted: true,
			},
			mock: func(m *mocks.MockDeviceManagementFeature) {
				m.EXPECT().AddCertificate(
					context.Background(),
					"valid-guid",
					dto.CertInfo{
						Cert:      "-----BEGIN CERTIFICATE-----\nMIIDtTM=\n-----END CERTIFICATE-----",
						IsTrusted: true,
					},
				).Return("", ErrGeneral)
			},
			expectedCode: http.StatusInternalServerError,
			response:     nil,
		},
		{
			name:   "addCertificate - missing required field",
			url:    "/api/v1/amt/certificates/valid-guid",
			method: http.MethodPost,
			requestBody: map[string]interface{}{
				"isTrusted": true,
			},
			mock: func(m *mocks.MockDeviceManagementFeature) {
				m.EXPECT().AddCertificate(
					context.Background(),
					"valid-guid",
					gomock.Any(),
				).Return("", ErrGeneral).AnyTimes()
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

			if tc.expectedCode == http.StatusOK || tc.expectedCode == http.StatusCreated {
				jsonBytes, _ := json.Marshal(tc.response)
				require.Equal(t, string(jsonBytes), w.Body.String())
			}
		})
	}
}
