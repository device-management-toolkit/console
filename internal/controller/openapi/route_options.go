package openapi

import (
	"net/http"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-fuego/fuego"
	"github.com/go-fuego/fuego/param"

	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/internal/controller/httpapi/middleware"
	"github.com/device-management-toolkit/console/internal/tenant"
)

// specCookieAuthEnabled mirrors the auth middleware. cmd/openapi-gen runs
// without config, so nil documents the shipped defaults.
func specCookieAuthEnabled() bool {
	if config.ConsoleConfig == nil {
		return true
	}

	return config.ConsoleConfig.CookieAuthEnabled()
}

func apiRouteOptions() fuego.RouteOption {
	return routeOptionGroup(
		errorResponseOption(http.StatusUnauthorized, "Unauthorized _(authentication error)_"),
	)
}

func protectedRouteOptions() fuego.RouteOption {
	// Alternatives, not both: a bearer header or the session cookie.
	security := []openapi3.SecurityRequirement{
		{bearerAuthScheme: []string{}},
	}

	if specCookieAuthEnabled() {
		security = append(security, openapi3.SecurityRequirement{cookieAuthScheme: []string{}})
	}

	return routeOptionGroup(
		apiRouteOptions(),
		fuego.OptionSecurity(security...),
		tenantHeaderOption(),
		errorResponseOption(http.StatusBadRequest, "Bad Request"),
		errorResponseOption(http.StatusNotFound, "Not Found"),
		errorResponseOption(http.StatusRequestTimeout, "Request Timeout"),
		errorResponseOption(http.StatusConflict, "Conflict"),
		errorResponseOption(http.StatusNotImplemented, "Not Implemented"),
		errorResponseOption(http.StatusGatewayTimeout, "Gateway Timeout"),
	)
}

// tenantHeaderOption documents the optional tenant scope. Omitting the header
// selects the default tenant, which is where single-tenant data lives.
func tenantHeaderOption() fuego.RouteOption {
	return fuego.OptionHeader(
		middleware.TenantHeaderName,
		"Scopes the request to a tenant. "+tenant.Hint+". Omit for the default tenant.",
		param.Nullable(),
		param.Example("tenant", "acme-corp"),
	)
}

func errorResponseOption(statusCode int, description string) fuego.RouteOption {
	return fuego.OptionAddResponse(statusCode, description, fuego.Response{Type: fuego.HTTPError{}})
}

func routeOptionGroup(options ...fuego.RouteOption) fuego.RouteOption {
	return func(route *fuego.BaseRoute) {
		for _, option := range options {
			option(route)
		}
	}
}
