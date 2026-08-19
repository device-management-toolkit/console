package openapi

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/device-management-toolkit/console/internal/controller/httpapi/middleware"
)

// tenantHeaderNames returns, per operation, whether x-tenant-id is declared.
func tenantHeaderNames(t *testing.T, spec map[string]interface{}) map[string]bool {
	t.Helper()

	paths, ok := spec["paths"].(map[string]interface{})
	require.True(t, ok)

	declared := map[string]bool{}

	for path, item := range paths {
		operations, ok := item.(map[string]interface{})
		require.True(t, ok)

		for method, raw := range operations {
			operation, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}

			found := false

			if params, ok := operation["parameters"].([]interface{}); ok {
				for _, p := range params {
					param, ok := p.(map[string]interface{})
					if ok && param["name"] == middleware.TenantHeaderName && param["in"] == "header" {
						found = true
					}
				}
			}

			declared[method+" "+path] = found
		}
	}

	return declared
}

func TestProtectedRoutesDeclareTenantHeader(t *testing.T) {
	t.Parallel()

	f := newTestAdapter()
	f.RegisterRoutes()

	specBytes, err := f.GetOpenAPISpec()
	require.NoError(t, err)

	var spec map[string]interface{}
	require.NoError(t, json.Unmarshal(specBytes, &spec))

	declared := tenantHeaderNames(t, spec)
	require.NotEmpty(t, declared)

	// Only the public authorize endpoints are unscoped.
	public := map[string]bool{
		"post /api/v1/authorize":        true,
		"post /api/v1/authorize/logout": true,
	}

	for operation, hasHeader := range declared {
		if public[operation] {
			require.False(t, hasHeader, "%s is public and must not advertise a tenant header", operation)

			continue
		}

		require.True(t, hasHeader, "%s must declare the tenant header", operation)
	}
}
