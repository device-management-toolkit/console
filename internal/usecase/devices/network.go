package devices

import (
	"context"
	"strings"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/ethernetport"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/usecase/devices/wsman"
)

func (uc *UseCase) GetNetworkSettings(c context.Context, guid string) (dto.NetworkSettings, error) {
	item, err := uc.repo.GetByID(c, guid, "")
	if err != nil {
		return dto.NetworkSettings{}, err
	}

	if item == nil || item.GUID == "" {
		return dto.NetworkSettings{}, ErrNotFound
	}

	device := uc.device.SetupWsmanClient(*item, false, true)

	response, err := device.GetNetworkSettings()
	if err != nil {
		return dto.NetworkSettings{}, err
	}

	ns := dto.NetworkSettings{}

	for i := range response.EthernetPortSettingsResult {
		portSetting := &response.EthernetPortSettingsResult[i]

		if strings.Contains(portSetting.InstanceID, "Intel(r) AMT Ethernet Port Settings 0") {
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

		if strings.Contains(portSetting.InstanceID, "Intel(r) AMT Ethernet Port Settings 1") {
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
