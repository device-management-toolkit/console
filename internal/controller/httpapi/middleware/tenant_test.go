package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/device-management-toolkit/console/internal/controller/httpapi/middleware"
	"github.com/device-management-toolkit/console/internal/tenant"
)

func serveWithDefault(t *testing.T, defaultTenant, headerValue string) (*httptest.ResponseRecorder, string) {
	t.Helper()

	var seen string

	engine := gin.New()
	engine.Use(middleware.Tenant(defaultTenant))
	engine.GET("/", func(c *gin.Context) {
		seen = tenant.FromContext(c.Request.Context())
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	if headerValue != "" {
		req.Header.Set(middleware.TenantHeaderName, headerValue)
	}

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	return recorder, seen
}

func serve(t *testing.T, headerValue string) (*httptest.ResponseRecorder, string) {
	t.Helper()

	return serveWithDefault(t, "", headerValue)
}

func TestTenantScopesRequestContext(t *testing.T) {
	t.Parallel()

	recorder, seen := serve(t, "tenant-a")

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "tenant-a", seen)
}

func TestTenantWithoutHeaderYieldsEmptyTenant(t *testing.T) {
	t.Parallel()

	recorder, seen := serve(t, "")

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Empty(t, seen)
}

func TestTenantRejectsMalformedHeader(t *testing.T) {
	t.Parallel()

	recorder, seen := serve(t, "tenant a")

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Contains(t, recorder.Body.String(), "x-tenant-id must be")
	require.Empty(t, seen)
}

func TestTenantFallsBackToDefaultTenant(t *testing.T) {
	t.Parallel()

	recorder, seen := serveWithDefault(t, "acme-corp", "")

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "acme-corp", seen)
}

func TestTenantHeaderOverridesDefaultTenant(t *testing.T) {
	t.Parallel()

	recorder, seen := serveWithDefault(t, "acme-corp", "globex")

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "globex", seen)
}

// A misconfigured default must fail loudly rather than silently widening scope.
func TestTenantRejectsMalformedDefaultTenant(t *testing.T) {
	t.Parallel()

	recorder, seen := serveWithDefault(t, "bad tenant", "")

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Empty(t, seen)
}
