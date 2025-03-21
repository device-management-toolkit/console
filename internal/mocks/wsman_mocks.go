// Code generated by MockGen. DO NOT EDIT.
// Source: ./internal/usecase/devices/wsman/interfaces.go
//
// Generated by this command:
//
//	mockgen -source ./internal/usecase/devices/wsman/interfaces.go -package mocks
//

// Package mocks is a generated GoMock package.
package mocks

import (
	tls "crypto/tls"
	reflect "reflect"
	time "time"

	dto "github.com/open-amt-cloud-toolkit/console/internal/entity/dto/v1"
	wsman "github.com/open-amt-cloud-toolkit/console/internal/usecase/devices/wsman"
	alarmclock "github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/amt/alarmclock"
	auditlog "github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/amt/auditlog"
	boot "github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/amt/boot"
	messagelog "github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/amt/messagelog"
	redirection "github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/amt/redirection"
	setupandconfiguration "github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/amt/setupandconfiguration"
	tls0 "github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/amt/tls"
	boot0 "github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/cim/boot"
	concrete "github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/cim/concrete"
	credential "github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/cim/credential"
	kvm "github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/cim/kvm"
	power "github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/cim/power"
	service "github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/cim/service"
	software "github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/cim/software"
	alarmclock0 "github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/ips/alarmclock"
	optin "github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/ips/optin"
	gomock "go.uber.org/mock/gomock"
)

// MockManagement is a mock of Management interface.
type MockManagement struct {
	ctrl     *gomock.Controller
	recorder *MockManagementMockRecorder
	isgomock struct{}
}

// MockManagementMockRecorder is the mock recorder for MockManagement.
type MockManagementMockRecorder struct {
	mock *MockManagement
}

// NewMockManagement creates a new mock instance.
func NewMockManagement(ctrl *gomock.Controller) *MockManagement {
	mock := &MockManagement{ctrl: ctrl}
	mock.recorder = &MockManagementMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockManagement) EXPECT() *MockManagementMockRecorder {
	return m.recorder
}

