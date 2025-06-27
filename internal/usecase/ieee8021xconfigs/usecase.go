package ieee8021xconfigs

import (
	"context"

	"github.com/device-management-toolkit/console/internal/entity"
	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/usecase/sqldb"
	"github.com/device-management-toolkit/console/pkg/consoleerrors"
	"github.com/device-management-toolkit/console/pkg/logger"
)

// UseCase -.
type UseCase struct {
	repo Repository
	log  logger.Interface
}

var (
	ErrDomainsUseCase = consoleerrors.CreateConsoleError("IEEE8021xUseCase")
	ErrDatabase       = sqldb.DatabaseError{Console: consoleerrors.CreateConsoleError("IEEE8021xUseCase")}
	ErrNotFound       = sqldb.NotFoundError{Console: consoleerrors.CreateConsoleError("IEEE8021xUseCase")}
)

// New -.
func New(r Repository, log logger.Interface) *UseCase {
	return &UseCase{
		repo: r,
		log:  log,
	}
}

// History - getting translate history from store.
func (uc *UseCase) CheckProfileExists(ctx context.Context, profileName, tenantID string) (bool, error) {
	data, err := uc.repo.CheckProfileExists(ctx, profileName, tenantID)
	if err != nil {
		return false, ErrDatabase.Wrap("Count", "uc.repo.GetCount", err)
	}

	return data, nil
}

func (uc *UseCase) GetCount(ctx context.Context, tenantID string) (int, error) {
	count, err := uc.repo.GetCount(ctx, tenantID)
	if err != nil {
		return 0, ErrDatabase.Wrap("Count", "uc.repo.GetCount", err)
	}

	return count, nil
}

func (uc *UseCase) Get(ctx context.Context, top, skip int, tenantID string) ([]dto.IEEE8021xConfig, error) {
	data, err := uc.repo.Get(ctx, top, skip, tenantID)
	if err != nil {
		return nil, ErrDatabase.Wrap("Get", "uc.repo.Get", err)
	}

	if data == nil {
		return nil, ErrNotFound
	}

	// iterate over the data and convert each entity to dto
	d1 := make([]dto.IEEE8021xConfig, len(data))

	for i := range data {
		tmpEntity := data[i] // create a new variable to avoid memory aliasing
		d1[i] = *uc.entityToDTO(&tmpEntity)
	}

	return d1, nil
}

func (uc *UseCase) GetByName(ctx context.Context, profileName, tenantID string) (*dto.IEEE8021xConfig, error) {
	data, err := uc.repo.GetByName(ctx, profileName, tenantID)
	if err != nil {
		return nil, ErrDatabase.Wrap("GetByName", "uc.repo.GetByName", err)
	}

	if data == nil {
		return nil, ErrNotFound
	}

	d2 := uc.entityToDTO(data)

	return d2, nil
}

func (uc *UseCase) Delete(ctx context.Context, profileName, tenantID string) error {
	isSuccessful, err := uc.repo.Delete(ctx, profileName, tenantID)
	if err != nil {
		return ErrDatabase.Wrap("Delete", "uc.repo.Delete", err)
	}

	if !isSuccessful {
		return ErrNotFound
	}

	return nil
}

func (uc *UseCase) Update(ctx context.Context, d *dto.IEEE8021xConfig) (*dto.IEEE8021xConfig, error) {
	d1 := uc.dtoToEntity(d)

	updated, err := uc.repo.Update(ctx, d1)
	if err != nil {
		return nil, ErrDatabase.Wrap("Update", "uc.repo.Update", err)
	}

	if !updated {
		return nil, ErrNotFound
	}

	updatedCiraConfig, err := uc.repo.GetByName(ctx, d.ProfileName, d.TenantID)
	if err != nil {
		return nil, err
	}

	d2 := uc.entityToDTO(updatedCiraConfig)

	return d2, nil
}

func (uc *UseCase) Insert(ctx context.Context, d *dto.IEEE8021xConfig) (*dto.IEEE8021xConfig, error) {
	d1 := uc.dtoToEntity(d)

	_, err := uc.repo.Insert(ctx, d1)
	if err != nil {
		return nil, ErrDatabase.Wrap("Insert", "uc.repo.Insert", err)
	}

	newConfig, err := uc.repo.GetByName(ctx, d.ProfileName, d.TenantID)
	if err != nil {
		return nil, err
	}

	d2 := uc.entityToDTO(newConfig)

	return d2, nil
}

// convert dto.Domain to entity.Domain.
func (uc *UseCase) dtoToEntity(d *dto.IEEE8021xConfig) *entity.IEEE8021xConfig {
	d1 := &entity.IEEE8021xConfig{
		ProfileName:            d.ProfileName,
		AuthenticationProtocol: d.AuthenticationProtocol,
		PXETimeout:             d.PXETimeout,
		WiredInterface:         d.WiredInterface,
		TenantID:               d.TenantID,
		Version:                d.Version,
	}

	return d1
}

// convert entity.Domain to dto.Domain.
func (uc *UseCase) entityToDTO(d *entity.IEEE8021xConfig) *dto.IEEE8021xConfig {
	d1 := &dto.IEEE8021xConfig{
		ProfileName:            d.ProfileName,
		AuthenticationProtocol: d.AuthenticationProtocol,
		PXETimeout:             d.PXETimeout,
		WiredInterface:         d.WiredInterface,
		TenantID:               d.TenantID,
		Version:                d.Version,
	}

	return d1
}
