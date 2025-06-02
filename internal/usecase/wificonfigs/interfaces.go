package wificonfigs

import (
	"context"

	"github.com/device-management-toolkit/console/internal/entity"
	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
)

type (
	Repository interface {
		CheckProfileExists(ctx context.Context, profileName, tenantID string) (bool, error)
		GetCount(ctx context.Context, tenantID string) (int, error)
		Get(ctx context.Context, top, skip int, tenantID string) ([]entity.WirelessConfig, error)
		GetByName(ctx context.Context, guid, tenantID string) (*entity.WirelessConfig, error)
		Delete(ctx context.Context, profileName, tenantID string) (bool, error)
		Update(ctx context.Context, p *entity.WirelessConfig) (bool, error)
		Insert(ctx context.Context, p *entity.WirelessConfig) (string, error)
	}

	Feature interface {
		CheckProfileExists(ctx context.Context, profileName, tenantID string) (bool, error)
		GetCount(ctx context.Context, tenantID string) (int, error)
		Get(ctx context.Context, top, skip int, tenantID string) ([]dto.WirelessConfig, error)
		GetByName(ctx context.Context, guid, tenantID string) (*dto.WirelessConfig, error)
		Delete(ctx context.Context, profileName, tenantID string) error
		Update(ctx context.Context, p *dto.WirelessConfig) (*dto.WirelessConfig, error)
		Insert(ctx context.Context, p *dto.WirelessConfig) (*dto.WirelessConfig, error)
	}
)
