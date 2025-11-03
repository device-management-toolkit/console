package wsman

import (
	gotls "crypto/tls"
	"sync"
	"time"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/security"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman"
	amtAlarmClock "github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/alarmclock"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/auditlog"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/authorization"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/boot"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/environmentdetection"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/ethernetport"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/general"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/ieee8021x"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/kerberos"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/managementpresence"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/messagelog"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/mps"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/publickey"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/publicprivate"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/redirection"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/remoteaccess"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/setupandconfiguration"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/timesynchronization"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/tls"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/userinitiatedconnection"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/wifiportconfiguration"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/bios"
	cimBoot "github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/boot"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/card"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/chassis"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/chip"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/computer"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/concrete"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/credential"
	cimIEEE8021x "github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/ieee8021x"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/kvm"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/mediaaccess"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/models"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/physical"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/power"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/processor"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/service"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/software"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/system"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/wifi"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/client"
	ipsAlarmClock "github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/ips/alarmclock"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/ips/hostbasedsetup"
	ipsIEEE8021x "github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/ips/ieee8021x"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/ips/kvmredirection"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/ips/optin"
	ipspower "github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/ips/power"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/ips/screensetting"

	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/internal/entity"
	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/pkg/logger"
)

const (
	deviceCallBuffer = 100
	maxReadRecords   = 390
)

var (
	connections         = make(map[string]*ConnectionEntry)
	connectionsMu       sync.Mutex
	waitForAuthTickTime = 1 * time.Second
	queueTickTime       = 500 * time.Millisecond
	expireAfter         = 30 * time.Second                    // expire the stored connection after 30 seconds
	waitForAuth         = 3 * time.Second                     // wait for 3 seconds for the connection to authenticate, prevents multiple api calls trying to auth at the same time
	requestQueue        = make(chan func(), deviceCallBuffer) // Buffered channel to queue requests
	shutdownSignal      = make(chan struct{})
)

type ConnectionEntry struct {
	WsmanMessages wsman.Messages
	Timer         *time.Timer
}

type GoWSMANMessages struct {
	log              logger.Interface
	safeRequirements security.Cryptor
}

func NewGoWSMANMessages(log logger.Interface, safeRequirements security.Cryptor) *GoWSMANMessages {
	return &GoWSMANMessages{
		log:              log,
		safeRequirements: safeRequirements,
	}
}

func (g GoWSMANMessages) DestroyWsmanClient(device dto.Device) {
	if entry, ok := connections[device.GUID]; ok {
		entry.Timer.Stop()
		removeConnection(device.GUID)
	}
}

func (g GoWSMANMessages) Worker() {
	for {
		select {
		case request := <-requestQueue:
			request()
			time.Sleep(queueTickTime)
		case <-shutdownSignal:
			return
		}
	}
}

func (g GoWSMANMessages) SetupWsmanClient(device entity.Device, isRedirection, logAMTMessages bool) Management {
	resultChan := make(chan *ConnectionEntry)
	// Queue the request
	requestQueue <- func() {
		device.Password, _ = g.safeRequirements.Decrypt(device.Password)
		resultChan <- g.setupWsmanClientInternal(device, isRedirection, logAMTMessages)
	}

	return <-resultChan
}

func (g GoWSMANMessages) setupWsmanClientInternal(device entity.Device, isRedirection, logAMTMessages bool) *ConnectionEntry {
	clientParams := client.Parameters{
		Target:                    device.Hostname,
		Username:                  device.Username,
		Password:                  device.Password,
		UseDigest:                 true,
		UseTLS:                    device.UseTLS,
		SelfSignedAllowed:         device.AllowSelfSigned,
		LogAMTMessages:            logAMTMessages,
		IsRedirection:             isRedirection,
		AllowInsecureCipherSuites: config.ConsoleConfig.AllowInsecureCiphers,
	}

	if device.CertHash != nil && *device.CertHash != "" {
		clientParams.PinnedCert = *device.CertHash
	}

	timer := time.AfterFunc(expireAfter, func() {
		removeConnection(device.GUID)
	})

	if entry, ok := connections[device.GUID]; ok {
		if entry.WsmanMessages.Client.IsAuthenticated() {
			entry.Timer.Stop() // Stop the previous timer
			entry.Timer = time.AfterFunc(expireAfter, func() {
				removeConnection(device.GUID)
			})

			return connections[device.GUID]
		}

		ticker := time.NewTicker(waitForAuthTickTime)

		defer ticker.Stop()

		timeout := time.After(waitForAuth)

		for {
			select {
			case <-ticker.C:
				if entry.WsmanMessages.Client.IsAuthenticated() {
					// Your logic when the function check is successful
					return connections[device.GUID]
				}
			case <-timeout:
				connectionsMu.Lock()

				connections[device.GUID] = &ConnectionEntry{
					WsmanMessages: wsman.NewMessages(clientParams),
					Timer:         timer,
				}

				connectionsMu.Unlock()

				return connections[device.GUID]
			}
		}
	}

	wsmanMsgs := wsman.NewMessages(clientParams)

	connectionsMu.Lock()

	connections[device.GUID] = &ConnectionEntry{
		WsmanMessages: wsmanMsgs,
		Timer:         timer,
	}
	connections[device.GUID].WsmanMessages.Client.IsAuthenticated()
	connectionsMu.Unlock()

	return connections[device.GUID]
}

func removeConnection(guid string) {
	connectionsMu.Lock()
	defer connectionsMu.Unlock()

	delete(connections, guid)
}

func (g *ConnectionEntry) GetAMTVersion() ([]software.SoftwareIdentity, error) {
	response, err := g.WsmanMessages.CIM.SoftwareIdentity.Enumerate()
	if err != nil {
		return []software.SoftwareIdentity{}, err
	}

	response, err = g.WsmanMessages.CIM.SoftwareIdentity.Pull(response.Body.EnumerateResponse.EnumerationContext)
	if err != nil {
		return []software.SoftwareIdentity{}, err
	}

	return response.Body.PullResponse.SoftwareIdentityItems, nil
}

func (g *ConnectionEntry) GetSetupAndConfiguration() ([]setupandconfiguration.SetupAndConfigurationServiceResponse, error) {
	response, err := g.WsmanMessages.AMT.SetupAndConfigurationService.Enumerate()
	if err != nil {
		return []setupandconfiguration.SetupAndConfigurationServiceResponse{}, err
	}

	response, err = g.WsmanMessages.AMT.SetupAndConfigurationService.Pull(response.Body.EnumerateResponse.EnumerationContext)
	if err != nil {
		return []setupandconfiguration.SetupAndConfigurationServiceResponse{}, err
	}

	return response.Body.PullResponse.SetupAndConfigurationServiceItems, nil
}

func (g *ConnectionEntry) GetDeviceCertificate() (*gotls.Certificate, error) {
	return g.WsmanMessages.Client.GetServerCertificate()
}

func (g *ConnectionEntry) RequestAMTRedirectionServiceStateChange(ider, sol bool) (redirection.RequestedState, int, error) {
	requestedState := redirection.DisableIDERAndSOL
	listenerEnabled := 0

	if ider {
		requestedState++
		listenerEnabled = 1
	}

	if sol {
		requestedState += 2
		listenerEnabled = 1
	}

	_, err := g.WsmanMessages.AMT.RedirectionService.RequestStateChange(requestedState)
	if err != nil {
		return 0, 0, err
	}

	return requestedState, listenerEnabled, nil
}

func (g *ConnectionEntry) GetKVMRedirection() (kvm.Response, error) {
	response, err := g.WsmanMessages.CIM.KVMRedirectionSAP.Get()
	if err != nil {
		return kvm.Response{}, err
	}

	return response, nil
}

func (g *ConnectionEntry) SetKVMRedirection(enable bool) (int, error) {
	requestedState := kvm.RedirectionSAPDisable
	listenerEnabled := 0

	if enable {
		requestedState = kvm.RedirectionSAPEnable
		listenerEnabled = 1
	}

	_, err := g.WsmanMessages.CIM.KVMRedirectionSAP.RequestStateChange(requestedState)
	if err != nil {
		return 0, err
	}

	return listenerEnabled, nil
}

