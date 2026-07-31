package devices

import (
	"context"
	"strings"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/ethernetport"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/usecase/devices/wsman"
)

// Instance ID substrings that identify the AMT ethernet port interfaces.
const (
	wiredEthernetInstanceID    = "Intel(r) AMT Ethernet Port Settings 0"
	wirelessEthernetInstanceID = "Intel(r) AMT Ethernet Port Settings 1"
)

func (uc *UseCase) GetNetworkSettings(c context.Context, guid string) (dto.NetworkSettings, error) {
	item, err := uc.repo.GetByID(c, guid, "")
	if err != nil {
		return dto.NetworkSettings{}, err
	}

	if item == nil || item.GUID == "" {
		return dto.NetworkSettings{}, ErrNotFound
	}

	device, err := uc.device.SetupWsmanClient(c, *item, false, true)
	if err != nil {
		return dto.NetworkSettings{}, err
	}

	response, err := device.GetNetworkSettings()
	if err != nil {
		return dto.NetworkSettings{}, err
	}

	ns := dto.NetworkSettings{}

	for i := range response.EthernetPortSettingsResult {
		portSetting := &response.EthernetPortSettingsResult[i]

		if strings.Contains(portSetting.InstanceID, wiredEthernetInstanceID) {
			// Wired network
			ns.Wired = &dto.WiredNetworkInfo{
				IEEE8021x: dto.IEEE8021x{
					Enabled:       response.IPSIEEE8021xSettingsResult.Enabled.String(),
					AvailableInS0: response.IPSIEEE8021xSettingsResult.AvailableInS0,
					PxeTimeout:    response.IPSIEEE8021xSettingsResult.PxeTimeout,
				},
			}
			ns.Wired.NetworkInfo = convertToNetworkInfo(*portSetting)
		}

		if strings.Contains(portSetting.InstanceID, wirelessEthernetInstanceID) {
			// Wireless network
			ns.Wireless = &dto.WirelessNetworkInfo{}
			ns.Wireless.NetworkInfo = convertToNetworkInfo(*portSetting)
			ns.Wireless.LinkPreference = portSetting.LinkPreference.String()
			ns.Wireless.LinkControl = portSetting.LinkControl.String()
			ns.Wireless.WLANLinkProtectionLevel = portSetting.WLANLinkProtectionLevel.String()
			ns.Wireless.WiFiNetworks = uc.processWiFiSettings(response)
			ns.Wireless.IEEE8021xSettings = uc.processIEEE8021xSettings(response)
			ns.Wireless.WiFiPortConfigService = uc.processWiFiPortConfigService(response)
		}
	}

	return ns, nil
}

func (uc *UseCase) processWiFiPortConfigService(response wsman.NetworkResults) dto.WiFiPortConfigService {
	return dto.WiFiPortConfigService{
		RequestedState:                     int(response.WiFiPortConfigServiceResult.RequestedState),
		EnabledState:                       int(response.WiFiPortConfigServiceResult.EnabledState),
		HealthState:                        int(response.WiFiPortConfigServiceResult.HealthState),
		ElementName:                        response.WiFiPortConfigServiceResult.ElementName,
		SystemCreationClassName:            response.WiFiPortConfigServiceResult.SystemCreationClassName,
		SystemName:                         response.WiFiPortConfigServiceResult.SystemName,
		CreationClassName:                  response.WiFiPortConfigServiceResult.CreationClassName,
		Name:                               response.WiFiPortConfigServiceResult.Name,
		LocalProfileSynchronizationEnabled: int(response.WiFiPortConfigServiceResult.LocalProfileSynchronizationEnabled),
		LastConnectedSsidUnderMeControl:    response.WiFiPortConfigServiceResult.LastConnectedSsidUnderMeControl,
		NoHostCsmeSoftwarePolicy:           int(response.WiFiPortConfigServiceResult.NoHostCsmeSoftwarePolicy),
		UEFIWiFiProfileShareEnabled:        response.WiFiPortConfigServiceResult.UEFIWiFiProfileShareEnabled,
	}
}

func convertToNetworkInfo(portSetting ethernetport.SettingsResponse) dto.NetworkInfo {
	return dto.NetworkInfo{
		ElementName:                  portSetting.ElementName,
		InstanceID:                   portSetting.InstanceID,
		VLANTag:                      portSetting.VLANTag,
		SharedMAC:                    portSetting.SharedMAC,
		MACAddress:                   portSetting.MACAddress,
		LinkIsUp:                     portSetting.LinkIsUp,
		SharedStaticIP:               portSetting.SharedStaticIp,
		SharedDynamicIP:              portSetting.SharedDynamicIP,
		IPSyncEnabled:                portSetting.IpSyncEnabled,
		DHCPEnabled:                  portSetting.DHCPEnabled,
		IPAddress:                    portSetting.IPAddress,
		SubnetMask:                   portSetting.SubnetMask,
		DefaultGateway:               portSetting.DefaultGateway,
		PrimaryDNS:                   portSetting.PrimaryDNS,
		SecondaryDNS:                 portSetting.SecondaryDNS,
		ConsoleTCPMaxRetransmissions: portSetting.ConsoleTcpMaxRetransmissions,
		PhysicalConnectionType:       portSetting.PhysicalConnectionType.String(),
		PhysicalNicMedium:            portSetting.PhysicalNicMedium.String(),
		LinkPolicy:                   convertLinkPolicy(portSetting.LinkPolicy),
	}
}

