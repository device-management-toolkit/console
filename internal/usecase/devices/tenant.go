package devices

import (
	"context"

	"github.com/device-management-toolkit/console/internal/entity"
	"github.com/device-management-toolkit/console/internal/tenant"
)

// deviceInTenant is the single tenant-scoped device lookup used by every
// management call, so scoping cannot be forgotten per-operation.
func (uc *UseCase) deviceInTenant(ctx context.Context, guid string) (*entity.Device, error) {
	return uc.repo.GetByID(ctx, guid, tenant.FromContext(ctx))
}