func (g *ConnectionEntry) GetAlarmOccurrences() ([]ipsAlarmClock.AlarmClockOccurrence, error) {
	response, err := g.WsmanMessages.IPS.AlarmClockOccurrence.Enumerate()
	if err != nil {
		return []ipsAlarmClock.AlarmClockOccurrence{}, err
	}

	response, err = g.WsmanMessages.IPS.AlarmClockOccurrence.Pull(response.Body.EnumerateResponse.EnumerationContext)
	if err != nil {
		return []ipsAlarmClock.AlarmClockOccurrence{}, err
	}

	return response.Body.PullResponse.Items, nil
}

func (g *ConnectionEntry) CreateAlarmOccurrences(name string, startTime time.Time, interval int, deleteOnCompletion bool) (amtAlarmClock.AddAlarmOutput, error) {
	alarmOccurrence := amtAlarmClock.AlarmClockOccurrence{
		InstanceID:         name,
		ElementName:        name,
		StartTime:          startTime,
		Interval:           interval,
		DeleteOnCompletion: deleteOnCompletion,
	}

	response, err := g.WsmanMessages.AMT.AlarmClockService.AddAlarm(alarmOccurrence)
	if err != nil {
		return amtAlarmClock.AddAlarmOutput{}, err
	}

	return response.Body.AddAlarmOutput, nil
}

func (g *ConnectionEntry) DeleteAlarmOccurrences(instanceID string) error {
	_, err := g.WsmanMessages.IPS.AlarmClockOccurrence.Delete(instanceID)
	if err != nil {
		return err
	}

	return nil
}

func (g *ConnectionEntry) hardwareGets() (GetHWResults, error) {
	results := GetHWResults{}

	var err error

	results.ChassisResult, err = g.WsmanMessages.CIM.Chassis.Get()
	if err != nil {
		return results, err
	}

	results.CardResult, err = g.WsmanMessages.CIM.Card.Get()
	if err != nil {
		return results, err
	}

	results.ChipResult, err = g.WsmanMessages.CIM.Chip.Get()
	if err != nil {
		return results, err
	}

	results.BiosResult, err = g.WsmanMessages.CIM.BIOSElement.Get()
	if err != nil {
		return results, err
	}

	results.ProcessorResult, err = g.WsmanMessages.CIM.Processor.Get()
	if err != nil {
		return results, err
	}

	return results, nil
}

func (g *ConnectionEntry) hardwarePulls() (PullHWResults, error) {
	results := PullHWResults{}

	var err error

	pmEnumerateResult, err := g.WsmanMessages.CIM.PhysicalMemory.Enumerate()
	if err != nil {
		return results, err
	}

	results.PhysicalMemoryResult, err = g.WsmanMessages.CIM.PhysicalMemory.Pull(pmEnumerateResult.Body.EnumerateResponse.EnumerationContext)
	if err != nil {
		return results, err
	}

	return results, nil
}

func (g *ConnectionEntry) GetHardwareInfo() (interface{}, error) {
	getHWResults, err := g.hardwareGets()
	if err != nil {
		return nil, err
	}

	pullHWResults, err := g.hardwarePulls()
	if err != nil {
		return nil, err
	}

	hwResults := HWResults{
		ChassisResult:        getHWResults.ChassisResult,
		ChipResult:           getHWResults.ChipResult,
		CardResult:           getHWResults.CardResult,
		PhysicalMemoryResult: pullHWResults.PhysicalMemoryResult,
		BiosResult:           getHWResults.BiosResult,
		ProcessorResult:      getHWResults.ProcessorResult,
	}

	return createMapInterfaceForHWInfo(hwResults)
}

type GetHWResults struct {
	ChassisResult   chassis.Response
	ChipResult      chip.Response
	CardResult      card.Response
	BiosResult      bios.Response
	ProcessorResult processor.Response
}
type PullHWResults struct {
	PhysicalMemoryResult physical.Response
}
type HWResults struct {
	ChassisResult        chassis.Response
	ChipResult           chip.Response
	CardResult           card.Response
	PhysicalMemoryResult physical.Response
	BiosResult           bios.Response
	ProcessorResult      processor.Response
}

func createMapInterfaceForHWInfo(hwResults HWResults) (interface{}, error) {
	return map[string]interface{}{
		"CIM_Chassis": map[string]interface{}{
			"response":  hwResults.ChassisResult.Body.PackageResponse,
			"responses": []interface{}{},
		}, "CIM_Chip": map[string]interface{}{
			"responses": []interface{}{hwResults.ChipResult.Body.PackageResponse},
		}, "CIM_Card": map[string]interface{}{
			"response":  hwResults.CardResult.Body.PackageResponse,
			"responses": []interface{}{},
		}, "CIM_BIOSElement": map[string]interface{}{
			"response":  hwResults.BiosResult.Body.GetResponse,
			"responses": []interface{}{},
		}, "CIM_Processor": map[string]interface{}{
			"responses": []interface{}{hwResults.ProcessorResult.Body.PackageResponse},
		}, "CIM_PhysicalMemory": map[string]interface{}{
			"responses": hwResults.PhysicalMemoryResult.Body.PullResponse.MemoryItems,
		},
	}, nil
}

func createMapInterfaceForDiskInfo(diskResults DiskResults) (interface{}, error) {
	return map[string]interface{}{
		"CIM_MediaAccessDevice": map[string]interface{}{
			"responses": []interface{}{diskResults.MediaAccessPullResult.Body.PullResponse.MediaAccessDevices},
		}, "CIM_PhysicalPackage": map[string]interface{}{
			"responses": []interface{}{diskResults.PPPullResult.Body.PullResponse.PhysicalPackage},
		},
	}, nil
}

type DiskResults struct {
	MediaAccessPullResult mediaaccess.Response
	PPPullResult          physical.Response
}

func (g *ConnectionEntry) GetDiskInfo() (interface{}, error) {
	results := DiskResults{}

	var err error

	maEnumerateResult, err := g.WsmanMessages.CIM.MediaAccessDevice.Enumerate()
	if err != nil {
		return results, err
	}

	results.MediaAccessPullResult, err = g.WsmanMessages.CIM.MediaAccessDevice.Pull(maEnumerateResult.Body.EnumerateResponse.EnumerationContext)
	if err != nil {
		return results, err
	}

	ppEnumerateResult, err := g.WsmanMessages.CIM.PhysicalPackage.Enumerate()
	if err != nil {
		return results, err
	}

	results.PPPullResult, err = g.WsmanMessages.CIM.PhysicalPackage.Pull(ppEnumerateResult.Body.EnumerateResponse.EnumerationContext)
	if err != nil {
		return results, err
	}

	diskResults := DiskResults{
		MediaAccessPullResult: results.MediaAccessPullResult,
		PPPullResult:          results.PPPullResult,
	}

	return createMapInterfaceForDiskInfo(diskResults)
}

func (g *ConnectionEntry) GetPowerState() ([]service.CIM_AssociatedPowerManagementService, error) {
	response, err := g.WsmanMessages.CIM.ServiceAvailableToElement.Enumerate()
	if err != nil {
		return []service.CIM_AssociatedPowerManagementService{}, err
	}

	response, err = g.WsmanMessages.CIM.ServiceAvailableToElement.Pull(response.Body.EnumerateResponse.EnumerationContext)
	if err != nil {
		return []service.CIM_AssociatedPowerManagementService{}, err
	}

	return response.Body.PullResponse.AssociatedPowerManagementService, nil
}

func (g *ConnectionEntry) GetOSPowerSavingState() (ipspower.OSPowerSavingState, error) {
	response, err := g.GetIPSPowerManagementService()
	if err != nil {
		return 0, err
	}

	return response.OSPowerSavingState, nil
}

func (g *ConnectionEntry) GetIPSPowerManagementService() (ipspower.PowerManagementService, error) {
	response, err := g.WsmanMessages.IPS.PowerManagementService.Get()
	if err != nil {
		return ipspower.PowerManagementService{}, err
	}

	return response.Body.GetResponse, nil
}

func (g *ConnectionEntry) RequestOSPowerSavingStateChange(newOSPowerStavingState ipspower.OSPowerSavingState) (ipspower.PowerActionResponse, error) {
	response, err := g.WsmanMessages.IPS.PowerManagementService.RequestOSPowerSavingStateChange(newOSPowerStavingState)
	if err != nil {
		return ipspower.PowerActionResponse{}, err
	}

	return response.Body.RequestOSPowerSavingStateChangeResponse, nil
}

