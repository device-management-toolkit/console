package openapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gin-gonic/gin"
	"github.com/go-fuego/fuego"

	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/internal/usecase"
	"github.com/device-management-toolkit/console/pkg/logger"
)

// Shared constants for OpenAPI example responses and defaults.
const (
	defaultVersion    = "1.0.0"
	defaultTenantID   = "default"
	exampleDeviceGUID = "example-guid-1"
	exampleDeviceHost = "device1.example.com"

	bearerAuthScheme = "bearerAuth"
	cookieAuthScheme = "cookieAuth"
)

type FuegoAdapter struct {
	server   *fuego.Server
	usecases usecase.Usecases
	logger   logger.Interface
}

func NewFuegoAdapter(usecases usecase.Usecases, log logger.Interface) *FuegoAdapter {
	server := fuego.NewServer(
		fuego.WithoutStartupMessages(),
		fuego.WithSecurity(securitySchemes()),
	)

	return &FuegoAdapter{
		server:   server,
		usecases: usecases,
		logger:   log,
	}
}

// securitySchemes declares only the mechanisms the server actually accepts, so
// cookieAuth is omitted where the middleware would ignore a cookie.
func securitySchemes() openapi3.SecuritySchemes {
	schemes := openapi3.SecuritySchemes{
		bearerAuthScheme: &openapi3.SecuritySchemeRef{
			Value: openapi3.NewSecurityScheme().
				WithType("http").
				WithScheme("bearer").
				WithBearerFormat("JWT"),
		},
	}

	if !specCookieAuthEnabled() {
		return schemes
	}

	// The name is the default; a build-time spec cannot know a runtime override.
	schemes[cookieAuthScheme] = &openapi3.SecuritySchemeRef{
		Value: openapi3.NewSecurityScheme().
			WithType("apiKey").
			WithIn("cookie").
			WithName(config.DefaultSessionCookieName).
			WithDescription("Session cookie issued by POST /api/v1/authorize. Named `" +
				config.DefaultSessionCookieName + "` by default; a deployment may override it with " +
				"`auth.cookieName`."),
	}

	return schemes
}

// Registers API routes with Fuego for automatic OpenAPI generation.
func (f *FuegoAdapter) RegisterRoutes() {
	// Domains
	f.RegisterDomainRoutes()

	// Profiles
	f.RegisterProfileRoutes()

	// Wireless
	f.RegisterWirelessConfigRoutes()

	// IEEE 802.1X
	f.RegisterIEEE8021xConfigRoutes()

	// CIRA
	f.RegisterCIRAConfigRoutes()

	// Devices
	f.RegisterDeviceRoutes()

	// CIRA certificate
	f.RegisterCIRACertRoutes()

	// Device Management
	f.RegisterDeviceManagementRoutes()

	// Server features
	f.RegisterServerRoutes()
}

// Generates OpenAPI specification as JSON.
func (f *FuegoAdapter) GetOpenAPISpec() ([]byte, error) {
	spec := f.server.OutputOpenAPISpec()

	// Default
	version := defaultVersion
	if config.ConsoleConfig != nil && config.ConsoleConfig.Version != "" {
		version = config.ConsoleConfig.Version
	}

	validSpec := map[string]interface{}{
		"openapi": "3.0.3", // default
		"info": map[string]interface{}{
			"title":       "Console API",
			"version":     version,
			"description": "API for managing console resources",
		},
		"paths": make(map[string]interface{}),
		"components": map[string]interface{}{
			"schemas": make(map[string]interface{}),
		},
	}

	if spec == nil {
		return json.MarshalIndent(validSpec, "", "  ")
	}

	specBytes, err := json.Marshal(spec)
	if err != nil {
		return json.MarshalIndent(validSpec, "", "  ")
	}

	var fuegoSpec map[string]interface{}
	if err := json.Unmarshal(specBytes, &fuegoSpec); err == nil {
		if v, ok := fuegoSpec["openapi"]; ok {
			validSpec["openapi"] = v
		}

		if paths, ok := fuegoSpec["paths"]; ok {
			cleanedPaths := f.cleanDescriptions(paths)
			validSpec["paths"] = cleanedPaths
		}

		if components, ok := fuegoSpec["components"]; ok {
			validSpec["components"] = components
		}
	}

	return json.MarshalIndent(validSpec, "", "  ")
}

func (f *FuegoAdapter) cleanDescriptions(paths interface{}) interface{} {
	pathsMap, ok := paths.(map[string]interface{})
	if !ok {
		return paths
	}

	for _, path := range pathsMap {
		pathObj, ok := path.(map[string]interface{})
		if !ok {
			continue
		}

		for _, method := range pathObj {
			f.cleanMethodDescription(method)
		}
	}

	return paths
}

func (f *FuegoAdapter) cleanMethodDescription(m interface{}) {
	methodObj, ok := m.(map[string]interface{})
	if !ok {
		return
	}

	if desc, exists := methodObj["description"]; exists {
		descStr, ok := desc.(string)
		if !ok {
			return
		}

		parts := strings.Split(descStr, "\n\n---\n\n")
		if len(parts) > 1 {
			methodObj["description"] = parts[len(parts)-1]
		}
	}
}

// Adds Fuego-generated OpenAPI endpoints to existing Gin router.
func (f *FuegoAdapter) AddToGinRouter(router *gin.Engine) {
	router.GET("/api/openapi.json", func(c *gin.Context) {
		spec, err := f.GetOpenAPISpec()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate OpenAPI spec"})

			return
		}

		c.Data(http.StatusOK, "application/json", spec)
	})
}
