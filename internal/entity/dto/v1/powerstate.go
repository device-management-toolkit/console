package dto

type PowerState struct {
	PowerState         int `json:"powerstate" binding:"required" example:"0"`
	OSPowerSavingState int `json:"osPowerSavingState" binding:"required" example:"0"`
}
