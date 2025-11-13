package sqldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/device-management-toolkit/console/internal/entity"
	"github.com/device-management-toolkit/console/pkg/consoleerrors"
	"github.com/device-management-toolkit/console/pkg/db"
	"github.com/device-management-toolkit/console/pkg/logger"
)

// DomainRepo -.
type DomainRepo struct {
	*db.SQL
	log logger.Interface
}

var (
	ErrDomainDatabase  = DatabaseError{Console: consoleerrors.CreateConsoleError("DomainRepo")}
	ErrDomainNotUnique = NotUniqueError{Console: consoleerrors.CreateConsoleError("DomainRepo")}
)

// New -.
func NewDomainRepo(database *db.SQL, log logger.Interface) *DomainRepo {
	return &DomainRepo{database, log}
}

// GetCount -.
func (r *DomainRepo) GetCount(_ context.Context, tenantID string) (int, error) {
	sqlQuery, _, err := r.Builder.
		Select("COUNT(*) OVER() AS total_count").
		From("domains").
		Where("tenant_id = ?", tenantID).
		ToSql()
	if err != nil {
		return 0, ErrDomainDatabase.Wrap("GetCount", "r.Builder: ", err)
	}

	var count int

	err = r.Pool.QueryRowContext(context.Background(), sqlQuery, tenantID).Scan(&count)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}

		return 0, ErrDomainDatabase.Wrap("GetCount", "r.Pool.QueryRow", err)
	}

	return count, nil
}

// Get -.
func (r *DomainRepo) Get(_ context.Context, top, skip int, tenantID string) ([]entity.Domain, error) {
	const defaultTop = 100

	if top == 0 {
		top = defaultTop
	}

	limitedTop := uint64(defaultTop)
	if top > 0 {
		limitedTop = uint64(top)
	}

	limitedSkip := uint64(0)
	if skip > 0 {
		limitedSkip = uint64(skip)
	}

	sqlQuery, _, err := r.Builder.
		Select("name",
			"domain_suffix",
			"provisioning_cert",
			"provisioning_cert_storage_format",
			"provisioning_cert_key",
			"expiration_date",
			"tenant_id").
		From("domains").
		Where("tenant_id = ?", tenantID).
		OrderBy("name").
		Limit(limitedTop).
		Offset(limitedSkip).
		ToSql()
	if err != nil {
		return nil, ErrDomainDatabase.Wrap("Get", "r.Builder: ", err)
	}

	rows, err := r.Pool.QueryContext(context.Background(), sqlQuery, tenantID)
	if err != nil {
		return nil, ErrDomainDatabase.Wrap("Get", "r.Pool.Query", err)
	}

	defer rows.Close()

	if rows.Err() != nil {
		return nil, ErrDeviceDatabase.Wrap("Get", "rows.Err", rows.Err())
	}

	domains := make([]entity.Domain, 0)

	for rows.Next() {
		d := entity.Domain{}

		err = rows.Scan(&d.ProfileName, &d.DomainSuffix, &d.ProvisioningCert, &d.ProvisioningCertStorageFormat, &d.ProvisioningCertPassword, &d.ExpirationDate, &d.TenantID)
		if err != nil {
			return nil, ErrDomainDatabase.Wrap("Get", "rows.Scan: ", err)
		}

		domains = append(domains, d)
	}

	return domains, nil
}

// GetDomainByDomainSuffix -.
func (r *DomainRepo) GetDomainByDomainSuffix(_ context.Context, domainSuffix, tenantID string) (*entity.Domain, error) {
	sqlQuery, _, err := r.Builder.
		Select("name",
			"domain_suffix",
			"provisioning_cert",
			"provisioning_cert_storage_format",
			"provisioning_cert_key",
			"expiration_date",
			"tenant_id",
		).
		From("domains").
		Where("domain_suffix = ? AND tenant_id = ?", domainSuffix, tenantID).
		ToSql()
	if err != nil {
		return nil, ErrDomainDatabase.Wrap("GetDomainByDomainSuffix", "r.Builder: ", err)
	}

	row := r.Pool.QueryRowContext(context.Background(), sqlQuery)

	d := entity.Domain{}

	err = row.Scan(&d.ProfileName, &d.DomainSuffix, &d.ProvisioningCert, &d.ProvisioningCertStorageFormat, &d.ProvisioningCertPassword, &d.ExpirationDate, &d.TenantID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, ErrDomainDatabase.Wrap("GetDomainByDomainSuffix", "row.Scan: ", err)
	}

	return &d, nil
}

