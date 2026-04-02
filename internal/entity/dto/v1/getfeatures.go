package dto

type GetFeaturesResponse struct {
	Redirection           bool   `json:"redirection" binding:"required" example:"true"`
	KVM                   bool   `json:"KVM" binding:"required" example:"true"`
	SOL                   bool   `json:"SOL" binding:"required" example:"true"`
	IDER                  bool   `json:"IDER" binding:"required" example:"true"`
	OptInState            int    `json:"optInState" binding:"required" example:"0"`
	UserConsent           string `json:"userConsent" binding:"required" example:"none"`
	KVMAvailable          bool   `json:"kvmAvailable" binding:"required" example:"true"`
	OCR                   bool   `json:"ocr" binding:"required" example:"false"`
	HTTPSBootSupported    bool   `json:"httpsBootSupported" binding:"required" example:"false"`
	WinREBootSupported    bool   `json:"winREBootSupported" binding:"required" example:"false"`
	LocalPBABootSupported bool   `json:"localPBABootSupported" binding:"required" example:"false"`
	RemoteErase           bool   `json:"remoteErase" binding:"required" example:"false"`
}
