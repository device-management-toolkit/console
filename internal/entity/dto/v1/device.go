package dto

import (
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
	UseTLS           bool        `json:"useTLS"`
	AllowSelfSigned  bool        `json:"allowSelfSigned"`
	CertHash         string      `json:"certHash"`
}

type DeviceInfo struct {
	FWVersion   string    `json:"fwVersion"`
	FWBuild     string    `json:"fwBuild"`
	FWSku       string    `json:"fwSku"`
	CurrentMode string    `json:"currentMode"`
	Features    string    `json:"features"`
	IPAddress   string    `json:"ipAddress"`
	LastUpdated time.Time `json:"lastUpdated"`
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
