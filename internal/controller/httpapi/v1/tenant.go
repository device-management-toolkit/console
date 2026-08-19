package v1

import (
	"github.com/gin-gonic/gin"
)

const tenantHeaderName = "x-tenant-id"

func tenantIDFromHeader(c *gin.Context) string {
	return c.GetHeader(tenantHeaderName)
}

// applyTenantID overwrites target with the request's tenant header. An absent
// header leaves the body-supplied value untouched.
func applyTenantID(c *gin.Context, target *string) {
	if tenantID := tenantIDFromHeader(c); tenantID != "" {
		*target = tenantID
	}
}
