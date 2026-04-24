package openapi

import (
	"github.com/go-fuego/fuego"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	dtov2 "github.com/device-management-toolkit/console/internal/entity/dto/v2"
)

type V2VersionResponse dtov2.Version

type V2FeaturesResponse dtov2.Features

func (f *FuegoAdapter) RegisterV2AMTRoutes() {
	fuego.Get(f.server, "/api/v2/amt/version/{guid}", f.getV2Version,
		fuego.OptionTags("Device Management V2"),
		fuego.OptionSummary("Get Version (v2)"),
		fuego.OptionDescription("Retrieve AMT/software version information for a device (v2)"),
		fuego.OptionPath("guid", "Device GUID"),
	)

	fuego.Get(f.server, "/api/v2/amt/features/{guid}", f.getV2Features,
		fuego.OptionTags("Device Management V2"),
		fuego.OptionSummary("Get Features (v2)"),
		fuego.OptionDescription("Retrieve feature flags for a device (v2)"),
		fuego.OptionPath("guid", "Device GUID"),
	)

	fuego.Post(f.server, "/api/v2/amt/features/{guid}", f.setV2Features,
		fuego.OptionTags("Device Management V2"),
		fuego.OptionSummary("Set Features (v2)"),
		fuego.OptionDescription("Update feature flags for a device (v2)"),
		fuego.OptionPath("guid", "Device GUID"),
	)
}

func (f *FuegoAdapter) getV2Version(_ fuego.ContextNoBody) (V2VersionResponse, error) {
	return V2VersionResponse{}, nil
}

func (f *FuegoAdapter) getV2Features(_ fuego.ContextNoBody) (V2FeaturesResponse, error) {
	return V2FeaturesResponse{}, nil
}

func (f *FuegoAdapter) setV2Features(c fuego.ContextWithBody[dto.Features]) (V2FeaturesResponse, error) {
	_, err := c.Body()
	if err != nil {
		return V2FeaturesResponse{}, err
	}

	return V2FeaturesResponse{}, nil
}