// BootServiceStateChange mocks base method.
func (m *MockManagement) BootServiceStateChange(requestedState int) (boot0.BootService, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BootServiceStateChange", requestedState)
	ret0, _ := ret[0].(boot0.BootService)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// BootServiceStateChange indicates an expected call of BootServiceStateChange.
func (mr *MockManagementMockRecorder) BootServiceStateChange(requestedState any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BootServiceStateChange", reflect.TypeOf((*MockManagement)(nil).BootServiceStateChange), requestedState)
}

// CancelUserConsentRequest mocks base method.
func (m *MockManagement) CancelUserConsentRequest() (dto.UserConsentMessage, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CancelUserConsentRequest")
	ret0, _ := ret[0].(dto.UserConsentMessage)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CancelUserConsentRequest indicates an expected call of CancelUserConsentRequest.
func (mr *MockManagementMockRecorder) CancelUserConsentRequest() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CancelUserConsentRequest", reflect.TypeOf((*MockManagement)(nil).CancelUserConsentRequest))
}

// ChangeBootOrder mocks base method.
func (m *MockManagement) ChangeBootOrder(bootSource string) (boot0.ChangeBootOrder_OUTPUT, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ChangeBootOrder", bootSource)
	ret0, _ := ret[0].(boot0.ChangeBootOrder_OUTPUT)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ChangeBootOrder indicates an expected call of ChangeBootOrder.
func (mr *MockManagementMockRecorder) ChangeBootOrder(bootSource any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ChangeBootOrder", reflect.TypeOf((*MockManagement)(nil).ChangeBootOrder), bootSource)
}

// CreateAlarmOccurrences mocks base method.
func (m *MockManagement) CreateAlarmOccurrences(name string, startTime time.Time, interval int, deleteOnCompletion bool) (alarmclock.AddAlarmOutput, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateAlarmOccurrences", name, startTime, interval, deleteOnCompletion)
	ret0, _ := ret[0].(alarmclock.AddAlarmOutput)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateAlarmOccurrences indicates an expected call of CreateAlarmOccurrences.
func (mr *MockManagementMockRecorder) CreateAlarmOccurrences(name, startTime, interval, deleteOnCompletion any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateAlarmOccurrences", reflect.TypeOf((*MockManagement)(nil).CreateAlarmOccurrences), name, startTime, interval, deleteOnCompletion)
}

// DeleteAlarmOccurrences mocks base method.
func (m *MockManagement) DeleteAlarmOccurrences(instanceID string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteAlarmOccurrences", instanceID)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteAlarmOccurrences indicates an expected call of DeleteAlarmOccurrences.
func (mr *MockManagementMockRecorder) DeleteAlarmOccurrences(instanceID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteAlarmOccurrences", reflect.TypeOf((*MockManagement)(nil).DeleteAlarmOccurrences), instanceID)
}

// GetAMTRedirectionService mocks base method.
func (m *MockManagement) GetAMTRedirectionService() (redirection.Response, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAMTRedirectionService")
	ret0, _ := ret[0].(redirection.Response)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetAMTRedirectionService indicates an expected call of GetAMTRedirectionService.
func (mr *MockManagementMockRecorder) GetAMTRedirectionService() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAMTRedirectionService", reflect.TypeOf((*MockManagement)(nil).GetAMTRedirectionService))
}

// GetAMTVersion mocks base method.
func (m *MockManagement) GetAMTVersion() ([]software.SoftwareIdentity, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAMTVersion")
	ret0, _ := ret[0].([]software.SoftwareIdentity)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetAMTVersion indicates an expected call of GetAMTVersion.
func (mr *MockManagementMockRecorder) GetAMTVersion() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAMTVersion", reflect.TypeOf((*MockManagement)(nil).GetAMTVersion))
}

// GetAlarmOccurrences mocks base method.
func (m *MockManagement) GetAlarmOccurrences() ([]alarmclock0.AlarmClockOccurrence, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAlarmOccurrences")
	ret0, _ := ret[0].([]alarmclock0.AlarmClockOccurrence)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetAlarmOccurrences indicates an expected call of GetAlarmOccurrences.
func (mr *MockManagementMockRecorder) GetAlarmOccurrences() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAlarmOccurrences", reflect.TypeOf((*MockManagement)(nil).GetAlarmOccurrences))
}

// GetAuditLog mocks base method.
func (m *MockManagement) GetAuditLog(startIndex int) (auditlog.Response, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAuditLog", startIndex)
	ret0, _ := ret[0].(auditlog.Response)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetAuditLog indicates an expected call of GetAuditLog.
func (mr *MockManagementMockRecorder) GetAuditLog(startIndex any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAuditLog", reflect.TypeOf((*MockManagement)(nil).GetAuditLog), startIndex)
}

// GetBootData mocks base method.
func (m *MockManagement) GetBootData() (boot.BootSettingDataResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBootData")
	ret0, _ := ret[0].(boot.BootSettingDataResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetBootData indicates an expected call of GetBootData.
func (mr *MockManagementMockRecorder) GetBootData() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBootData", reflect.TypeOf((*MockManagement)(nil).GetBootData))
}

// GetBootService mocks base method.
func (m *MockManagement) GetBootService() (boot0.BootService, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBootService")
	ret0, _ := ret[0].(boot0.BootService)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetBootService indicates an expected call of GetBootService.
func (mr *MockManagementMockRecorder) GetBootService() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBootService", reflect.TypeOf((*MockManagement)(nil).GetBootService))
}

// GetCIMBootSourceSetting mocks base method.
func (m *MockManagement) GetCIMBootSourceSetting() (boot0.Response, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCIMBootSourceSetting")
	ret0, _ := ret[0].(boot0.Response)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetCIMBootSourceSetting indicates an expected call of GetCIMBootSourceSetting.
func (mr *MockManagementMockRecorder) GetCIMBootSourceSetting() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCIMBootSourceSetting", reflect.TypeOf((*MockManagement)(nil).GetCIMBootSourceSetting))
}

// GetCertificates mocks base method.
func (m *MockManagement) GetCertificates() (wsman.Certificates, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCertificates")
	ret0, _ := ret[0].(wsman.Certificates)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetCertificates indicates an expected call of GetCertificates.
func (mr *MockManagementMockRecorder) GetCertificates() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCertificates", reflect.TypeOf((*MockManagement)(nil).GetCertificates))
}

// GetConcreteDependencies mocks base method.
func (m *MockManagement) GetConcreteDependencies() ([]concrete.ConcreteDependency, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetConcreteDependencies")
	ret0, _ := ret[0].([]concrete.ConcreteDependency)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetConcreteDependencies indicates an expected call of GetConcreteDependencies.
func (mr *MockManagementMockRecorder) GetConcreteDependencies() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetConcreteDependencies", reflect.TypeOf((*MockManagement)(nil).GetConcreteDependencies))
}

// GetCredentialRelationships mocks base method.
func (m *MockManagement) GetCredentialRelationships() (credential.Items, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCredentialRelationships")
	ret0, _ := ret[0].(credential.Items)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetCredentialRelationships indicates an expected call of GetCredentialRelationships.
func (mr *MockManagementMockRecorder) GetCredentialRelationships() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCredentialRelationships", reflect.TypeOf((*MockManagement)(nil).GetCredentialRelationships))
}

// GetDeviceCertificate mocks base method.
func (m *MockManagement) GetDeviceCertificate() (*tls.Certificate, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetDeviceCertificate")
	ret0, _ := ret[0].(*tls.Certificate)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetDeviceCertificate indicates an expected call of GetDeviceCertificate.
func (mr *MockManagementMockRecorder) GetDeviceCertificate() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetDeviceCertificate", reflect.TypeOf((*MockManagement)(nil).GetDeviceCertificate))
}

// GetDiskInfo mocks base method.
func (m *MockManagement) GetDiskInfo() (any, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetDiskInfo")
	ret0, _ := ret[0].(any)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetDiskInfo indicates an expected call of GetDiskInfo.
func (mr *MockManagementMockRecorder) GetDiskInfo() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetDiskInfo", reflect.TypeOf((*MockManagement)(nil).GetDiskInfo))
}

// GetEventLog mocks base method.
func (m *MockManagement) GetEventLog(startIndex, maxReadRecords int) (messagelog.GetRecordsResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetEventLog", startIndex, maxReadRecords)
	ret0, _ := ret[0].(messagelog.GetRecordsResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetEventLog indicates an expected call of GetEventLog.
func (mr *MockManagementMockRecorder) GetEventLog(startIndex, maxReadRecords any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetEventLog", reflect.TypeOf((*MockManagement)(nil).GetEventLog), startIndex, maxReadRecords)
}

// GetGeneralSettings mocks base method.
func (m *MockManagement) GetGeneralSettings() (any, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetGeneralSettings")
	ret0, _ := ret[0].(any)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetGeneralSettings indicates an expected call of GetGeneralSettings.
func (mr *MockManagementMockRecorder) GetGeneralSettings() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetGeneralSettings", reflect.TypeOf((*MockManagement)(nil).GetGeneralSettings))
}

// GetHardwareInfo mocks base method.
func (m *MockManagement) GetHardwareInfo() (wsman.HWResults, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetHardwareInfo")
	ret0, _ := ret[0].(wsman.HWResults)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetHardwareInfo indicates an expected call of GetHardwareInfo.
func (mr *MockManagementMockRecorder) GetHardwareInfo() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetHardwareInfo", reflect.TypeOf((*MockManagement)(nil).GetHardwareInfo))
}

// GetIPSOptInService mocks base method.
func (m *MockManagement) GetIPSOptInService() (optin.Response, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetIPSOptInService")
	ret0, _ := ret[0].(optin.Response)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetIPSOptInService indicates an expected call of GetIPSOptInService.
func (mr *MockManagementMockRecorder) GetIPSOptInService() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetIPSOptInService", reflect.TypeOf((*MockManagement)(nil).GetIPSOptInService))
}

// GetKVMRedirection mocks base method.
func (m *MockManagement) GetKVMRedirection() (kvm.Response, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetKVMRedirection")
	ret0, _ := ret[0].(kvm.Response)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetKVMRedirection indicates an expected call of GetKVMRedirection.
func (mr *MockManagementMockRecorder) GetKVMRedirection() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetKVMRedirection", reflect.TypeOf((*MockManagement)(nil).GetKVMRedirection))
}

// GetNetworkSettings mocks base method.
func (m *MockManagement) GetNetworkSettings() (wsman.NetworkResults, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetNetworkSettings")
	ret0, _ := ret[0].(wsman.NetworkResults)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetNetworkSettings indicates an expected call of GetNetworkSettings.
func (mr *MockManagementMockRecorder) GetNetworkSettings() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetNetworkSettings", reflect.TypeOf((*MockManagement)(nil).GetNetworkSettings))
}

// GetPowerCapabilities mocks base method.
func (m *MockManagement) GetPowerCapabilities() (boot.BootCapabilitiesResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetPowerCapabilities")
	ret0, _ := ret[0].(boot.BootCapabilitiesResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetPowerCapabilities indicates an expected call of GetPowerCapabilities.
func (mr *MockManagementMockRecorder) GetPowerCapabilities() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetPowerCapabilities", reflect.TypeOf((*MockManagement)(nil).GetPowerCapabilities))
}

// GetPowerState mocks base method.
func (m *MockManagement) GetPowerState() ([]service.CIM_AssociatedPowerManagementService, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetPowerState")
	ret0, _ := ret[0].([]service.CIM_AssociatedPowerManagementService)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetPowerState indicates an expected call of GetPowerState.
func (mr *MockManagementMockRecorder) GetPowerState() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetPowerState", reflect.TypeOf((*MockManagement)(nil).GetPowerState))
}

// GetSetupAndConfiguration mocks base method.
func (m *MockManagement) GetSetupAndConfiguration() ([]setupandconfiguration.SetupAndConfigurationServiceResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetSetupAndConfiguration")
	ret0, _ := ret[0].([]setupandconfiguration.SetupAndConfigurationServiceResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetSetupAndConfiguration indicates an expected call of GetSetupAndConfiguration.
func (mr *MockManagementMockRecorder) GetSetupAndConfiguration() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSetupAndConfiguration", reflect.TypeOf((*MockManagement)(nil).GetSetupAndConfiguration))
}

// GetTLSSettingData mocks base method.
func (m *MockManagement) GetTLSSettingData() ([]tls0.SettingDataResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTLSSettingData")
	ret0, _ := ret[0].([]tls0.SettingDataResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetTLSSettingData indicates an expected call of GetTLSSettingData.
func (mr *MockManagementMockRecorder) GetTLSSettingData() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTLSSettingData", reflect.TypeOf((*MockManagement)(nil).GetTLSSettingData))
}

// GetUserConsentCode mocks base method.
func (m *MockManagement) GetUserConsentCode() (optin.StartOptIn_OUTPUT, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUserConsentCode")
	ret0, _ := ret[0].(optin.StartOptIn_OUTPUT)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetUserConsentCode indicates an expected call of GetUserConsentCode.
func (mr *MockManagementMockRecorder) GetUserConsentCode() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUserConsentCode", reflect.TypeOf((*MockManagement)(nil).GetUserConsentCode))
}

// RequestAMTRedirectionServiceStateChange mocks base method.
func (m *MockManagement) RequestAMTRedirectionServiceStateChange(ider, sol bool) (redirection.RequestedState, int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RequestAMTRedirectionServiceStateChange", ider, sol)
	ret0, _ := ret[0].(redirection.RequestedState)
	ret1, _ := ret[1].(int)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// RequestAMTRedirectionServiceStateChange indicates an expected call of RequestAMTRedirectionServiceStateChange.
func (mr *MockManagementMockRecorder) RequestAMTRedirectionServiceStateChange(ider, sol any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RequestAMTRedirectionServiceStateChange", reflect.TypeOf((*MockManagement)(nil).RequestAMTRedirectionServiceStateChange), ider, sol)
}

// SendConsentCode mocks base method.
func (m *MockManagement) SendConsentCode(code int) (dto.UserConsentMessage, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SendConsentCode", code)
	ret0, _ := ret[0].(dto.UserConsentMessage)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// SendConsentCode indicates an expected call of SendConsentCode.
func (mr *MockManagementMockRecorder) SendConsentCode(code any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendConsentCode", reflect.TypeOf((*MockManagement)(nil).SendConsentCode), code)
}

// SendPowerAction mocks base method.
func (m *MockManagement) SendPowerAction(action int) (power.PowerActionResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SendPowerAction", action)
	ret0, _ := ret[0].(power.PowerActionResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// SendPowerAction indicates an expected call of SendPowerAction.
func (mr *MockManagementMockRecorder) SendPowerAction(action any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendPowerAction", reflect.TypeOf((*MockManagement)(nil).SendPowerAction), action)
}

// SetAMTRedirectionService mocks base method.
func (m *MockManagement) SetAMTRedirectionService(arg0 redirection.RedirectionRequest) (redirection.Response, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetAMTRedirectionService", arg0)
	ret0, _ := ret[0].(redirection.Response)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// SetAMTRedirectionService indicates an expected call of SetAMTRedirectionService.
func (mr *MockManagementMockRecorder) SetAMTRedirectionService(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetAMTRedirectionService", reflect.TypeOf((*MockManagement)(nil).SetAMTRedirectionService), arg0)
}

// SetBootConfigRole mocks base method.
func (m *MockManagement) SetBootConfigRole(role int) (any, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetBootConfigRole", role)
	ret0, _ := ret[0].(any)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// SetBootConfigRole indicates an expected call of SetBootConfigRole.
func (mr *MockManagementMockRecorder) SetBootConfigRole(role any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetBootConfigRole", reflect.TypeOf((*MockManagement)(nil).SetBootConfigRole), role)
}

// SetBootData mocks base method.
func (m *MockManagement) SetBootData(data boot.BootSettingDataRequest) (any, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetBootData", data)
	ret0, _ := ret[0].(any)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// SetBootData indicates an expected call of SetBootData.
func (mr *MockManagementMockRecorder) SetBootData(data any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetBootData", reflect.TypeOf((*MockManagement)(nil).SetBootData), data)
}

// SetIPSOptInService mocks base method.
func (m *MockManagement) SetIPSOptInService(arg0 optin.OptInServiceRequest) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetIPSOptInService", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetIPSOptInService indicates an expected call of SetIPSOptInService.
func (mr *MockManagementMockRecorder) SetIPSOptInService(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetIPSOptInService", reflect.TypeOf((*MockManagement)(nil).SetIPSOptInService), arg0)
}

// SetKVMRedirection mocks base method.
func (m *MockManagement) SetKVMRedirection(enable bool) (int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetKVMRedirection", enable)
	ret0, _ := ret[0].(int)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// SetKVMRedirection indicates an expected call of SetKVMRedirection.
func (mr *MockManagementMockRecorder) SetKVMRedirection(enable any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetKVMRedirection", reflect.TypeOf((*MockManagement)(nil).SetKVMRedirection), enable)
}
