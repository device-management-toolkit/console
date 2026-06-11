package dto

type RPERequest struct {
	Enabled bool `json:"enabled"`
}

type RemoteEraseRequest struct {
	SecureEraseAllSSDs bool   `json:"secureEraseAllSSDs"`
	TPMClear           bool   `json:"tpmClear"`
	RestoreBIOSToEOM   bool   `json:"restoreBIOSToEOM"`
	UnconfigureCSME    bool   `json:"unconfigureCSME"`
	SSDPassword        string `json:"ssdPassword,omitempty"`
}

type BootCapabilities struct {
	SecureEraseAllSSDs bool `json:"secureEraseAllSSDs"`
	TPMClear           bool `json:"tpmClear"`
	RestoreBIOSToEOM   bool `json:"restoreBIOSToEOM"`
	UnconfigureCSME    bool `json:"unconfigureCSME"`
}
