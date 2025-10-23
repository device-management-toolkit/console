package openapi

import (
	"github.com/go-fuego/fuego"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
)

func (f *FuegoAdapter) RegisterCIRAConfigRoutes() {
	fuego.Get(f.server, "/api/v1/admin/ciraconfigs", f.getCIRAConfigs,
		fuego.OptionTags("CIRA"),
		fuego.OptionSummary("List CIRA Configurations"),
		fuego.OptionDescription("Retrieve all CIRA configurations with optional pagination"),
		fuego.OptionQueryInt("$top", "Number of records to return"),
		fuego.OptionQueryInt("$skip", "Number of records to skip"),
		fuego.OptionQueryBool("$count", "Include total count"),
	)

	fuego.Get(f.server, "/api/v1/admin/ciraconfigs/{name}", f.getCIRAConfigByName,
		fuego.OptionTags("CIRA"),
		fuego.OptionSummary("Get CIRA Configuration by Name"),
		fuego.OptionDescription("Retrieve a specific CIRA configuration by profile name"),
		fuego.OptionPath("name", "Profile name"),
	)

	fuego.Post(f.server, "/api/v1/admin/ciraconfigs", f.createCIRAConfig,
		fuego.OptionTags("CIRA"),
		fuego.OptionSummary("Create CIRA Configuration"),
		fuego.OptionDescription("Create a new CIRA configuration"),
	)

	fuego.Patch(f.server, "/api/v1/admin/ciraconfigs", f.updateCIRAConfig,
		fuego.OptionTags("CIRA"),
		fuego.OptionSummary("Update CIRA Configuration"),
		fuego.OptionDescription("Update an existing CIRA configuration"),
	)

	fuego.Delete(f.server, "/api/v1/admin/ciraconfigs/{name}", f.deleteCIRAConfig,
		fuego.OptionTags("CIRA"),
		fuego.OptionSummary("Delete CIRA Configuration"),
		fuego.OptionDescription("Delete a CIRA configuration by profile name"),
		fuego.OptionPath("name", "Profile name"),
	)
}

func (f *FuegoAdapter) getCIRAConfigs(_ fuego.ContextNoBody) (dto.CIRAConfigCountResponse, error) {
	configs := []dto.CIRAConfig{
		{
			ConfigName:          "My CIRA Config",
			MPSAddress:          "https://example.com",
			MPSPort:             4433,
			Username:            "my_username",
			Password:            "my_password",
			CommonName:          "example.com",
			ServerAddressFormat: 201, // 3 = IPV4, 4 = IPV6, 201 = FQDN
			AuthMethod:          2,   // 1 = Mutual Auth, 2 = Username and Password
			MPSRootCertificate:  "-----BEGIN CERTIFICATE-----\n...",
			ProxyDetails:        "http://example.com",
			TenantID:            "abc123",
			RegeneratePassword:  true,
		},
	}

	return dto.CIRAConfigCountResponse{
		Count: len(configs),
		Data:  configs,
	}, nil
}

func (f *FuegoAdapter) getCIRAConfigByName(c fuego.ContextNoBody) (dto.CIRAConfig, error) {
	profileName := c.PathParam("name")

	return dto.CIRAConfig{
		ConfigName:          profileName,
		MPSAddress:          "https://example.com",
		MPSPort:             4433,
		Username:            "my_username",
		Password:            "my_password",
		CommonName:          "example.com",
		ServerAddressFormat: 201,
		AuthMethod:          2,
		MPSRootCertificate:  "-----BEGIN CERTIFICATE-----\n...",
		ProxyDetails:        "http://example.com",
		TenantID:            "abc123",
		RegeneratePassword:  false,
	}, nil
}

func (f *FuegoAdapter) createCIRAConfig(c fuego.ContextWithBody[dto.CIRAConfig]) (dto.CIRAConfig, error) {
	config, err := c.Body()
	if err != nil {
		return dto.CIRAConfig{}, err
	}

	return config, nil
}

func (f *FuegoAdapter) updateCIRAConfig(c fuego.ContextWithBody[dto.CIRAConfig]) (dto.CIRAConfig, error) {
	config, err := c.Body()
	if err != nil {
		return dto.CIRAConfig{}, err
	}

	return config, nil
}

func (f *FuegoAdapter) deleteCIRAConfig(_ fuego.ContextNoBody) (any, error) {
	return nil, nil
}
