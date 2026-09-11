package v1

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/device-management-toolkit/console/internal/controller/httpapi/middleware"
	dto "github.com/device-management-toolkit/console/internal/entity/dto/v1"
)

func tenantContext(t *testing.T, tenantID string) *gin.Context {
	t.Helper()

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.Header.Set(middleware.TenantHeaderName, tenantID)
	c.Request = req
	c.Set("tenant-id", tenantID)

	return c
}

func TestTenantIDFromRequest(t *testing.T) {
	t.Parallel()

	require.Equal(t, "tenant-a", tenantIDFromRequest(tenantContext(t, "tenant-a")))
	require.Empty(t, tenantIDFromRequest(tenantContext(t, "")))
}

func TestApplyTenantID(t *testing.T) {
	t.Parallel()

	c := tenantContext(t, "tenant-a")

	profile := dto.Profile{}
	require.NoError(t, applyTenantID(c, &profile.TenantID))
	require.Equal(t, "tenant-a", profile.TenantID)

	// Matching body value is accepted.
	require.NoError(t, applyTenantID(c, &profile.TenantID))
	require.Equal(t, "tenant-a", profile.TenantID)
}

func TestApplyTenantIDRejectsBodyMismatch(t *testing.T) {
	t.Parallel()

	c := tenantContext(t, "tenant-a")

	profile := dto.Profile{TenantID: "tenant-b"}
	require.ErrorIs(t, applyTenantID(c, &profile.TenantID), errTenantMismatch)
	require.Equal(t, "tenant-b", profile.TenantID)
}

func TestApplyTenantIDWithoutTenantKeepsBodyValue(t *testing.T) {
	t.Parallel()

	c := tenantContext(t, "")

	profile := dto.Profile{TenantID: "tenant-b"}
	require.NoError(t, applyTenantID(c, &profile.TenantID))
	require.Equal(t, "tenant-b", profile.TenantID)

	profileEmpty := dto.Profile{TenantID: ""}
	require.NoError(t, applyTenantID(c, &profileEmpty.TenantID))
	require.Equal(t, "", profileEmpty.TenantID)
}

func TestApplyTenantIDWithoutTenantRejectsInvalidBodyValue(t *testing.T) {
	t.Parallel()

	c := tenantContext(t, "")

	profile := dto.Profile{TenantID: "../../other-tenant"}
	require.ErrorIs(t, applyTenantID(c, &profile.TenantID), errInvalidTenantID)
	require.Equal(t, "../../other-tenant", profile.TenantID)
}
