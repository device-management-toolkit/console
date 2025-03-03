package dto

type BootSetting struct {
	Action int    `json:"action" binding:"required" example:"8"`
	Value  string `json:"value" binding:"omitempty,required" example:"http://"`
	UseSOL bool   `json:"useSOL" binding:"omitempty,required" example:"true"`
}
