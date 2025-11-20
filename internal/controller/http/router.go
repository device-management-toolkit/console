// Package v1 implements routing paths. Each services in own file.
package http

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/device-management-toolkit/console/config"
	v1 "github.com/device-management-toolkit/console/internal/controller/http/v1"
	v2 "github.com/device-management-toolkit/console/internal/controller/http/v2"
	openapi "github.com/device-management-toolkit/console/internal/controller/openapi"
	"github.com/device-management-toolkit/console/internal/usecase"
	"github.com/device-management-toolkit/console/pkg/db"
	"github.com/device-management-toolkit/console/pkg/logger"
	redfish "github.com/device-management-toolkit/console/redfish"
)

//go:embed all:ui
var content embed.FS

const (
	protocolHTTP  = "http://"
	protocolHTTPS = "https://"
)

// NewRouter sets up the HTTP router with redfish support.
func NewRouter(handler *gin.Engine, l logger.Interface, t usecase.Usecases, cfg *config.Config, database *db.SQL) { //nolint:funlen // This function is responsible for setting up the router, so it's expected to be long
	// Options
	handler.Use(gin.Logger())
	handler.Use(gin.Recovery())

	// Initialize redfish directly
	if err := redfish.Initialize(handler, l, database, &t, cfg); err != nil {
		l.Fatal("Failed to initialize redfish: " + err.Error())
	}

	// Register redfish routes FIRST (before static files)
	if err := redfish.RegisterRoutes(handler, l); err != nil {
		l.Fatal("Failed to register redfish routes: " + err.Error())
	}

	// Initialize Fuego adapter
	fuegoAdapter := openapi.NewFuegoAdapter(t, l)
	fuegoAdapter.RegisterRoutes()
	fuegoAdapter.AddToGinRouter(handler)

	// Public routes
	login := v1.NewLoginRoute(cfg)
	handler.POST("/api/v1/authorize", login.Login)
	// Static files
	// Serve static assets (js, css, images, etc.)
	// Create subdirectory view of the embedded file system
	staticFiles, err := fs.Sub(content, "ui")
	if err != nil {
		l.Fatal(err)
	}

	modifiedMainJS := injectConfigToMainJS(l, cfg)
	handler.StaticFile("/main.js", modifiedMainJS)

	handler.StaticFileFS("/polyfills.js", "./polyfills.js", http.FS(staticFiles))
	handler.StaticFileFS("/media/kJEhBvYX7BgnkSrUwT8OhrdQw4oELdPIeeII9v6oFsI.woff2", "./media/kJEhBvYX7BgnkSrUwT8OhrdQw4oELdPIeeII9v6oFsI.woff2", http.FS(staticFiles))
	handler.StaticFileFS("/runtime.js", "./runtime.js", http.FS(staticFiles))
	handler.StaticFileFS("/styles.css", "./styles.css", http.FS(staticFiles))
	handler.StaticFileFS("/vendor.js", "./vendor.js", http.FS(staticFiles))
	handler.StaticFileFS("/favicon.ico", "./favicon.ico", http.FS(staticFiles))
	handler.StaticFileFS("/assets/logo.png", "./assets/logo.png", http.FS(staticFiles))
	handler.StaticFileFS("/assets/monaco/min/vs/loader.js", "./assets/monaco/min/vs/loader.js", http.FS(staticFiles))
	handler.StaticFileFS("/assets/monaco/min/vs/editor/editor.main.js", "./assets/monaco/min/vs/editor/editor.main.js", http.FS(staticFiles))
	handler.StaticFileFS("/assets/monaco/min/vs/editor/editor.main.css", "./assets/monaco/min/vs/editor/editor.main.css", http.FS(staticFiles))
	handler.StaticFileFS("/assets/monaco/min/vs/editor/editor.main.nls.js", "./assets/monaco/min/vs/editor/editor.main.nls.js", http.FS(staticFiles))
	handler.StaticFileFS("/assets/monaco/min/vs/base/worker/workerMain.js", "./assets/monaco/min/vs/base/worker/workerMain.js", http.FS(staticFiles))
	handler.StaticFileFS("/assets/monaco/min/vs/base/common/worker/simpleWorker.nls.js", "./assets/monaco/min/vs/base/common/worker/simpleWorker.nls.js", http.FS(staticFiles))
	handler.StaticFileFS("/assets/monaco/min/vs/base/browser/ui/codicons/codicon/codicon.ttf", "./assets/monaco/min/vs/base/browser/ui/codicons/codicon/codicon.ttf", http.FS(staticFiles))
	handler.StaticFileFS("/assets/monaco/min/vs/basic-languages/xml/xml.js", "./assets/monaco/min/vs/basic-languages/xml/xml.js", http.FS(staticFiles))

	langs := []string{"en", "fr", "de", "ar", "es", "fi", "he", "it", "ja", "nl", "ru", "sv"}
	for _, lang := range langs {
		relativePath := "/assets/i18n/" + lang + ".json"
		filePath := "." + relativePath
		handler.StaticFileFS(relativePath, filePath, http.FS(staticFiles))
	}

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

	// Serve SPA only for root path
	handler.GET("/", func(c *gin.Context) {
		c.FileFromFS("./index.html", http.FS(staticFiles))
	})

	handler.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path

		// Handle Redfish API routes with Redfish-compliant 404 errors
		if len(path) >= 10 && path[:10] == "/redfish/v" {
			c.Header("Content-Type", "application/json; charset=utf-8")
			c.Header("OData-Version", "4.0")

			errorMessage := "Resource not found"
			c.JSON(http.StatusNotFound, gin.H{
				"error": gin.H{
					"code":    "Base.1.22.0.ResourceMissingAtURI",
					"message": errorMessage,
					"@Message.ExtendedInfo": []gin.H{
						{
							"@odata.type": "#Message.v1_0_0.Message",
							"MessageId":   "Base.1.22.0.ResourceMissingAtURI",
							"Message":     errorMessage,
							"MessageArgs": []string{path},
							"Severity":    "Warning",
							"Resolution":  "Remove the URI from the request and resubmit the request.",
						},
					},
				},
			})

			return
		}

		// Handle API routes with regular JSON errors
		if len(path) >= 4 && path[:4] == "/api" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})

			return
		}

		// Serve SPA for all other routes (client-side routing)
		c.FileFromFS("./index.html", http.FS(staticFiles))
	})
}

