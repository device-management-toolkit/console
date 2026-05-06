package openapi

import (
	"net/http"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-fuego/fuego"
)

func apiRouteOptions() fuego.RouteOption {
	return routeOptionGroup(
		errorResponseOption(http.StatusUnauthorized, "Unauthorized _(authentication error)_"),
	)
}

func protectedRouteOptions() fuego.RouteOption {
	return routeOptionGroup(
		apiRouteOptions(),
		fuego.OptionSecurity(openapi3.SecurityRequirement{"bearerAuth": []string{}}),
		errorResponseOption(http.StatusNotFound, "Not Found"),
		errorResponseOption(http.StatusRequestTimeout, "Request Timeout"),
		errorResponseOption(http.StatusConflict, "Conflict"),
		errorResponseOption(http.StatusNotImplemented, "Not Implemented"),
		errorResponseOption(http.StatusGatewayTimeout, "Gateway Timeout"),
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