// GetByName -.
func (r *DomainRepo) GetByName(_ context.Context, domainName, tenantID string) (*entity.Domain, error) {
	sqlQuery, args, err := r.Builder.
		Select(
			"name",
			"domain_suffix",
			"provisioning_cert",
			"provisioning_cert_storage_format",
			"provisioning_cert_key",
			"expiration_date",
			"tenant_id",
		).
		From("domains").
		Where("LOWER(name) = LOWER(?) AND tenant_id = ?", domainName, tenantID).
		ToSql()
	if err != nil {
		return nil, ErrDomainDatabase.Wrap("GetByName", "r.Builder: ", err)
	}

	row := r.Pool.QueryRowContext(context.Background(), sqlQuery, args...)

	d := entity.Domain{}

	err = row.Scan(&d.ProfileName, &d.DomainSuffix, &d.ProvisioningCert, &d.ProvisioningCertStorageFormat, &d.ProvisioningCertPassword, &d.ExpirationDate, &d.TenantID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, ErrDomainDatabase.Wrap("GetByName", "row.Scan: ", err)
	}

	return &d, nil
}

// Delete -.
func (r *DomainRepo) Delete(_ context.Context, domainName, tenantID string) (bool, error) {
	sqlQuery, args, err := r.Builder.
		Delete("domains").
		Where("LOWER(name) = LOWER(?) AND tenant_id = ?", domainName, tenantID).
		ToSql()
	if err != nil {
		return false, ErrDomainDatabase.Wrap("Delete", "r.Builder: ", err)
	}

	res, err := r.Pool.ExecContext(context.Background(), sqlQuery, args...)
	if err != nil {
		return false, ErrDomainDatabase.Wrap("Delete", "r.Pool.Exec", err)
	}

	result, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("DomainRepo - Delete - r.Pool.Exec: %w", err)
	}

	return result > 0, nil
}

// Update -.
func (r *DomainRepo) Update(_ context.Context, d *entity.Domain) (bool, error) {
	sqlQuery, args, err := r.Builder.
		Update("domains").
		Set("name", d.ProfileName).
		Set("domain_suffix", d.DomainSuffix).
		Set("provisioning_cert", d.ProvisioningCert).
		Set("provisioning_cert_storage_format", d.ProvisioningCertStorageFormat).
		Set("provisioning_cert_key", d.ProvisioningCertPassword).
		Set("expiration_date", d.ExpirationDate).
		Where("name = ? AND tenant_id = ?", d.ProfileName, d.TenantID).
		ToSql()
	if err != nil {
		return false, ErrDomainDatabase.Wrap("Update", "r.Builder: ", err)
	}

	res, err := r.Pool.ExecContext(context.Background(), sqlQuery, args...)
	if err != nil {
		if db.CheckNotUnique(err) {
			return false, ErrProfileNotUnique.Wrap(err.Error())
		}

		return false, ErrDomainDatabase.Wrap("Update", "r.Pool.Exec", err)
	}

	result, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("DomainRepo - Update - r.Pool.Exec: %w", err)
	}

	return result > 0, nil
}

// Insert -.
func (r *DomainRepo) Insert(_ context.Context, d *entity.Domain) (string, error) {
	insertBuilder := r.Builder.
		Insert("domains").
		Columns("name", "domain_suffix", "provisioning_cert", "provisioning_cert_storage_format", "provisioning_cert_key", "expiration_date", "tenant_id").
		Values(d.ProfileName, d.DomainSuffix, d.ProvisioningCert, d.ProvisioningCertStorageFormat, d.ProvisioningCertPassword, d.ExpirationDate, d.TenantID)

	if !r.IsEmbedded {
		insertBuilder = insertBuilder.Suffix("RETURNING xmin::text")
	}

	sqlQuery, args, err := insertBuilder.ToSql()
	if err != nil {
		return "", ErrDomainDatabase.Wrap("Insert", "r.Builder: ", err)
	}

	version := ""

	if r.IsEmbedded {
		_, err = r.Pool.ExecContext(context.Background(), sqlQuery, args...)
	} else {
		err = r.Pool.QueryRowContext(context.Background(), sqlQuery, args...).Scan(&version)
	}

	if err != nil {
		if db.CheckNotUnique(err) {
			return "", ErrProfileNotUnique.Wrap(err.Error())
		}

		return "", ErrDomainDatabase.Wrap("Insert", "r.Pool.QueryRow", err)
	}

	return version, nil
}
