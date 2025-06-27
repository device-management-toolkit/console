package ciraconfigs

import (
	"context"

	"github.com/device-management-toolkit/console/internal/entity"
	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
)

type (
	Repository interface {
		GetCount(ctx context.Context, tenantID string) (int, error)
		Get(ctx context.Context, top, skip int, tenantID string) ([]entity.CIRAConfig, error)
		GetByName(ctx context.Context, configName, tenantID string) (*entity.CIRAConfig, error)
		Delete(ctx context.Context, profileName, tenantID string) (bool, error)
		Update(ctx context.Context, p *entity.CIRAConfig) (bool, error)
		Insert(ctx context.Context, p *entity.CIRAConfig) (string, error)
	}
	Feature interface {
		GetCount(ctx context.Context, tenantID string) (int, error)
		Get(ctx context.Context, top, skip int, tenantID string) ([]dto.CIRAConfig, error)
		GetByName(ctx context.Context, configName, tenantID string) (*dto.CIRAConfig, error)
		Delete(ctx context.Context, profileName, tenantID string) error
		Update(ctx context.Context, p *dto.CIRAConfig) (*dto.CIRAConfig, error)
		Insert(ctx context.Context, p *dto.CIRAConfig) (*dto.CIRAConfig, error)
	}
)
