package openapi

import (
	"net/http"

	"github.com/go-fuego/fuego"
)

func (f *FuegoAdapter) RegisterCIRACertRoutes() {
	fuego.Get(f.server, "/api/v1/ciracert", f.getCIRACert,
		fuego.OptionTags("CIRA"),
		fuego.OptionSummary("Get CIRA Root Certificate"),
		fuego.OptionDescription("Retrieve the root CIRA certificate as plain text"),
		fuego.OptionAddResponse(http.StatusOK, "OK", fuego.Response{Type: "", ContentTypes: []string{"text/plain"}}),
		protectedRouteOptions(),
	)
}

func (f *FuegoAdapter) getCIRACert(_ fuego.ContextNoBody) (string, error) {
	return "", nil
}
