package devices

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/usecase/sqldb"
	"github.com/device-management-toolkit/console/pkg/consoleerrors"
)

var (
	ErrDeviceUseCase = consoleerrors.CreateConsoleError("DevicesUseCase")
	ErrDatabase      = sqldb.DatabaseError{Console: consoleerrors.CreateConsoleError("DevicesUseCase")}
	ErrNotFound      = sqldb.NotFoundError{Console: consoleerrors.CreateConsoleError("DevicesUseCase")}
)

// History - getting translate history from store.
func (uc *UseCase) GetCount(ctx context.Context, tenantID string) (int, error) {
	count, err := uc.repo.GetCount(ctx, tenantID)
	if err != nil {
		return 0, ErrDatabase.Wrap("Count", "uc.repo.GetCount", err)
	}

	return count, nil
}

func (uc *UseCase) Get(ctx context.Context, top, skip int, tenantID string) ([]dto.Device, error) {
	data, err := uc.repo.Get(ctx, top, skip, tenantID)
	if err != nil {
		return nil, ErrDatabase.Wrap("Get", "uc.repo.Get", err)
	}

	// iterate over the data and convert each entity to dto
	d1 := make([]dto.Device, len(data))

	for i := range data {
		tmpEntity := data[i] // create a new variable to avoid memory aliasing
		d1[i] = *uc.entityToDTO(&tmpEntity)
	}

	return d1, nil
}

func (uc *UseCase) GetByColumn(ctx context.Context, columnName, queryValue, tenantID string) ([]dto.Device, error) {
	data, err := uc.repo.GetByColumn(ctx, columnName, queryValue, tenantID)
	if err != nil {
		return nil, ErrDatabase.Wrap("GetByColumn", "uc.repo.GetByColumn", err)
	}

	// iterate over the data and convert each entity to dto
	d1 := make([]dto.Device, len(data))

	for i := range data {
		tmpEntity := data[i] // create a new variable to avoid memory aliasing
		d1[i] = *uc.entityToDTO(&tmpEntity)
	}

	return d1, nil
}

func (uc *UseCase) GetByID(ctx context.Context, guid, tenantID string, includeSecrets bool) (*dto.Device, error) {
	data, err := uc.repo.GetByID(ctx, guid, tenantID)
	if err != nil {
		return nil, ErrDatabase.Wrap("GetByID", "uc.repo.GetByID", err)
	}

	if data == nil || data.GUID == "" {
		return nil, ErrNotFound
	}

	d2 := uc.entityToDTO(data)
	if includeSecrets {
		d2.Password, err = uc.safeRequirements.Decrypt(data.Password)
		if err != nil {
			return nil, ErrDeviceUseCase.Wrap("GetByID", "uc.safeRequirements.Decrypt", err)
		}
	}

	return d2, nil
}

func (uc *UseCase) GetDistinctTags(ctx context.Context, tenantID string) ([]string, error) {
	data, err := uc.repo.GetDistinctTags(ctx, tenantID)
	if err != nil {
		return nil, ErrDatabase.Wrap("GetDistinctTags", "uc.repo.GetDistinctTags", err)
	}

	allTags := make([]string, 0)

	for _, v := range data {
		tags := strings.Split(v, ",")

		allTags = append(allTags, tags...)
	}

	return allTags, nil
}

func (uc *UseCase) GetByTags(ctx context.Context, tags, method string, limit, offset int, tenantID string) ([]dto.Device, error) {
	splitTags := strings.Split(tags, ",")

	data, err := uc.repo.GetByTags(ctx, splitTags, method, limit, offset, tenantID)
	if err != nil {
		return nil, fmt.Errorf("DevicesUseCase - GetByTags - uc.repo.GetByTags: %w", err)
	}

	// iterate over the data and convert each entity to dto
	d1 := make([]dto.Device, len(data))

	for i := range data {
		tmpEntity := data[i] // create a new variable to avoid memory aliasing
		d1[i] = *uc.entityToDTO(&tmpEntity)
	}

	return d1, nil
}

func (uc *UseCase) Delete(ctx context.Context, guid, tenantID string) error {
	isSuccessful, err := uc.repo.Delete(ctx, guid, tenantID)
	if err != nil {
		return ErrDatabase.Wrap("Delete", "uc.repo.Delete", err)
	}

	if !isSuccessful {
		return ErrNotFound
	}

	return nil
}

func (uc *UseCase) Update(ctx context.Context, d *dto.Device) (*dto.Device, error) {
	d1 := uc.dtoToEntity(d)

	updated, err := uc.repo.Update(ctx, d1)
	if err != nil {
		return nil, ErrDatabase.Wrap("Update", "uc.repo.Update", err)
	}

	if !updated {
		return nil, ErrNotFound.Wrap("Update", "uc.repo.Update", nil)
	}

	updateDevice, err := uc.repo.GetByID(ctx, d1.GUID, d1.TenantID)
	if err != nil {
		return nil, err
	}

	d2 := uc.entityToDTO(updateDevice)

	// invalidate connection cache
	uc.device.DestroyWsmanClient(*d2)

	return d2, nil
}

func (uc *UseCase) Insert(ctx context.Context, d *dto.Device) (*dto.Device, error) {
	d1 := uc.dtoToEntity(d)

	if d1.GUID == "" {
		d1.GUID = uuid.New().String()
	}

	_, err := uc.repo.Insert(ctx, d1)
	if err != nil {
		return nil, ErrDatabase.Wrap("Insert", "uc.repo.Insert", err)
	}

	newDevice, err := uc.repo.GetByID(ctx, d1.GUID, d1.TenantID)
	if err != nil {
		return nil, err
	}

	d2 := uc.entityToDTO(newDevice)
	if newDevice.Tags == "" {
		d2.Tags = []string{}
	}

	return d2, nil
}
