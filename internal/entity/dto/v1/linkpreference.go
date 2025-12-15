package dto

// LinkPreferenceRequest represents the request to set link preference on a device.
type LinkPreferenceRequest struct {
	LinkPreference int `json:"linkPreference" binding:"required,min=1,max=2"` // 1 for ME, 2 for HOST
	Timeout        int `json:"timeout" binding:"required,min=0"`              // Timeout in seconds
}

// LinkPreferenceResponse represents the response from setting link preference.
type LinkPreferenceResponse struct {
	ReturnValue int `json:"returnValue" example:"0"` // Return code. 0 indicates success
}

// LinkPreference enumeration values
const (
	LinkPreferenceME   = 1 // Management Engine
	LinkPreferenceHost = 2 // Host
)

// Console-specific return value for no WiFi port found
const ReturnValueNoWiFiPort = -1
