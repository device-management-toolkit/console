package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/internal/controller/httpapi/middleware"
	"github.com/device-management-toolkit/console/internal/tenant"
	"github.com/device-management-toolkit/console/pkg/logger"
)

func serve(t *testing.T, headerValue string) (recorder *httptest.ResponseRecorder, seen string) {
	t.Helper()

	engine := gin.New()
	engine.Use(middleware.Tenant(logger.New("error")))
	engine.GET("/", func(c *gin.Context) {
		seen = tenant.FromContext(c.Request.Context())
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	if headerValue != "" {
		req.Header.Set(middleware.TenantHeaderName, headerValue)
	}

	recorder = httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	return recorder, seen
}

func TestTenantScopesRequestContext(t *testing.T) {
	t.Parallel()

	recorder, seen := serve(t, "tenant-a")

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "tenant-a", seen)
}

func TestTenantWithoutHeaderYieldsEmptyTenant(t *testing.T) {
	prev := config.ConsoleConfig
	t.Cleanup(func() { config.ConsoleConfig = prev })
	config.ConsoleConfig = &config.Config{App: config.App{DefaultTenant: ""}}

	recorder, seen := serve(t, "")

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Empty(t, seen)
}

func TestTenantWithoutHeaderUsesDefaultTenant(t *testing.T) {
	prev := config.ConsoleConfig
	t.Cleanup(func() { config.ConsoleConfig = prev })
	config.ConsoleConfig = &config.Config{App: config.App{DefaultTenant: "tenant-default"}}

	recorder, seen := serve(t, "")

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "tenant-default", seen)
}

func TestTenantRejectsMalformedHeader(t *testing.T) {
	t.Parallel()

	recorder, seen := serve(t, "tenant a")

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Contains(t, recorder.Body.String(), "x-tenant-id must match")
	require.Empty(t, seen)
}