func (g *ConnectionEntry) GetPowerCapabilities() (boot.BootCapabilitiesResponse, error) {
	response, err := g.WsmanMessages.AMT.BootCapabilities.Get()
	if err != nil {
		return boot.BootCapabilitiesResponse{}, err
	}

	return response.Body.BootCapabilitiesGetResponse, nil
}

func (g *ConnectionEntry) GetGeneralSettings() (interface{}, error) {
	response, err := g.WsmanMessages.AMT.GeneralSettings.Get()
	if err != nil {
		return nil, err
	}

	return response.Body.GetResponse, nil
}

func (g *ConnectionEntry) CancelUserConsentRequest() (optin.Response, error) {
	response, err := g.WsmanMessages.IPS.OptInService.CancelOptIn()
	if err != nil {
		return optin.Response{}, err
	}

	return response, nil
}

func (g *ConnectionEntry) GetUserConsentCode() (optin.Response, error) {
	response, err := g.WsmanMessages.IPS.OptInService.StartOptIn()
	if err != nil {
		return optin.Response{}, err
	}

	return response, nil
}

func (g *ConnectionEntry) SendConsentCode(code int) (optin.Response, error) {
	response, err := g.WsmanMessages.IPS.OptInService.SendOptInCode(code)
	if err != nil {
		return optin.Response{}, err
	}

	return response, nil
}

func (g *ConnectionEntry) GetBootData() (boot.BootSettingDataResponse, error) {
	bootSettingData, err := g.WsmanMessages.AMT.BootSettingData.Get()
	if err != nil {
		return boot.BootSettingDataResponse{}, err
	}

	return bootSettingData.Body.BootSettingDataGetResponse, nil
}

func (g *ConnectionEntry) SetBootData(data boot.BootSettingDataRequest) (interface{}, error) {
	bootSettingData, err := g.WsmanMessages.AMT.BootSettingData.Put(data)
	if err != nil {
		return nil, err
	}

	return bootSettingData.Body, nil
}

func (g *ConnectionEntry) GetBootService() (cimBoot.BootService, error) {
	bootService, err := g.WsmanMessages.CIM.BootService.Get()
	if err != nil {
		return cimBoot.BootService{}, err
	}

	return bootService.Body.ServiceGetResponse, nil
}

func (g *ConnectionEntry) BootServiceStateChange(requestedState int) (cimBoot.BootService, error) {
	bootService, err := g.WsmanMessages.CIM.BootService.RequestStateChange(requestedState)
	if err != nil {
		return cimBoot.BootService{}, err
	}

	return bootService.Body.ServiceGetResponse, nil
}

func (g *ConnectionEntry) SetBootConfigRole(role int) (interface{}, error) {
	response, err := g.WsmanMessages.CIM.BootService.SetBootConfigRole("Intel(r) AMT: Boot Configuration 0", role)
	if err != nil {
		return cimBoot.ChangeBootOrder_OUTPUT{}, err
	}

	return response.Body.ChangeBootOrder_OUTPUT, nil
}

func (g *ConnectionEntry) ChangeBootOrder(bootSource string) (cimBoot.ChangeBootOrder_OUTPUT, error) {
	response, err := g.WsmanMessages.CIM.BootConfigSetting.ChangeBootOrder(cimBoot.Source(bootSource))
	if err != nil {
		return cimBoot.ChangeBootOrder_OUTPUT{}, err
	}

	return response.Body.ChangeBootOrder_OUTPUT, nil
}

func (g *ConnectionEntry) GetAuditLog(startIndex int) (auditlog.Response, error) {
	response, err := g.WsmanMessages.AMT.AuditLog.ReadRecords(startIndex)
	if err != nil {
		return auditlog.Response{}, err
	}

	return response, nil
}

func (g *ConnectionEntry) GetEventLog(startIndex, maxReadRecords int) (messagelog.GetRecordsResponse, error) {
	response, err := g.WsmanMessages.AMT.MessageLog.GetRecords(startIndex, maxReadRecords)
	if err != nil {
		return messagelog.GetRecordsResponse{}, err
	}

	return response.Body.GetRecordsResponse, nil
}

func (g *ConnectionEntry) SendPowerAction(action int) (power.PowerActionResponse, error) {
	response, err := g.WsmanMessages.CIM.PowerManagementService.RequestPowerStateChange(power.PowerState(action))
	if err != nil {
		return power.PowerActionResponse{}, err
	}

	return response.Body.RequestPowerStateChangeResponse, nil
}

func (g *ConnectionEntry) GetPublicKeyCerts() ([]publickey.PublicKeyCertificateResponse, error) {
	response, err := g.WsmanMessages.AMT.PublicKeyCertificate.Enumerate()
	if err != nil {
		return nil, err
	}

	response, err = g.WsmanMessages.AMT.PublicKeyCertificate.Pull(response.Body.EnumerateResponse.EnumerationContext)
	if err != nil {
		return nil, err
	}

	return response.Body.PullResponse.PublicKeyCertificateItems, nil
}

func (g *ConnectionEntry) GenerateKeyPair(keyAlgorithm publickey.KeyAlgorithm, keyLength publickey.KeyLength) (response publickey.Response, err error) {
	return g.WsmanMessages.AMT.PublicKeyManagementService.GenerateKeyPair(keyAlgorithm, keyLength)
}

func (g *ConnectionEntry) UpdateAMTPassword(digestPassword string) (authorization.Response, error) {
	return g.WsmanMessages.AMT.AuthorizationService.SetAdminAclEntryEx("admin", digestPassword)
}

func (g *ConnectionEntry) CreateTLSCredentialContext(certHandle string) (response tls.Response, err error) {
	return g.WsmanMessages.AMT.TLSCredentialContext.Create(certHandle)
}

// GetPublicPrivateKeyPairs

// NOTE: RSA Key encoded as DES PKCS#1. The Exponent (E) is 65537 (0x010001).

// When this structure is used as an output parameter (GET or PULL method),

// only the public section of the key is exported.

func (g *ConnectionEntry) GetPublicPrivateKeyPairs() ([]publicprivate.PublicPrivateKeyPair, error) {
	response, err := g.WsmanMessages.AMT.PublicPrivateKeyPair.Enumerate()
	if err != nil {
		return nil, err
	}

	response, err = g.WsmanMessages.AMT.PublicPrivateKeyPair.Pull(response.Body.EnumerateResponse.EnumerationContext)
	if err != nil {
		return nil, err
	}

	return response.Body.PullResponse.PublicPrivateKeyPairItems, nil
}

func (g *ConnectionEntry) GetWiFiSettings() ([]wifi.WiFiEndpointSettingsResponse, error) {
	response, err := g.WsmanMessages.CIM.WiFiEndpointSettings.Enumerate()
	if err != nil {
		return nil, err
	}

	response, err = g.WsmanMessages.CIM.WiFiEndpointSettings.Pull(response.Body.EnumerateResponse.EnumerationContext)
	if err != nil {
		return nil, err
	}

	return response.Body.PullResponse.EndpointSettingsItems, nil
}

func (g *ConnectionEntry) GetEthernetPortSettings() ([]ethernetport.SettingsResponse, error) {
	response, err := g.WsmanMessages.AMT.EthernetPortSettings.Enumerate()
	if err != nil {
		return nil, err
	}

	response, err = g.WsmanMessages.AMT.EthernetPortSettings.Pull(response.Body.EnumerateResponse.EnumerationContext)
	if err != nil {
		return nil, err
	}

	return response.Body.PullResponse.EthernetPortItems, nil
}

func (g *ConnectionEntry) PutEthernetPortSettings(ethernetPortSettings ethernetport.SettingsRequest, instanceID string) (ethernetport.Response, error) {
	return g.WsmanMessages.AMT.EthernetPortSettings.Put(instanceID, ethernetPortSettings)
}

func (g *ConnectionEntry) DeletePublicPrivateKeyPair(instanceID string) error {
	_, err := g.WsmanMessages.AMT.PublicPrivateKeyPair.Delete(instanceID)

	return err
}

