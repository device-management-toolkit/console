package openapi

import (
	"github.com/go-fuego/fuego"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
)

func (f *FuegoAdapter) RegisterIEEE8021xConfigRoutes() {
	fuego.Get(f.server, "/api/v1/admin/ieee8021xconfigs", f.getIEEE8021xConfigs,
		fuego.OptionTags("IEEE 802.1x"),
		fuego.OptionSummary("List IEEE 802.1x Configurations"),
		fuego.OptionDescription("Retrieve all IEEE 802.1x configurations with optional pagination"),
		fuego.OptionQueryInt("$top", "Number of records to return"),
		fuego.OptionQueryInt("$skip", "Number of records to skip"),
		fuego.OptionQueryBool("$count", "Include total count"),
	)

	fuego.Post(f.server, "/api/v1/admin/ieee8021xconfigs", f.createIEEE8021xConfig,
		fuego.OptionTags("IEEE 802.1x"),
		fuego.OptionSummary("Create IEEE 802.1x Configuration"),
		fuego.OptionDescription("Create a new IEEE 802.1x configuration"),
	)

	fuego.Get(f.server, "/api/v1/admin/ieee8021xconfigs/{name}", f.getIEEE8021xConfigByName,
		fuego.OptionTags("IEEE 802.1x"),
		fuego.OptionSummary("Get IEEE 802.1x Configuration by Name"),
		fuego.OptionDescription("Retrieve a specific IEEE 802.1x configuration by name"),
		fuego.OptionPath("name", "Configuration name"),
	)

	fuego.Patch(f.server, "/api/v1/admin/ieee8021xconfigs", f.updateIEEE8021xConfig,
		fuego.OptionTags("IEEE 802.1x"),
		fuego.OptionSummary("Update IEEE 802.1x Configuration"),
		fuego.OptionDescription("Update an existing IEEE 802.1x configuration"),
	)

	fuego.Delete(f.server, "/api/v1/admin/ieee8021xconfigs/{name}", f.deleteIEEE8021xConfig,
		fuego.OptionTags("IEEE 802.1x"),
		fuego.OptionSummary("Delete IEEE 802.1x Configuration"),
		fuego.OptionDescription("Delete an IEEE 802.1x configuration by name"),
		fuego.OptionPath("name", "Configuration name"),
	)
}

func (f *FuegoAdapter) getIEEE8021xConfigs(_ fuego.ContextNoBody) (dto.IEEE8021xConfigCountResponse, error) {
	timeout := 60
	configs := []dto.IEEE8021xConfig{
		{
			ProfileName:            "example-8021x",
			AuthenticationProtocol: 2,
			PXETimeout:             &timeout,
			WiredInterface:         true,
			TenantID:               "default",
			Version:                "1.0.0",
		},
	}

	return dto.IEEE8021xConfigCountResponse{
		Count: len(configs),
		Data:  configs,
	}, nil
}

func (f *FuegoAdapter) getIEEE8021xConfigByName(_ fuego.ContextNoBody) (dto.IEEE8021xConfig, error) {
	timeout := 60

	return dto.IEEE8021xConfig{
		ProfileName:            "example-8021x",
		AuthenticationProtocol: 2,
		PXETimeout:             &timeout,
		WiredInterface:         true,
		TenantID:               "default",
		Version:                "1.0.0",
	}, nil
}

func (f *FuegoAdapter) createIEEE8021xConfig(c fuego.ContextWithBody[dto.IEEE8021xConfig]) (dto.IEEE8021xConfig, error) {
	body, err := c.Body()
	if err != nil {
		return dto.IEEE8021xConfig{}, err
	}

	return body, nil
}

func (f *FuegoAdapter) updateIEEE8021xConfig(c fuego.ContextWithBody[dto.IEEE8021xConfig]) (dto.IEEE8021xConfig, error) {
	body, err := c.Body()
	if err != nil {
		return dto.IEEE8021xConfig{}, err
	}

	return body, nil
}

func (f *FuegoAdapter) deleteIEEE8021xConfig(_ fuego.ContextNoBody) (any, error) {
	return nil, nil
}
