// Package redfish implements the Redfish component for DMT.
package redfish

import (
	_ "embed"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	dmtconfig "github.com/device-management-toolkit/console/config"
	dmtusecase "github.com/device-management-toolkit/console/internal/usecase"
	"github.com/device-management-toolkit/console/internal/usecase/devices"
	"github.com/device-management-toolkit/console/pkg/db"
	"github.com/device-management-toolkit/console/pkg/logger"
	redfishgenerated "github.com/device-management-toolkit/console/redfish/internal/controller/http/v1/generated"
	v1 "github.com/device-management-toolkit/console/redfish/internal/controller/http/v1/handler"
	sessioninfra "github.com/device-management-toolkit/console/redfish/internal/infrastructure/sessions"
	"github.com/device-management-toolkit/console/redfish/internal/mocks"
	redfishusecase "github.com/device-management-toolkit/console/redfish/internal/usecase"
	"github.com/device-management-toolkit/console/redfish/internal/usecase/sessions"
)

// Embed the OpenAPI specification at build time
//
//go:embed openapi/redfish-openapi.yaml
var embeddedOpenAPISpec []byte

// ErrDevicesCastFailed is returned when the devices use case cannot be cast to the expected type.
var ErrDevicesCastFailed = errors.New("failed to cast devices use case")

// ComponentConfig holds component-specific configuration.
type ComponentConfig struct {
	Enabled      bool   `yaml:"enabled" env:"REDFISH_ENABLED"`
	AuthRequired bool   `yaml:"auth_required" env:"REDFISH_AUTH_REQUIRED"`
	BaseURL      string `yaml:"base_url" env:"REDFISH_BASE_URL"`
}

const (
	// HTTP status codes.
	statusBadRequest       = 400
	statusUnauthorized     = 401
	statusForbidden        = 403
	statusMethodNotAllowed = 405
)

var (
	server          *v1.RedfishServer
	componentConfig *ComponentConfig
)

// Initialize initializes the Redfish component with DMT infrastructure.
func Initialize(_ *gin.Engine, log logger.Interface, _ *db.SQL, usecases *dmtusecase.Usecases, config *dmtconfig.Config) error {
	// Initialize configuration with defaults
	auth := config.Auth
	componentConfig = &ComponentConfig{
		Enabled:      true,
		AuthRequired: !auth.Disabled,
		BaseURL:      "/redfish/v1",
	}

	// Check if we should use mock repository (for testing)
	useMock := os.Getenv("REDFISH_USE_MOCK") == "true"

	var repo redfishusecase.ComputerSystemRepository

	if useMock {
		log.Info("Using mock WSMAN repository for Redfish API")

		repo = mocks.NewMockComputerSystemRepo()
	} else {
		// Create Redfish-specific repository and use case using DMT's device management
		devicesUC, ok := usecases.Devices.(*devices.UseCase)
		if !ok {
			log.Error("Failed to cast Devices usecase to *devices.UseCase")

			return nil // Return nil to not block other components
		}

		repo = redfishusecase.NewWsmanComputerSystemRepo(devicesUC, log)
	}

	computerSystemUC := &redfishusecase.ComputerSystemUseCase{Repo: repo}

	// Create session repository and use case
	const sessionCleanupInterval = 5 * time.Minute

	sessionRepo := sessioninfra.NewInMemoryRepository(sessionCleanupInterval)
	sessionUseCase := sessions.NewUseCase(sessionRepo, config)

	// Initialize the Redfish server with configuration
	server = &v1.RedfishServer{
		ComputerSystemUC: computerSystemUC,
		SessionUC:        sessionUseCase,
		Config:           config,
		Logger:           log,
	}

	// Load OData services from embedded OpenAPI spec
	services, err := v1.ExtractServicesFromOpenAPIData(embeddedOpenAPISpec)
	if err != nil {
		log.Warn("Failed to load services from embedded OpenAPI spec: %v, using defaults", err)

		services = v1.GetDefaultServices()
	}

	server.Services = services

	log.Info("Redfish component initialized successfully with %d OData services", len(server.Services))

	return nil
}

