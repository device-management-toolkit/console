package dto

import (
	"encoding/json"
	"time"
)

// DeviceExport is the top-level response of GET /api/v1/devices/export.
type DeviceExport struct {
	Metadata ExportMetadata       `json:"metadata"`
	Summary  ExportSummary        `json:"summary"`
	Data     []DeviceExportRecord `json:"data"`
}

// ExportMetadata carries auditing/version-traceability info for the export.
type ExportMetadata struct {
	ExportedAt time.Time `json:"exportedAt"`
	SwVersion  string    `json:"swVersion"`
}

// ExportSummary holds the total count of exported devices.
type ExportSummary struct {
	TotalCount int `json:"totalCount"`
}

// DeviceExportRecord is the per-device allowlisted export shape. Only the
// fields listed here can be exported to users; credential fields (password,
// mpsPassword, mebxPassword, certHash) are intentionally excluded.
type DeviceExportRecord struct {
	GUID            string           `json:"guid"`
	Hostname        string           `json:"hostname"`
	FriendlyName    string           `json:"friendlyName"`
	Tags            []string         `json:"tags"`
	TenantID        string           `json:"tenantId"`
	FirstDiscovered *time.Time       `json:"firstDiscovered"`
	LastSynced      *time.Time       `json:"lastSynced"`
	LastUpdated     *time.Time       `json:"lastUpdated"` // export-gap: no writer yet; always null until a last-update timestamp is stamped.
	DeviceInfo      DeviceExportInfo `json:"deviceInfo"`
}

// DeviceExportInfo groups device details by OS/pheripherals categorie. A nil subsystem is
// rendered as JSON null (e.g. me/bmc for non-ME devices).
type DeviceExportInfo struct {
	ME       *ExportME       `json:"me"`
	OS       *ExportOS       `json:"os"`
	Platform *ExportPlatform `json:"platform"`
	BMC      *ExportBMC      `json:"bmc"` // export-gap: BMC data is not collected yet; always null.
}

// ExportME holds Management Engine (AMT/ISM) details. A device without a
// when `me` is present its fields are always shown, null/empty where unknown.
type ExportME struct {
	DNSSuffix         string                     `json:"dnsSuffix"`
	CurrentMode       string                     `json:"currentMode"`
	MEBXEnabledInBIOS *bool                      `json:"mebxEnabledInBIOS"`
	FWVersion         string                     `json:"fwVersion"`
	FWBuild           string                     `json:"fwBuild"`
	FWSku             string                     `json:"fwSku"`
	Features          string                     `json:"features"`
	TLSMode           string                     `json:"tlsMode"`
	DHCPEnabled       *bool                      `json:"dhcpEnabled"`
	CertHashes        []string                   `json:"certHashes"`
	UPID              map[string]json.RawMessage `json:"upid"`
	Network           *ExportMENetwork           `json:"network"`
}

// ExportMENetwork groups ME wired/wireless network settings.
type ExportMENetwork struct {
	Wired    *ExportMEInterface `json:"wired"`
	Wireless *ExportMEInterface `json:"wireless"` // export-gap: no wireless-ME source field yet; always null.
}

// ExportMEInterface is a single ME network interface.
type ExportMEInterface struct {
	IPAddress   string `json:"ipAddress"`
	DHCPEnabled *bool  `json:"dhcpEnabled"`
	DHCPMode    string `json:"dhcpMode"`   // export-gap: no source field yet.
	LinkStatus  string `json:"linkStatus"` // export-gap: no source field yet.
	MACAddress  string `json:"macAddress"` // export-gap: no source field yet.
}

// ExportOS holds operating-system-reported details.
type ExportOS struct {
	DNSSuffix          string           `json:"dnsSuffix"` // export-gap: no OS-reported DNS suffix source field yet.
	Name               string           `json:"name"`
	Version            string           `json:"version"`
	Distro             string           `json:"distro"`
	LMSInstalled       *bool            `json:"lmsInstalled"`
	LMSVersion         string           `json:"lmsVersion"`
	MEInterfaceVersion string           `json:"meInterfaceVersion"`
	MonitorConnected   *bool            `json:"monitorConnected"`
	IEEE8021XEnabled   *bool            `json:"ieee8021xEnabled"`
	Network            *ExportOSNetwork `json:"network"`
}

// ExportOSNetwork groups OS wired/wireless network interfaces.
type ExportOSNetwork struct {
	Wired    []ExportOSInterface `json:"wired"`
	Wireless *ExportOSInterface  `json:"wireless"` // export-gap: no wireless-OS source field yet; always null.
}

// ExportOSInterface is a single OS network interface.
type ExportOSInterface struct {
	Name        string `json:"name"` // export-gap: no source field yet.
	IPAddress   string `json:"ipAddress"`
	DHCPEnabled *bool  `json:"dhcpEnabled"` // export-gap: no source field yet.
	LinkStatus  string `json:"linkStatus"`  // export-gap: no source field yet.
	MACAddress  string `json:"macAddress"`  // export-gap: no source field yet.
}

// ExportPlatform holds platform-level details (CPU, adapters).
type ExportPlatform struct {
	CPU                  string                  `json:"cpu"`
	EthernetAdapterCount *int                    `json:"ethernetAdapterCount"`
	Adapters             *ExportPlatformAdapters `json:"adapters"` // export-gap: no adapter-name source fields yet; always null.
}

// ExportPlatformAdapters summarizes adapter names.
type ExportPlatformAdapters struct {
	Wired    string `json:"wired"`
	Wireless string `json:"wireless"`
}

// ExportBMC holds a high-level baseboard management controller summary.
type ExportBMC struct {
	Vendor          string `json:"vendor"`
	Model           string `json:"model"`
	FirmwareVersion string `json:"firmwareVersion"`
}
