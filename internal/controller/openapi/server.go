package openapi

import (
	"github.com/go-fuego/fuego"

	dto "github.com/device-management-toolkit/console/internal/entity/dto/v1"
)

func (f *FuegoAdapter) RegisterServerRoutes() {
	fuego.Get(f.server, "/api/v1/server/features", f.getServerFeatures,
		fuego.OptionTags("Server"),
		fuego.OptionSummary("Get Server Features"),
		fuego.OptionDescription("Retrieve server-level capability flags so clients can adapt their UI (e.g. show or hide the CIRA tab)"),
		protectedRouteOptions(),
	)
}

func (f *FuegoAdapter) getServerFeatures(_ fuego.ContextNoBody) (dto.ServerFeatures, error) {
	return dto.ServerFeatures{}, nil
}
