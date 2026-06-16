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
	req := httptest.NewRequest("GET", "/", http.NoBody)
	req.Header.Set(tenantHeaderName, "tenant-a")
	c.Request = req

	require.Equal(t, "tenant-a", tenantIDFromHeader(c))
}

func TestSetTenantID(t *testing.T) {
	t.Parallel()

	profile := dto.Profile{}
	setTenantID(&profile, "tenant-a")
	require.Equal(t, "tenant-a", profile.TenantID)

	setTenantID(&profile, "")
	require.Equal(t, "tenant-a", profile.TenantID)
}