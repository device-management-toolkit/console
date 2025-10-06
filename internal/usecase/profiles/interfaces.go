package profiles

import (
	"context"

	"github.com/device-management-toolkit/console/internal/entity"
	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
)

type (
	Repository interface {
		GetCount(ctx context.Context, tenantID string) (int, error)
		Get(ctx context.Context, top, skip int, tenantID string) ([]entity.Profile, error)
		GetByName(ctx context.Context, profileName, tenantID string) (*entity.Profile, error)
		Delete(ctx context.Context, profileName, tenantID string) (bool, error)
		Update(ctx context.Context, p *entity.Profile) (bool, error)
		Insert(ctx context.Context, p *entity.Profile) (string, error)
	}

	Feature interface {
		GetCount(ctx context.Context, tenantID string) (int, error)
		Get(ctx context.Context, top, skip int, tenantID string) ([]dto.Profile, error)
		GetByName(ctx context.Context, profileName, tenantID string) (*dto.Profile, error)
		Delete(ctx context.Context, profileName, tenantID string) error
		Update(ctx context.Context, p *dto.Profile) (*dto.Profile, error)
		Insert(ctx context.Context, p *dto.Profile) (*dto.Profile, error)
		Export(ctx context.Context, profileName, domainName, tenantID string) (string, string, error)
	}
)
