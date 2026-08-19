package v1

import (
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/tenant"
	"github.com/device-management-toolkit/console/pkg/consoleerrors"
)

var (
	ErrValidationTenant = dto.NotValidError{Console: consoleerrors.CreateConsoleError("TenantAPI")}
	errTenantMismatch   = errors.New("tenantId in body conflicts with x-tenant-id header")
)

func tenantIDFromHeader(c *gin.Context) string {
	return tenant.FromContext(c.Request.Context())
}

// applyTenantID overwrites target with the request's tenant. An absent header
// leaves the body-supplied value untouched; a conflicting one is rejected so a
// caller cannot write outside the tenant it asked for.
func applyTenantID(c *gin.Context, target *string) error {
	tenantID := tenantIDFromHeader(c)
	if tenantID == "" {
		return nil
	}

	if *target != "" && *target != tenantID {
		return ErrValidationTenant.Wrap("applyTenantID", "tenantId", errTenantMismatch)
	}

	*target = tenantID

	return nil
}
