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

func TestUIFallbackAddsNoCacheHeaders(t *testing.T) {
	t.Parallel()

	engine := gin.New()
	setupUIRoutes(engine, logger.New("error"), &config.Config{})

	req := httptest.NewRequest(http.MethodGet, "/random-non-asset-route", http.NoBody)
	w := httptest.NewRecorder()

	engine.ServeHTTP(w, req)

	require.Contains(t, []int{http.StatusOK, http.StatusMovedPermanently, http.StatusNotFound}, w.Code)
	require.Equal(t, cacheControlNoStore, w.Header().Get("Cache-Control"))
	require.Equal(t, pragmaNoCache, w.Header().Get("Pragma"))
	require.Equal(t, expiresNoCache, w.Header().Get("Expires"))
}

func TestUIRootAddsNoCacheHeaders(t *testing.T) {
	t.Parallel()

	engine := gin.New()
	setupUIRoutes(engine, logger.New("error"), &config.Config{})

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	w := httptest.NewRecorder()

	engine.ServeHTTP(w, req)

	require.Contains(t, []int{http.StatusOK, http.StatusMovedPermanently, http.StatusNotFound}, w.Code)
	require.Equal(t, cacheControlNoStore, w.Header().Get("Cache-Control"))
	require.Equal(t, pragmaNoCache, w.Header().Get("Pragma"))
	require.Equal(t, expiresNoCache, w.Header().Get("Expires"))
}

func TestUIAssetsNoRouteReturns404WithoutNoCacheHeaders(t *testing.T) {
	t.Parallel()

	engine := gin.New()
	setupUIRoutes(engine, logger.New("error"), &config.Config{})

	req := httptest.NewRequest(http.MethodGet, "/assets/does-not-exist", http.NoBody)
	w := httptest.NewRecorder()

	engine.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
	require.Empty(t, w.Header().Get("Cache-Control"))
	require.Empty(t, w.Header().Get("Pragma"))
	require.Empty(t, w.Header().Get("Expires"))
}

func TestNoCacheHeadersMiddleware(t *testing.T) {
	t.Parallel()

	engine := gin.New()
	engine.Use(noCacheHeadersMiddleware())
	engine.GET("/api/v1/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", http.NoBody)
	w := httptest.NewRecorder()

	engine.ServeHTTP(w, req)

	require.Equal(t, cacheControlNoStore, w.Header().Get("Cache-Control"))
	require.Equal(t, pragmaNoCache, w.Header().Get("Pragma"))
	require.Equal(t, expiresNoCache, w.Header().Get("Expires"))
}

func TestNoCacheHeadersMiddlewareSkipsNonAPIPaths(t *testing.T) {
	t.Parallel()

	engine := gin.New()
	engine.Use(noCacheHeadersMiddleware())
	engine.GET("/healthz", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/healthz", http.NoBody)
	w := httptest.NewRecorder()

	engine.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Empty(t, w.Header().Get("Cache-Control"))
	require.Empty(t, w.Header().Get("Pragma"))
	require.Empty(t, w.Header().Get("Expires"))
}

//nolint:paralleltest // mutates shared global config.ConsoleConfig
func TestNewRouterAuthorizeRouteHasNoCacheHeaders(t *testing.T) {
	prev := config.ConsoleConfig
	config.ConsoleConfig = &config.Config{}

	t.Cleanup(func() { config.ConsoleConfig = prev })

	engine := gin.New()
	NewRouter(engine, logger.New("error"), usecase.Usecases{}, &config.Config{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/authorize", http.NoBody)
	w := httptest.NewRecorder()

	engine.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Equal(t, cacheControlNoStore, w.Header().Get("Cache-Control"))
	require.Equal(t, pragmaNoCache, w.Header().Get("Pragma"))
	require.Equal(t, expiresNoCache, w.Header().Get("Expires"))
}
