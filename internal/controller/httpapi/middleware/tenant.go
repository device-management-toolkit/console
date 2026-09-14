package middleware

import (
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"

	"github.com/device-management-toolkit/console/pkg/logger"
)

const (
	// TenantHeaderName is the request header carrying the tenant identifier.
	TenantHeaderName  = "x-tenant-id"
	MaxTenantIDLength = 64
	TenantIDPattern   = `^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`
	TenantIDHint      = "x-tenant-id must match " + TenantIDPattern
	tenantIDKey       = "tenant-id"
)

var tenantIDPattern = regexp.MustCompile(TenantIDPattern)

func ValidTenantID(tenantID string) bool {
	return tenantID == "" || tenantIDPattern.MatchString(tenantID)
}

// TenantID returns the tenant resolved by Tenant for the current request.
func TenantID(c *gin.Context) string {
	tenantID, _ := c.Get(tenantIDKey)

	value, _ := tenantID.(string)

	return value
}

// ResolveTenant resolves, validates, and logs the request tenant. An absent header
// yields the empty tenant, which is what existing single-tenant rows are stored
// under. JWT claim resolution can replace the header lookup here without
// changing handler or use-case interfaces.
func ResolveTenant(l logger.Interface) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := c.GetHeader(TenantHeaderName)

		if !ValidTenantID(tenantID) {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": TenantIDHint, "message": TenantIDHint})

			return
		}

		l.Debug("REST request tenant ID", "tenant_id", tenantID)
		c.Set(tenantIDKey, tenantID)

		c.Next()
	}
}
