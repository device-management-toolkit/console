package openapi

import (
	"net/http"

	"github.com/go-fuego/fuego"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
)

type DomainCountResponse struct {
	Count int          `json:"totalCount"`
	Data  []dto.Domain `json:"data"`
}

func (f *FuegoAdapter) RegisterDomainRoutes() {
	fuego.Get(f.server, "/api/v1/admin/domains", f.getDomains,
		fuego.OptionTags("Domains"),
		fuego.OptionSummary("List Domains"),
		fuego.OptionDescription("Retrieve all domains with optional pagination"),
		fuego.OptionQueryInt("$top", "Number of records to return"),
		fuego.OptionQueryInt("$skip", "Number of records to skip"),
		fuego.OptionQueryBool("$count", "Include total count"),
	)

	fuego.Get(f.server, "/api/v1/admin/domains/{name}", f.getDomainByName,
		fuego.OptionTags("Domains"),
		fuego.OptionSummary("Get Domain by Name"),
		fuego.OptionDescription("Retrieve a specific domain by name"),
		fuego.OptionPath("name", "Domain profile name"),
	)

	fuego.Post(f.server, "/api/v1/admin/domains", f.createDomain,
		fuego.OptionTags("Domains"),
		fuego.OptionSummary("Create Domain"),
		fuego.OptionDescription("Create a new domain"),
		fuego.OptionDefaultStatusCode(http.StatusCreated),
	)

	fuego.Patch(f.server, "/api/v1/admin/domains", f.updateDomain,
		fuego.OptionTags("Domains"),
		fuego.OptionSummary("Update Domain"),
		fuego.OptionDescription("Update an existing domain"),
	)

	fuego.Delete(f.server, "/api/v1/admin/domains/{name}", f.deleteDomain,
		fuego.OptionTags("Domains"),
		fuego.OptionSummary("Delete Domain"),
		fuego.OptionDescription("Delete a domain by name"),
		fuego.OptionPath("name", "Domain profile name"),
		fuego.OptionDefaultStatusCode(http.StatusNoContent),
	)
}

func (f *FuegoAdapter) getDomains(_ fuego.ContextNoBody) (DomainCountResponse, error) {
	return DomainCountResponse{Count: 0, Data: []dto.Domain{}}, nil
}

func (f *FuegoAdapter) getDomainByName(_ fuego.ContextNoBody) (dto.Domain, error) {
	return dto.Domain{}, nil
}

func (f *FuegoAdapter) createDomain(c fuego.ContextWithBody[dto.Domain]) (dto.Domain, error) {
	body, err := c.Body()
	if err != nil {
		return dto.Domain{}, err
	}

	return body, nil
}

func (f *FuegoAdapter) updateDomain(c fuego.ContextWithBody[dto.Domain]) (dto.Domain, error) {
	body, err := c.Body()
	if err != nil {
		return dto.Domain{}, err
	}

	return body, nil
}

func (f *FuegoAdapter) deleteDomain(_ fuego.ContextNoBody) (any, error) {
	return nil, nil
}
