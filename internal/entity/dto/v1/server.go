package dto

type (
	// ServerFeatures reports server-level capability flags so clients (e.g. the
	// Sample Web UI) can toggle features that depend on how Console was started.
	// New flags should be added here as additional fields rather than via a new
	// endpoint.
	ServerFeatures struct {
		CIRAEnabled bool `json:"ciraEnabled" example:"true"`
	}
)
