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

func TestClickjackingProtectionMiddlewareSetsHeaders(t *testing.T) {
	t.Parallel()

	engine := gin.New()
	engine.Use(clickjackingProtectionMiddleware())
	engine.GET("/healthz", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/healthz", http.NoBody)
	w := httptest.NewRecorder()

	engine.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, xFrameOptionsHeaderValue, w.Header().Get("X-Frame-Options"))
	require.Equal(t, contentSecurityPolicyHeaderValue, w.Header().Get("Content-Security-Policy"))
}

//nolint:paralleltest // mutates shared global config.ConsoleConfig
func TestNewRouterAuthorizeHasClickjackingHeaders(t *testing.T) {
	prev := config.ConsoleConfig
	config.ConsoleConfig = &config.Config{}

	t.Cleanup(func() { config.ConsoleConfig = prev })

	engine := gin.New()
	NewRouter(engine, logger.New("error"), usecase.Usecases{}, &config.Config{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/authorize", http.NoBody)
	w := httptest.NewRecorder()

	engine.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Equal(t, xFrameOptionsHeaderValue, w.Header().Get("X-Frame-Options"))
	require.Equal(t, contentSecurityPolicyHeaderValue, w.Header().Get("Content-Security-Policy"))
}
