package v1

import (
	"errors"
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/pkg/consoleerrors"
)

const (
	tenantHeaderName = "x-tenant-id"
	// tenantContextKey holds the validated tenant ID for the current request.
	tenantContextKey = "tenantID"
	tenantIDHint     = "x-tenant-id must be 1-64 characters of A-Z, a-z, 0-9, dot, underscore or hyphen"
)

// tenantIDPattern constrains the header to characters that are safe as part of
// the composite primary key used across Postgres, SQLite and MongoDB.
var tenantIDPattern = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)

var (
	ErrValidationTenant = dto.NotValidError{Console: consoleerrors.CreateConsoleError("TenantAPI")}
	errTenantMismatch   = errors.New("tenantId in body conflicts with x-tenant-id header")
)

// TenantMiddleware validates x-tenant-id and stores the result on the request
// context. An absent header yields the empty tenant, which is what existing
// single-tenant rows are stored under.
func TenantMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := c.GetHeader(tenantHeaderName)
		if tenantID != "" && !tenantIDPattern.MatchString(tenantID) {
			c.AbortWithStatusJSON(http.StatusBadRequest, response{Error: tenantIDHint, Message: tenantIDHint})

			return
		}

		c.Set(tenantContextKey, tenantID)
		c.Next()
	}
}

func tenantIDFromHeader(c *gin.Context) string {
	tenantID, _ := c.Value(tenantContextKey).(string)

	return tenantID
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
