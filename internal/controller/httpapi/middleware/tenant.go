package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/internal/tenant"
	"github.com/device-management-toolkit/console/pkg/logger"
)

// TenantHeaderName is the request header carrying the tenant identifier.
const TenantHeaderName = "x-tenant-id"

// Tenant validates the tenant header and scopes the request context to it. An
// absent header yields the empty tenant, which is what existing single-tenant
// rows are stored under.
func Tenant(l logger.Interface) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := c.GetHeader(TenantHeaderName)
		if tenantID == "" && config.ConsoleConfig != nil && config.ConsoleConfig.DefaultTenant != "" {
			tenantID = config.ConsoleConfig.DefaultTenant
		}

		l.Debug("REST request tenant ID", "tenant_id", tenantID)

		if tenantID != "" {
			c.Request = c.Request.WithContext(tenant.WithContext(c.Request.Context(), tenantID))
		}

		c.Next()
	}
}
