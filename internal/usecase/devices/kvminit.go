package devices

import (
	"context"
	"time"

	"github.com/device-management-toolkit/console/internal/cache"
	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
)

const kvmInitCacheTTL = 30 * time.Second

// GetKVMInitData retrieves all data needed to initialize a KVM session in a single call.
// This combines display settings, power state, redirection status, and features.
func (uc *UseCase) GetKVMInitData(ctx context.Context, guid string) (dto.KVMInitResponse, error) {
	// Check cache first - use medium TTL since this is initialization data
	cacheKey := cache.MakeKVMInitKey(guid)
	if cached, found := uc.cache.Get(cacheKey); found {
		if initData, ok := cached.(dto.KVMInitResponse); ok {
			uc.log.Info("Cache hit for KVM init data", "guid", guid)

			return initData, nil
		}
	}

	uc.log.Info("Cache miss for KVM init data, fetching from AMT", "guid", guid)

	// Fetch all required data
	displaySettings, err := uc.GetKVMScreenSettings(ctx, guid)
	if err != nil {
		return dto.KVMInitResponse{}, err
	}

	powerState, err := uc.GetPowerState(ctx, guid)
	if err != nil {
		return dto.KVMInitResponse{}, err
	}

	// Redirection status is currently just placeholders
	redirectionStatus := dto.KVMRedirectionStatus{
		IsSOLConnected:  false,
		IsIDERConnected: false,
	}

	// Get features (v1 version for compatibility)
	features, _, err := uc.GetFeatures(ctx, guid)
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

	// Cache with 30 second TTL - short since power state can change
	uc.cache.Set(cacheKey, response, kvmInitCacheTTL)

	return response, nil
}
