package dto

type BootDetails struct {
	URL               string `json:"url" example:"https://"`
	Username          string `json:"username" example:"admin"`
	Password          string `json:"password" example:"password"`
	BootPath          string `json:"bootPath" example:"\\OemPba.efi"`
	EnforceSecureBoot bool   `json:"enforceSecureBoot" example:"true"`
}

type BootSetting struct {
	Action      int         `json:"action" binding:"required" example:"8"`
	BootDetails BootDetails `json:"bootDetails" binding:"omitempty"`
	UseSOL      bool        `json:"useSOL" binding:"omitempty,required" example:"true"`
}
