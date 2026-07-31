package dto

import (
	"encoding/json"
	"time"
)

type DeviceCountResponse struct {
	Count int      `json:"totalCount"`
	Data  []Device `json:"data"`
}
type DeviceStatResponse struct {
	TotalCount        int `json:"totalCount"`
	ConnectedCount    int `json:"connectedCount"`
	DisconnectedCount int `json:"disconnectedCount"`
}
type Device struct {
	ConnectionStatus bool        `json:"connectionStatus"`
	MPSInstance      string      `json:"mpsInstance"`
	Hostname         string      `json:"hostname"`
	GUID             string      `json:"guid"`
	MPSUsername      string      `json:"mpsusername"`
	Tags             []string    `json:"tags"`
	TenantID         string      `json:"tenantId"`
	FriendlyName     string      `json:"friendlyName"`
	DNSSuffix        string      `json:"dnsSuffix"`
	LastConnected    *time.Time  `json:"lastConnected,omitempty"`
	LastSeen         *time.Time  `json:"lastSeen,omitempty"`
	LastDisconnected *time.Time  `json:"lastDisconnected,omitempty"`
	DeviceInfo       *DeviceInfo `json:"deviceInfo,omitempty"`
	Username         string      `json:"username" binding:"max=16"`
	Password         string      `json:"password"`
	MPSPassword      string      `json:"mpspassword"`
	MEBXPassword     string      `json:"mebxpassword"`
	UseTLS           bool        `json:"useTLS"`
	AllowSelfSigned  bool        `json:"allowSelfSigned"`
	CertHash         string      `json:"certHash"`
}

type DeviceInfo struct {
	FWVersion            string                     `json:"fwVersion"`
	FWBuild              string                     `json:"fwBuild"`
	FWSku                string                     `json:"fwSku"`
	Discovered           *bool                      `json:"discovered,omitempty"`
	FirstDiscovered      *time.Time                 `json:"firstDiscovered,omitempty"`
	CurrentMode          string                     `json:"currentMode"`
	Features             string                     `json:"features"`
	IPAddress            string                     `json:"ipAddress"`
	LastSynced           *time.Time                 `json:"lastSynced,omitempty"`
	LMSInstalled         *bool                      `json:"lmsInstalled,omitempty"`
	LMSVersion           string                     `json:"lmsVersion,omitempty"`
	TLSMode              string                     `json:"tlsMode,omitempty"`
	UPID                 map[string]json.RawMessage `json:"upid,omitempty"`
	AMTEnabledInBIOS     *bool                      `json:"amtEnabledInBIOS,omitempty"`
	MEInterfaceVersion   string                     `json:"meInterfaceVersion,omitempty"`
	DHCPEnabled          *bool                      `json:"dhcpEnabled,omitempty"`
	CertHashes           []string                   `json:"certHashes,omitempty"`
	OSName               string                     `json:"osName,omitempty"`
	OSVersion            string                     `json:"osVersion,omitempty"`
	OSDistro             string                     `json:"osDistro,omitempty"`
	CPUModel             string                     `json:"cpuModel,omitempty"`
	OSIPAddress          string                     `json:"osIpAddress,omitempty"`
	EthernetAdapterCount *int                       `json:"ethernetAdapterCount,omitempty"`
	MonitorConnected     *bool                      `json:"monitorConnected,omitempty"`
	IEEE8021XEnabled     *bool                      `json:"ieee8021xEnabled,omitempty"`
}

// UnmarshalJSON implements custom JSON deserialization to support backwards compatibility
// for the lastUpdated -> lastSynced field rename. Existing clients may send the old
// "lastUpdated" key; this method migrates it to the new "lastSynced" field if present.
func (d *DeviceInfo) UnmarshalJSON(data []byte) error {
	type Alias DeviceInfo

	type deviceInfoCompat struct {
		*Alias
		LegacyLastUpdated *time.Time `json:"lastUpdated,omitempty"`
	}

	aux := deviceInfoCompat{Alias: (*Alias)(d)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if d.LastSynced == nil && aux.LegacyLastUpdated != nil {
		d.LastSynced = aux.LegacyLastUpdated
	}

	return nil
}

type Explorer struct {
	XMLInput  string `json:"xmlInput"`
	XMLOutput string `json:"xmlOutput"`
}
type Certificate struct {
	GUID               string    `json:"guid"`
	CommonName         string    `json:"commonName"`
	IssuerName         string    `json:"issuerName"`
	SerialNumber       string    `json:"serialNumber"`
	NotBefore          time.Time `json:"notBefore"`
	NotAfter           time.Time `json:"notAfter"`
	DNSNames           []string  `json:"dnsNames"`
	SHA1Fingerprint    string    `json:"sha1Fingerprint"`
	SHA256Fingerprint  string    `json:"sha256Fingerprint"`
	PublicKeyAlgorithm string    `json:"publicKeyAlgorithm"`
	PublicKeySize      int       `json:"publicKeySize"`
}

type PinCertificate struct {
	SHA256Fingerprint string `json:"sha256Fingerprint"`
}
