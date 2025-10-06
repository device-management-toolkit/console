package domains

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"time"

	"software.sslmate.com/src/go-pkcs12"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/security"

	"github.com/device-management-toolkit/console/internal/entity"
	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/usecase/sqldb"
	"github.com/device-management-toolkit/console/pkg/consoleerrors"
	"github.com/device-management-toolkit/console/pkg/logger"
)

// UseCase -.
type UseCase struct {
	repo             Repository
	log              logger.Interface
	safeRequirements security.Cryptor
}

// New -.
func New(r Repository, log logger.Interface, safeRequirements security.Cryptor) *UseCase {
	return &UseCase{
		repo:             r,
		log:              log,
		safeRequirements: safeRequirements,
	}
}

var (
	ErrDomainsUseCase = consoleerrors.CreateConsoleError("DomainsUseCase")
	ErrDatabase       = sqldb.DatabaseError{Console: ErrDomainsUseCase}
	ErrNotFound       = sqldb.NotFoundError{Console: ErrDomainsUseCase}
	ErrCertPassword   = CertPasswordError{Console: ErrDomainsUseCase}
	ErrCertExpiration = CertExpirationError{Console: ErrDomainsUseCase}
)

// History - getting translate history from store.
func (uc *UseCase) GetCount(ctx context.Context, tenantID string) (int, error) {
	count, err := uc.repo.GetCount(ctx, tenantID)
	if err != nil {
		return 0, ErrDatabase.Wrap("Get", "uc.repo.GetCount", err)
	}

	return count, nil
}

func (uc *UseCase) Get(ctx context.Context, top, skip int, tenantID string) ([]dto.Domain, error) {
	data, err := uc.repo.Get(ctx, top, skip, tenantID)
	if err != nil {
		return nil, ErrDatabase.Wrap("Get", "uc.repo.Get", err)
	}

	// iterate over the data and convert each entity to dto
	d1 := make([]dto.Domain, len(data))

	for i := range data {
		tmpEntity := data[i] // create a new variable to avoid memory aliasing
		d1[i] = *uc.entityToDTO(&tmpEntity)
	}

	return d1, nil
}

func (uc *UseCase) GetDomainByDomainSuffix(ctx context.Context, domainSuffix, tenantID string) (*dto.Domain, error) {
	data, err := uc.repo.GetDomainByDomainSuffix(ctx, domainSuffix, tenantID)
	if err != nil {
		return nil, ErrDatabase.Wrap("GetDomainByDomainSuffix", "uc.repo.GetDomainByDomainSuffix", err)
	}

	if data == nil {
		return nil, ErrNotFound
	}

	d2 := uc.entityToDTO(data)

	return d2, nil
}

func (uc *UseCase) GetByName(ctx context.Context, domainName, tenantID string) (*dto.Domain, error) {
	data, err := uc.repo.GetByName(ctx, domainName, tenantID)
	if err != nil {
		return nil, ErrDatabase.Wrap("GetByName", "uc.repo.GetByName", err)
	}

	if data == nil {
		return nil, ErrNotFound
	}

	d2 := uc.entityToDTO(data)

	return d2, nil
}

func (uc *UseCase) Delete(ctx context.Context, domainName, tenantID string) error {
	isSuccessful, err := uc.repo.Delete(ctx, domainName, tenantID)
	if err != nil {
		return ErrDatabase.Wrap("Delete", "uc.repo.Delete", err)
	}

	if !isSuccessful {
		return ErrNotFound
	}

	return nil
}

func (uc *UseCase) Update(ctx context.Context, d *dto.Domain) (*dto.Domain, error) {
	d1 := uc.dtoToEntity(d)

	updated, err := uc.repo.Update(ctx, d1)
	if err != nil {
		return nil, ErrDatabase.Wrap("Update", "uc.repo.Update", err)
	}

	if !updated {
		return nil, ErrNotFound
	}

	updateDomain, err := uc.repo.GetByName(ctx, d.ProfileName, d.TenantID)
	if err != nil {
		return nil, err
	}

	d2 := uc.entityToDTO(updateDomain)

	return d2, nil
}

func (uc *UseCase) Insert(ctx context.Context, d *dto.Domain) (*dto.Domain, error) {
	cert, err := DecryptAndCheckCertExpiration(*d)
	if err != nil {
		return nil, err
	}

	d1 := uc.dtoToEntity(d)

	d1.ExpirationDate = cert.NotAfter.Format(time.RFC3339)

	_, err = uc.repo.Insert(ctx, d1)
	if err != nil {
		return nil, ErrDatabase.Wrap("Insert", "uc.repo.Insert", err)
	}

	newDomain, err := uc.repo.GetByName(ctx, d.ProfileName, d.TenantID)
	if err != nil {
		return nil, err
	}

	d2 := uc.entityToDTO(newDomain)

	return d2, nil
}

func DecryptAndCheckCertExpiration(domain dto.Domain) (*x509.Certificate, error) {
	// Decode the base64 encoded PFX certificate
	pfxData, err := base64.StdEncoding.DecodeString(domain.ProvisioningCert)
	if err != nil {
		return nil, err
	}

	// Convert the PFX data to x509 cert
	_, cert, err := pkcs12.Decode(pfxData, domain.ProvisioningCertPassword)
	if err != nil && cert == nil {
		return nil, ErrCertPassword.Wrap("DecryptAndCheckCertExpiration", "pkcs12.Decode", err)
	}

	// Check the expiration date of the certificate
	if cert.NotAfter.Before(time.Now()) {
		return nil, ErrCertExpiration.Wrap("DecryptAndCheckCertExpiration", "x509Cert.NotAfter.Before", nil)
	}

	return cert, nil
}

// convert dto.Domain to entity.Domain.
func (uc *UseCase) dtoToEntity(d *dto.Domain) *entity.Domain {
	d1 := &entity.Domain{
		ProfileName:                   d.ProfileName,
		DomainSuffix:                  d.DomainSuffix,
		ProvisioningCert:              d.ProvisioningCert,
		ProvisioningCertPassword:      d.ProvisioningCertPassword,
		ProvisioningCertStorageFormat: d.ProvisioningCertStorageFormat,
		TenantID:                      d.TenantID,
		Version:                       d.Version,
	}

	d1.ProvisioningCertPassword, _ = uc.safeRequirements.Encrypt(d.ProvisioningCertPassword)

	return d1
}

// convert entity.Domain to dto.Domain.
func (uc *UseCase) entityToDTO(d *entity.Domain) *dto.Domain {
	// parse expiration date
	var expirationDate time.Time

	var err error

	if d.ExpirationDate != "" {
		expirationDate, err = time.Parse(time.RFC3339, d.ExpirationDate)
		if err != nil {
			uc.log.Warn("failed to parse expiration date")
		}
	}

	d1 := &dto.Domain{
		ProfileName:  d.ProfileName,
		DomainSuffix: d.DomainSuffix,
		// ProvisioningCert:              d.ProvisioningCert,
		// ProvisioningCertPassword:      d.ProvisioningCertPassword,
		ProvisioningCertStorageFormat: d.ProvisioningCertStorageFormat,
		ExpirationDate:                expirationDate,
		TenantID:                      d.TenantID,
		Version:                       d.Version,
	}

	return d1
}
