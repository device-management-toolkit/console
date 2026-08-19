package v1

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	dto "github.com/device-management-toolkit/console/internal/entity/dto/v1"
)

func TestTenantIDFromHeader(t *testing.T) {
	t.Parallel()

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.Header.Set(tenantHeaderName, "tenant-a")
	c.Request = req

	require.Equal(t, "tenant-a", tenantIDFromHeader(c))
}

func TestApplyTenantID(t *testing.T) {
	t.Parallel()

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.Header.Set(tenantHeaderName, "tenant-a")
	c.Request = req

	profile := dto.Profile{}
	applyTenantID(c, &profile.TenantID)
	require.Equal(t, "tenant-a", profile.TenantID)

	req.Header.Del(tenantHeaderName)
	applyTenantID(c, &profile.TenantID)
	require.Equal(t, "tenant-a", profile.TenantID)
}
