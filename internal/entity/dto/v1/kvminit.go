package dto

// KVMInitResponse combines all data needed to initialize KVM session.
// This reduces multiple API calls into a single request.
type KVMInitResponse struct {
	DisplaySettings   KVMScreenSettings    `json:"displaySettings"`
	PowerState        PowerState           `json:"powerState"`
	RedirectionStatus KVMRedirectionStatus `json:"redirectionStatus"`
	Features          GetFeaturesResponse  `json:"features"`
}

// KVMRedirectionStatus represents the status of redirection services.
type KVMRedirectionStatus struct {
	IsSOLConnected  bool `json:"isSOLConnected"`
	IsIDERConnected bool `json:"isIDERConnected"`
}
