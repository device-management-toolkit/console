package dto

type PowerState struct {
	PowerState         int `json:"powerstate" example:"0"`
	OSPowerSavingState int `json:"osPowerSavingState" example:"0"`
}
