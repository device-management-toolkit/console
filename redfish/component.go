// Package redfish implements the Redfish component for DMT.
package redfish

import (
	"errors"
	"os"
	"strings"

	"github.com/gin-gonic/gin"

	dmtconfig "github.com/device-management-toolkit/console/config"
	dmtusecase "github.com/device-management-toolkit/console/internal/usecase"
	"github.com/device-management-toolkit/console/internal/usecase/devices"
	"github.com/device-management-toolkit/console/pkg/db"
	"github.com/device-management-toolkit/console/pkg/logger"
	redfishgenerated "github.com/device-management-toolkit/console/redfish/internal/controller/http/v1/generated"
	v1 "github.com/device-management-toolkit/console/redfish/internal/controller/http/v1/handler"
	redfishusecase "github.com/device-management-toolkit/console/redfish/internal/usecase"
)

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

		repo = redfishusecase.NewMockComputerSystemRepo()
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

	// Initialize the Redfish server with shared infrastructure
	server = &v1.RedfishServer{
		ComputerSystemUC: computerSystemUC,
		Config:           config,
		Logger:           log,
	}

	log.Info("Redfish component initialized successfully")

	return nil
}

// RegisterRoutes registers Redfish API routes.
func RegisterRoutes(router *gin.Engine, _ logger.Interface) error {
	if !componentConfig.Enabled {
		server.Logger.Info("Redfish component is disabled, skipping route registration")

		return nil
	}

	if componentConfig.AuthRequired {
		// Apply Basic Auth middleware to OpenAPI-defined protected endpoints
		// Use actual admin credentials from the DMT configuration
		auth := server.Config.Auth
		adminUsername := auth.AdminUsername
		adminPassword := auth.AdminPassword
		basicAuthMiddleware := v1.BasicAuthValidator(adminUsername, adminPassword)

		// Register handlers with OpenAPI-spec-compliant middleware
		redfishgenerated.RegisterHandlersWithOptions(router, server, redfishgenerated.GinServerOptions{
			BaseURL:      "",
			ErrorHandler: createErrorHandler(),
			Middlewares: []redfishgenerated.MiddlewareFunc{
				// OpenAPI-spec-driven selective authentication
				func(c *gin.Context) {
					path := c.Request.URL.Path

					// Public endpoints as defined in OpenAPI spec (security: [{}])
					if path == "/redfish/v1/" || path == "/redfish/v1/$metadata" {
						c.Next()

						return
					}

					// Protected endpoints as defined in OpenAPI spec (security: [{"BasicAuth": []}])
					if strings.HasPrefix(path, "/redfish/v1/") {
						basicAuthMiddleware(c)

						return
					}

					// Default: no authentication
					c.Next()
				},
			},
		})

		server.Logger.Info("Redfish API routes registered with OpenAPI-spec-driven Basic Auth")
	} else {
		// Register without authentication (all endpoints public)
		redfishgenerated.RegisterHandlersWithOptions(router, server, redfishgenerated.GinServerOptions{
			BaseURL:      "",
			ErrorHandler: createErrorHandler(),
		})

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
