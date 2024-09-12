// Code generated by MockGen. DO NOT EDIT.
// Source: ./internal/usecase/devices/interfaces.go
//
// Generated by this command:
//
//	mockgen -source ./internal/usecase/devices/interfaces.go -package devices_test
//

// Package devices_test is a generated GoMock package.
package devices_test

import (
	context "context"
	reflect "reflect"

	websocket "github.com/gorilla/websocket"
	entity "github.com/open-amt-cloud-toolkit/console/internal/entity"
	dto "github.com/open-amt-cloud-toolkit/console/internal/entity/dto"
	devices "github.com/open-amt-cloud-toolkit/console/internal/usecase/devices"
	wsman "github.com/open-amt-cloud-toolkit/console/internal/usecase/devices/wsman"
	wsman0 "github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman"
	power "github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/cim/power"
	gomock "go.uber.org/mock/gomock"
)

// MockWSMAN is a mock of WSMAN interface.
type MockWSMAN struct {
	ctrl     *gomock.Controller
	recorder *MockWSMANMockRecorder
}

// MockWSMANMockRecorder is the mock recorder for MockWSMAN.
type MockWSMANMockRecorder struct {
	mock *MockWSMAN
}

// NewMockWSMAN creates a new mock instance.
func NewMockWSMAN(ctrl *gomock.Controller) *MockWSMAN {
	mock := &MockWSMAN{ctrl: ctrl}
	mock.recorder = &MockWSMANMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockWSMAN) EXPECT() *MockWSMANMockRecorder {
	return m.recorder
}

// DestroyWsmanClient mocks base method.
func (m *MockWSMAN) DestroyWsmanClient(device dto.Device) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "DestroyWsmanClient", device)
}

// DestroyWsmanClient indicates an expected call of DestroyWsmanClient.
func (mr *MockWSMANMockRecorder) DestroyWsmanClient(device any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DestroyWsmanClient", reflect.TypeOf((*MockWSMAN)(nil).DestroyWsmanClient), device)
}

// SetupWsmanClient mocks base method.
func (m *MockWSMAN) SetupWsmanClient(device dto.Device, isRedirection, logMessages bool) wsman.Management {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetupWsmanClient", device, isRedirection, logMessages)
	ret0, _ := ret[0].(wsman.Management)
	return ret0
}

// SetupWsmanClient indicates an expected call of SetupWsmanClient.
func (mr *MockWSMANMockRecorder) SetupWsmanClient(device, isRedirection, logMessages any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetupWsmanClient", reflect.TypeOf((*MockWSMAN)(nil).SetupWsmanClient), device, isRedirection, logMessages)
}

// Worker mocks base method.
func (m *MockWSMAN) Worker() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Worker")
}

// Worker indicates an expected call of Worker.
func (mr *MockWSMANMockRecorder) Worker() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Worker", reflect.TypeOf((*MockWSMAN)(nil).Worker))
}

// MockWebSocketConn is a mock of WebSocketConn interface.
type MockWebSocketConn struct {
	ctrl     *gomock.Controller
	recorder *MockWebSocketConnMockRecorder
}

// MockWebSocketConnMockRecorder is the mock recorder for MockWebSocketConn.
type MockWebSocketConnMockRecorder struct {
	mock *MockWebSocketConn
}

// NewMockWebSocketConn creates a new mock instance.
func NewMockWebSocketConn(ctrl *gomock.Controller) *MockWebSocketConn {
	mock := &MockWebSocketConn{ctrl: ctrl}
	mock.recorder = &MockWebSocketConnMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockWebSocketConn) EXPECT() *MockWebSocketConnMockRecorder {
	return m.recorder
}

// Close mocks base method.
func (m *MockWebSocketConn) Close() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Close")
	ret0, _ := ret[0].(error)
	return ret0
}

// Close indicates an expected call of Close.
func (mr *MockWebSocketConnMockRecorder) Close() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Close", reflect.TypeOf((*MockWebSocketConn)(nil).Close))
}

