package domains

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"software.sslmate.com/src/go-pkcs12"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/security"

	"github.com/device-management-toolkit/console/internal/entity"
	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/repoerrors"
	"github.com/device-management-toolkit/console/pkg/consoleerrors"
	"github.com/device-management-toolkit/console/pkg/logger"
	"github.com/device-management-toolkit/console/pkg/principal"
)

// Object storage field name constants for domain certificates.
const (
	certStoreFieldCert = "cert"
)

// ObjectStorager extends security.Storager with object storage capabilities.
type ObjectStorager interface {
	security.Storager
	GetObject(key string) (map[string]string, error)
	SetObject(key string, data map[string]string) error
}

// UseCase -.
type UseCase struct {
	repo             Repository
	log              logger.Interface
	safeRequirements security.Cryptor
	certStore        security.Storager
}

// New -.
func New(r Repository, log logger.Interface, safeRequirements security.Cryptor, certStore security.Storager) *UseCase {
	return &UseCase{
		repo:             r,
		log:              log,
		safeRequirements: safeRequirements,
		certStore:        certStore,
	}
}

var (
	ErrDomainsUseCase   = consoleerrors.CreateConsoleError("DomainsUseCase")
	ErrDatabase         = repoerrors.DatabaseError{Console: ErrDomainsUseCase}
	ErrNotFound         = repoerrors.NotFoundError{Console: ErrDomainsUseCase}
	ErrCertPassword     = CertPasswordError{Console: ErrDomainsUseCase}
	ErrCertExpiration   = CertExpirationError{Console: ErrDomainsUseCase}
	ErrCertDomainSuffix = CertDomainSuffixError{Console: ErrDomainsUseCase}
	ErrCertStore        = CertStoreError{Console: ErrDomainsUseCase}
)

// domainCertKey generates the key path for storing domain certificates in Vault.
// Format: certs/domains/{tenantID}/{profileName}.
func domainCertKey(tenantID, profileName string) string {
	return fmt.Sprintf("certs/domains/%s/%s", tenantID, profileName)
}

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

// GetByNameWithCert retrieves a domain and its certificate from Vault.
// This should be used when the certificate data is needed (e.g., for provisioning).
func (uc *UseCase) GetByNameWithCert(ctx context.Context, domainName, tenantID string) (*entity.Domain, error) {
	data, err := uc.repo.GetByName(ctx, domainName, tenantID)
	if err != nil {
		return nil, ErrDatabase.Wrap("GetByNameWithCert", "uc.repo.GetByName", err)
	}

	if data == nil {
		return nil, ErrNotFound
	}

	// If cert store is available and cert is not in DB, fetch from Vault
	if uc.certStore != nil && data.ProvisioningCert == "" {
		certKey := domainCertKey(tenantID, domainName)

		// Use object storage if available
		if objStore, ok := uc.certStore.(ObjectStorager); ok {
			certData, err := objStore.GetObject(certKey)
			if err != nil {
				uc.log.Warn("Failed to retrieve domain certificate from Vault: %v", err)
				// Continue without cert - it may be a legacy domain stored in DB
			} else {
				data.ProvisioningCert = certData[certStoreFieldCert]
				data.ProvisioningCertPassword = certData["password"]
			}
		}
	}

	return data, nil
}

func (uc *UseCase) Delete(ctx context.Context, domainName, tenantID string) error {
	isSuccessful, err := uc.repo.Delete(ctx, domainName, tenantID)
	if err != nil {
		return ErrDatabase.Wrap("Delete", "uc.repo.Delete", err)
	}

	if !isSuccessful {
		return ErrNotFound
	}

	// Delete certificate from Vault if available
	if uc.certStore != nil {
		certKey := domainCertKey(tenantID, domainName)
		if err := uc.certStore.DeleteKeyValue(certKey); err != nil {
			// Log but don't fail - the DB record is already deleted
			uc.log.Warn("Failed to delete domain certificate from Vault: %v", err)
		} else {
			uc.log.Info("Domain certificate deleted from Vault: %s", certKey)
		}
	}

	return nil
}

