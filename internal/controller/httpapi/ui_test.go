//go:build !noui

package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestApplyUIConfigAuthMode(t *testing.T) {
	t.Parallel()

	const bundle = `{cloud:!1,useOAuth:!1,authDisabled:!1,mpsServer:"##CONSOLE_SERVER_API##",auth:{clientId:"##CLIENTID##"}}`

	tests := []struct {
		name         string
		disabled     bool
		clientID     string
		wantDisabled bool
		wantOAuth    bool
	}{
		{name: "auth enabled without OAuth leaves both flags off"},
		{name: "auth disabled turns on authDisabled", disabled: true, wantDisabled: true},
		{name: "OAuth client turns on useOAuth", clientID: "client", wantOAuth: true},
		{name: "auth disabled wins over OAuth", disabled: true, clientID: "client", wantDisabled: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg := &config.Config{}
			cfg.Disabled = tc.disabled
			cfg.ClientID = tc.clientID

			got := string(applyUIConfig([]byte(bundle), cfg))

			require.Equal(t, tc.wantDisabled, strings.Contains(got, ",authDisabled:!0,"))
			require.Equal(t, tc.wantOAuth, strings.Contains(got, ",useOAuth:!0,"))
		})
	}
}
