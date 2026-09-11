package v1

import (
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/device-management-toolkit/console/internal/controller/httpapi/middleware"
	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/pkg/consoleerrors"
)

var (
	ErrValidationTenant = dto.NotValidError{Console: consoleerrors.CreateConsoleError("TenantAPI")}
	errTenantMismatch   = errors.New("tenantId in body conflicts with x-tenant-id header")
	errInvalidTenantID  = errors.New(middleware.TenantIDHint)
)

func tenantIDFromRequest(c *gin.Context) string {
	return middleware.TenantID(c)
}

// applyTenantID overwrites target with the request's tenant. An absent header
// leaves the body-supplied value untouched if valid; a conflicting or invalid
// tenant ID is rejected so a caller cannot write outside the tenant it asked for
// or supply a malformed tenant ID.
func applyTenantID(c *gin.Context, target *string) error {
	tenantID := tenantIDFromRequest(c)

	if tenantID == "" {
		if !middleware.ValidTenantID(*target) {
			return ErrValidationTenant.Wrap("applyTenantID", "tenantId", errInvalidTenantID)
		}

		return nil
	}

	if *target != "" && *target != tenantID {
		return ErrValidationTenant.Wrap("applyTenantID", "tenantId", errTenantMismatch)
	}

	*target = tenantID

	return nil
}