// ReadMessage mocks base method.
func (m *MockWebSocketConn) ReadMessage() (int, []byte, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReadMessage")
	ret0, _ := ret[0].(int)
	ret1, _ := ret[1].([]byte)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// ReadMessage indicates an expected call of ReadMessage.
func (mr *MockWebSocketConnMockRecorder) ReadMessage() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReadMessage", reflect.TypeOf((*MockWebSocketConn)(nil).ReadMessage))
}

// WriteMessage mocks base method.
func (m *MockWebSocketConn) WriteMessage(messageType int, data []byte) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WriteMessage", messageType, data)
	ret0, _ := ret[0].(error)
	return ret0
}

// WriteMessage indicates an expected call of WriteMessage.
func (mr *MockWebSocketConnMockRecorder) WriteMessage(messageType, data any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WriteMessage", reflect.TypeOf((*MockWebSocketConn)(nil).WriteMessage), messageType, data)
}

// MockRedirection is a mock of Redirection interface.
type MockRedirection struct {
	ctrl     *gomock.Controller
	recorder *MockRedirectionMockRecorder
}

// MockRedirectionMockRecorder is the mock recorder for MockRedirection.
type MockRedirectionMockRecorder struct {
	mock *MockRedirection
}

// NewMockRedirection creates a new mock instance.
func NewMockRedirection(ctrl *gomock.Controller) *MockRedirection {
	mock := &MockRedirection{ctrl: ctrl}
	mock.recorder = &MockRedirectionMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockRedirection) EXPECT() *MockRedirectionMockRecorder {
	return m.recorder
}

// RedirectClose mocks base method.
func (m *MockRedirection) RedirectClose(ctx context.Context, deviceConnection *devices.DeviceConnection) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RedirectClose", ctx, deviceConnection)
	ret0, _ := ret[0].(error)
	return ret0
}

// RedirectClose indicates an expected call of RedirectClose.
func (mr *MockRedirectionMockRecorder) RedirectClose(ctx, deviceConnection any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RedirectClose", reflect.TypeOf((*MockRedirection)(nil).RedirectClose), ctx, deviceConnection)
}

// RedirectConnect mocks base method.
func (m *MockRedirection) RedirectConnect(ctx context.Context, deviceConnection *devices.DeviceConnection) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RedirectConnect", ctx, deviceConnection)
	ret0, _ := ret[0].(error)
	return ret0
}

// RedirectConnect indicates an expected call of RedirectConnect.
func (mr *MockRedirectionMockRecorder) RedirectConnect(ctx, deviceConnection any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RedirectConnect", reflect.TypeOf((*MockRedirection)(nil).RedirectConnect), ctx, deviceConnection)
}

// RedirectListen mocks base method.
func (m *MockRedirection) RedirectListen(ctx context.Context, deviceConnection *devices.DeviceConnection) ([]byte, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RedirectListen", ctx, deviceConnection)
	ret0, _ := ret[0].([]byte)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// RedirectListen indicates an expected call of RedirectListen.
func (mr *MockRedirectionMockRecorder) RedirectListen(ctx, deviceConnection any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RedirectListen", reflect.TypeOf((*MockRedirection)(nil).RedirectListen), ctx, deviceConnection)
}

// RedirectSend mocks base method.
func (m *MockRedirection) RedirectSend(ctx context.Context, deviceConnection *devices.DeviceConnection, message []byte) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RedirectSend", ctx, deviceConnection, message)
	ret0, _ := ret[0].(error)
	return ret0
}

// RedirectSend indicates an expected call of RedirectSend.
func (mr *MockRedirectionMockRecorder) RedirectSend(ctx, deviceConnection, message any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RedirectSend", reflect.TypeOf((*MockRedirection)(nil).RedirectSend), ctx, deviceConnection, message)
}

// SetupWsmanClient mocks base method.
func (m *MockRedirection) SetupWsmanClient(device dto.Device, isRedirection, logMessages bool) wsman0.Messages {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetupWsmanClient", device, isRedirection, logMessages)
	ret0, _ := ret[0].(wsman0.Messages)
	return ret0
}

// SetupWsmanClient indicates an expected call of SetupWsmanClient.
func (mr *MockRedirectionMockRecorder) SetupWsmanClient(device, isRedirection, logMessages any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetupWsmanClient", reflect.TypeOf((*MockRedirection)(nil).SetupWsmanClient), device, isRedirection, logMessages)
}

