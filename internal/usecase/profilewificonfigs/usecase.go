package profilewificonfigs

import (
	"context"

	"github.com/device-management-toolkit/console/internal/entity"
	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/usecase/sqldb"
	"github.com/device-management-toolkit/console/pkg/consoleerrors"
	"github.com/device-management-toolkit/console/pkg/logger"
)

type UseCase struct {
	repo Repository
	log  logger.Interface
}

var (
	ErrProfileWiFiConfigsUseCase = consoleerrors.CreateConsoleError("ProfilesWiFiUseCase")
	ErrDatabase                  = sqldb.DatabaseError{Console: consoleerrors.CreateConsoleError("ProfilesWiFiUseCase")}
	ErrNotFound                  = sqldb.NotFoundError{Console: consoleerrors.CreateConsoleError("ProfilesWiFiUseCase")}
)

func New(r Repository, log logger.Interface) *UseCase {
	return &UseCase{
		repo: r,
		log:  log,
	}
}

func (uc *UseCase) GetByProfileName(ctx context.Context, profileName, tenantID string) ([]dto.ProfileWiFiConfigs, error) {
	data, err := uc.repo.GetByProfileName(ctx, profileName, tenantID)
	if err != nil {
		return nil, ErrDatabase.Wrap("Get", "uc.repo.Get", err)
	}

	if data == nil {
		return nil, ErrNotFound
	}

	// iterate over the data and convert each entity to dto
	d1 := make([]dto.ProfileWiFiConfigs, len(data))

	for i := range data {
		tmpEntity := data[i] // create a new variable to avoid memory aliasing
		d1[i] = *uc.entityToDTO(&tmpEntity)
	}

	return d1, nil
}

func (uc *UseCase) DeleteByProfileName(ctx context.Context, profileName, tenantID string) error {
	_, err := uc.repo.DeleteByProfileName(ctx, profileName, tenantID)
	if err != nil {
		return ErrDatabase.Wrap("Delete", "uc.repo.Delete", err)
	}

	return nil
}

func (uc *UseCase) Insert(ctx context.Context, d *dto.ProfileWiFiConfigs) error {
	d1 := uc.dtoToEntity(d)

	_, err := uc.repo.Insert(ctx, d1)
	if err != nil {
		return ErrDatabase.Wrap("Insert", "uc.repo.Insert", err)
	}

	return nil
}

// convert dto.ProfileWiFiConfigs to entity.ProfileWiFiConfigs.
func (uc *UseCase) dtoToEntity(d *dto.ProfileWiFiConfigs) *entity.ProfileWiFiConfigs {
	d1 := &entity.ProfileWiFiConfigs{
		Priority:            d.Priority,
		WirelessProfileName: d.WirelessProfileName,
		ProfileName:         d.ProfileName,
		TenantID:            d.TenantID,
	}

	return d1
}

// convert entity.ProfileWiFiConfigs to dto.ProfileWiFiConfigs.
func (uc *UseCase) entityToDTO(d *entity.ProfileWiFiConfigs) *dto.ProfileWiFiConfigs {
	d1 := &dto.ProfileWiFiConfigs{
		Priority:            d.Priority,
		WirelessProfileName: d.WirelessProfileName,
		ProfileName:         d.ProfileName,
		TenantID:            d.TenantID,
	}

	return d1
}
