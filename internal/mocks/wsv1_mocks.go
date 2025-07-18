// Code generated by MockGen. DO NOT EDIT.
// Source: ./internal/controller/ws/v1/interface.go
//
// Generated by this command:
//
//	mockgen -source ./internal/controller/ws/v1/interface.go -package mocks
//

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	http "net/http"
	reflect "reflect"

	dto "github.com/device-management-toolkit/console/internal/entity/dto/v1"
	v2 "github.com/device-management-toolkit/console/internal/entity/dto/v2"
	power "github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/power"
	ipspower "github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/ips/power"
	gin "github.com/gin-gonic/gin"
	websocket "github.com/gorilla/websocket"
	gomock "go.uber.org/mock/gomock"
)

// MockUpgrader is a mock of Upgrader interface.
type MockUpgrader struct {
	ctrl     *gomock.Controller
	recorder *MockUpgraderMockRecorder
	isgomock struct{}
}

// MockUpgraderMockRecorder is the mock recorder for MockUpgrader.
type MockUpgraderMockRecorder struct {
	mock *MockUpgrader
}

// NewMockUpgrader creates a new mock instance.
func NewMockUpgrader(ctrl *gomock.Controller) *MockUpgrader {
	mock := &MockUpgrader{ctrl: ctrl}
	mock.recorder = &MockUpgraderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockUpgrader) EXPECT() *MockUpgraderMockRecorder {
	return m.recorder
}

// Upgrade mocks base method.
func (m *MockUpgrader) Upgrade(w http.ResponseWriter, r *http.Request, hdr http.Header) (*websocket.Conn, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Upgrade", w, r, hdr)
	ret0, _ := ret[0].(*websocket.Conn)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Upgrade indicates an expected call of Upgrade.
func (mr *MockUpgraderMockRecorder) Upgrade(w, r, hdr any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Upgrade", reflect.TypeOf((*MockUpgrader)(nil).Upgrade), w, r, hdr)
}

// MockRedirect is a mock of Redirect interface.
type MockRedirect struct {
	ctrl     *gomock.Controller
	recorder *MockRedirectMockRecorder
	isgomock struct{}
}

// MockRedirectMockRecorder is the mock recorder for MockRedirect.
type MockRedirectMockRecorder struct {
	mock *MockRedirect
}

// NewMockRedirect creates a new mock instance.
func NewMockRedirect(ctrl *gomock.Controller) *MockRedirect {
	mock := &MockRedirect{ctrl: ctrl}
	mock.recorder = &MockRedirectMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockRedirect) EXPECT() *MockRedirectMockRecorder {
	return m.recorder
}

// Redirect mocks base method.
func (m *MockRedirect) Redirect(c *gin.Context, conn *websocket.Conn, host, mode string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Redirect", c, conn, host, mode)
	ret0, _ := ret[0].(error)
	return ret0
}

// Redirect indicates an expected call of Redirect.
func (mr *MockRedirectMockRecorder) Redirect(c, conn, host, mode any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Redirect", reflect.TypeOf((*MockRedirect)(nil).Redirect), c, conn, host, mode)
}

// MockFeature is a mock of Feature interface.
type MockFeature struct {
	ctrl     *gomock.Controller
	recorder *MockFeatureMockRecorder
	isgomock struct{}
}

// MockFeatureMockRecorder is the mock recorder for MockFeature.
type MockFeatureMockRecorder struct {
	mock *MockFeature
}

// NewMockFeature creates a new mock instance.
func NewMockFeature(ctrl *gomock.Controller) *MockFeature {
	mock := &MockFeature{ctrl: ctrl}
	mock.recorder = &MockFeatureMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockFeature) EXPECT() *MockFeatureMockRecorder {
	return m.recorder
}

// AddCertificate mocks base method.
func (m *MockFeature) AddCertificate(c context.Context, guid string, certInfo dto.CertInfo) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AddCertificate", c, guid, certInfo)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// AddCertificate indicates an expected call of AddCertificate.
func (mr *MockFeatureMockRecorder) AddCertificate(c, guid, certInfo any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddCertificate", reflect.TypeOf((*MockFeature)(nil).AddCertificate), c, guid, certInfo)
}

// CancelUserConsent mocks base method.
func (m *MockFeature) CancelUserConsent(ctx context.Context, guid string) (dto.UserConsentMessage, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CancelUserConsent", ctx, guid)
	ret0, _ := ret[0].(dto.UserConsentMessage)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CancelUserConsent indicates an expected call of CancelUserConsent.
func (mr *MockFeatureMockRecorder) CancelUserConsent(ctx, guid any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CancelUserConsent", reflect.TypeOf((*MockFeature)(nil).CancelUserConsent), ctx, guid)
}

// CreateAlarmOccurrences mocks base method.
func (m *MockFeature) CreateAlarmOccurrences(ctx context.Context, guid string, alarm dto.AlarmClockOccurrenceInput) (dto.AddAlarmOutput, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateAlarmOccurrences", ctx, guid, alarm)
	ret0, _ := ret[0].(dto.AddAlarmOutput)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateAlarmOccurrences indicates an expected call of CreateAlarmOccurrences.
func (mr *MockFeatureMockRecorder) CreateAlarmOccurrences(ctx, guid, alarm any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateAlarmOccurrences", reflect.TypeOf((*MockFeature)(nil).CreateAlarmOccurrences), ctx, guid, alarm)
}

// Delete mocks base method.
func (m *MockFeature) Delete(ctx context.Context, guid, tenantID string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", ctx, guid, tenantID)
	ret0, _ := ret[0].(error)
	return ret0
}

// Delete indicates an expected call of Delete.
func (mr *MockFeatureMockRecorder) Delete(ctx, guid, tenantID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockFeature)(nil).Delete), ctx, guid, tenantID)
}

