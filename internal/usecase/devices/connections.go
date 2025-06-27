package devices

import (
	"context"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/tls"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
)

func (uc *UseCase) GetTLSSettingData(c context.Context, guid string) ([]dto.SettingDataResponse, error) {
	item, err := uc.repo.GetByID(c, guid, "")
	if err != nil {
		return nil, err
	}

	if item == nil || item.GUID == "" {
		return nil, ErrNotFound
	}

	device := uc.device.SetupWsmanClient(*item, false, true)

	response, err := device.GetTLSSettingData()
	if err != nil {
		return nil, err
	}

	// iterate over the data and convert each entity to dto
	d1 := make([]dto.SettingDataResponse, len(response))

	for i := range response {
		tmpEntity := response[i] // create a new variable to avoid memory aliasing
		d1[i] = *uc.tlsSettingDataEntityToDTO(&tmpEntity)
	}

	return d1, nil
}

func (uc *UseCase) tlsSettingDataEntityToDTO(d *tls.SettingDataResponse) *dto.SettingDataResponse {
	d1 := &dto.SettingDataResponse{
		ElementName:                   d.ElementName,
		InstanceID:                    d.InstanceID,
		MutualAuthentication:          d.MutualAuthentication,
		Enabled:                       d.Enabled,
		TrustedCN:                     d.TrustedCN,
		AcceptNonSecureConnections:    d.AcceptNonSecureConnections,
		NonSecureConnectionsSupported: d.NonSecureConnectionsSupported,
	}

	return d1
}
