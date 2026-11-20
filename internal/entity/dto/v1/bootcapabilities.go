package dto

type RemoteEraseRequest struct {
	SecureEraseAllSSDs bool   `json:"secureEraseAllSSDs"`
	TPMClear           bool   `json:"tpmClear"`
	RestoreBIOSToEOM   bool   `json:"restoreBIOSToEOM"`
	UnconfigureCSME    bool   `json:"unconfigureCSME"`
	SSDPassword        string `json:"ssdPassword,omitempty"`
	PSID               string `json:"psid,omitempty"`
}

type BootCapabilities struct {
	SecureEraseAllSSDs bool `json:"secureEraseAllSSDs"`
	TPMClear           bool `json:"tpmClear"`
	RestoreBIOSToEOM   bool `json:"restoreBIOSToEOM"`
	UnconfigureCSME    bool `json:"unconfigureCSME"`
}