func convertLinkPolicy(linkPolicy []ethernetport.LinkPolicy) []string {
	var linkPolicyStr []string
	for _, v := range linkPolicy {
		linkPolicyStr = append(linkPolicyStr, v.String())
	}

	return linkPolicyStr
}

func (uc *UseCase) processWiFiSettings(response wsman.NetworkResults) []dto.WiFiNetwork {
	var wifiNetworks []dto.WiFiNetwork

	for _, v := range response.WiFiSettingsResult {
		// Skip Endpoint User Settings and show only Admin Endpoint Settings
		if v.ElementName != "Endpoint User Settings" {
			wifiNetworks = append(wifiNetworks, dto.WiFiNetwork{
				ElementName:          v.ElementName,
				SSID:                 v.SSID,
				AuthenticationMethod: v.AuthenticationMethod.String(),
				EncryptionMethod:     v.EncryptionMethod.String(),
				Priority:             v.Priority,
				BSSType:              v.BSSType.String(),
			})
		}
	}

	return wifiNetworks
}

func (uc *UseCase) processIEEE8021xSettings(response wsman.NetworkResults) []dto.IEEE8021xSettings {
	var ieee8021xSettings []dto.IEEE8021xSettings

	for i := range response.CIMIEEE8021xSettingsResult.IEEE8021xSettingsItems {
		v := &response.CIMIEEE8021xSettingsResult.IEEE8021xSettingsItems[i]
		ieee8021xSettings = append(ieee8021xSettings, dto.IEEE8021xSettings{
			AuthenticationProtocol:          v.AuthenticationProtocol,
			RoamingIdentity:                 v.RoamingIdentity,
			ServerCertificateName:           v.ServerCertificateName,
			ServerCertificateNameComparison: v.ServerCertificateNameComparison,
			Username:                        v.Username,
			Password:                        v.Password,
			Domain:                          v.Domain,
			ProtectedAccessCredential:       v.ProtectedAccessCredential,
		})
	}

	return ieee8021xSettings
}

// GetWiredNetworkSettings returns the wired (Intel® AMT Ethernet Port Settings 0)
// portion of a device's network settings.
func (uc *UseCase) GetWiredNetworkSettings(c context.Context, guid string) (dto.WiredNetworkInfo, error) {
	ns, err := uc.GetNetworkSettings(c, guid)
	if err != nil {
		return dto.WiredNetworkInfo{}, err
	}

	if ns.Wired == nil {
		return dto.WiredNetworkInfo{}, ErrNotFound
	}

	return *ns.Wired, nil
}

// PatchWiredNetworkSettings updates a device's wired IPv4 configuration (DHCP or
// static IP). It returns no content on success.
func (uc *UseCase) PatchWiredNetworkSettings(c context.Context, guid string, req dto.WiredNetworkConfigRequest) error {
	// Wired 802.1X configuration is a forward-looking API contract that is not yet
	// implemented. Reject requests that supply the ieee8021x object rather than
	// silently ignoring it, so the behavior is unambiguous for clients.
	if req.IEEE8021x != nil {
		return ErrNotSupportedUseCase.Wrap("PatchWiredNetworkSettings", "ieee8021x", "wired 802.1X configuration is not yet supported")
	}

	if err := validateWiredNetworkConfig(req); err != nil {
		return err
	}

	item, err := uc.repo.GetByID(c, guid, "")
	if err != nil {
		return err
	}

	if item == nil || item.GUID == "" {
		return ErrNotFound
	}

	device, err := uc.device.SetupWsmanClient(c, *item, false, true)
	if err != nil {
		return err
	}

	settings, err := device.GetEthernetPortSettings()
	if err != nil {
		return err
	}

	var current *ethernetport.SettingsResponse

	for i := range settings {
		if strings.Contains(settings[i].InstanceID, wiredEthernetInstanceID) {
			current = &settings[i]

			break
		}
	}

	if current == nil {
		return ErrNotFound
	}

	settingsRequest := buildWiredSettingsRequest(*current, req)

	if _, err = device.PutEthernetPortSettings(settingsRequest, current.InstanceID); err != nil {
		return err
	}

	return nil
}