// MockRepository is a mock of Repository interface.
type MockRepository struct {
	ctrl     *gomock.Controller
	recorder *MockRepositoryMockRecorder
}

// MockRepositoryMockRecorder is the mock recorder for MockRepository.
type MockRepositoryMockRecorder struct {
	mock *MockRepository
}

// NewMockRepository creates a new mock instance.
func NewMockRepository(ctrl *gomock.Controller) *MockRepository {
	mock := &MockRepository{ctrl: ctrl}
	mock.recorder = &MockRepositoryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockRepository) EXPECT() *MockRepositoryMockRecorder {
	return m.recorder
}

// Delete mocks base method.
func (m *MockRepository) Delete(ctx context.Context, guid, tenantID string) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", ctx, guid, tenantID)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Delete indicates an expected call of Delete.
func (mr *MockRepositoryMockRecorder) Delete(ctx, guid, tenantID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockRepository)(nil).Delete), ctx, guid, tenantID)
}

// Get mocks base method.
func (m *MockRepository) Get(ctx context.Context, top, skip int, tenantID string) ([]entity.Device, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", ctx, top, skip, tenantID)
	ret0, _ := ret[0].([]entity.Device)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get.
func (mr *MockRepositoryMockRecorder) Get(ctx, top, skip, tenantID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockRepository)(nil).Get), ctx, top, skip, tenantID)
}

