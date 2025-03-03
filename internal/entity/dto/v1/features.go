package dto

type Features struct {
	UserConsent           string `json:"userConsent" example:"kvm"`
	EnableSOL             bool   `json:"enableSOL" example:"true"`
	EnableIDER            bool   `json:"enableIDER" example:"true"`
	EnableKVM             bool   `json:"enableKVM" example:"true"`
	Redirection           bool   `json:"redirection" example:"true"`
	OptInState            int    `json:"optInState" example:"0"`
	KVMAvailable          bool   `json:"kvmAvailable" example:"true"`
	OCR                   bool   `json:"ocr" example:"true"`
	HTTPSBootSupported    bool   `json:"httpsBootSupported" example:"true"`
	WinREBootSupported    bool   `json:"winREBootSupported" example:"true"`
	LocalPBABootSupported bool   `json:"localPBABootSupported" example:"true"`
	RemoteErase           bool   `json:"remoteErase" example:"true"`
}

type FeaturesRequest struct {
	UserConsent string `json:"userConsent" example:"kvm"`
	EnableSOL   bool   `json:"enableSOL" example:"true"`
	EnableIDER  bool   `json:"enableIDER" example:"true"`
	EnableKVM   bool   `json:"enableKVM" example:"true"`
	OCR         bool   `json:"ocr" example:"true"`
	RemoteErase bool   `json:"remoteErase" example:"true"`
}
