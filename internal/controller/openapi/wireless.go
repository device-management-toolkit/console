package openapi

import (
	"github.com/go-fuego/fuego"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
)

func (f *FuegoAdapter) RegisterWirelessConfigRoutes() {
	fuego.Get(f.server, "/api/v1/admin/wirelessconfigs", f.getWirelessConfigs,
		fuego.OptionTags("Wireless"),
		fuego.OptionSummary("List Wireless Configurations"),
		fuego.OptionDescription("Retrieve all wireless configurations with optional pagination"),
		fuego.OptionQueryInt("$top", "Number of records to return"),
		fuego.OptionQueryInt("$skip", "Number of records to skip"),
		fuego.OptionQueryBool("$count", "Include total count"),
	)

	fuego.Get(f.server, "/api/v1/admin/wirelessconfigs/{name}", f.getWirelessConfigByName,
		fuego.OptionTags("Wireless"),
		fuego.OptionSummary("Get Wireless Configuration by Name"),
		fuego.OptionDescription("Retrieve a specific wireless configuration by profile name"),
		fuego.OptionPath("name", "Profile name"),
	)

	fuego.Post(f.server, "/api/v1/admin/wirelessconfigs", f.createWirelessConfig,
		fuego.OptionTags("Wireless"),
		fuego.OptionSummary("Create Wireless Configuration"),
		fuego.OptionDescription("Create a new wireless configuration"),
	)

	fuego.Patch(f.server, "/api/v1/admin/wirelessconfigs", f.updateWirelessConfig,
		fuego.OptionTags("Wireless"),
		fuego.OptionSummary("Update Wireless Configuration"),
		fuego.OptionDescription("Update an existing wireless configuration"),
	)

	fuego.Delete(f.server, "/api/v1/admin/wirelessconfigs/{name}", f.deleteWirelessConfig,
		fuego.OptionTags("Wireless"),
		fuego.OptionSummary("Delete Wireless Configuration"),
		fuego.OptionDescription("Delete a wireless configuration by profile name"),
		fuego.OptionPath("name", "Profile name"),
	)
}

func (f *FuegoAdapter) getWirelessConfigs(_ fuego.ContextNoBody) (dto.WirelessConfigCountResponse, error) {
	configs := []dto.WirelessConfig{
		{
			ProfileName:          "example-wifi",
			SSID:                 "ExampleSSID",
			AuthenticationMethod: 6, // WPA2-Personal
			EncryptionMethod:     4,
			TenantID:             "default",
			Version:              "1.0",
		},
	}

	return dto.WirelessConfigCountResponse{
		Count: len(configs),
		Data:  configs,
	}, nil
}

func (f *FuegoAdapter) getWirelessConfigByName(_ fuego.ContextNoBody) (dto.WirelessConfig, error) {
	return dto.WirelessConfig{
		ProfileName:          "example-wifi",
		SSID:                 "ExampleSSID",
		AuthenticationMethod: 6,
		EncryptionMethod:     4,
		TenantID:             "default",
		Version:              "1.0",
	}, nil
}

func (f *FuegoAdapter) createWirelessConfig(c fuego.ContextWithBody[dto.WirelessConfig]) (dto.WirelessConfig, error) {
	config, err := c.Body()
	if err != nil {
		return dto.WirelessConfig{}, err
	}

	return config, nil
}

func (f *FuegoAdapter) updateWirelessConfig(c fuego.ContextWithBody[dto.WirelessConfig]) (dto.WirelessConfig, error) {
	config, err := c.Body()
	if err != nil {
		return dto.WirelessConfig{}, err
	}

	return config, nil
}

func (f *FuegoAdapter) deleteWirelessConfig(c fuego.ContextNoBody) (any, error) {
	profileName := c.PathParam("name")
	f.logger.Info("Deleting wireless config: " + profileName)

	return nil, nil
}
