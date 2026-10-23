//go:build !noui

package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/device-management-toolkit/console/config"
)

const testUIShell = "<app-root></app-root>"

// newUITestEngine serves a stand-in UI build through the media and SPA fallback routes.
func newUITestEngine(t *testing.T) *gin.Engine {
	t.Helper()

	staticFiles := fstest.MapFS{
		"index.html":              {Data: []byte(testUIShell)},
		"media/dmt-logo_dark.svg": {Data: []byte("<svg/>")},
		"media/font.woff2":        {Data: []byte("font")},
	}

	engine := gin.New()
	require.NoError(t, serveMedia(engine, staticFiles))
	engine.NoRoute(spaFallback(staticFiles))

	return engine
}

func TestUIServesFiles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "media file is served", path: "/media/dmt-logo_dark.svg", want: "<svg/>"},
		{name: "every media file is served, not a fixed list", path: "/media/font.woff2", want: "font"},
		{name: "client-side route gets the HTML shell", path: "/devices", want: testUIShell},
	}

	engine := newUITestEngine(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			w := httptest.NewRecorder()
			engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, tt.path, http.NoBody))

			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, tt.want, w.Body.String())
		})
	}
}

func TestUIMissingStaticFileIsNotFound(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		path string
	}{
		{name: "missing media file", path: "/media/missing.svg"},
		{name: "missing asset", path: "/assets/missing.png"},
	}

	engine := newUITestEngine(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			w := httptest.NewRecorder()
			engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, tt.path, http.NoBody))

			require.Equal(t, http.StatusNotFound, w.Code)
		})
	}
}

func TestConsoleServerAPIBase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		protocol string
		host     string
		port     string
		want     string
	}{
		{
			name:     "wildcard empty host returns relative URL",
			protocol: "https://",
			host:     "",
			port:     "8181",
			want:     "",
		},
		{
			name:     "wildcard 0.0.0.0 returns relative URL",
			protocol: "http://",
			host:     "0.0.0.0",
			port:     "8181",
			want:     "",
		},
		{
			name:     "wildcard :: returns relative URL",
			protocol: "https://",
			host:     "::",
			port:     "8181",
			want:     "",
		},
		{
			name:     "localhost returns absolute URL",
			protocol: "https://",
			host:     "localhost",
			port:     "8181",
			want:     "https://localhost:8181",
		},
		{
			name:     "specific IP returns absolute URL",
			protocol: "http://",
			host:     "192.168.10.13",
			port:     "8181",
			want:     "http://192.168.10.13:8181",
		},
		{
			name:     "IPv6 address is bracketed",
			protocol: "https://",
			host:     "fe80::1",
			port:     "8181",
			want:     "https://[fe80::1]:8181",
		},
		{
			name:     "already-bracketed IPv6 is not double-wrapped",
			protocol: "https://",
			host:     "[::1]",
			port:     "8181",
			want:     "https://[::1]:8181",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := consoleServerAPIBase(tt.protocol, tt.host, tt.port)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestOAuthUIReplacements(t *testing.T) {
	t.Parallel()

	t.Run("basic auth injects no OAuth values", func(t *testing.T) {
		t.Parallel()

		cfg := &config.Config{}
		cfg.Auth.UI.ClientID = "spa-client"
		cfg.Auth.UI.Audience = "https://api.example.com"

		require.Empty(t, oauthUIReplacements(cfg))
	})

	t.Run("OAuth injects the UI values including the audience", func(t *testing.T) {
		t.Parallel()

		cfg := &config.Config{}
		cfg.ClientID = "api-client"
		cfg.Auth.UI.ClientID = "spa-client"
		cfg.Auth.UI.Issuer = "https://idp.example.com"
		cfg.Auth.UI.Scope = "openid profile"
		cfg.Auth.UI.RedirectURI = "https://console.example.com/"
		cfg.Auth.UI.Audience = "https://api.example.com"

		got := oauthUIReplacements(cfg)

		require.Equal(t, ",useOAuth:!0,", got[",useOAuth:!1,"])
		require.Equal(t, "spa-client", got["##CLIENTID##"])
		require.Equal(t, "https://idp.example.com", got["##ISSUER##"])
		require.Equal(t, "openid profile", got["##SCOPE##"])
		require.Equal(t, "https://console.example.com/", got["##REDIRECTURI##"])
		require.Equal(t, "https://api.example.com", got["##AUDIENCE##"])
	})

	t.Run("OAuth without an audience replaces the placeholder with empty", func(t *testing.T) {
		t.Parallel()

		cfg := &config.Config{}
		cfg.ClientID = "api-client"

		got := oauthUIReplacements(cfg)

		value, ok := got["##AUDIENCE##"]
		require.True(t, ok)
		require.Empty(t, value)
	})
}

func TestOAuthUIReplacementsFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		requireHTTPS, strictDiscovery bool
		wantHTTPS, wantStrict         string
	}{
		{true, true, ",requireHttps:!0", ",strictDiscoveryDocumentValidation:!0"},
		{false, false, ",requireHttps:!1", ",strictDiscoveryDocumentValidation:!1"},
	}

	for _, tc := range tests {
		cfg := &config.Config{}
		cfg.ClientID = "api-client"
		cfg.Auth.UI.RequireHTTPS = tc.requireHTTPS
		cfg.Auth.UI.StrictDiscoveryDocumentValidation = tc.strictDiscovery

		got := oauthUIReplacements(cfg)

		require.Equal(t, tc.wantHTTPS, got[",requireHttps:!0"])
		require.Equal(t, tc.wantStrict, got[",strictDiscoveryDocumentValidation:!0"])
	}
}
