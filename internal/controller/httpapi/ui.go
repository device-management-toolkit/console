//go:build !noui

package httpapi

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/pkg/logger"
)

//go:embed all:ui
var content embed.FS

const (
	protocolHTTP  = "http://"
	protocolHTTPS = "https://"
)

// setupUIRoutes sets up all UI-related routes and static file serving.
func setupUIRoutes(handler *gin.Engine, l logger.Interface, cfg *config.Config) {
	// Static files
	// Serve static assets (js, css, images, etc.)
	// Create subdirectory view of the embedded file system
	staticFiles, err := fs.Sub(content, "ui")
	if err != nil {
		l.Fatal(err)
	}

	// Serve index.html at root
	handler.StaticFileFS("/", "./", http.FS(staticFiles))

	// main.js needs config injection, so it's handled specially
	modifiedMainJS := injectConfigToMainJS(l, cfg)
	handler.StaticFile("/main.js", modifiedMainJS)

	// Serve all other static files dynamically via NoRoute handler
	// This handles chunk files, assets, and any other embedded files
	handler.NoRoute(func(c *gin.Context) {
		path := strings.TrimPrefix(c.Request.URL.Path, "/")
		if path == "" {
			path = "."
		}

		// Try to serve the actual file if it exists
		if file, err := staticFiles.Open(path); err == nil {
			file.Close()
			c.FileFromFS(path, http.FS(staticFiles))

			return
		}

		// Fallback to index.html for SPA routing
		c.FileFromFS("./", http.FS(staticFiles))
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
	if cfg.Auth.UI.RequireHTTPS {
		requireHTTPSReplacement = ",requireHttps:!0"
		protocol = protocolHTTPS
	}

	if cfg.TLS.Enabled {
		protocol = protocolHTTPS
	}

	// if there is a clientID, we assume oauth will be configured, so inject UI config values from YAML
	if cfg.ClientID != "" {
		strictDiscoveryReplacement := ",strictDiscoveryDocumentValidation:!1"
		if cfg.Auth.UI.StrictDiscoveryDocumentValidation {
			strictDiscoveryReplacement = ",strictDiscoveryDocumentValidation:!0"
		}

		data = injectPlaceholders(data, map[string]string{
			",useOAuth:!1,":                         ",useOAuth:!0,",
			",requireHttps:!0":                      requireHTTPSReplacement,
			",strictDiscoveryDocumentValidation:!0": strictDiscoveryReplacement,
			"##CLIENTID##":                          cfg.Auth.UI.ClientID,
			"##ISSUER##":                            cfg.Auth.UI.Issuer,
			"##SCOPE##":                             cfg.Auth.UI.Scope,
			"##REDIRECTURI##":                       cfg.Auth.UI.RedirectURI,
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
