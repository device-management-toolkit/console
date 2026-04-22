package dto

type RPERequest struct {
	Enabled bool `json:"enabled"`
}

type RemoteEraseRequest struct {
	EraseMask int `json:"eraseMask"`
}

type BootCapabilities struct {
	PlatformErase int `json:"PlatformErase,omitempty"`
}