func (g *ConnectionEntry) DeletePublicCert(instanceID string) error {
	_, err := g.WsmanMessages.AMT.PublicKeyCertificate.Delete(instanceID)

	return err
}

func (g *ConnectionEntry) GetCredentialRelationships() (credential.Items, error) {
	response, err := g.WsmanMessages.CIM.CredentialContext.Enumerate()
	if err != nil {
		return credential.Items{}, err
	}

	response, err = g.WsmanMessages.CIM.CredentialContext.Pull(response.Body.EnumerateResponse.EnumerationContext)
	if err != nil {
		return credential.Items{}, err
	}

	return response.Body.PullResponse.Items, nil
}

func (g *ConnectionEntry) GetConcreteDependencies() ([]concrete.ConcreteDependency, error) {
	response, err := g.WsmanMessages.CIM.ConcreteDependency.Enumerate()
	if err != nil {
		return nil, err
	}

	response, err = g.WsmanMessages.CIM.ConcreteDependency.Pull(response.Body.EnumerateResponse.EnumerationContext)
	if err != nil {
		return nil, err
	}

	return response.Body.PullResponse.Items, nil
}

func (g *ConnectionEntry) DeleteWiFiSetting(instanceID string) error {
	_, err := g.WsmanMessages.CIM.WiFiEndpointSettings.Delete(instanceID)

	return err
}

func (g *ConnectionEntry) AddTrustedRootCert(caCert string) (handle string, err error) {
	response, err := g.WsmanMessages.AMT.PublicKeyManagementService.AddTrustedRootCertificate(caCert)
	if err != nil {
		return "", err
	}

	if len(response.Body.AddTrustedRootCertificate_OUTPUT.CreatedCertificate.ReferenceParameters.SelectorSet.Selectors) > 0 {
		handle = response.Body.AddTrustedRootCertificate_OUTPUT.CreatedCertificate.ReferenceParameters.SelectorSet.Selectors[0].Text
	}

	return handle, nil
}

func (g *ConnectionEntry) AddClientCert(clientCert string) (handle string, err error) {
	response, err := g.WsmanMessages.AMT.PublicKeyManagementService.AddCertificate(clientCert)
	if err != nil {
		return "", err
	}

	if len(response.Body.AddCertificate_OUTPUT.CreatedCertificate.ReferenceParameters.SelectorSet.Selectors) > 0 {
		handle = response.Body.AddCertificate_OUTPUT.CreatedCertificate.ReferenceParameters.SelectorSet.Selectors[0].Text
	}

	return handle, nil
}

func (g *ConnectionEntry) AddPrivateKey(privateKey string) (handle string, err error) {
	response, err := g.WsmanMessages.AMT.PublicKeyManagementService.AddKey(privateKey)
	if err != nil {
		return "", err
	}

	if len(response.Body.AddKey_OUTPUT.CreatedKey.ReferenceParameters.SelectorSet.Selectors) > 0 {
		handle = response.Body.AddKey_OUTPUT.CreatedKey.ReferenceParameters.SelectorSet.Selectors[0].Text
	}

	return handle, nil
}

func (g *ConnectionEntry) DeleteKeyPair(instanceID string) error {
	_, err := g.WsmanMessages.AMT.PublicKeyManagementService.Delete(instanceID)

	return err
}

func (g *ConnectionEntry) GetWiFiPortConfigurationService() (wifiportconfiguration.WiFiPortConfigurationServiceResponse, error) {
	response, err := g.WsmanMessages.AMT.WiFiPortConfigurationService.Get()
	if err != nil {
		return wifiportconfiguration.WiFiPortConfigurationServiceResponse{}, err
	}

	return response.Body.WiFiPortConfigurationService, nil
}

func (g *ConnectionEntry) PutWiFiPortConfigurationService(request wifiportconfiguration.WiFiPortConfigurationServiceRequest) (wifiportconfiguration.WiFiPortConfigurationServiceResponse, error) {
	// if local sync not enable, enable it
	// if response.Body.WiFiPortConfigurationService.LocalProfileSynchronizationEnabled == wifiportconfiguration.LocalSyncDisabled {
	// 	putRequest := wifiportconfiguration.WiFiPortConfigurationServiceRequest{
	// 		RequestedState:                     response.Body.WiFiPortConfigurationService.RequestedState,
	// 		EnabledState:                       response.Body.WiFiPortConfigurationService.EnabledState,
	// 		HealthState:                        response.Body.WiFiPortConfigurationService.HealthState,
	// 		ElementName:                        response.Body.WiFiPortConfigurationService.ElementName,
	// 		SystemCreationClassName:            response.Body.WiFiPortConfigurationService.SystemCreationClassName,
	// 		SystemName:                         response.Body.WiFiPortConfigurationService.SystemName,
	// 		CreationClassName:                  response.Body.WiFiPortConfigurationService.CreationClassName,
	// 		Name:                               response.Body.WiFiPortConfigurationService.Name,
	// 		LocalProfileSynchronizationEnabled: wifiportconfiguration.UnrestrictedSync,
	// 		LastConnectedSsidUnderMeControl:    response.Body.WiFiPortConfigurationService.LastConnectedSsidUnderMeControl,
	// 		NoHostCsmeSoftwarePolicy:           response.Body.WiFiPortConfigurationService.NoHostCsmeSoftwarePolicy,
	// 		UEFIWiFiProfileShareEnabled:        response.Body.WiFiPortConfigurationService.UEFIWiFiProfileShareEnabled,
	// 	}
	response, err := g.WsmanMessages.AMT.WiFiPortConfigurationService.Put(request)
	if err != nil {
		return wifiportconfiguration.WiFiPortConfigurationServiceResponse{}, err
	}

	return response.Body.WiFiPortConfigurationService, nil
}

func (g *ConnectionEntry) WiFiRequestStateChange() (err error) {
	// always turn wifi on via state change request
	// Enumeration 32769 - WiFi is enabled in S0 + Sx/AC
	_, err = g.WsmanMessages.CIM.WiFiPort.RequestStateChange(int(wifi.EnabledStateWifiEnabledS0SxAC))
	if err != nil {
		return err // utils.WSMANMessageError
	}

	return nil
}

func (g *ConnectionEntry) AddWiFiSettings(wifiEndpointSettings wifi.WiFiEndpointSettingsRequest, ieee8021xSettings models.IEEE8021xSettings, wifiEndpoint, clientCredential, caCredential string) (response wifiportconfiguration.Response, err error) {
	return g.WsmanMessages.AMT.WiFiPortConfigurationService.AddWiFiSettings(wifiEndpointSettings, ieee8021xSettings, wifiEndpoint, clientCredential, caCredential)
}

func (g *ConnectionEntry) PUTTLSSettings(instanceID string, tlsSettingData tls.SettingDataRequest) (response tls.Response, err error) {
	return g.WsmanMessages.AMT.TLSSettingData.Put(instanceID, tlsSettingData)
}

func (g *ConnectionEntry) GetLowAccuracyTimeSynch() (response timesynchronization.Response, err error) {
	return g.WsmanMessages.AMT.TimeSynchronizationService.GetLowAccuracyTimeSynch()
}

func (g *ConnectionEntry) SetHighAccuracyTimeSynch(ta0, tm1, tm2 int64) (response timesynchronization.Response, err error) {
	return g.WsmanMessages.AMT.TimeSynchronizationService.SetHighAccuracyTimeSynch(ta0, tm1, tm2)
}

func (g *ConnectionEntry) EnumerateTLSSettingData() (response tls.Response, err error) {
	return g.WsmanMessages.AMT.TLSSettingData.Enumerate()
}

func (g *ConnectionEntry) PullTLSSettingData(enumerationContext string) (response tls.Response, err error) {
	return g.WsmanMessages.AMT.TLSSettingData.Pull(enumerationContext)
}

func (g *ConnectionEntry) CommitChanges() (response setupandconfiguration.Response, err error) {
	return g.WsmanMessages.AMT.SetupAndConfigurationService.CommitChanges()
}

func (g *ConnectionEntry) GeneratePKCS10RequestEx(keyPair, nullSignedCertificateRequest string, signingAlgorithm publickey.SigningAlgorithm) (response publickey.Response, err error) {
	return g.WsmanMessages.AMT.PublicKeyManagementService.GeneratePKCS10RequestEx(keyPair, nullSignedCertificateRequest, signingAlgorithm)
}

