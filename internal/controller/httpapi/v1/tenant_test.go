package v1

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	dto "github.com/device-management-toolkit/console/internal/entity/dto/v1"
)

// tenantContext returns a context that has already passed TenantMiddleware.
func tenantContext(t *testing.T, headerValue string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/", http.NoBody)

	if headerValue != "" {
		c.Request.Header.Set(tenantHeaderName, headerValue)
	}

	TenantMiddleware()(c)

	return c, recorder
}

func TestTenantIDFromHeader(t *testing.T) {
	t.Parallel()

	c, _ := tenantContext(t, "tenant-a")
	require.Equal(t, "tenant-a", tenantIDFromHeader(c))

	c, _ = tenantContext(t, "")
	require.Empty(t, tenantIDFromHeader(c))
}

func TestTenantMiddlewareRejectsMalformedHeader(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
	}{
		{"single space", " "},
		{"embedded space", "tenant a"},
		{"leading space", " tenant-a"},
		{"trailing space", "tenant-a "},
		{"tab", "tenant\tb"},
		{"newline", "tenant\nb"},
		{"carriage return", "tenant\rb"},
		{"null byte", "tenant\x00b"},
		{"forward slash", "tenant/a"},
		{"backslash", "tenant\\a"},
		{"colon", "tenant:a"},
		{"at sign", "tenant@a"},
		{"percent encoding", "tenant%20a"},
		{"sql quote", "tenant'a"},
		{"sql injection", "a' OR '1'='1"},
		{"wildcard", "tenant*"},
		{"comma", "tenant,a"},
		{"path traversal", "../tenant"},
		{"cyrillic homoglyph", "tenant-\u0430"},
		{"zero width space", "tenant\u200ba"},
		{"emoji", "tenant-\U0001F600"},
		{"one over max length", strings.Repeat("a", 65)},
		{"far over max length", strings.Repeat("a", 4096)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c, recorder := tenantContext(t, tt.value)

			require.True(t, c.IsAborted())
			require.Equal(t, http.StatusBadRequest, recorder.Code)
			require.Empty(t, tenantIDFromHeader(c))
		})
	}
}

func TestTenantMiddlewareAcceptsValidHeader(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
	}{
		{"single character", "a"},
		{"single digit", "1"},
		{"lower case", "tenant"},
		{"upper case", "TENANT"},
		{"mixed case", "TenantA"},
		{"hyphen", "tenant-a"},
		{"underscore", "tenant_a"},
		{"dot", "tenant.a"},
		{"all separators", "a-b_c.d"},
		{"uuid", "3a1b0c8e-4f2d-4a6b-9c1e-7d5f8a0b2c3d"},
		{"at max length", strings.Repeat("a", 64)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c, recorder := tenantContext(t, tt.value)

			require.False(t, c.IsAborted())
			require.Equal(t, http.StatusOK, recorder.Code)
			require.Equal(t, tt.value, tenantIDFromHeader(c))
		})
	}
}

func TestApplyTenantID(t *testing.T) {
	t.Parallel()

	c, _ := tenantContext(t, "tenant-a")

	profile := dto.Profile{}
	require.NoError(t, applyTenantID(c, &profile.TenantID))
	require.Equal(t, "tenant-a", profile.TenantID)

	// Matching body value is accepted.
	require.NoError(t, applyTenantID(c, &profile.TenantID))
	require.Equal(t, "tenant-a", profile.TenantID)
}

func TestApplyTenantIDRejectsBodyMismatch(t *testing.T) {
	t.Parallel()

	c, _ := tenantContext(t, "tenant-a")

	profile := dto.Profile{TenantID: "tenant-b"}
	require.ErrorIs(t, applyTenantID(c, &profile.TenantID), errTenantMismatch)
	require.Equal(t, "tenant-b", profile.TenantID)
}

func TestApplyTenantIDWithoutHeaderKeepsBodyValue(t *testing.T) {
	t.Parallel()

	c, _ := tenantContext(t, "")

	profile := dto.Profile{TenantID: "tenant-b"}
	require.NoError(t, applyTenantID(c, &profile.TenantID))
	require.Equal(t, "tenant-b", profile.TenantID)
}
