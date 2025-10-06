package v2

type BootParams struct {
	InstanceID     string
	BIOSBootString string
	BootString     string
}

type BootSettings struct {
	IsHTTPSBootExists bool
	IsPBAExists       bool
	IsWinREExists     bool
}

type Features struct {
	UserConsent           string `json:"userConsent" example:"kvm"`
	EnableSOL             bool   `json:"enableSOL" example:"true"`
	EnableIDER            bool   `json:"enableIDER" example:"true"`
	EnableKVM             bool   `json:"enableKVM" example:"true"`
	Redirection           bool   `json:"redirection" example:"true"`
	OptInState            int    `json:"optInState" example:"0"`
	KVMAvailable          bool   `json:"kvmAvailable" example:"true"`
	OCR                   bool   `json:"httpBoot" example:"true"`
	HTTPSBootSupported    bool   `json:"httpBootSupported,omitempty" example:"true"`
	WinREBootSupported    bool   `json:"winREBootSupported,omitempty" example:"true"`
	LocalPBABootSupported bool   `json:"localPBABootSupported,omitempty" example:"true"`
	RemoteErase           bool   `json:"remoteErase" example:"true"`
}
