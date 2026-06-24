package devices

import (
	"context"
	"errors"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/wifiportconfiguration"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
)

// ErrUEFIProfileSyncNotSupported is returned when a caller requests UEFI WiFi
// profile sharing on a device whose power capabilities do not support it. The
// HTTP layer maps this to 409 Conflict and the whole request is rejected (no
// partial update).
var ErrUEFIProfileSyncNotSupported = errors.New("UEFI WiFi profile sync is not supported on this device")

func (uc *UseCase) GetWirelessProfileSync(c context.Context, guid string) (dto.WirelessProfileSyncResponse, error) {
	device, err := uc.setupWirelessProfileManagement(c, guid)
	if err != nil {
		return dto.WirelessProfileSyncResponse{}, err
	}

	if _, err := device.GetWiFiPorts(); err != nil {
		return dto.WirelessProfileSyncResponse{}, err
	}

	response, err := device.GetWiFiPortConfigurationService()
	if err != nil {
		return dto.WirelessProfileSyncResponse{}, err
	}

	powerCapabilities, err := device.GetBootCapabilities()
	if err != nil {
		return dto.WirelessProfileSyncResponse{}, err
	}

	uefiSupported := powerCapabilities.UEFIWiFiCoExistenceAndProfileShare

	return buildProfileSyncResponse(response, uefiSupported), nil
}

func (uc *UseCase) SetWirelessProfileSync(c context.Context, guid string, req dto.WirelessProfileSyncRequest) (dto.WirelessProfileSyncResponse, error) {
	device, err := uc.setupWirelessProfileManagement(c, guid)
	if err != nil {
		return dto.WirelessProfileSyncResponse{}, err
	}

	if _, err := device.GetWiFiPorts(); err != nil {
		return dto.WirelessProfileSyncResponse{}, err
	}

	current, err := device.GetWiFiPortConfigurationService()
	if err != nil {
		return dto.WirelessProfileSyncResponse{}, err
	}

	powerCapabilities, err := device.GetBootCapabilities()
	if err != nil {
		return dto.WirelessProfileSyncResponse{}, err
	}

	uefiSupported := powerCapabilities.UEFIWiFiCoExistenceAndProfileShare

	// Reject the entire request (no partial update) when UEFI sync is explicitly
	// requested on a device that does not support it.
	if req.UEFIProfileSync != nil && *req.UEFIProfileSync && !uefiSupported {
		return dto.WirelessProfileSyncResponse{}, ErrUEFIProfileSyncNotSupported
	}

	localSyncState := current.LocalProfileSynchronizationEnabled
	if req.LocalProfileSync != nil {
		localSyncState = wifiportconfiguration.LocalSyncDisabled
		if *req.LocalProfileSync {
			localSyncState = wifiportconfiguration.LocalUserProfileSynchronizationEnabled
		}
	}

	uefiSyncState := current.UEFIWiFiProfileShareEnabled
	if req.UEFIProfileSync != nil {
		uefiSyncState = *req.UEFIProfileSync
	}

	if localSyncState != current.LocalProfileSynchronizationEnabled || uefiSyncState != current.UEFIWiFiProfileShareEnabled {
		putRequest := buildWiFiPortConfigRequest(current, localSyncState, uefiSyncState)

		updatedResponse, err := device.PutWiFiPortConfigurationService(putRequest)
		if err != nil {
			return dto.WirelessProfileSyncResponse{}, err
		}

		return buildProfileSyncResponse(updatedResponse, uefiSupported), nil
	}

	return buildProfileSyncResponse(current, uefiSupported), nil
}

func buildProfileSyncResponse(cfg wifiportconfiguration.WiFiPortConfigurationServiceResponse, uefiSupported bool) dto.WirelessProfileSyncResponse {
	return dto.WirelessProfileSyncResponse{
		LocalProfileSync:         isLocalProfileSyncEnabled(cfg.LocalProfileSynchronizationEnabled),
		UEFIProfileSync:          cfg.UEFIWiFiProfileShareEnabled,
		UEFIProfileSyncSupported: uefiSupported,
	}
}

func buildWiFiPortConfigRequest(
	current wifiportconfiguration.WiFiPortConfigurationServiceResponse,
	localSyncState wifiportconfiguration.LocalProfileSynchronizationEnabled,
	uefiWiFiSyncState bool,
) wifiportconfiguration.WiFiPortConfigurationServiceRequest {
	return wifiportconfiguration.WiFiPortConfigurationServiceRequest{
		RequestedState:                     current.RequestedState,
		EnabledState:                       current.EnabledState,
		HealthState:                        current.HealthState,
		ElementName:                        current.ElementName,
		SystemCreationClassName:            current.SystemCreationClassName,
		SystemName:                         current.SystemName,
		CreationClassName:                  current.CreationClassName,
		Name:                               current.Name,
		LocalProfileSynchronizationEnabled: localSyncState,
		LastConnectedSsidUnderMeControl:    current.LastConnectedSsidUnderMeControl,
		NoHostCsmeSoftwarePolicy:           current.NoHostCsmeSoftwarePolicy,
		UEFIWiFiProfileShareEnabled:        uefiWiFiSyncState,
	}
}

func isLocalProfileSyncEnabled(state wifiportconfiguration.LocalProfileSynchronizationEnabled) bool {
	return state == wifiportconfiguration.LocalUserProfileSynchronizationEnabled
}