// validateWiredNetworkConfig enforces the DHCP vs static-IP combination rules that
// cannot be expressed with field-level binding tags.
func validateWiredNetworkConfig(req dto.WiredNetworkConfigRequest) error {
	dhcpEnabled := req.DHCPEnabled != nil && *req.DHCPEnabled
	ipSyncEnabled := req.IPSyncEnabled != nil && *req.IPSyncEnabled
	staticIPProvided := hasStaticIPSettings(req)

	if dhcpEnabled && staticIPProvided {
		return ErrValidationUseCase.Wrap("PatchWiredNetworkSettings", "validate", "cannot specify static IP settings when DHCP is enabled")
	}

	if ipSyncEnabled && staticIPProvided {
		return ErrValidationUseCase.Wrap("PatchWiredNetworkSettings", "validate", "cannot specify static IP settings when IP sync is enabled")
	}

	if !dhcpEnabled && !ipSyncEnabled && !staticIPProvided {
		return ErrValidationUseCase.Wrap("PatchWiredNetworkSettings", "validate", "must enable DHCP, enable IP sync, or provide static IP settings")
	}

	if !dhcpEnabled && staticIPProvided {
		return validateStaticIPFields(req)
	}

	return nil
}

// hasStaticIPSettings reports whether the request supplies any static IPv4 field.
func hasStaticIPSettings(req dto.WiredNetworkConfigRequest) bool {
	return req.IPAddress != "" || req.SubnetMask != "" ||
		req.DefaultGateway != "" || req.PrimaryDNS != "" || req.SecondaryDNS != ""
}

// validateStaticIPFields ensures the fields required for a static IP
// configuration are present.
func validateStaticIPFields(req dto.WiredNetworkConfigRequest) error {
	required := []struct {
		value string
		name  string
	}{
		{req.IPAddress, "ipAddress"},
		{req.SubnetMask, "subnetMask"},
		{req.DefaultGateway, "defaultGateway"},
		{req.PrimaryDNS, "primaryDNS"},
	}

	for _, field := range required {
		if field.value == "" {
			return ErrValidationUseCase.Wrap("PatchWiredNetworkSettings", "validate", field.name+" is required for static IP configuration")
		}
	}

	return nil
}

// buildWiredSettingsRequest builds an ethernet port settings Put request by
// overlaying the requested IPv4 changes on top of the device's current settings.
func buildWiredSettingsRequest(current ethernetport.SettingsResponse, req dto.WiredNetworkConfigRequest) ethernetport.SettingsRequest {
	settingsRequest := ethernetport.SettingsRequest{
		XMLName:        current.XMLName,
		ElementName:    current.ElementName,
		InstanceID:     current.InstanceID,
		SharedMAC:      current.SharedMAC,
		SharedStaticIp: current.SharedStaticIp,
		IpSyncEnabled:  current.IpSyncEnabled,
		DHCPEnabled:    current.DHCPEnabled,
		IPAddress:      current.IPAddress,
		SubnetMask:     current.SubnetMask,
		DefaultGateway: current.DefaultGateway,
		PrimaryDNS:     current.PrimaryDNS,
		SecondaryDNS:   current.SecondaryDNS,
	}

	if req.DHCPEnabled != nil && *req.DHCPEnabled {
		// DHCP mode: AMT acquires IP settings, host/ME stay in sync.
		settingsRequest.DHCPEnabled = true
		settingsRequest.IpSyncEnabled = true
		settingsRequest.SharedStaticIp = false
	} else {
		// Static IP mode.
		settingsRequest.DHCPEnabled = false

		ipSyncEnabled := current.IpSyncEnabled
		if req.IPSyncEnabled != nil {
			ipSyncEnabled = *req.IPSyncEnabled
		}

		// SharedStaticIp always follows IpSyncEnabled; the AMT firmware does not
		// support sharing a static IP without host sync.
		settingsRequest.IpSyncEnabled = ipSyncEnabled
		settingsRequest.SharedStaticIp = ipSyncEnabled

		settingsRequest.IPAddress = req.IPAddress
		settingsRequest.SubnetMask = req.SubnetMask
		settingsRequest.DefaultGateway = req.DefaultGateway
		settingsRequest.PrimaryDNS = req.PrimaryDNS
		settingsRequest.SecondaryDNS = req.SecondaryDNS
	}

	// When IP settings come from DHCP or are synced with the host, AMT rejects
	// explicit IP fields, so they must be cleared.
	if settingsRequest.IpSyncEnabled || settingsRequest.DHCPEnabled {
		settingsRequest.IPAddress = ""
		settingsRequest.SubnetMask = ""
		settingsRequest.DefaultGateway = ""
		settingsRequest.PrimaryDNS = ""
		settingsRequest.SecondaryDNS = ""
	}

	return settingsRequest
}