func injectConfigToMainJS(l logger.Interface, cfg *config.Config) string {
	data, err := fs.ReadFile(content, "ui/main.js")
	if err != nil {
		l.Warn("Could not read embedded main.js: %v", err)

		return ""
	}

	protocol := protocolHTTP

	requireHTTPSReplacement := ",requireHttps:!1"
	if cfg.UI.RequireHTTPS {
		requireHTTPSReplacement = ",requireHttps:!0"
		protocol = protocolHTTPS
	}

	if cfg.TLS.Enabled {
		protocol = protocolHTTPS
	}

	// if there is a clientID, we assume oauth will be configured, so inject UI config values from YAML
	if cfg.ClientID != "" {
		strictDiscoveryReplacement := ",strictDiscoveryDocumentValidation:!1"
		if cfg.UI.StrictDiscoveryDocumentValidation {
			strictDiscoveryReplacement = ",strictDiscoveryDocumentValidation:!0"
		}

		data = injectPlaceholders(data, map[string]string{
			",useOAuth:!1,":                         ",useOAuth:!0,",
			",requireHttps:!0":                      requireHTTPSReplacement,
			",strictDiscoveryDocumentValidation:!0": strictDiscoveryReplacement,
			"##CLIENTID##":                          cfg.UI.ClientID,
			"##ISSUER##":                            cfg.UI.Issuer,
			"##SCOPE##":                             cfg.UI.Scope,
			"##REDIRECTURI##":                       cfg.UI.RedirectURI,
		})
	}

	data = injectPlaceholders(data, map[string]string{
		"##CONSOLE_SERVER_API##": protocol + cfg.Host + ":" + cfg.Port,
	})

	// Write to /tmp
	permissions := 0o600

	tempFile := filepath.Join(os.TempDir(), "main.js")

	if err := os.WriteFile(tempFile, data, os.FileMode(permissions)); err != nil {
		log.Fatalf("Could not write modified main.js: %v", err)
	}

	return tempFile
}

func injectPlaceholders(content []byte, replacements map[string]string) []byte {
	result := string(content)
	for placeholder, value := range replacements {
		result = strings.ReplaceAll(result, placeholder, value)
	}

	return []byte(result)
}
