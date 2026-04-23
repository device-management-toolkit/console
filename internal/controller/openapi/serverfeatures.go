package openapi

import (
	"github.com/go-fuego/fuego"

	"github.com/device-management-toolkit/console/config"
	dto "github.com/device-management-toolkit/console/internal/entity/dto/v1"
)

func (f *FuegoAdapter) RegisterServerFeaturesRoutes() {
	fuego.Get(f.server, "/api/v1/features", f.getServerFeatures,
		fuego.OptionTags("Server"),
		fuego.OptionSummary("Get Server Feature Flags"),
		fuego.OptionDescription("Returns which optional server capabilities are enabled. The UI reads this at bootstrap to decide which tabs/routes to render."),
	)
}

func (f *FuegoAdapter) getServerFeatures(_ fuego.ContextNoBody) (dto.ServerFeaturesResponse, error) {
	disabled := false
	if config.ConsoleConfig != nil {
		disabled = config.ConsoleConfig.DisableCIRA
	}

	return dto.ServerFeaturesResponse{
		CIRA: !disabled,
	}, nil
}
