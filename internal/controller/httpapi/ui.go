//go:build !noui

package httpapi

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net"
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

func HasUI() bool {
	_, err := fs.Stat(content, "ui/index.html")

	return err == nil
}

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

	handler.StaticFileFS("/", "./", http.FS(staticFiles)) // Serve static files from "/" route

	modifiedMainJS := injectConfigToMainJS(l, cfg)
	handler.StaticFile("/main.js", modifiedMainJS)

	handler.StaticFileFS("/polyfills.js", "./polyfills.js", http.FS(staticFiles))
	handler.StaticFileFS("/runtime.js", "./runtime.js", http.FS(staticFiles))
	handler.StaticFileFS("/styles.css", "./styles.css", http.FS(staticFiles))
	handler.StaticFileFS("/vendor.js", "./vendor.js", http.FS(staticFiles))
	handler.StaticFileFS("/favicon.ico", "./favicon.ico", http.FS(staticFiles))
	handler.StaticFileFS("/assets/logo.png", "./assets/logo.png", http.FS(staticFiles))

	// Serve the whole monaco subtree; its AMD loader requests files dynamically
	// and the exact set changes between monaco versions.
	monacoFS, err := fs.Sub(staticFiles, "assets/monaco")
	if err != nil {
		l.Fatal(err)
	}

	handler.StaticFS("/assets/monaco", http.FS(monacoFS))

	if err := serveMedia(handler, staticFiles); err != nil {
		l.Fatal(err)
	}

	langs := []string{"en", "fr", "de", "ar", "es", "fi", "he", "it", "ja", "nl", "ru", "sv"}
	for _, lang := range langs {
		relativePath := "/assets/i18n/" + lang + ".json"
		filePath := "." + relativePath
		handler.StaticFileFS(relativePath, filePath, http.FS(staticFiles))
	}

	handler.NoRoute(spaFallback(staticFiles))
}

// serveMedia serves the whole media subtree; the UI build emits every file its
// stylesheets reference there.
func serveMedia(handler *gin.Engine, staticFiles fs.FS) error {
	mediaFS, err := fs.Sub(staticFiles, "media")
	if err != nil {
		return fmt.Errorf("httpapi.serveMedia: %w", err)
	}

	handler.StaticFS("/media", http.FS(mediaFS))

	return nil
}

// spaFallback serves the HTML shell for unmatched routes. It skips /assets/ and
// /media/ so a missing static file 404s instead of silently returning the shell.
func spaFallback(staticFiles fs.FS) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/assets/") || strings.HasPrefix(path, "/media/") {
			c.Status(http.StatusNotFound)

			return
		}

		c.FileFromFS("./", http.FS(staticFiles))
	}
}

func injectConfigToMainJS(l logger.Interface, cfg *config.Config) string {
	data, err := fs.ReadFile(content, "ui/main.js")
	if err != nil {
		l.Warn("Could not read embedded main.js: %v", err)

		return ""
	}

	protocol := protocolHTTP

	if cfg.Auth.UI.RequireHTTPS || cfg.TLS.Enabled {
		protocol = protocolHTTPS
	}

	data = injectPlaceholders(data, oauthUIReplacements(cfg))

	data = injectPlaceholders(data, map[string]string{
		"##CONSOLE_SERVER_API##": consoleServerAPIBase(protocol, cfg.Host, cfg.Port),
	})

	// Write to /tmp
	permissions := 0o600

	tempFile := filepath.Join(os.TempDir(), "main.js")

	if err := os.WriteFile(tempFile, data, os.FileMode(permissions)); err != nil {
		log.Fatalf("Could not write modified main.js: %v", err)
	}

	return tempFile
}

// oauthUIReplacements returns the main.js replacements that switch the UI to
// OAuth with the auth.ui values. A clientID means OAuth is configured; without
// one nothing is replaced and the UI keeps basic auth.
func oauthUIReplacements(cfg *config.Config) map[string]string {
	if cfg.ClientID == "" {
		return map[string]string{}
	}

	requireHTTPSReplacement := ",requireHttps:!1"
	if cfg.Auth.UI.RequireHTTPS {
		requireHTTPSReplacement = ",requireHttps:!0"
	}

	strictDiscoveryReplacement := ",strictDiscoveryDocumentValidation:!1"
	if cfg.Auth.UI.StrictDiscoveryDocumentValidation {
		strictDiscoveryReplacement = ",strictDiscoveryDocumentValidation:!0"
	}

	return map[string]string{
		",useOAuth:!1,":                         ",useOAuth:!0,",
		",requireHttps:!0":                      requireHTTPSReplacement,
		",strictDiscoveryDocumentValidation:!0": strictDiscoveryReplacement,
		"##CLIENTID##":                          cfg.Auth.UI.ClientID,
		"##ISSUER##":                            cfg.Auth.UI.Issuer,
		"##SCOPE##":                             cfg.Auth.UI.Scope,
		"##REDIRECTURI##":                       cfg.Auth.UI.RedirectURI,
		"##AUDIENCE##":                          cfg.Auth.UI.Audience,
	}
}

// Returns "" on wildcard hosts so the UI uses same-origin requests matching the user's URL/SNI.
func consoleServerAPIBase(protocol, host, port string) string {
	if isWildcardHost(host) {
		return ""
	}

	return protocol + net.JoinHostPort(unbracket(host), port)
}

// unbracket strips a single pair of surrounding square brackets from a literal
// IPv6 host so net.JoinHostPort doesn't double-wrap (e.g. "[::1]" → "::1").
func unbracket(host string) string {
	if len(host) >= 2 && host[0] == '[' && host[len(host)-1] == ']' {
		return host[1 : len(host)-1]
	}

	return host
}

func isWildcardHost(host string) bool {
	switch host {
	case "", "0.0.0.0", "::", "[::]":
		return true
	}

	return false
}

func injectPlaceholders(content []byte, replacements map[string]string) []byte {
	result := string(content)
	for placeholder, value := range replacements {
		result = strings.ReplaceAll(result, placeholder, value)
	}

	return []byte(result)
}
