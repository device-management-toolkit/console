package openapi

import (
	"net/http"

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
		protectedRouteOptions(),
	)

	fuego.Get(f.server, "/api/v1/admin/wirelessconfigs/{profileName}", f.getWirelessConfigByName,
		fuego.OptionTags("Wireless"),
		fuego.OptionSummary("Get Wireless Configuration by Name"),
		fuego.OptionDescription("Retrieve a specific wireless configuration by profile name"),
		fuego.OptionPath("profileName", "Profile name"),
		protectedRouteOptions(),
	)

	fuego.Post(f.server, "/api/v1/admin/wirelessconfigs", f.createWirelessConfig,
		fuego.OptionTags("Wireless"),
		fuego.OptionSummary("Create Wireless Configuration"),
		fuego.OptionDescription("Create a new wireless configuration"),
		fuego.OptionDefaultStatusCode(http.StatusCreated),
		protectedRouteOptions(),
	)

	fuego.Patch(f.server, "/api/v1/admin/wirelessconfigs", f.updateWirelessConfig,
		fuego.OptionTags("Wireless"),
		fuego.OptionSummary("Update Wireless Configuration"),
		fuego.OptionDescription("Update an existing wireless configuration"),
		protectedRouteOptions(),
	)

	fuego.Delete(f.server, "/api/v1/admin/wirelessconfigs/{profileName}", f.deleteWirelessConfig,
		fuego.OptionTags("Wireless"),
		fuego.OptionSummary("Delete Wireless Configuration"),
		fuego.OptionDescription("Delete a wireless configuration by profile name"),
		fuego.OptionPath("profileName", "Profile name"),
		fuego.OptionDefaultStatusCode(http.StatusNoContent),
		protectedRouteOptions(),
	)
}

func (f *FuegoAdapter) getWirelessConfigs(_ fuego.ContextNoBody) (dto.WirelessConfigCountResponse, error) {
	configs := []dto.WirelessConfig{
		{
			ProfileName:          "example-wifi",
			SSID:                 "ExampleSSID",
			AuthenticationMethod: 6, // WPA2-Personal
			EncryptionMethod:     4,
			TenantID:             defaultTenantID,
			Version:              "1.0",
		},
	}

	return dto.WirelessConfigCountResponse{
		Count: len(configs),
		Data:  configs,
	}, nil
}

func (f *FuegoAdapter) getWirelessConfigByName(c fuego.ContextNoBody) (dto.WirelessConfig, error) {
	profileName := c.PathParam("profileName")

	return dto.WirelessConfig{
		ProfileName:          profileName,
		SSID:                 "ExampleSSID",
		AuthenticationMethod: 6,
		EncryptionMethod:     4,
		TenantID:             defaultTenantID,
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

func (f *FuegoAdapter) deleteWirelessConfig(c fuego.ContextNoBody) (NoContentResponse, error) {
	profileName := c.PathParam("profileName")
	f.logger.Info("Deleting wireless config: " + profileName)

	return NoContentResponse{}, nil
}
