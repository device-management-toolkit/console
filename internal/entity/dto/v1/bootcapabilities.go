package dto

type RPERequest struct {
	Enabled bool `json:"enabled"`
}

type RemoteEraseRequest struct {
	EraseMask int `json:"eraseMask"`
}

type BootCapabilities struct {
	IDER                               bool `json:"IDER,omitempty"`
	SOL                                bool `json:"SOL,omitempty"`
	BIOSReflash                        bool `json:"BIOSReflash,omitempty"`
	BIOSSetup                          bool `json:"BIOSSetup,omitempty"`
	BIOSPause                          bool `json:"BIOSPause,omitempty"`
	ForcePXEBoot                       bool `json:"ForcePXEBoot,omitempty"`
	ForceHardDriveBoot                 bool `json:"ForceHardDriveBoot,omitempty"`
	ForceHardDriveSafeModeBoot         bool `json:"ForceHardDriveSafeModeBoot,omitempty"`
	ForceDiagnosticBoot                bool `json:"ForceDiagnosticBoot,omitempty"`
	ForceCDorDVDBoot                   bool `json:"ForceCDorDVDBoot,omitempty"`
	VerbosityScreenBlank               bool `json:"VerbosityScreenBlank,omitempty"`
	PowerButtonLock                    bool `json:"PowerButtonLock,omitempty"`
	ResetButtonLock                    bool `json:"ResetButtonLock,omitempty"`
	KeyboardLock                       bool `json:"KeyboardLock,omitempty"`
	SleepButtonLock                    bool `json:"SleepButtonLock,omitempty"`
	UserPasswordBypass                 bool `json:"UserPasswordBypass,omitempty"`
	ForcedProgressEvents               bool `json:"ForcedProgressEvents,omitempty"`
	VerbosityVerbose                   bool `json:"VerbosityVerbose,omitempty"`
	VerbosityQuiet                     bool `json:"VerbosityQuiet,omitempty"`
	ConfigurationDataReset             bool `json:"ConfigurationDataReset,omitempty"`
	BIOSSecureBoot                     bool `json:"BIOSSecureBoot,omitempty"`
	SecureErase                        bool `json:"SecureErase,omitempty"`
	ForceWinREBoot                     bool `json:"ForceWinREBoot,omitempty"`
	ForceUEFILocalPBABoot              bool `json:"ForceUEFILocalPBABoot,omitempty"`
	ForceUEFIHTTPSBoot                 bool `json:"ForceUEFIHTTPSBoot,omitempty"`
	AMTSecureBootControl               bool `json:"AMTSecureBootControl,omitempty"`
	UEFIWiFiCoExistenceAndProfileShare bool `json:"UEFIWiFiCoExistenceAndProfileShare,omitempty"`
	PlatformErase                      int  `json:"PlatformErase,omitempty"`
}
