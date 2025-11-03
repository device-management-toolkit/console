package dto

type PowerAction struct {
	Action int `json:"action" binding:"required" example:"8"`
}

type BootSources struct {
	BIOSBootString       string `json:"biosBootString" example:"string"`
	BootString           string `json:"bootString" example:"string"`
	ElementName          string `json:"elementName" example:"Intel® AMT: Boot Source"`
	FailThroughSupported int    `json:"failThroughSupported" example:"2"`
	InstanceID           string `json:"instanceID" example:"Intel® AMT: Force Hard-drive Boot"`
	StructuredBootString string `json:"structuredBiosBootString" example:"CIM:Hard-Disk:1"`
}

type PowerActionResponse struct {
	ReturnValue int `json:"ReturnValue" example:"0"` // Return code. 0 indicates success
}
