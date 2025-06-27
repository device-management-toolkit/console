package profilewificonfigs

import (
	"context"

	"github.com/device-management-toolkit/console/internal/entity"
	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
)

type (
	Repository interface {
		GetByProfileName(ctx context.Context, profileName, tenantID string) ([]entity.ProfileWiFiConfigs, error)
		DeleteByProfileName(ctx context.Context, profileName, tenantID string) (bool, error)
		Insert(ctx context.Context, p *entity.ProfileWiFiConfigs) (string, error)
	}

	Feature interface {
		GetByProfileName(ctx context.Context, profileName, tenantID string) ([]dto.ProfileWiFiConfigs, error)
		DeleteByProfileName(ctx context.Context, profileName, tenantID string) error
		Insert(ctx context.Context, p *dto.ProfileWiFiConfigs) error
	}
)
