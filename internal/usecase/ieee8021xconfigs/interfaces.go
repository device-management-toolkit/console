package ieee8021xconfigs

import (
	"context"

	"github.com/device-management-toolkit/console/internal/entity"
	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
)

type (
	Repository interface {
		CheckProfileExists(ctx context.Context, profileName, tenantID string) (bool, error)
		GetCount(ctx context.Context, tenantID string) (int, error)
		Get(ctx context.Context, top, skip int, tenantID string) ([]entity.IEEE8021xConfig, error)
		GetByName(ctx context.Context, profileName, tenantID string) (*entity.IEEE8021xConfig, error)
		Delete(ctx context.Context, profileName, tenantID string) (bool, error)
		Update(ctx context.Context, p *entity.IEEE8021xConfig) (bool, error)
		Insert(ctx context.Context, p *entity.IEEE8021xConfig) (string, error)
	}

	Feature interface {
		CheckProfileExists(ctx context.Context, profileName, tenantID string) (bool, error)
		GetCount(ctx context.Context, tenantID string) (int, error)
		Get(ctx context.Context, top, skip int, tenantID string) ([]dto.IEEE8021xConfig, error)
		GetByName(ctx context.Context, profileName, tenantID string) (*dto.IEEE8021xConfig, error)
		Delete(ctx context.Context, profileName, tenantID string) error
		Update(ctx context.Context, p *dto.IEEE8021xConfig) (*dto.IEEE8021xConfig, error)
		Insert(ctx context.Context, p *dto.IEEE8021xConfig) (*dto.IEEE8021xConfig, error)
	}
)
