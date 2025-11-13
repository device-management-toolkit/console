package dto

// KVMScreenDisplay represents one display's status and geometry.
type KVMScreenDisplay struct {
	DisplayIndex int    `json:"displayIndex"`
	IsActive     bool   `json:"isActive"`
	ResolutionX  int    `json:"resolutionX"`
	ResolutionY  int    `json:"resolutionY"`
	UpperLeftX   int    `json:"upperLeftX"`
	UpperLeftY   int    `json:"upperLeftY"`
	Role         string `json:"role,omitempty"` // primary, secondary, tertiary, quaternary
	IsDefault    bool   `json:"isDefault"`
}

// KVMScreenSettings represents a simplified view of IPS_ScreenSettingData
// Displays contains a flattened view across all returned ScreenSettingData items.
type KVMScreenSettings struct {
	Displays []KVMScreenDisplay `json:"displays"`
}

// KVMScreenSettingsRequest allows updating screen settings; schema may vary by platform.
// Keep it generic for now until wsman Put support exists.
type KVMScreenSettingsRequest struct {
	DisplayIndex int `json:"displayIndex,omitempty"`
}