// GetByColumn mocks base method.
func (m *MockRepository) GetByColumn(ctx context.Context, columnName, queryValue, tenantID string) ([]entity.Device, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetByColumn", ctx, columnName, queryValue, tenantID)
	ret0, _ := ret[0].([]entity.Device)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetByColumn indicates an expected call of GetByColumn.
func (mr *MockRepositoryMockRecorder) GetByColumn(ctx, columnName, queryValue, tenantID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetByColumn", reflect.TypeOf((*MockRepository)(nil).GetByColumn), ctx, columnName, queryValue, tenantID)
}

// GetByID mocks base method.
func (m *MockRepository) GetByID(ctx context.Context, guid, tenantID string) (*entity.Device, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetByID", ctx, guid, tenantID)
	ret0, _ := ret[0].(*entity.Device)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetByID indicates an expected call of GetByID.
func (mr *MockRepositoryMockRecorder) GetByID(ctx, guid, tenantID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetByID", reflect.TypeOf((*MockRepository)(nil).GetByID), ctx, guid, tenantID)
}

// GetByTags mocks base method.
func (m *MockRepository) GetByTags(ctx context.Context, tags []string, method string, limit, offset int, tenantID string) ([]entity.Device, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetByTags", ctx, tags, method, limit, offset, tenantID)
	ret0, _ := ret[0].([]entity.Device)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetByTags indicates an expected call of GetByTags.
func (mr *MockRepositoryMockRecorder) GetByTags(ctx, tags, method, limit, offset, tenantID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetByTags", reflect.TypeOf((*MockRepository)(nil).GetByTags), ctx, tags, method, limit, offset, tenantID)
}

// GetCount mocks base method.
func (m *MockRepository) GetCount(arg0 context.Context, arg1 string) (int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCount", arg0, arg1)
	ret0, _ := ret[0].(int)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetCount indicates an expected call of GetCount.
func (mr *MockRepositoryMockRecorder) GetCount(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCount", reflect.TypeOf((*MockRepository)(nil).GetCount), arg0, arg1)
}

// GetDistinctTags mocks base method.
func (m *MockRepository) GetDistinctTags(ctx context.Context, tenantID string) ([]string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetDistinctTags", ctx, tenantID)
	ret0, _ := ret[0].([]string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetDistinctTags indicates an expected call of GetDistinctTags.
func (mr *MockRepositoryMockRecorder) GetDistinctTags(ctx, tenantID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetDistinctTags", reflect.TypeOf((*MockRepository)(nil).GetDistinctTags), ctx, tenantID)
}

// Insert mocks base method.
func (m *MockRepository) Insert(ctx context.Context, d *entity.Device) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Insert", ctx, d)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Insert indicates an expected call of Insert.
func (mr *MockRepositoryMockRecorder) Insert(ctx, d any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Insert", reflect.TypeOf((*MockRepository)(nil).Insert), ctx, d)
}

// Update mocks base method.
func (m *MockRepository) Update(ctx context.Context, d *entity.Device) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Update", ctx, d)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Update indicates an expected call of Update.
func (mr *MockRepositoryMockRecorder) Update(ctx, d any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Update", reflect.TypeOf((*MockRepository)(nil).Update), ctx, d)
}

// MockFeature is a mock of Feature interface.
type MockFeature struct {
	ctrl     *gomock.Controller
	recorder *MockFeatureMockRecorder
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

// CancelUserConsent mocks base method.
func (m *MockFeature) CancelUserConsent(ctx context.Context, guid string) (any, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CancelUserConsent", ctx, guid)
	ret0, _ := ret[0].(any)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CancelUserConsent indicates an expected call of CancelUserConsent.
func (mr *MockFeatureMockRecorder) CancelUserConsent(ctx, guid any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CancelUserConsent", reflect.TypeOf((*MockFeature)(nil).CancelUserConsent), ctx, guid)
}

// CreateAlarmOccurrences mocks base method.
func (m *MockFeature) CreateAlarmOccurrences(ctx context.Context, guid string, alarm dto.AlarmClockOccurrence) (dto.AddAlarmOutput, error) {
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
func (m *MockFeature) GetEventLog(ctx context.Context, guid string) ([]dto.EventLog, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetEventLog", ctx, guid)
	ret0, _ := ret[0].([]dto.EventLog)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetEventLog indicates an expected call of GetEventLog.
func (mr *MockFeatureMockRecorder) GetEventLog(ctx, guid any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetEventLog", reflect.TypeOf((*MockFeature)(nil).GetEventLog), ctx, guid)
}

// GetFeatures mocks base method.
func (m *MockFeature) GetFeatures(ctx context.Context, guid string) (dto.Features, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetFeatures", ctx, guid)
	ret0, _ := ret[0].(dto.Features)
	ret1, _ := ret[1].(error)
	return ret0, ret1
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

// GetPowerCapabilities mocks base method.
func (m *MockFeature) GetPowerCapabilities(ctx context.Context, guid string) (map[string]any, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetPowerCapabilities", ctx, guid)
	ret0, _ := ret[0].(map[string]any)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetPowerCapabilities indicates an expected call of GetPowerCapabilities.
func (mr *MockFeatureMockRecorder) GetPowerCapabilities(ctx, guid any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetPowerCapabilities", reflect.TypeOf((*MockFeature)(nil).GetPowerCapabilities), ctx, guid)
}

// GetPowerState mocks base method.
func (m *MockFeature) GetPowerState(ctx context.Context, guid string) (map[string]any, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetPowerState", ctx, guid)
	ret0, _ := ret[0].(map[string]any)
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
func (m *MockFeature) GetUserConsentCode(ctx context.Context, guid string) (map[string]any, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUserConsentCode", ctx, guid)
	ret0, _ := ret[0].(map[string]any)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetUserConsentCode indicates an expected call of GetUserConsentCode.
func (mr *MockFeatureMockRecorder) GetUserConsentCode(ctx, guid any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUserConsentCode", reflect.TypeOf((*MockFeature)(nil).GetUserConsentCode), ctx, guid)
}

// GetVersion mocks base method.
func (m *MockFeature) GetVersion(ctx context.Context, guid string) (map[string]any, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetVersion", ctx, guid)
	ret0, _ := ret[0].(map[string]any)
	ret1, _ := ret[1].(error)
	return ret0, ret1
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

// SendConsentCode mocks base method.
func (m *MockFeature) SendConsentCode(ctx context.Context, code dto.UserConsent, guid string) (any, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SendConsentCode", ctx, code, guid)
	ret0, _ := ret[0].(any)
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
func (m *MockFeature) SetFeatures(ctx context.Context, guid string, features dto.Features) (dto.Features, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetFeatures", ctx, guid, features)
	ret0, _ := ret[0].(dto.Features)
	ret1, _ := ret[1].(error)
	return ret0, ret1
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
