package openapi

import "github.com/go-fuego/fuego"

func (f *FuegoAdapter) RegisterCIRACertRoutes() {
	fuego.Get(f.server, "/api/v1/ciracert", f.getCIRACert,
		fuego.OptionTags("CIRA"),
		fuego.OptionSummary("Get CIRA Root Certificate"),
		fuego.OptionDescription("Retrieve the root CIRA certificate as plain text"),
	)
}

func (f *FuegoAdapter) getCIRACert(_ fuego.ContextNoBody) (string, error) {
	return "", nil
}
