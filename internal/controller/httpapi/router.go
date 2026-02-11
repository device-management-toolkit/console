// Package httpapi implements routing paths. Each services in own file.
package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/device-management-toolkit/console/config"
	v1 "github.com/device-management-toolkit/console/internal/controller/httpapi/v1"
	v2 "github.com/device-management-toolkit/console/internal/controller/httpapi/v2"
	openapi "github.com/device-management-toolkit/console/internal/controller/openapi"
	"github.com/device-management-toolkit/console/internal/usecase"
	"github.com/device-management-toolkit/console/pkg/db"
	"github.com/device-management-toolkit/console/pkg/logger"
	redfish "github.com/device-management-toolkit/console/redfish"
)

// NewRouter sets up the HTTP router with redfish support.
func NewRouter(handler *gin.Engine, l logger.Interface, t usecase.Usecases, cfg *config.Config, database *db.SQL) {
	// Options
	handler.Use(gin.Logger())
	handler.Use(gin.Recovery())

	// Initialize redfish directly
	if err := redfish.Initialize(handler, l, database, &t, cfg); err != nil {
		l.Fatal("Failed to initialize redfish: " + err.Error())
	}

	// Initialize Fuego adapter
	fuegoAdapter := openapi.NewFuegoAdapter(t, l)
	fuegoAdapter.RegisterRoutes()
	fuegoAdapter.AddToGinRouter(handler)

	// Public routes
	login := v1.NewLoginRoute(cfg)
	handler.POST("/api/v1/authorize", login.Login)

	// Setup UI routes (no-op in noui builds)
	setupUIRoutes(handler, l, cfg)

	// K8s probe
	handler.GET("/healthz", func(c *gin.Context) { c.Status(http.StatusOK) })

	// Prometheus metrics
	handler.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// version info
	vr := v1.NewVersionRoute(cfg)
	handler.GET("/version", vr.LatestReleaseHandler)

	// Protected routes using JWT middleware
	var protected *gin.RouterGroup
	if cfg.Disabled {
		protected = handler.Group("/api")
	} else {
		protected = handler.Group("/api", login.JWTAuthMiddleware())
	}

	// Routers
	h2 := protected.Group("/v1")
	{
		v1.NewDeviceRoutes(h2, t.Devices, l)
		v1.NewAmtRoutes(h2, t.Devices, t.AMTExplorer, t.Exporter, l)
		v1.NewCIRACertRoutes(h2, l)
	}

	h := protected.Group("/v1/admin")
	{
		v1.NewDomainRoutes(h, t.Domains, l)
		v1.NewCIRAConfigRoutes(h, t.CIRAConfigs, l)
		v1.NewProfileRoutes(h, t.Profiles, l)
		v1.NewWirelessConfigRoutes(h, t.WirelessProfiles, l)
		v1.NewIEEE8021xConfigRoutes(h, t.IEEE8021xProfiles, l)
	}

	h3 := protected.Group("/v2")
	{
		v2.NewAmtRoutes(h3, t.Devices, l)
	}

	// Register redfish routes directly
	if err := redfish.RegisterRoutes(handler, l); err != nil {
		l.Fatal("Failed to register redfish routes: " + err.Error())
	}
}
