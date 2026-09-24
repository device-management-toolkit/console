package openapi

import (
	"net/http"

	"github.com/go-fuego/fuego"

	dto "github.com/device-management-toolkit/console/internal/entity/dto/v1"
)

// RegisterPackageRoutes declares the Download RPC endpoints. They sit under
// /api/package rather than a versioned group, matching the Gin routes.
func (f *FuegoAdapter) RegisterPackageRoutes() {
	fuego.Get(f.server, "/api/package/rpc-versions", f.listRPCVersions,
		fuego.OptionTags("Package"),
		fuego.OptionSummary("List RPC Versions"),
		fuego.OptionDescription("List the most recent rpc-go releases (v3 and above) available for packaging, with the OS/arch builds each one publishes"),
		protectedRouteOptions(),
	)

	fuego.Post(f.server, "/api/package", f.buildPackage,
		fuego.OptionTags("Package"),
		fuego.OptionSummary("Build RPC Package"),
		fuego.OptionDescription("Build a zip containing the requested rpc-go binary and a generated config.yaml pointing at this Console. The response is the zip itself, not JSON."),
		fuego.OptionAddResponse(http.StatusOK, "OK", fuego.Response{Type: "", ContentTypes: []string{"application/zip"}}),
		protectedRouteOptions(),
	)
}

func (f *FuegoAdapter) listRPCVersions(_ fuego.ContextNoBody) ([]dto.RPCRelease, error) {
	return nil, nil
}

func (f *FuegoAdapter) buildPackage(_ fuego.ContextWithBody[dto.PackageRequest]) (string, error) {
	return "", nil
}
