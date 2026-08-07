//go:build !noui

package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/internal/usecase"
	"github.com/device-management-toolkit/console/pkg/logger"
)

func TestSecurityHeadersSetsNoSniff(t *testing.T) {
	t.Parallel()

	r := gin.New()
	r.Use(securityHeaders())
	r.GET("/ok", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"a": 1}) })
	// SPA fallback, the path CM-348 was reported against.
	r.NoRoute(func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte("<html></html>"))
	})
	r.GET("/unauth", func(c *gin.Context) {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
	})

	for _, path := range []string{"/ok", "/no/such/route", "/unauth"} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, http.NoBody))

		require.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"), path)
	}
}

// NewRouter must register securityHeaders, not just define it.
//
//nolint:paralleltest // mutates the config.ConsoleConfig singleton
func TestNewRouterRegistersSecurityHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	cfg.Disabled = true

	prev := config.ConsoleConfig

	t.Cleanup(func() { config.ConsoleConfig = prev })

	config.ConsoleConfig = cfg

	handler := gin.New()
	NewRouter(handler, logger.New("error"), usecase.Usecases{}, cfg)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/healthz", http.NoBody))

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
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