func (g *ConnectionEntry) RequestRedirectionStateChange(requestedState redirection.RequestedState) (response redirection.Response, err error) {
	return g.WsmanMessages.AMT.RedirectionService.RequestStateChange(requestedState)
}

func (g *ConnectionEntry) RequestKVMStateChange(requestedState kvm.KVMRedirectionSAPRequestStateChangeInput) (response kvm.Response, err error) {
	return g.WsmanMessages.CIM.KVMRedirectionSAP.RequestStateChange(requestedState)
}

func (g *ConnectionEntry) GetRedirectionService() (response redirection.Response, err error) {
	return g.WsmanMessages.AMT.RedirectionService.Get()
}

func (g *ConnectionEntry) GetIpsOptInService() (response optin.Response, err error) {
	return g.WsmanMessages.IPS.OptInService.Get()
}

func (g *ConnectionEntry) GetIPSIEEE8021xSettings() (response ipsIEEE8021x.Response, err error) {
	return g.WsmanMessages.IPS.IEEE8021xSettings.Get()
}

type NetworkResults struct {
	EthernetPortSettingsResult  []ethernetport.SettingsResponse
	IPSIEEE8021xSettingsResult  ipsIEEE8021x.IEEE8021xSettingsResponse
	WiFiSettingsResult          []wifi.WiFiEndpointSettingsResponse
	CIMIEEE8021xSettingsResult  cimIEEE8021x.PullResponse
	WiFiPortConfigServiceResult wifiportconfiguration.WiFiPortConfigurationServiceResponse
	NetworkInterfaces           InterfaceTypes
}

type InterfaceTypes struct {
	hasWired    bool
	hasWireless bool
}

func (g *ConnectionEntry) GetCIMIEEE8021xSettings() (response cimIEEE8021x.Response, err error) {
	response, err = g.WsmanMessages.CIM.IEEE8021xSettings.Enumerate()
	if err != nil {
		return cimIEEE8021x.Response{}, err
	}

	response, err = g.WsmanMessages.CIM.IEEE8021xSettings.Pull(response.Body.EnumerateResponse.EnumerationContext)
	if err != nil {
		return cimIEEE8021x.Response{}, err
	}

	return response, nil
}

func (g *ConnectionEntry) GetNetworkSettings() (NetworkResults, error) {
	networkResults := NetworkResults{}

	var err error

	networkResults.EthernetPortSettingsResult, err = g.GetEthernetPortSettings()
	if err != nil {
		return networkResults, err
	}

	networkResults.NetworkInterfaces = g.determineInterfaceTypes(networkResults.EthernetPortSettingsResult)

	if networkResults.NetworkInterfaces.hasWired {
		response, err := g.GetIPSIEEE8021xSettings()
		if err != nil {
			return networkResults, err
		}

		networkResults.IPSIEEE8021xSettingsResult = response.Body.IEEE8021xSettingsResponse
	}

	if networkResults.NetworkInterfaces.hasWireless {
		networkResults.WiFiSettingsResult, err = g.GetWiFiSettings()
		if err != nil {
			return networkResults, err
		}

		cimResponse, err := g.GetCIMIEEE8021xSettings()
		if err != nil {
			return networkResults, err
		}

		networkResults.CIMIEEE8021xSettingsResult = cimResponse.Body.PullResponse

		wifiPortConfigService, err := g.WsmanMessages.AMT.WiFiPortConfigurationService.Get()
		if err != nil {
			return networkResults, err
		}

		networkResults.WiFiPortConfigServiceResult = wifiPortConfigService.Body.WiFiPortConfigurationService
	}

	return networkResults, nil
}

func (g *ConnectionEntry) determineInterfaceTypes(ethernetSettings []ethernetport.SettingsResponse) InterfaceTypes {
	types := InterfaceTypes{}

	for i := range ethernetSettings {
		switch ethernetSettings[i].InstanceID {
		case "Intel(r) AMT Ethernet Port Settings 0":
			types.hasWired = true
		case "Intel(r) AMT Ethernet Port Settings 1":
			types.hasWireless = true
		}
	}

	return types
}

// AMT Explorer Functions.
func (g *ConnectionEntry) GetAMT8021xCredentialContext() (ieee8021x.Response, error) {
	enum, err := g.WsmanMessages.AMT.IEEE8021xCredentialContext.Enumerate()
	if err != nil {
		return ieee8021x.Response{}, err
	}

	pull, err := g.WsmanMessages.AMT.IEEE8021xCredentialContext.Pull(enum.Body.EnumerateResponse.EnumerationContext)
	if err != nil {
		return ieee8021x.Response{}, err
	}

	return pull, nil
}

func (g *ConnectionEntry) GetAMT8021xProfile() (ieee8021x.Response, error) {
	enum, err := g.WsmanMessages.AMT.IEEE8021xProfile.Enumerate()
	if err != nil {
		return ieee8021x.Response{}, err
	}

	pull, err := g.WsmanMessages.AMT.IEEE8021xProfile.Pull(enum.Body.EnumerateResponse.EnumerationContext)
	if err != nil {
		return ieee8021x.Response{}, err
	}

	return pull, nil
}

func (g *ConnectionEntry) GetAMTAlarmClockService() (amtAlarmClock.Response, error) {
	enum, err := g.WsmanMessages.AMT.AlarmClockService.Enumerate()
	if err != nil {
		return amtAlarmClock.Response{}, err
	}

	pull, err := g.WsmanMessages.AMT.AlarmClockService.Pull(enum.Body.EnumerateResponse.EnumerationContext)
	if err != nil {
		return amtAlarmClock.Response{}, err
	}

	return pull, nil
}

func (g *ConnectionEntry) GetAMTAuditLog() (auditlog.Response, error) {
	readrecords, err := g.WsmanMessages.AMT.AuditLog.ReadRecords(1)
	if err != nil {
		return auditlog.Response{}, err
	}

	return readrecords, nil
}

func (g *ConnectionEntry) GetAMTAuthorizationService() (authorization.Response, error) {
	enum, err := g.WsmanMessages.AMT.AuthorizationService.Enumerate()
	if err != nil {
		return authorization.Response{}, err
	}

	pull, err := g.WsmanMessages.AMT.AuthorizationService.Pull(enum.Body.EnumerateResponse.EnumerationContext)
	if err != nil {
		return authorization.Response{}, err
	}

	return pull, nil
}

func (g *ConnectionEntry) GetAMTBootCapabilities() (boot.Response, error) {
	enum, err := g.WsmanMessages.AMT.BootCapabilities.Enumerate()
	if err != nil {
		return boot.Response{}, err
	}

	pull, err := g.WsmanMessages.AMT.BootCapabilities.Pull(enum.Body.EnumerateResponse.EnumerationContext)
	if err != nil {
		return boot.Response{}, err
	}

	return pull, nil
}

func (g *ConnectionEntry) GetAMTBootSettingData() (boot.Response, error) {
	enum, err := g.WsmanMessages.AMT.BootSettingData.Enumerate()
	if err != nil {
		return boot.Response{}, err
	}

	pull, err := g.WsmanMessages.AMT.BootSettingData.Pull(enum.Body.EnumerateResponse.EnumerationContext)
	if err != nil {
		return boot.Response{}, err
	}

	return pull, nil
}

func (g *ConnectionEntry) GetAMTEnvironmentDetectionSettingData() (environmentdetection.Response, error) {
	enum, err := g.WsmanMessages.AMT.EnvironmentDetectionSettingData.Enumerate()
	if err != nil {
		return environmentdetection.Response{}, err
	}

	pull, err := g.WsmanMessages.AMT.EnvironmentDetectionSettingData.Pull(enum.Body.EnumerateResponse.EnumerationContext)
	if err != nil {
		return environmentdetection.Response{}, err
	}

	return pull, nil
}

func (g *ConnectionEntry) GetAMTEthernetPortSettings() (ethernetport.Response, error) {
	enum, err := g.WsmanMessages.AMT.EthernetPortSettings.Enumerate()
	if err != nil {
		return ethernetport.Response{}, err
	}

	pull, err := g.WsmanMessages.AMT.EthernetPortSettings.Pull(enum.Body.EnumerateResponse.EnumerationContext)
	if err != nil {
		return ethernetport.Response{}, err
	}

	return pull, nil
}

