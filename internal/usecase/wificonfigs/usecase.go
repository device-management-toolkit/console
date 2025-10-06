package wificonfigs

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/security"

	"github.com/device-management-toolkit/console/internal/entity"
	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/usecase/ieee8021xconfigs"
	"github.com/device-management-toolkit/console/internal/usecase/sqldb"
	"github.com/device-management-toolkit/console/pkg/consoleerrors"
	"github.com/device-management-toolkit/console/pkg/logger"
)

// UseCase -.
type UseCase struct {
	repo             Repository
	ieee             ieee8021xconfigs.Feature
	log              logger.Interface
	safeRequirements security.Cryptor
}

// New -.
func New(r Repository, ieee ieee8021xconfigs.Feature, log logger.Interface, safeRequirements security.Cryptor) *UseCase {
	return &UseCase{
		repo:             r,
		ieee:             ieee,
		log:              log,
		safeRequirements: safeRequirements,
	}
}

var (
	ErrCountNotUnique = sqldb.NotUniqueError{Console: consoleerrors.CreateConsoleError("WifiConfigs")}
	ErrDomainsUseCase = consoleerrors.CreateConsoleError("WificonfigsUseCase")
	ErrDatabase       = sqldb.DatabaseError{Console: consoleerrors.CreateConsoleError("WificonfigsUseCase")}
	ErrNotFound       = sqldb.NotFoundError{Console: consoleerrors.CreateConsoleError("WificonfigsUseCase")}
)

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

func (uc *UseCase) Get(ctx context.Context, top, skip int, tenantID string) ([]dto.WirelessConfig, error) {
	data, err := uc.repo.Get(ctx, top, skip, tenantID)
	if err != nil {
		return nil, ErrDatabase.Wrap("Get", "uc.repo.Get", err)
	}

	// iterate over the data and convert each entity to dto
	d1 := make([]dto.WirelessConfig, len(data))

	for i := range data {
		tmpEntity := data[i] // create a new variable to avoid memory aliasing
		d1[i] = *uc.entityToDTO(&tmpEntity)
	}

	return d1, nil
}

func (uc *UseCase) GetByName(ctx context.Context, profileName, tenantID string) (*dto.WirelessConfig, error) {
	data, err := uc.repo.GetByName(ctx, profileName, tenantID)
	if err != nil {
		return nil, ErrDatabase.Wrap("GetByName", "uc.repo.GetByName", err)
	}

	if data == nil {
		return nil, ErrNotFound
	}

	d1 := uc.entityToDTO(data)

	return d1, nil
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

func (uc *UseCase) Update(ctx context.Context, d *dto.WirelessConfig) (*dto.WirelessConfig, error) {
	d1 := uc.dtoToEntity(d)

	// check if the IEEE profile is exists in the database
	if d1.IEEE8021xProfileName != nil && *d1.IEEE8021xProfileName != "" {
		_, err := uc.ieee.GetByName(ctx, *d1.IEEE8021xProfileName, d.TenantID)
		if err != nil {
			return nil, err
		}
	}

	updated, err := uc.repo.Update(ctx, d1)
	if err != nil {
		return nil, ErrDatabase.Wrap("Update", "uc.repo.Update", err)
	}

	if !updated {
		return nil, ErrNotFound
	}

	updatedConfig, err := uc.repo.GetByName(ctx, d1.ProfileName, d.TenantID)
	if err != nil {
		return nil, err
	}

	d2 := uc.entityToDTO(updatedConfig)

	return d2, nil
}

func (uc *UseCase) Insert(ctx context.Context, d *dto.WirelessConfig) (*dto.WirelessConfig, error) {
	d1 := uc.dtoToEntity(d)

	// check if the IEEE profile is exists in the database
	if d1.IEEE8021xProfileName != nil && *d1.IEEE8021xProfileName != "" {
		_, err := uc.ieee.GetByName(ctx, *d1.IEEE8021xProfileName, d.TenantID)
		if err != nil {
			return nil, err
		}
	}

	_, err := uc.repo.Insert(ctx, d1)
	if err != nil {
		return nil, ErrDatabase.Wrap("Insert", "uc.repo.Insert", err)
	}

	insertedConfig, err := uc.repo.GetByName(ctx, d.ProfileName, d.TenantID)
	if err != nil {
		return nil, err
	}

	d2 := uc.entityToDTO(insertedConfig)

	return d2, nil
}

// convert dto.WirelessConfig to entity.WirelessConfig.
func (uc *UseCase) dtoToEntity(d *dto.WirelessConfig) *entity.WirelessConfig {
	// convert []int to comma separated string
	linkPolicy := strings.Trim(strings.Join(strings.Fields(fmt.Sprint(d.LinkPolicy)), ","), "[]")

	d1 := &entity.WirelessConfig{
		ProfileName:          d.ProfileName,
		AuthenticationMethod: d.AuthenticationMethod,
		EncryptionMethod:     d.EncryptionMethod,
		SSID:                 d.SSID,
		PSKValue:             d.PSKValue,
		PSKPassphrase:        d.PSKPassphrase,
		LinkPolicy:           &linkPolicy,
		TenantID:             d.TenantID,
		IEEE8021xProfileName: d.IEEE8021xProfileName,
		Version:              d.Version,
	}

	d1.PSKPassphrase, _ = uc.safeRequirements.Encrypt(d.PSKPassphrase)

	return d1
}

// convert entity.WirelessConfig to dto.WirelessConfig.
func (uc *UseCase) entityToDTO(d *entity.WirelessConfig) *dto.WirelessConfig {
	// convert comma separated string to []int
	linkPolicyInt := []int{}

	if d.LinkPolicy != nil && *d.LinkPolicy != "" {
		linkPolicy := strings.Split(*d.LinkPolicy, ",")

		// convert []string to []int
		linkPolicyInt = make([]int, len(linkPolicy))

		for i, v := range linkPolicy {
			val, err := strconv.Atoi(v)
			if err != nil {
				// handle the error, e.g. log or return an error
				uc.log.Error("error converting string to int")
			}

			linkPolicyInt[i] = val
		}
	}

	d1 := &dto.WirelessConfig{
		ProfileName:          d.ProfileName,
		AuthenticationMethod: d.AuthenticationMethod,
		EncryptionMethod:     d.EncryptionMethod,
		SSID:                 d.SSID,
		PSKValue:             d.PSKValue,
		LinkPolicy:           linkPolicyInt,
		TenantID:             d.TenantID,
		IEEE8021xProfileName: d.IEEE8021xProfileName,
		Version:              d.Version,
	}

	if d.IEEE8021xProfileName != nil && *d.IEEE8021xProfileName != "" {
		val := &dto.IEEE8021xConfig{
			AuthenticationProtocol: *d.AuthenticationProtocol,
			PXETimeout:             d.PXETimeout,
			WiredInterface:         *d.WiredInterface,
		}
		d1.IEEE8021xProfileObject = val
	}

	return d1
}