func (uc *UseCase) Update(ctx context.Context, d *dto.Domain) (*dto.Domain, error) {
	oldDomain, err := uc.GetByNameWithCert(ctx, d.ProfileName, d.TenantID)
	if err != nil {
		return nil, err
	}

	d1, err := uc.dtoToEntity(d)
	if err != nil {
		return nil, err
	}

	// Capture the encrypted password generated by dtoToEntity
	encryptedNewPassword := d1.ProvisioningCertPassword

	d1.ExpirationDate = oldDomain.ExpirationDate
	d1.ProvisioningCert = oldDomain.ProvisioningCert
	d1.ProvisioningCertPassword = oldDomain.ProvisioningCertPassword

	if d.ProvisioningCert != "" {
		if err := uc.applyCertUpdate(d, d1, encryptedNewPassword); err != nil {
			return nil, err
		}
	} else if uc.certStore != nil {
		// Only clear DB cert fields when using object storage and the existing
		// record is already Vault-backed (empty cert in DB). Legacy records that
		// still carry their cert in the DB must not be wiped.
		if _, ok := uc.certStore.(ObjectStorager); ok && oldDomain.ProvisioningCert == "" {
			d1.ProvisioningCert = ""
			d1.ProvisioningCertPassword = ""
		}
	}

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

	if err := CheckCertDomainSuffix(cert, d.DomainSuffix); err != nil {
		return nil, err
	}

	d1, err := uc.dtoToEntity(d)
	if err != nil {
		return nil, err
	}

	d1.ExpirationDate = cert.NotAfter.Format(time.RFC3339)
	d1.CreatedBy = principal.User(ctx)

	// Store certificate in Vault (if available) - cert goes to Vault, not DB
	if uc.certStore != nil {
		certKey := domainCertKey(d.TenantID, d.ProfileName)

		// Use object storage if available
		if objStore, ok := uc.certStore.(ObjectStorager); ok {
			err = objStore.SetObject(certKey, map[string]string{
				certStoreFieldCert: d.ProvisioningCert,
				"password":         d.ProvisioningCertPassword,
			})
			if err != nil {
				return nil, ErrCertStore.Wrap("Insert", "objStore.SetObject", err)
			}

			// Clear cert data from entity - don't store in DB when using Vault
			d1.ProvisioningCert = ""
			d1.ProvisioningCertPassword = ""

			uc.log.Info("Domain certificate stored in Vault: %s", certKey)
		}
	}

	_, err = uc.repo.Insert(ctx, d1)
	if err != nil {
		// If DB insert fails and we stored in Vault, try to clean up
		if uc.certStore != nil {
			certKey := domainCertKey(d.TenantID, d.ProfileName)
			_ = uc.certStore.DeleteKeyValue(certKey)
		}

		return nil, ErrDatabase.Wrap("Insert", "uc.repo.Insert", err)
	}

	newDomain, err := uc.repo.GetByName(ctx, d.ProfileName, d.TenantID)
	if err != nil {
		return nil, err
	}

	d2 := uc.entityToDTO(newDomain)

	return d2, nil
}

// applyCertUpdate validates the replacement certificate against the request,
// copies it (and its freshly encrypted password) onto the entity, and stores it
// in Vault when object storage is configured.
func (uc *UseCase) applyCertUpdate(d *dto.Domain, d1 *entity.Domain, encryptedPassword string) error {
	cert, err := DecryptAndCheckCertExpiration(*d)
	if err != nil {
		return err
	}

	if err := CheckCertDomainSuffix(cert, d.DomainSuffix); err != nil {
		return err
	}

	d1.ExpirationDate = cert.NotAfter.Format(time.RFC3339)
	d1.ProvisioningCert = d.ProvisioningCert
	d1.ProvisioningCertPassword = encryptedPassword

	return uc.storeCertInVault("Update", d, d1)
}

