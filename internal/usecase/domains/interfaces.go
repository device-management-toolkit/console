package domains

import (
	"context"

	"github.com/device-management-toolkit/console/internal/entity"
	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
)

type (
	Repository interface {
		GetCount(context.Context, string) (int, error)
		Get(ctx context.Context, top, skip int, tenantID string) ([]entity.Domain, error)
		GetDomainByDomainSuffix(ctx context.Context, domainSuffix, tenantID string) (*entity.Domain, error)
		GetByName(ctx context.Context, name, tenantID string) (*entity.Domain, error)
		Delete(ctx context.Context, name, tenantID string) (bool, error)
		Update(ctx context.Context, d *entity.Domain) (bool, error)
		Insert(ctx context.Context, d *entity.Domain) (string, error)
	}
	Feature interface {
		GetCount(context.Context, string) (int, error)
		Get(ctx context.Context, top, skip int, tenantID string) ([]dto.Domain, error)
		GetDomainByDomainSuffix(ctx context.Context, domainSuffix, tenantID string) (*dto.Domain, error)
		GetByName(ctx context.Context, name, tenantID string) (*dto.Domain, error)
		Delete(ctx context.Context, name, tenantID string) error
		Update(ctx context.Context, d *dto.Domain) (*dto.Domain, error)
		Insert(ctx context.Context, d *dto.Domain) (*dto.Domain, error)
	}
)