func (g *ConnectionEntry) GetAMTGeneralSettings() (general.Response, error) {
	get, err := g.WsmanMessages.AMT.GeneralSettings.Get()
	if err != nil {
		return general.Response{}, err
	}

	return get, nil
}

func (g *ConnectionEntry) GetAMTKerberosSettingData() (kerberos.Response, error) {
	enum, err := g.WsmanMessages.AMT.KerberosSettingData.Enumerate()
	if err != nil {
		return kerberos.Response{}, err
	}

	pull, err := g.WsmanMessages.AMT.KerberosSettingData.Pull(enum.Body.EnumerateResponse.EnumerationContext)
	if err != nil {
		return kerberos.Response{}, err
	}

	return pull, nil
}

func (g *ConnectionEntry) GetAMTManagementPresenceRemoteSAP() (managementpresence.Response, error) {
	enum, err := g.WsmanMessages.AMT.ManagementPresenceRemoteSAP.Enumerate()
	if err != nil {
		return managementpresence.Response{}, err
	}

	pull, err := g.WsmanMessages.AMT.ManagementPresenceRemoteSAP.Pull(enum.Body.EnumerateResponse.EnumerationContext)
	if err != nil {
		return managementpresence.Response{}, err
	}

	return pull, nil
}

func (g *ConnectionEntry) GetAMTMessageLog() (messagelog.Response, error) {
	get, err := g.WsmanMessages.AMT.MessageLog.GetRecords(1, maxReadRecords)
	if err != nil {
		return messagelog.Response{}, err
	}

	return get, nil
}

func (g *ConnectionEntry) GetAMTMPSUsernamePassword() (mps.Response, error) {
	enum, err := g.WsmanMessages.AMT.MPSUsernamePassword.Enumerate()
	if err != nil {
		return mps.Response{}, err
	}

	pull, err := g.WsmanMessages.AMT.MPSUsernamePassword.Pull(enum.Body.EnumerateResponse.EnumerationContext)
	if err != nil {
		return mps.Response{}, err
	}

	return pull, nil
}

func (g *ConnectionEntry) GetAMTPublicKeyCertificate() (publickey.Response, error) {
	enum, err := g.WsmanMessages.AMT.PublicKeyCertificate.Enumerate()
	if err != nil {
		return publickey.Response{}, err
	}

	pull, err := g.WsmanMessages.AMT.PublicKeyCertificate.Pull(enum.Body.EnumerateResponse.EnumerationContext)
	if err != nil {
		return publickey.Response{}, err
	}

	return pull, nil
}

func (g *ConnectionEntry) GetAMTPublicKeyManagementService() (publickey.Response, error) {
	get, err := g.WsmanMessages.AMT.PublicKeyManagementService.Get()
	if err != nil {
		return publickey.Response{}, err
	}

	return get, nil
}

func (g *ConnectionEntry) GetAMTPublicPrivateKeyPair() (publicprivate.Response, error) {
	enum, err := g.WsmanMessages.AMT.PublicPrivateKeyPair.Enumerate()
	if err != nil {
		return publicprivate.Response{}, err
	}

	pull, err := g.WsmanMessages.AMT.PublicPrivateKeyPair.Pull(enum.Body.EnumerateResponse.EnumerationContext)
	if err != nil {
		return publicprivate.Response{}, err
	}

	return pull, nil
}

func (g *ConnectionEntry) GetAMTRedirectionService() (redirection.Response, error) {
	get, err := g.WsmanMessages.AMT.RedirectionService.Get()
	if err != nil {
		return redirection.Response{}, err
	}

	return get, nil
}

func (g *ConnectionEntry) SetAMTRedirectionService(request *redirection.RedirectionRequest) (redirection.Response, error) {
	response, err := g.WsmanMessages.AMT.RedirectionService.Put(request)
	if err != nil {
		return redirection.Response{}, err
	}

	return response, nil
}

func (g *ConnectionEntry) GetAMTRemoteAccessPolicyAppliesToMPS() (remoteaccess.Response, error) {
	enum, err := g.WsmanMessages.AMT.RemoteAccessPolicyAppliesToMPS.Enumerate()
	if err != nil {
		return remoteaccess.Response{}, err
	}

	pull, err := g.WsmanMessages.AMT.RemoteAccessPolicyAppliesToMPS.Pull(enum.Body.EnumerateResponse.EnumerationContext)
	if err != nil {
		return remoteaccess.Response{}, err
	}

	return pull, nil
}

func (g *ConnectionEntry) GetAMTRemoteAccessPolicyRule() (remoteaccess.Response, error) {
	enum, err := g.WsmanMessages.AMT.RemoteAccessPolicyRule.Enumerate()
	if err != nil {
		return remoteaccess.Response{}, err
	}

	pull, err := g.WsmanMessages.AMT.RemoteAccessPolicyRule.Pull(enum.Body.EnumerateResponse.EnumerationContext)
	if err != nil {
		return remoteaccess.Response{}, err
	}

	return pull, nil
}

func (g *ConnectionEntry) GetAMTRemoteAccessService() (remoteaccess.Response, error) {
	get, err := g.WsmanMessages.AMT.RemoteAccessService.Get()
	if err != nil {
		return remoteaccess.Response{}, err
	}

	return get, nil
}

func (g *ConnectionEntry) GetAMTSetupAndConfigurationService() (setupandconfiguration.Response, error) {
	get, err := g.WsmanMessages.AMT.SetupAndConfigurationService.Get()
	if err != nil {
		return setupandconfiguration.Response{}, err
	}

	return get, nil
}

func (g *ConnectionEntry) GetAMTTimeSynchronizationService() (timesynchronization.Response, error) {
	get, err := g.WsmanMessages.AMT.TimeSynchronizationService.Get()
	if err != nil {
		return timesynchronization.Response{}, err
	}

	return get, nil
}

func (g *ConnectionEntry) GetAMTTLSCredentialContext() (tls.Response, error) {
	enum, err := g.WsmanMessages.AMT.TLSCredentialContext.Enumerate()
	if err != nil {
		return tls.Response{}, err
	}

	pull, err := g.WsmanMessages.AMT.TLSCredentialContext.Pull(enum.Body.EnumerateResponse.EnumerationContext)
	if err != nil {
		return tls.Response{}, err
	}

	return pull, nil
}

func (g *ConnectionEntry) GetAMTTLSProtocolEndpointCollection() (tls.Response, error) {
	enum, err := g.WsmanMessages.AMT.TLSProtocolEndpointCollection.Enumerate()
	if err != nil {
		return tls.Response{}, err
	}

	pull, err := g.WsmanMessages.AMT.TLSProtocolEndpointCollection.Pull(enum.Body.EnumerateResponse.EnumerationContext)
	if err != nil {
		return tls.Response{}, err
	}

	return pull, nil
}

func (g *ConnectionEntry) GetAMTTLSSettingData() (tls.Response, error) {
	enum, err := g.WsmanMessages.AMT.TLSSettingData.Enumerate()
	if err != nil {
		return tls.Response{}, err
	}

	pull, err := g.WsmanMessages.AMT.TLSSettingData.Pull(enum.Body.EnumerateResponse.EnumerationContext)
	if err != nil {
		return tls.Response{}, err
	}

	return pull, nil
}

func (g *ConnectionEntry) GetAMTUserInitiatedConnectionService() (userinitiatedconnection.Response, error) {
	get, err := g.WsmanMessages.AMT.UserInitiatedConnectionService.Get()
	if err != nil {
		return userinitiatedconnection.Response{}, err
	}

	return get, nil
}

func (g *ConnectionEntry) GetAMTWiFiPortConfigurationService() (wifiportconfiguration.Response, error) {
	enum, err := g.WsmanMessages.AMT.WiFiPortConfigurationService.Enumerate()
	if err != nil {
		return wifiportconfiguration.Response{}, err
	}

	pull, err := g.WsmanMessages.AMT.WiFiPortConfigurationService.Pull(enum.Body.EnumerateResponse.EnumerationContext)
	if err != nil {
		return wifiportconfiguration.Response{}, err
	}

	return pull, nil
}

