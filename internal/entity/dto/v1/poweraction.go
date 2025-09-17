package dto

type PowerAction struct {
	Action int `json:"action" binding:"required" example:"8"`
}

type BootSources struct {
	BootOption  string `json:"bootOption" binding:"required" example:"Hard-drive Boot"`
	BootPath    string `json:"bootPath" binding:"required" example:"\\OemPba.efi"`
	Description string `json:"description" binding:"required" example:"Boot from Hard Drive"`
}
