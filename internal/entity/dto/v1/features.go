package dto

type Features struct {
	UserConsent     string `json:"userConsent" example:"kvm"`
	EnableSOL       bool   `json:"enableSOL" example:"true"`
	EnableIDER      bool   `json:"enableIDER" example:"true"`
	EnableKVM       bool   `json:"enableKVM" example:"true"`
	Redirection     bool   `json:"redirection" example:"true"`
	OptInState      int    `json:"optInState" example:"0"`
	KVMAvailable    bool   `json:"kvmAvailable" example:"true"`
	HTTPBoot        bool   `json:"httpBoot,omitempty" example:"true"`
	HTTPBootSupport bool   `json:"httpBootSupport,omitempty" example:"true"`
	RemoteErase     bool   `json:"remoteErase,omitempty" example:"true"`
}

type FeaturesRequest struct {
	UserConsent string `json:"userConsent" example:"kvm"`
	EnableSOL   bool   `json:"enableSOL" example:"true"`
	EnableIDER  bool   `json:"enableIDER" example:"true"`
	EnableKVM   bool   `json:"enableKVM" example:"true"`
	HTTPBoot    bool   `json:"httpBoot,omitempty" example:"true"`
	RemoteErase bool   `json:"remoteErase,omitempty" example:"true"`
}
