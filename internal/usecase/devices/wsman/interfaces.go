package wsman

import (
	gotls "crypto/tls"
	"time"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/alarmclock"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/auditlog"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/boot"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/messagelog"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/redirection"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/setupandconfiguration"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/tls"
	cimBoot "github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/boot"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/concrete"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/credential"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/kvm"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/power"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/service"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/software"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/wifi"
	ipsAlarmClock "github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/ips/alarmclock"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/ips/kvmredirection"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/ips/optin"
	ipspower "github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/ips/power"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/ips/screensetting"
)

type Management interface {
	AddTrustedRootCert(caCert string) (string, error)
	AddClientCert(clientCert string) (string, error)
	GetAMTVersion() ([]software.SoftwareIdentity, error)
	GetSetupAndConfiguration() ([]setupandconfiguration.SetupAndConfigurationServiceResponse, error)
	GetAMTRedirectionService() (redirection.Response, error)
	SetAMTRedirectionService(*redirection.RedirectionRequest) (redirection.Response, error)
	RequestAMTRedirectionServiceStateChange(ider, sol bool) (redirection.RequestedState, int, error)
	GetIPSOptInService() (optin.Response, error)
	SetIPSOptInService(optin.OptInServiceRequest) error
	GetKVMRedirection() (kvm.Response, error)
	SetKVMRedirection(enable bool) (int, error)
	GetAlarmOccurrences() ([]ipsAlarmClock.AlarmClockOccurrence, error)
	CreateAlarmOccurrences(name string, startTime time.Time, interval int, deleteOnCompletion bool) (alarmclock.AddAlarmOutput, error)
	DeleteAlarmOccurrences(instanceID string) error
	GetHardwareInfo() (HWResults, error)
	GetPowerState() ([]service.CIM_AssociatedPowerManagementService, error)
	GetOSPowerSavingState() (ipspower.OSPowerSavingState, error)
	GetIPSPowerManagementService() (ipspower.PowerManagementService, error)
	RequestOSPowerSavingStateChange(osPowerSavingState ipspower.OSPowerSavingState) (ipspower.PowerActionResponse, error)
	GetBootCapabilities() (boot.BootCapabilitiesResponse, error)
	GetGeneralSettings() (interface{}, error)
	CancelUserConsentRequest() (optin.Response, error)
	GetUserConsentCode() (optin.Response, error)
	SendConsentCode(code int) (optin.Response, error)
	SendPowerAction(action int) (power.PowerActionResponse, error)
	GetBootData() (boot.BootSettingDataResponse, error)
	SetBootData(data boot.BootSettingDataRequest) (interface{}, error)
	GetBootService() (cimBoot.BootService, error)
	SetBootConfigRole(role int) (interface{}, error)
	ChangeBootOrder(bootSource string) (cimBoot.ChangeBootOrder_OUTPUT, error)
	GetAuditLog(startIndex int) (auditlog.Response, error)
	GetEventLog(startIndex, maxReadRecords int) (messagelog.GetRecordsResponse, error)
	GetNetworkSettings() (NetworkResults, error)
	EnumerateWiFiPort() (wifi.Response, error)
	PullWiFiPort(enumerationContext string) (wifi.Response, error)
	WiFiRequestStateChange(requestedState wifi.RequestedState) error
	GetCertificates() (Certificates, error)
	GetTLSSettingData() ([]tls.SettingDataResponse, error)
	GetCredentialRelationships() (credential.Items, error)
	GetConcreteDependencies() ([]concrete.ConcreteDependency, error)
	GetDiskInfo() (DiskResults, error)
	GetDeviceCertificate() (*gotls.Certificate, error)
	GetCIMBootSourceSetting() (cimBoot.Response, error)
	BootServiceStateChange(requestedState int) (cimBoot.BootService, error)
	GetIPSScreenSettingData() (screensetting.Response, error)
	GetIPSKVMRedirectionSettingData() (kvmredirection.Response, error)
	SetIPSKVMRedirectionSettingData(data *kvmredirection.KVMRedirectionSettingsRequest) (kvmredirection.Response, error)
	DeleteCertificate(instanceID string) error
	SetLinkPreference(linkPreference, timeout uint32) (int, error)
	SetRemoteEraseOptions(eraseMask int) error
}
