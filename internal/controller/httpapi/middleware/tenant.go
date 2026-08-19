package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/device-management-toolkit/console/internal/tenant"
)

// TenantHeaderName is the request header carrying the tenant identifier.
const TenantHeaderName = "x-tenant-id"

// Tenant validates the tenant header and scopes the request context to it.
// defaultTenant is applied when the header is absent; empty keeps single-tenant
// requests on the rows they already have.
func Tenant(defaultTenant string) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := c.GetHeader(TenantHeaderName)
		if tenantID == "" {
			tenantID = defaultTenant
		}

		if !tenant.Valid(tenantID) {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": tenant.Hint, "message": tenant.Hint})

			return
		}

		// Leave the context untouched for the default tenant so single-tenant
		// requests carry no extra value.
		if tenantID != "" {
			c.Request = c.Request.WithContext(tenant.WithContext(c.Request.Context(), tenantID))
		}

		c.Next()
	}
}