// isPublicEndpoint checks if the request path is a public endpoint.
func isPublicEndpoint(path, method string) bool {
	// Public endpoints as defined in OpenAPI spec (security: [{}])
	// - ServiceRoot, Metadata, OData (read-only discovery endpoints)
	// - SessionService Sessions POST (login endpoint - must be unauthenticated)
	if path == "/redfish/v1/" || path == "/redfish/v1/$metadata" || path == "/redfish/v1/odata" {
		return true
	}

	return path == "/redfish/v1/SessionService/Sessions" && method == "POST"
}

// createAuthMiddleware creates the authentication middleware for protected endpoints.
// Supports both X-Auth-Token (Redfish session) and Basic Auth.
func createAuthMiddleware() redfishgenerated.MiddlewareFunc {
	auth := server.Config.Auth
	basicAuthMiddleware := v1.BasicAuthValidator(auth.AdminUsername, auth.AdminPassword)
	sessionAuthMiddleware := v1.SessionAuthMiddleware(server.SessionUC)

	return func(c *gin.Context) {
		path := c.Request.URL.Path

		if isPublicEndpoint(path, c.Request.Method) {
			c.Next()

			return
		}

		// Protected endpoints: support both X-Auth-Token and Basic Auth
		// Per Redfish spec, X-Auth-Token takes precedence
		if strings.HasPrefix(path, "/redfish/v1/") {
			// Check for X-Auth-Token first (Redfish session authentication)
			if token := c.GetHeader("X-Auth-Token"); token != "" {
				sessionAuthMiddleware(c)

				return
			}

			// Fallback to Basic Auth
			basicAuthMiddleware(c)

			return
		}

		c.Next()
	}
}

// RegisterRoutes registers Redfish API routes.
func RegisterRoutes(router *gin.Engine, _ logger.Interface) error {
	if !componentConfig.Enabled {
		server.Logger.Info("Redfish component is disabled, skipping route registration")

		return nil
	}

	// Build middleware chain with OData header
	middlewares := []redfishgenerated.MiddlewareFunc{
		func(c *gin.Context) {
			c.Header("OData-Version", "4.0")
			c.Next()
		},
	}

	if componentConfig.AuthRequired {
		middlewares = append(middlewares, createAuthMiddleware())
	}

	// Register handlers with OpenAPI-spec-compliant middleware
	redfishgenerated.RegisterHandlersWithOptions(router, server, redfishgenerated.GinServerOptions{
		BaseURL:      "",
		ErrorHandler: createErrorHandler(),
		Middlewares:  middlewares,
	})

	if componentConfig.AuthRequired {
		server.Logger.Info("Redfish API routes registered with authentication")
	} else {
		server.Logger.Info("Redfish API routes registered without authentication")
	}

	// Enable HandleMethodNotAllowed to return 405 for wrong HTTP methods
	router.HandleMethodNotAllowed = true

	// Add NoMethod handler for Redfish routes to return 405 with proper error
	router.NoMethod(func(c *gin.Context) {
		// Only handle Redfish routes
		if len(c.Request.URL.Path) >= 10 && c.Request.URL.Path[:10] == "/redfish/v" {
			v1.MethodNotAllowedError(c)
		}
	})

	server.Logger.Info("Redfish API routes registered successfully")

	return nil
}

// createErrorHandler creates an error handler for OpenAPI-generated routes.
func createErrorHandler() func(*gin.Context, error, int) {
	return func(c *gin.Context, err error, statusCode int) {
		switch statusCode {
		case statusUnauthorized:
			v1.UnauthorizedError(c)
		case statusForbidden:
			v1.ForbiddenError(c)
		case statusMethodNotAllowed:
			v1.MethodNotAllowedError(c)
		case statusBadRequest:
			v1.BadRequestError(c, err.Error())
		default:
			v1.InternalServerError(c, err)
		}
	}
}
