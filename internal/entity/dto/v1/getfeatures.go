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
	RPE                   bool   `json:"rpe" binding:"required" example:"false"`
	RPESupported          bool   `json:"rpeSupported" example:"false"`
	RPECaps               int    `json:"rpeCaps" example:"0"`
	RPESecureErase        bool   `json:"rpeSecureErase" example:"false"`
	RPETPMClear           bool   `json:"rpeTPMClear" example:"false"`
	RPEClearBIOSNVM       bool   `json:"rpeClearBIOSNVM" example:"false"`
	RPEBIOSReload         bool   `json:"rpeBIOSReload" example:"false"`
}
