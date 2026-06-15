package dto

// NetworkSettings defines the network settings for a device.
type NetworkSettings struct {
	Wired    *WiredNetworkInfo    `json:"wired"`
	Wireless *WirelessNetworkInfo `json:"wireless"`
}

// NetworkResults defines the network results for a device.
type NetworkInfo struct {
	ElementName                  string   `json:"elementName"`                            // The user-friendly name for this instance of SettingData. In addition, the user-friendly name can be used as an index property for a search or query. (Note: The name does not have to be unique within a namespace.)
	InstanceID                   string   `json:"instanceID"`                             // Within the scope of the instantiating Namespace, InstanceID opaquely and uniquely identifies an instance of this class.
	VLANTag                      int      `json:"vlanTag"`                                // Indicates whether VLAN is in use and what is the VLAN tag when used.
	SharedMAC                    bool     `json:"sharedMAC"`                              // Indicates whether Intel® AMT shares it's MAC address with the host system.
	MACAddress                   string   `json:"macAddress"`                             // The MAC address used by Intel® AMT in a string format. For Example: 01-02-3f-b0-99-99. (This property can only be read and can't be changed.)
	LinkIsUp                     bool     `json:"linkIsUp"`                               // Indicates whether the network link is up
	LinkPolicy                   []string `json:"linkPolicy"`                             // Enumeration values for link policy restrictions for better power consumption. If Intel® AMT will not be able to determine the exact power state, the more restrictive closest configuration applies.
	LinkPreference               string   `json:"linkPreference,omitempty"`               // Determines whether the link is preferred to be owned by ME or host
	LinkControl                  string   `json:"linkControl,omitempty"`                  // Determines whether the link is owned by ME or host.  Additional Notes: This property is read-only.
	SharedStaticIP               bool     `json:"sharedStaticIP"`                         // Indicates whether the static host IP is shared with ME.
	SharedDynamicIP              bool     `json:"sharedDynamicIP"`                        // Indicates whether the dynamic host IP is shared with ME. This property is read only.
	IPSyncEnabled                bool     `json:"ipSyncEnabled"`                          // Indicates whether the IP synchronization between host and ME is enabled.
	DHCPEnabled                  bool     `json:"dhcpEnabled"`                            // Indicates whether DHCP is in use. Additional Notes: 'DHCPEnabled' is a required field for the Put command.
	IPAddress                    string   `json:"ipAddress"`                              // String representation of IP address. Get operation - reports the acquired IP address (whether in static or DHCP mode). Put operation - sets the IP address (in static mode only).
	SubnetMask                   string   `json:"subnetMask"`                             // Subnet mask in a string format.For example: 255.255.0.0
	DefaultGateway               string   `json:"defaultGateway"`                         // Default Gateway in a string format. For example: 10.12.232.1
	PrimaryDNS                   string   `json:"primaryDNS"`                             // Primary DNS in a string format. For example: 10.12.232.1
	SecondaryDNS                 string   `json:"secondaryDNS"`                           // Secondary DNS in a string format. For example: 10.12.232.1
	ConsoleTCPMaxRetransmissions int      `json:"consoleTCPMaxRetransmissions,omitempty"` // Indicates the number of retransmissions host TCP SW tries ifno ack is accepted
	WLANLinkProtectionLevel      string   `json:"wlanLinkProtectionLevel,omitempty"`      // Defines the level of the link protection feature activation. Read only property.
	PhysicalConnectionType       string   `json:"physicalConnectionType"`                 // Indicates the physical connection type of this network interface. Note: Applicable in Intel AMT 15.0 and later.
	PhysicalNicMedium            string   `json:"physicalNICMedium"`                      // Indicates which medium is currently used by Intel® AMT to communicate with the NIC. Note: Applicable in Intel AMT 15.0 and later.
}
type WiredNetworkInfo struct {
	NetworkInfo
	IEEE8021x IEEE8021x `json:"ieee8021x"`
}