// DeleteAlarmOccurrences mocks base method.
func (m *MockFeature) DeleteAlarmOccurrences(ctx context.Context, guid, instanceID string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteAlarmOccurrences", ctx, guid, instanceID)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteAlarmOccurrences indicates an expected call of DeleteAlarmOccurrences.
func (mr *MockFeatureMockRecorder) DeleteAlarmOccurrences(ctx, guid, instanceID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteAlarmOccurrences", reflect.TypeOf((*MockFeature)(nil).DeleteAlarmOccurrences), ctx, guid, instanceID)
}

// Get mocks base method.
func (m *MockFeature) Get(ctx context.Context, top, skip int, tenantID string) ([]dto.Device, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", ctx, top, skip, tenantID)
	ret0, _ := ret[0].([]dto.Device)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get.
func (mr *MockFeatureMockRecorder) Get(ctx, top, skip, tenantID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockFeature)(nil).Get), ctx, top, skip, tenantID)
}

// GetAlarmOccurrences mocks base method.
func (m *MockFeature) GetAlarmOccurrences(ctx context.Context, guid string) ([]dto.AlarmClockOccurrence, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAlarmOccurrences", ctx, guid)
	ret0, _ := ret[0].([]dto.AlarmClockOccurrence)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetAlarmOccurrences indicates an expected call of GetAlarmOccurrences.
func (mr *MockFeatureMockRecorder) GetAlarmOccurrences(ctx, guid any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAlarmOccurrences", reflect.TypeOf((*MockFeature)(nil).GetAlarmOccurrences), ctx, guid)
}

// GetAuditLog mocks base method.
func (m *MockFeature) GetAuditLog(ctx context.Context, startIndex int, guid string) (dto.AuditLog, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAuditLog", ctx, startIndex, guid)
	ret0, _ := ret[0].(dto.AuditLog)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetAuditLog indicates an expected call of GetAuditLog.
func (mr *MockFeatureMockRecorder) GetAuditLog(ctx, startIndex, guid any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAuditLog", reflect.TypeOf((*MockFeature)(nil).GetAuditLog), ctx, startIndex, guid)
}

// GetByColumn mocks base method.
func (m *MockFeature) GetByColumn(ctx context.Context, columnName, queryValue, tenantID string) ([]dto.Device, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetByColumn", ctx, columnName, queryValue, tenantID)
	ret0, _ := ret[0].([]dto.Device)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetByColumn indicates an expected call of GetByColumn.
func (mr *MockFeatureMockRecorder) GetByColumn(ctx, columnName, queryValue, tenantID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetByColumn", reflect.TypeOf((*MockFeature)(nil).GetByColumn), ctx, columnName, queryValue, tenantID)
}

// GetByID mocks base method.
func (m *MockFeature) GetByID(ctx context.Context, guid, tenantID string) (*dto.Device, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetByID", ctx, guid, tenantID)
	ret0, _ := ret[0].(*dto.Device)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetByID indicates an expected call of GetByID.
func (mr *MockFeatureMockRecorder) GetByID(ctx, guid, tenantID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetByID", reflect.TypeOf((*MockFeature)(nil).GetByID), ctx, guid, tenantID)
}

// GetByTags mocks base method.
func (m *MockFeature) GetByTags(ctx context.Context, tags, method string, limit, offset int, tenantID string) ([]dto.Device, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetByTags", ctx, tags, method, limit, offset, tenantID)
	ret0, _ := ret[0].([]dto.Device)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetByTags indicates an expected call of GetByTags.
func (mr *MockFeatureMockRecorder) GetByTags(ctx, tags, method, limit, offset, tenantID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetByTags", reflect.TypeOf((*MockFeature)(nil).GetByTags), ctx, tags, method, limit, offset, tenantID)
}

// GetCertificates mocks base method.
func (m *MockFeature) GetCertificates(c context.Context, guid string) (dto.SecuritySettings, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCertificates", c, guid)
	ret0, _ := ret[0].(dto.SecuritySettings)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetCertificates indicates an expected call of GetCertificates.
func (mr *MockFeatureMockRecorder) GetCertificates(c, guid any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCertificates", reflect.TypeOf((*MockFeature)(nil).GetCertificates), c, guid)
}

// GetCount mocks base method.
func (m *MockFeature) GetCount(arg0 context.Context, arg1 string) (int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCount", arg0, arg1)
	ret0, _ := ret[0].(int)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetCount indicates an expected call of GetCount.
func (mr *MockFeatureMockRecorder) GetCount(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCount", reflect.TypeOf((*MockFeature)(nil).GetCount), arg0, arg1)
}

// GetDeviceCertificate mocks base method.
func (m *MockFeature) GetDeviceCertificate(c context.Context, guid string) (dto.Certificate, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetDeviceCertificate", c, guid)
	ret0, _ := ret[0].(dto.Certificate)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetDeviceCertificate indicates an expected call of GetDeviceCertificate.
func (mr *MockFeatureMockRecorder) GetDeviceCertificate(c, guid any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetDeviceCertificate", reflect.TypeOf((*MockFeature)(nil).GetDeviceCertificate), c, guid)
}

// GetDiskInfo mocks base method.
func (m *MockFeature) GetDiskInfo(c context.Context, guid string) (any, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetDiskInfo", c, guid)
	ret0, _ := ret[0].(any)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetDiskInfo indicates an expected call of GetDiskInfo.
func (mr *MockFeatureMockRecorder) GetDiskInfo(c, guid any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetDiskInfo", reflect.TypeOf((*MockFeature)(nil).GetDiskInfo), c, guid)
}

// GetDistinctTags mocks base method.
func (m *MockFeature) GetDistinctTags(ctx context.Context, tenantID string) ([]string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetDistinctTags", ctx, tenantID)
	ret0, _ := ret[0].([]string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetDistinctTags indicates an expected call of GetDistinctTags.
func (mr *MockFeatureMockRecorder) GetDistinctTags(ctx, tenantID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetDistinctTags", reflect.TypeOf((*MockFeature)(nil).GetDistinctTags), ctx, tenantID)
}

// GetEventLog mocks base method.
func (m *MockFeature) GetEventLog(ctx context.Context, startIndex, maxReadRecords int, guid string) (dto.EventLogs, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetEventLog", ctx, startIndex, maxReadRecords, guid)
	ret0, _ := ret[0].(dto.EventLogs)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetEventLog indicates an expected call of GetEventLog.
func (mr *MockFeatureMockRecorder) GetEventLog(ctx, startIndex, maxReadRecords, guid any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetEventLog", reflect.TypeOf((*MockFeature)(nil).GetEventLog), ctx, startIndex, maxReadRecords, guid)
}

// GetFeatures mocks base method.
func (m *MockFeature) GetFeatures(ctx context.Context, guid string) (dto.Features, v2.Features, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetFeatures", ctx, guid)
	ret0, _ := ret[0].(dto.Features)
	ret1, _ := ret[1].(v2.Features)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// GetFeatures indicates an expected call of GetFeatures.
func (mr *MockFeatureMockRecorder) GetFeatures(ctx, guid any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetFeatures", reflect.TypeOf((*MockFeature)(nil).GetFeatures), ctx, guid)
}

// GetGeneralSettings mocks base method.
func (m *MockFeature) GetGeneralSettings(ctx context.Context, guid string) (any, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetGeneralSettings", ctx, guid)
	ret0, _ := ret[0].(any)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetGeneralSettings indicates an expected call of GetGeneralSettings.
func (mr *MockFeatureMockRecorder) GetGeneralSettings(ctx, guid any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetGeneralSettings", reflect.TypeOf((*MockFeature)(nil).GetGeneralSettings), ctx, guid)
}

// GetHardwareInfo mocks base method.
func (m *MockFeature) GetHardwareInfo(ctx context.Context, guid string) (any, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetHardwareInfo", ctx, guid)
	ret0, _ := ret[0].(any)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetHardwareInfo indicates an expected call of GetHardwareInfo.
func (mr *MockFeatureMockRecorder) GetHardwareInfo(ctx, guid any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetHardwareInfo", reflect.TypeOf((*MockFeature)(nil).GetHardwareInfo), ctx, guid)
}

// GetIPSPowerManagementService mocks base method.
func (m *MockFeature) GetIPSPowerManagementService() (ipspower.PowerManagementService, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetIPSPowerManagementService")
	ret0, _ := ret[0].(ipspower.PowerManagementService)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetIPSPowerManagementService indicates an expected call of GetIPSPowerManagementService.
func (mr *MockFeatureMockRecorder) GetIPSPowerManagementService() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetIPSPowerManagementService", reflect.TypeOf((*MockFeature)(nil).GetIPSPowerManagementService))
}

// GetNetworkSettings mocks base method.
func (m *MockFeature) GetNetworkSettings(c context.Context, guid string) (dto.NetworkSettings, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetNetworkSettings", c, guid)
	ret0, _ := ret[0].(dto.NetworkSettings)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetNetworkSettings indicates an expected call of GetNetworkSettings.
func (mr *MockFeatureMockRecorder) GetNetworkSettings(c, guid any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetNetworkSettings", reflect.TypeOf((*MockFeature)(nil).GetNetworkSettings), c, guid)
}

// GetOSPowerSavingState mocks base method.
func (m *MockFeature) GetOSPowerSavingState() (ipspower.OSPowerSavingState, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetOSPowerSavingState")
	ret0, _ := ret[0].(ipspower.OSPowerSavingState)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetOSPowerSavingState indicates an expected call of GetOSPowerSavingState.
func (mr *MockFeatureMockRecorder) GetOSPowerSavingState() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetOSPowerSavingState", reflect.TypeOf((*MockFeature)(nil).GetOSPowerSavingState))
}

// GetPowerCapabilities mocks base method.
func (m *MockFeature) GetPowerCapabilities(ctx context.Context, guid string) (dto.PowerCapabilities, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetPowerCapabilities", ctx, guid)
	ret0, _ := ret[0].(dto.PowerCapabilities)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetPowerCapabilities indicates an expected call of GetPowerCapabilities.
func (mr *MockFeatureMockRecorder) GetPowerCapabilities(ctx, guid any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetPowerCapabilities", reflect.TypeOf((*MockFeature)(nil).GetPowerCapabilities), ctx, guid)
}

// GetPowerState mocks base method.
func (m *MockFeature) GetPowerState(ctx context.Context, guid string) (dto.PowerState, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetPowerState", ctx, guid)
	ret0, _ := ret[0].(dto.PowerState)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetPowerState indicates an expected call of GetPowerState.
func (mr *MockFeatureMockRecorder) GetPowerState(ctx, guid any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetPowerState", reflect.TypeOf((*MockFeature)(nil).GetPowerState), ctx, guid)
}

// GetTLSSettingData mocks base method.
func (m *MockFeature) GetTLSSettingData(c context.Context, guid string) ([]dto.SettingDataResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTLSSettingData", c, guid)
	ret0, _ := ret[0].([]dto.SettingDataResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetTLSSettingData indicates an expected call of GetTLSSettingData.
func (mr *MockFeatureMockRecorder) GetTLSSettingData(c, guid any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTLSSettingData", reflect.TypeOf((*MockFeature)(nil).GetTLSSettingData), c, guid)
}

// GetUserConsentCode mocks base method.
func (m *MockFeature) GetUserConsentCode(ctx context.Context, guid string) (dto.GetUserConsentMessage, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUserConsentCode", ctx, guid)
	ret0, _ := ret[0].(dto.GetUserConsentMessage)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetUserConsentCode indicates an expected call of GetUserConsentCode.
func (mr *MockFeatureMockRecorder) GetUserConsentCode(ctx, guid any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUserConsentCode", reflect.TypeOf((*MockFeature)(nil).GetUserConsentCode), ctx, guid)
}

// GetVersion mocks base method.
func (m *MockFeature) GetVersion(ctx context.Context, guid string) (dto.Version, v2.Version, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetVersion", ctx, guid)
	ret0, _ := ret[0].(dto.Version)
	ret1, _ := ret[1].(v2.Version)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// GetVersion indicates an expected call of GetVersion.
func (mr *MockFeatureMockRecorder) GetVersion(ctx, guid any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetVersion", reflect.TypeOf((*MockFeature)(nil).GetVersion), ctx, guid)
}

// Insert mocks base method.
func (m *MockFeature) Insert(ctx context.Context, d *dto.Device) (*dto.Device, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Insert", ctx, d)
	ret0, _ := ret[0].(*dto.Device)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Insert indicates an expected call of Insert.
func (mr *MockFeatureMockRecorder) Insert(ctx, d any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Insert", reflect.TypeOf((*MockFeature)(nil).Insert), ctx, d)
}

// Redirect mocks base method.
func (m *MockFeature) Redirect(ctx context.Context, conn *websocket.Conn, guid, mode string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Redirect", ctx, conn, guid, mode)
	ret0, _ := ret[0].(error)
	return ret0
}

// Redirect indicates an expected call of Redirect.
func (mr *MockFeatureMockRecorder) Redirect(ctx, conn, guid, mode any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Redirect", reflect.TypeOf((*MockFeature)(nil).Redirect), ctx, conn, guid, mode)
}

// RequestOSPowerSavingStateChange mocks base method.
func (m *MockFeature) RequestOSPowerSavingStateChange(osPowerSavingState ipspower.OSPowerSavingState) (ipspower.PowerActionResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RequestOSPowerSavingStateChange", osPowerSavingState)
	ret0, _ := ret[0].(ipspower.PowerActionResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// RequestOSPowerSavingStateChange indicates an expected call of RequestOSPowerSavingStateChange.
func (mr *MockFeatureMockRecorder) RequestOSPowerSavingStateChange(osPowerSavingState interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RequestOSPowerSavingStateChange", reflect.TypeOf((*MockFeature)(nil).RequestOSPowerSavingStateChange), osPowerSavingState)
}

// SendConsentCode mocks base method.
func (m *MockFeature) SendConsentCode(ctx context.Context, code dto.UserConsentCode, guid string) (dto.UserConsentMessage, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SendConsentCode", ctx, code, guid)
	ret0, _ := ret[0].(dto.UserConsentMessage)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// SendConsentCode indicates an expected call of SendConsentCode.
func (mr *MockFeatureMockRecorder) SendConsentCode(ctx, code, guid any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendConsentCode", reflect.TypeOf((*MockFeature)(nil).SendConsentCode), ctx, code, guid)
}

// SendPowerAction mocks base method.
func (m *MockFeature) SendPowerAction(ctx context.Context, guid string, action int) (power.PowerActionResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SendPowerAction", ctx, guid, action)
	ret0, _ := ret[0].(power.PowerActionResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// SendPowerAction indicates an expected call of SendPowerAction.
func (mr *MockFeatureMockRecorder) SendPowerAction(ctx, guid, action any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendPowerAction", reflect.TypeOf((*MockFeature)(nil).SendPowerAction), ctx, guid, action)
}

// SetBootOptions mocks base method.
func (m *MockFeature) SetBootOptions(ctx context.Context, guid string, bootSetting dto.BootSetting) (power.PowerActionResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetBootOptions", ctx, guid, bootSetting)
	ret0, _ := ret[0].(power.PowerActionResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// SetBootOptions indicates an expected call of SetBootOptions.
func (mr *MockFeatureMockRecorder) SetBootOptions(ctx, guid, bootSetting any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetBootOptions", reflect.TypeOf((*MockFeature)(nil).SetBootOptions), ctx, guid, bootSetting)
}

// SetFeatures mocks base method.
func (m *MockFeature) SetFeatures(ctx context.Context, guid string, features dto.Features) (dto.Features, v2.Features, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetFeatures", ctx, guid, features)
	ret0, _ := ret[0].(dto.Features)
	ret1, _ := ret[1].(v2.Features)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// SetFeatures indicates an expected call of SetFeatures.
func (mr *MockFeatureMockRecorder) SetFeatures(ctx, guid, features any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetFeatures", reflect.TypeOf((*MockFeature)(nil).SetFeatures), ctx, guid, features)
}

// Update mocks base method.
func (m *MockFeature) Update(ctx context.Context, d *dto.Device) (*dto.Device, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Update", ctx, d)
	ret0, _ := ret[0].(*dto.Device)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Update indicates an expected call of Update.
func (mr *MockFeatureMockRecorder) Update(ctx, d any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Update", reflect.TypeOf((*MockFeature)(nil).Update), ctx, d)
}