func (g *ConnectionEntry) GetCIMBIOSElement() (bios.Response, error) {
	enum, err := g.WsmanMessages.CIM.BIOSElement.Enumerate()
	if err != nil {
		return bios.Response{}, err
	}

	pull, err := g.WsmanMessages.CIM.BIOSElement.Pull(enum.Body.EnumerateResponse.EnumerationContext)
	if err != nil {
		return bios.Response{}, err
	}

	return pull, nil
}

func (g *ConnectionEntry) GetCIMBootConfigSetting() (cimBoot.Response, error) {
	enum, err := g.WsmanMessages.CIM.BootConfigSetting.Enumerate()
	if err != nil {
		return cimBoot.Response{}, err
	}

	pull, err := g.WsmanMessages.CIM.BootConfigSetting.Pull(enum.Body.EnumerateResponse.EnumerationContext)
	if err != nil {
		return cimBoot.Response{}, err
	}

	return pull, nil
}

func (g *ConnectionEntry) GetCIMBootService() (cimBoot.Response, error) {
	enum, err := g.WsmanMessages.CIM.BootService.Enumerate()
	if err != nil {
		return cimBoot.Response{}, err
	}

	pull, err := g.WsmanMessages.CIM.BootService.Pull(enum.Body.EnumerateResponse.EnumerationContext)
	if err != nil {
		return cimBoot.Response{}, err
	}

	return pull, nil
}

func (g *ConnectionEntry) GetCIMBootSourceSetting() (cimBoot.Response, error) {
	enum, err := g.WsmanMessages.CIM.BootSourceSetting.Enumerate()
	if err != nil {
		return cimBoot.Response{}, err
	}

	pull, err := g.WsmanMessages.CIM.BootSourceSetting.Pull(enum.Body.EnumerateResponse.EnumerationContext)
	if err != nil {
		return cimBoot.Response{}, err
	}

	return pull, nil
}

func (g *ConnectionEntry) GetCIMCard() (card.Response, error) {
	enum, err := g.WsmanMessages.CIM.Card.Enumerate()
	if err != nil {
		return card.Response{}, err
	}

	pull, err := g.WsmanMessages.CIM.Card.Pull(enum.Body.EnumerateResponse.EnumerationContext)
	if err != nil {
		return card.Response{}, err
	}

	return pull, nil
}

func (g *ConnectionEntry) GetCIMChassis() (chassis.Response, error) {
	enum, err := g.WsmanMessages.CIM.Chassis.Enumerate()
	if err != nil {
		return chassis.Response{}, err
	}

	pull, err := g.WsmanMessages.CIM.Chassis.Pull(enum.Body.EnumerateResponse.EnumerationContext)
	if err != nil {
		return chassis.Response{}, err
	}

	return pull, nil
}

func (g *ConnectionEntry) GetCIMChip() (chip.Response, error) {
	enum, err := g.WsmanMessages.CIM.Chip.Enumerate()
	if err != nil {
		return chip.Response{}, err
	}

	pull, err := g.WsmanMessages.CIM.Chip.Pull(enum.Body.EnumerateResponse.EnumerationContext)
	if err != nil {
		return chip.Response{}, err
	}

	return pull, nil
}

func (g *ConnectionEntry) GetCIMComputerSystemPackage() (computer.Response, error) {
	enum, err := g.WsmanMessages.CIM.ComputerSystemPackage.Enumerate()
	if err != nil {
		return computer.Response{}, err
	}

	pull, err := g.WsmanMessages.CIM.ComputerSystemPackage.Pull(enum.Body.EnumerateResponse.EnumerationContext)
	if err != nil {
		return computer.Response{}, err
	}

	return pull, nil
}

func (g *ConnectionEntry) GetCIMConcreteDependency() (concrete.Response, error) {
	enum, err := g.WsmanMessages.CIM.ConcreteDependency.Enumerate()
	if err != nil {
		return concrete.Response{}, err
	}

	pull, err := g.WsmanMessages.CIM.ConcreteDependency.Pull(enum.Body.EnumerateResponse.EnumerationContext)
	if err != nil {
		return concrete.Response{}, err
	}

	return pull, nil
}

func (g *ConnectionEntry) GetCIMCredentialContext() (credential.Response, error) {
	enum, err := g.WsmanMessages.CIM.CredentialContext.Enumerate()
	if err != nil {
		return credential.Response{}, err
	}

	pull, err := g.WsmanMessages.CIM.CredentialContext.Pull(enum.Body.EnumerateResponse.EnumerationContext)
	if err != nil {
		return credential.Response{}, err
	}

	return pull, nil
}

func (g *ConnectionEntry) GetCIMKVMRedirectionSAP() (kvm.Response, error) {
	enum, err := g.WsmanMessages.CIM.KVMRedirectionSAP.Enumerate()
	if err != nil {
		return kvm.Response{}, err
	}

	pull, err := g.WsmanMessages.CIM.KVMRedirectionSAP.Pull(enum.Body.EnumerateResponse.EnumerationContext)
	if err != nil {
		return kvm.Response{}, err
	}

	return pull, nil
}

func (g *ConnectionEntry) GetCIMMediaAccessDevice() (mediaaccess.Response, error) {
	enum, err := g.WsmanMessages.CIM.MediaAccessDevice.Enumerate()
	if err != nil {
		return mediaaccess.Response{}, err
	}

	pull, err := g.WsmanMessages.CIM.MediaAccessDevice.Pull(enum.Body.EnumerateResponse.EnumerationContext)
	if err != nil {
		return mediaaccess.Response{}, err
	}

	return pull, nil
}

func (g *ConnectionEntry) GetCIMPhysicalMemory() (physical.Response, error) {
	enum, err := g.WsmanMessages.CIM.PhysicalMemory.Enumerate()
	if err != nil {
		return physical.Response{}, err
	}

	pull, err := g.WsmanMessages.CIM.PhysicalMemory.Pull(enum.Body.EnumerateResponse.EnumerationContext)
	if err != nil {
		return physical.Response{}, err
	}

	return pull, nil
}

func (g *ConnectionEntry) GetCIMPhysicalPackage() (physical.Response, error) {
	enum, err := g.WsmanMessages.CIM.PhysicalPackage.Enumerate()
	if err != nil {
		return physical.Response{}, err
	}

	pull, err := g.WsmanMessages.CIM.PhysicalPackage.Pull(enum.Body.EnumerateResponse.EnumerationContext)
	if err != nil {
		return physical.Response{}, err
	}

	return pull, nil
}

func (g *ConnectionEntry) GetCIMPowerManagementService() (power.Response, error) {
	get, err := g.WsmanMessages.CIM.PowerManagementService.Get()
	if err != nil {
		return power.Response{}, err
	}

	return get, nil
}

func (g *ConnectionEntry) GetCIMProcessor() (processor.Response, error) {
	enum, err := g.WsmanMessages.CIM.Processor.Enumerate()
	if err != nil {
		return processor.Response{}, err
	}

	pull, err := g.WsmanMessages.CIM.Processor.Pull(enum.Body.EnumerateResponse.EnumerationContext)
	if err != nil {
		return processor.Response{}, err
	}

	return pull, nil
}

func (g *ConnectionEntry) GetCIMServiceAvailableToElement() (service.Response, error) {
	enum, err := g.WsmanMessages.CIM.ServiceAvailableToElement.Enumerate()
	if err != nil {
		return service.Response{}, err
	}

	pull, err := g.WsmanMessages.CIM.ServiceAvailableToElement.Pull(enum.Body.EnumerateResponse.EnumerationContext)
	if err != nil {
		return service.Response{}, err
	}

	return pull, nil
}

func (g *ConnectionEntry) GetCIMSoftwareIdentity() (software.Response, error) {
	enum, err := g.WsmanMessages.CIM.SoftwareIdentity.Enumerate()
	if err != nil {
		return software.Response{}, err
	}

	pull, err := g.WsmanMessages.CIM.SoftwareIdentity.Pull(enum.Body.EnumerateResponse.EnumerationContext)
	if err != nil {
		return software.Response{}, err
	}

	return pull, nil
}