// storeCertInVault writes the domain certificate to Vault object storage and clears
// the cert fields on d1 so the certificate is not duplicated in the database.
// It is a no-op when no cert store or no ObjectStorager is configured.
func (uc *UseCase) storeCertInVault(operation string, d *dto.Domain, d1 *entity.Domain) error {
	if uc.certStore == nil {
		return nil
	}

	objStore, ok := uc.certStore.(ObjectStorager)
	if !ok {
		return nil
	}

	certKey := domainCertKey(d.TenantID, d.ProfileName)

	err := objStore.SetObject(certKey, map[string]string{
		certStoreFieldCert: d.ProvisioningCert,
		"password":         d.ProvisioningCertPassword,
	})
	if err != nil {
		return ErrCertStore.Wrap(operation, "objStore.SetObject", err)
	}

	d1.ProvisioningCert = ""
	d1.ProvisioningCertPassword = ""

	uc.log.Info("Domain certificate stored in Vault: %s", certKey)

	return nil
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

// CheckCertDomainSuffix verifies that domainSuffix is a DNS suffix the
// provisioning certificate is issued for, following Intel's remote
// configuration rules (Remote Configuration Certificate Selection white paper):
//
//   - a standard certificate covers exactly one suffix: its Common Name with
//     the host label removed (CN "intel.vprodemo.com" covers "vprodemo.com").
//     The CN itself is also accepted so certificates issued directly for the
//     suffix ("vprodemo.com") work. Sibling and child domains are rejected.
//   - a wildcard certificate "*.base" covers base, every domain under it and,
//     per Intel's figure 11, its parent (overlapping labels match).
//
// AMT refuses to provision when the suffix and certificate disagree, so the
// mismatch is caught here at domain creation time.
func CheckCertDomainSuffix(cert *x509.Certificate, domainSuffix string) error {
	if cert == nil || cert.Subject.CommonName == "" {
		return ErrCertDomainSuffix.Wrap("CheckCertDomainSuffix", "cert.Subject.CommonName", nil)
	}

	cn := normalizeDNSName(cert.Subject.CommonName)
	suffix := normalizeDNSName(domainSuffix)

	if suffix == "" || !certCoversSuffix(cn, suffix) {
		return ErrCertDomainSuffix.Wrap("CheckCertDomainSuffix", "certCoversSuffix", nil)
	}

	return nil
}

// certCoversSuffix reports whether a certificate with Common Name cn covers the
// DNS suffix. Both arguments must already be normalized.
func certCoversSuffix(cn, suffix string) bool {
	if base, isWildcard := strings.CutPrefix(cn, "*."); isWildcard {
		// "*.com" must not cover the whole TLD.
		if !strings.Contains(base, ".") {
			return false
		}

		return suffix == base ||
			strings.HasSuffix(suffix, "."+base) ||
			(strings.Contains(suffix, ".") && strings.HasSuffix(base, "."+suffix))
	}

	if suffix == cn {
		return true
	}

	_, domain, hasHost := strings.Cut(cn, ".")

	// The remainder must still be a domain ("vprodemo.com"), never a bare TLD.
	return hasHost && strings.Contains(domain, ".") && suffix == domain
}

// normalizeDNSName lower-cases a DNS name and strips surrounding whitespace and
// a trailing root dot so comparisons are purely structural.
func normalizeDNSName(name string) string {
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(name)), ".")
}

// convert dto.Domain to entity.Domain.
func (uc *UseCase) dtoToEntity(d *dto.Domain) (*entity.Domain, error) {
	d1 := &entity.Domain{
		ProfileName:                   d.ProfileName,
		DomainSuffix:                  d.DomainSuffix,
		ProvisioningCert:              d.ProvisioningCert,
		ProvisioningCertPassword:      d.ProvisioningCertPassword,
		ProvisioningCertStorageFormat: d.ProvisioningCertStorageFormat,
		TenantID:                      d.TenantID,
		Version:                       d.Version,
	}

	var err error

	d1.ProvisioningCertPassword, err = uc.safeRequirements.Encrypt(d.ProvisioningCertPassword)
	if err != nil {
		return nil, ErrDomainsUseCase.Wrap("dtoToEntity", "failed to encrypt provisioning cert password", err)
	}

	return d1, nil
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
