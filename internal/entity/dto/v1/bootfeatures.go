package dto

type BootFeatures struct {
	HTTPBoot bool `json:"httpBoot,omitempty" example:"true"`
	HTTPBootSupport bool `json:"httpBootSupport,omitempty" example:"true"`
	RemoteErase bool `json:"remoteErase,omitempty" example:"true"`
}

type BootFeaturesRequest struct {
	HTTPBoot bool `json:"httpBoot,omitempty" example:"true"`
	RemoteErase bool `json:"remoteErase,omitempty" example:"true"`
}