func (g *ConnectionEntry) GetCIMSystemPackaging() (system.Response, error) {
	enum, err := g.WsmanMessages.CIM.SystemPackaging.Enumerate()
	if err != nil {
		return system.Response{}, err
	}

	pull, err := g.WsmanMessages.CIM.SystemPackaging.Pull(enum.Body.EnumerateResponse.EnumerationContext)
	if err != nil {
		return system.Response{}, err
	}

	return pull, nil
}

func (g *ConnectionEntry) GetCIMWiFiEndpointSettings() (wifi.Response, error) {
	enum, err := g.WsmanMessages.CIM.WiFiEndpointSettings.Enumerate()
	if err != nil {
		return wifi.Response{}, err
	}

	pull, err := g.WsmanMessages.CIM.WiFiEndpointSettings.Pull(enum.Body.EnumerateResponse.EnumerationContext)
	if err != nil {
		return wifi.Response{}, err
	}

	return pull, nil
}

func (g *ConnectionEntry) GetCIMWiFiPort() (wifi.Response, error) {
	enum, err := g.WsmanMessages.CIM.WiFiPort.Enumerate()
	if err != nil {
		return wifi.Response{}, err
	}

	pull, err := g.WsmanMessages.CIM.WiFiPort.Pull(enum.Body.EnumerateResponse.EnumerationContext)
	if err != nil {
		return wifi.Response{}, err
	}

	return pull, nil
}

func (g *ConnectionEntry) GetIPS8021xCredentialContext() (ipsIEEE8021x.Response, error) {
	enum, err := g.WsmanMessages.IPS.IEEE8021xCredentialContext.Enumerate()
	if err != nil {
		return ipsIEEE8021x.Response{}, err
	}

	pull, err := g.WsmanMessages.IPS.IEEE8021xCredentialContext.Pull(enum.Body.EnumerateResponse.EnumerationContext)
	if err != nil {
		return ipsIEEE8021x.Response{}, err
	}

	return pull, nil
}

func (g *ConnectionEntry) GetIPSAlarmClockOccurrence() (ipsAlarmClock.Response, error) {
	enum, err := g.WsmanMessages.IPS.AlarmClockOccurrence.Enumerate()
	if err != nil {
		return ipsAlarmClock.Response{}, err
	}

	pull, err := g.WsmanMessages.IPS.AlarmClockOccurrence.Pull(enum.Body.EnumerateResponse.EnumerationContext)
	if err != nil {
		return ipsAlarmClock.Response{}, err
	}

	return pull, nil
}

func (g *ConnectionEntry) GetIPSHostBasedSetupService() (hostbasedsetup.Response, error) {
	get, err := g.WsmanMessages.IPS.HostBasedSetupService.Get()
	if err != nil {
		return hostbasedsetup.Response{}, err
	}

	return get, nil
}

func (g *ConnectionEntry) GetIPSOptInService() (optin.Response, error) {
	get, err := g.WsmanMessages.IPS.OptInService.Get()
	if err != nil {
		return optin.Response{}, err
	}

	return get, nil
}

func (g *ConnectionEntry) SetIPSOptInService(request optin.OptInServiceRequest) error {
	_, err := g.WsmanMessages.IPS.OptInService.Put(request)
	if err != nil {
		return err
	}

	return nil
}

type Certificates struct {
	ConcreteDependencyResponse   concrete.PullResponse
	PublicKeyCertificateResponse publickey.RefinedPullResponse
	PublicPrivateKeyPairResponse publicprivate.RefinedPullResponse
	CIMCredentialContextResponse credential.PullResponse
}

func (g *ConnectionEntry) GetCertificates() (Certificates, error) {
	concreteDepEnumResp, err := g.WsmanMessages.CIM.ConcreteDependency.Enumerate()
	if err != nil {
		return Certificates{}, err
	}

	concreteDepResponse, err := g.WsmanMessages.CIM.ConcreteDependency.Pull(concreteDepEnumResp.Body.EnumerateResponse.EnumerationContext)
	if err != nil {
		return Certificates{}, err
	}

	pubKeyCertEnumResp, err := g.WsmanMessages.AMT.PublicKeyCertificate.Enumerate()
	if err != nil {
		return Certificates{}, err
	}

	pubKeyCertResponse, err := g.WsmanMessages.AMT.PublicKeyCertificate.Pull(pubKeyCertEnumResp.Body.EnumerateResponse.EnumerationContext)
	if err != nil {
		return Certificates{}, err
	}

	pubPrivKeyPairEnumResp, err := g.WsmanMessages.AMT.PublicPrivateKeyPair.Enumerate()
	if err != nil {
		return Certificates{}, err
	}

	pubPrivKeyPairResponse, err := g.WsmanMessages.AMT.PublicPrivateKeyPair.Pull(pubPrivKeyPairEnumResp.Body.EnumerateResponse.EnumerationContext)
	if err != nil {
		return Certificates{}, err
	}

	cimCredContextEnumResp, err := g.WsmanMessages.CIM.CredentialContext.Enumerate()
	if err != nil {
		return Certificates{}, err
	}

	cimCredContextResponse, err := g.WsmanMessages.CIM.CredentialContext.Pull(cimCredContextEnumResp.Body.EnumerateResponse.EnumerationContext)
	if err != nil {
		return Certificates{}, err
	}

	certificates := Certificates{
		ConcreteDependencyResponse:   concreteDepResponse.Body.PullResponse,
		PublicKeyCertificateResponse: pubKeyCertResponse.Body.RefinedPullResponse,
		PublicPrivateKeyPairResponse: pubPrivKeyPairResponse.Body.RefinedPullResponse,
		CIMCredentialContextResponse: cimCredContextResponse.Body.PullResponse,
	}

	return certificates, nil
}

func (g *ConnectionEntry) GetTLSSettingData() ([]tls.SettingDataResponse, error) {
	tlsSettingDataEnumResp, err := g.WsmanMessages.AMT.TLSSettingData.Enumerate()
	if err != nil {
		return nil, err
	}

	tlsSettingDataResponse, err := g.WsmanMessages.AMT.TLSSettingData.Pull(tlsSettingDataEnumResp.Body.EnumerateResponse.EnumerationContext)
	if err != nil {
		return nil, err
	}

	return tlsSettingDataResponse.Body.PullResponse.SettingDataItems, nil
}

func (g *ConnectionEntry) GetIPSKVMRedirectionSettings() (kvmredirection.Response, error) {
	enum, err := g.WsmanMessages.IPS.KVMRedirectionSettingData.Enumerate()
	if err != nil {
		return kvmredirection.Response{}, err
	}

	pull, err := g.WsmanMessages.IPS.KVMRedirectionSettingData.Pull(enum.Body.EnumerateResponse.EnumerationContext)
	if err != nil {
		return kvmredirection.Response{}, err
	}

	return pull, nil
}

func (g *ConnectionEntry) GetIPSScreenSettingData() (screensetting.Response, error) {
	enum, err := g.WsmanMessages.IPS.ScreenSettingData.Enumerate()
	if err != nil {
		return screensetting.Response{}, err
	}

	pull, err := g.WsmanMessages.IPS.ScreenSettingData.Pull(enum.Body.EnumerateResponse.EnumerationContext)
	if err != nil {
		return screensetting.Response{}, err
	}

	return pull, nil
}

func (g *ConnectionEntry) GetIPSKVMRedirectionSettingData() (kvmredirection.Response, error) {
	enum, err := g.WsmanMessages.IPS.KVMRedirectionSettingData.Enumerate()
	if err != nil {
		return kvmredirection.Response{}, err
	}

	pull, err := g.WsmanMessages.IPS.KVMRedirectionSettingData.Pull(enum.Body.EnumerateResponse.EnumerationContext)
	if err != nil {
		return kvmredirection.Response{}, err
	}

	// Intentionally fetch the current settings to validate connectivity; response is unused.
	// Avoid printing to stdout per lint rules.
	_, err = g.WsmanMessages.IPS.KVMRedirectionSettingData.Get()
	if err != nil {
		return kvmredirection.Response{}, err
	}

	return pull, nil
}

func (g *ConnectionEntry) SetIPSKVMRedirectionSettingData(req *kvmredirection.KVMRedirectionSettingsRequest) (kvmredirection.Response, error) {
	return g.WsmanMessages.IPS.KVMRedirectionSettingData.Put(req)
}
