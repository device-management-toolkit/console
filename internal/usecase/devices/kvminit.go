package devices

import (
	"context"
	"time"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
)

// GetKVMInitData retrieves all data needed to initialize a KVM session in a single call.
// This combines display settings, power state, redirection status, and features.
// Note: This endpoint does NOT use cache because it needs to reflect current session state.
func (uc *UseCase) GetKVMInitData(ctx context.Context, guid string) (dto.KVMInitResponse, error) {
	// Fetch all required data (no caching for init endpoint)
	displaySettings, err := uc.GetKVMScreenSettings(ctx, guid)
	if err != nil {
		return dto.KVMInitResponse{}, err
	}

	powerState, err := uc.GetPowerState(ctx, guid)
	if err != nil {
		return dto.KVMInitResponse{}, err
	}

	// Check for active redirection sessions
	redirectionStatus := uc.getRedirectionStatus(guid)

	// Get features (v1 version for compatibility)
	features, _, err := uc.GetFeatures(ctx, guid, false)
	if err != nil {
		return dto.KVMInitResponse{}, err
	}

	response := dto.KVMInitResponse{
		DisplaySettings:   displaySettings,
		PowerState:        powerState,
		RedirectionStatus: redirectionStatus,
		Features: dto.GetFeaturesResponse{
			Redirection:           features.Redirection,
			KVM:                   features.EnableKVM,
			SOL:                   features.EnableSOL,
			IDER:                  features.EnableIDER,
			OptInState:            features.OptInState,
			UserConsent:           features.UserConsent,
			KVMAvailable:          features.KVMAvailable,
			OCR:                   features.OCR,
			HTTPSBootSupported:    features.HTTPSBootSupported,
			WinREBootSupported:    features.WinREBootSupported,
			LocalPBABootSupported: features.LocalPBABootSupported,
			RemoteErase:           features.RemoteErase,
		},
	}

	// Do NOT cache this endpoint - it needs to reflect current session state
	return response, nil
}

// getRedirectionStatus checks if there are active redirection sessions for the device.
func (uc *UseCase) getRedirectionStatus(guid string) dto.KVMRedirectionStatus {
	status := dto.KVMRedirectionStatus{
		IsSOLConnected:  false,
		IsIDERConnected: false,
	}

	uc.redirMutex.RLock()
	defer uc.redirMutex.RUnlock()

	// Check for SOL connection
	if conn, exists := uc.redirConnections[guid+"-sol"]; exists {
		conn.mu.RLock()
		// Consider connected if not expired
		status.IsSOLConnected = time.Since(conn.lastActivity) <= ConnectionTimeout
		conn.mu.RUnlock()
	}

	// Check for IDER connection
	if conn, exists := uc.redirConnections[guid+"-ider"]; exists {
		conn.mu.RLock()
		status.IsIDERConnected = time.Since(conn.lastActivity) <= ConnectionTimeout
		conn.mu.RUnlock()
	}

	return status
}