// WiredNetworkConfigRequest represents a request to update a device's wired
// (Intel® AMT Ethernet Port Settings 0) IPv4 configuration. Field-level format
// validation is enforced by the binding tags; the DHCP vs static-IP combination
// rules are enforced in the use case.
type WiredNetworkConfigRequest struct {
	DHCPEnabled    *bool                 `json:"dhcpEnabled" binding:"required"`          // Required: true selects DHCP mode (IP sync is forced on and explicit IP fields are ignored); false selects static IP mode, either manual or host-synced depending on ipSyncEnabled.
	IPSyncEnabled  *bool                 `json:"ipSyncEnabled"`                           // Optional, static mode only: true synchronizes the IP settings with the host OS and explicit static IP fields must not be supplied; false (or unset, defaulting to the device's current value) selects manual static IP. Ignored when DHCP is enabled.
	IPAddress      string                `json:"ipAddress" binding:"omitempty,ipv4"`      // Required for static IP mode (DHCP disabled, IP sync disabled).
	SubnetMask     string                `json:"subnetMask" binding:"omitempty,ipv4"`     // Required for static IP mode.
	DefaultGateway string                `json:"defaultGateway" binding:"omitempty,ipv4"` // Required for static IP mode.
	PrimaryDNS     string                `json:"primaryDNS" binding:"omitempty,ipv4"`     // Required for static IP mode.
	SecondaryDNS   string                `json:"secondaryDNS" binding:"omitempty,ipv4"`   // Optional for static IP mode.
	IEEE8021x      *WiredIEEE8021xConfig `json:"ieee8021x,omitempty"`                     // Optional: wired 802.1X authentication. Not yet supported; supplying this object returns HTTP 501.
}

// WiredIEEE8021xConfig represents a request to configure wired 802.1X (port-based
// network access control) authentication on the AMT Ethernet port.
//
// NOTE: This is a forward-looking API contract. Wired 802.1X configuration is not
// yet implemented; supplying this object causes PatchWiredNetworkSettings to return
// a "not supported" error (HTTP 501). The field shape is defined now so that the
// future implementation is purely additive and does not break existing clients.
type WiredIEEE8021xConfig struct {
	ProfileName            string `json:"profileName"`            // Friendly name of the 802.1X profile.
	AuthenticationProtocol int    `json:"authenticationProtocol"` // 0 = EAP-TLS, 2 = PEAPv0/EAP-MSCHAPv2.
	Username               string `json:"username"`               // 802.1X identity/username.
	Password               string `json:"password"`               // Required for PEAPv0/EAP-MSCHAPv2 (protocol 2).
	PrivateKey             string `json:"privateKey"`             // PEM-encoded private key. Required for EAP-TLS (protocol 0).
	ClientCert             string `json:"clientCert"`             // PEM-encoded client certificate. Required for EAP-TLS (protocol 0).
	CACert                 string `json:"caCert"`                 // PEM-encoded CA certificate.
}

type WirelessNetworkInfo struct {
	NetworkInfo
	WiFiNetworks          []WiFiNetwork         `json:"wifiNetworks"`
	IEEE8021xSettings     []IEEE8021xSettings   `json:"ieee8021xSettings"`
	WiFiPortConfigService WiFiPortConfigService `json:"wifiPortConfigService"`
}
type WiFiNetwork struct {
	ElementName          string `json:"elementName"`
	SSID                 string `json:"ssid"`
	AuthenticationMethod string `json:"authenticationMethod"`
	EncryptionMethod     string `json:"encryptionMethod"`
	Priority             int    `json:"priority"`
	BSSType              string `json:"bsstype"`
}
type IEEE8021x struct {
	Enabled       string `json:"enabled"`
	AvailableInS0 bool   `json:"availableInS0"`
	PxeTimeout    int    `json:"pxeTimeout"`
}
type IEEE8021xSettings struct {
	AuthenticationProtocol          int    `json:"authenticationProtocol"`
	RoamingIdentity                 string `json:"roamingIdentity"`
	ServerCertificateName           string `json:"serverCertificateName"`
	ServerCertificateNameComparison int    `json:"serverCertificateNameComparison"`
	Username                        string `json:"username"`
	Password                        string `json:"password"`
	Domain                          string `json:"domain"`
	ProtectedAccessCredential       string `json:"protectedAccessCredential"`
	PACPassword                     string `json:"pacPassword"`
	PSK                             string `json:"psk"`
}

type WiFiPortConfigService struct {
	RequestedState                     int    `json:"requestedState"`
	EnabledState                       int    `json:"enabledState"`
	HealthState                        int    `json:"healthState"`
	ElementName                        string `json:"elementName"`
	SystemCreationClassName            string `json:"systemCreationClassName"`
	SystemName                         string `json:"systemName"`
	CreationClassName                  string `json:"creationClassName"`
	Name                               string `json:"name"`
	LocalProfileSynchronizationEnabled int    `json:"localProfileSynchronizationEnabled"`
	LastConnectedSsidUnderMeControl    string `json:"lastConnectedSsidUnderMeControl"`
	NoHostCsmeSoftwarePolicy           int    `json:"noHostCsmeSoftwarePolicy"`
	UEFIWiFiProfileShareEnabled        bool   `json:"uefiWiFiProfileShareEnabled"`
}
