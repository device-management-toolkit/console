package sqldb

import (
	"context"
	"database/sql"
	"errors"

	"github.com/device-management-toolkit/console/internal/entity"
	"github.com/device-management-toolkit/console/pkg/consoleerrors"
	"github.com/device-management-toolkit/console/pkg/db"
	"github.com/device-management-toolkit/console/pkg/logger"
)

// CIRARepo -.
type CIRARepo struct {
	*db.SQL
	log logger.Interface
}

// New -.
func NewCIRARepo(database *db.SQL, log logger.Interface) *CIRARepo {
	return &CIRARepo{database, log}
}

var (
	ErrCIRARepo          = consoleerrors.CreateConsoleError("CIRARepo")
	ErrCIRARepoDatabase  = DatabaseError{Console: consoleerrors.CreateConsoleError("CIRARepo")}
	ErrCIRARepoNotUnique = NotUniqueError{Console: consoleerrors.CreateConsoleError("CIRARepo")}
)

// GetCount -.
func (r *CIRARepo) GetCount(_ context.Context, tenantID string) (int, error) {
	sqlQuery, _, err := r.Builder.
		Select("COUNT(*) OVER() AS total_count").
		From("ciraconfigs").
		Where("tenant_id = ?", tenantID).
		ToSql()
	if err != nil {
		return 0, ErrCIRARepoDatabase.Wrap("GetCount", "r.Builder", err)
	}

	var count int

	err = r.Pool.QueryRowContext(context.Background(), sqlQuery, tenantID).Scan(&count)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}

		return 0, ErrCIRARepoDatabase.Wrap("GetCount", "r.Pool.QueryRow", err)
	}

	return count, nil
}

// Get -.
func (r *CIRARepo) Get(_ context.Context, top, skip int, tenantID string) ([]entity.CIRAConfig, error) {
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
		Select("cira_config_name",
			"mps_server_address",
			"mps_port",
			"user_name",
			"password",
			"common_name",
			"server_address_format",
			"auth_method",
			"mps_root_certificate",
			"proxydetails",
			"tenant_id").
		From("ciraconfigs").
		Where("tenant_id = ?", tenantID).
		OrderBy("cira_config_name").
		Limit(limitedTop).
		Offset(limitedSkip).
		ToSql()
	if err != nil {
		return nil, ErrCIRARepoDatabase.Wrap("Get", "r.Builder", err)
	}

	rows, err := r.Pool.QueryContext(context.Background(), sqlQuery, tenantID)
	if err != nil {
		return nil, ErrCIRARepoDatabase.Wrap("Get", "r.Pool.Query", err)
	}

	defer rows.Close()

	if rows.Err() != nil {
		return nil, ErrDeviceDatabase.Wrap("Get", "rows.Err", rows.Err())
	}

	configs := make([]entity.CIRAConfig, 0)

	for rows.Next() {
		p := entity.CIRAConfig{}

		err = rows.Scan(&p.ConfigName, &p.MPSAddress, &p.MPSPort, &p.Username, &p.Password, &p.CommonName, &p.ServerAddressFormat, &p.AuthMethod, &p.MPSRootCertificate, &p.ProxyDetails, &p.TenantID)
		if err != nil {
			return nil, ErrCIRARepoDatabase.Wrap("Get", "rows.Scan", err)
		}

		configs = append(configs, p)
	}

	return configs, nil
}

// GetByName -.
func (r *CIRARepo) GetByName(_ context.Context, configName, tenantID string) (*entity.CIRAConfig, error) {
	sqlQuery, _, err := r.Builder.
		Select("cira_config_name",
			"mps_server_address",
			"mps_port",
			"user_name",
			"password",
			"common_name",
			"server_address_format",
			"auth_method",
			"mps_root_certificate",
			"proxydetails",
			"tenant_id").
		From("ciraconfigs").
		Where("cira_config_name = ? and tenant_id = ?", configName, tenantID).
		ToSql()
	if err != nil {
		return nil, ErrCIRARepoDatabase.Wrap("GetByName", "r.Builder", err)
	}

	rows, err := r.Pool.QueryContext(context.Background(), sqlQuery, configName, tenantID)
	if err != nil {
		return nil, ErrCIRARepoDatabase.Wrap("GetByName", "r.Pool.Query", err)
	}

	defer rows.Close()

	if rows.Err() != nil {
		return nil, ErrDeviceDatabase.Wrap("Get", "rows.Err", rows.Err())
	}

	configs := make([]*entity.CIRAConfig, 0)

	for rows.Next() {
		p := &entity.CIRAConfig{}

		err = rows.Scan(&p.ConfigName, &p.MPSAddress, &p.MPSPort, &p.Username, &p.Password, &p.CommonName, &p.ServerAddressFormat, &p.AuthMethod, &p.MPSRootCertificate, &p.ProxyDetails, &p.TenantID)
		if err != nil {
			return p, ErrCIRARepoDatabase.Wrap("GetByName", "rows.Scan", err)
		}

		configs = append(configs, p)
	}

	if len(configs) == 0 {
		return nil, nil
	}

	return configs[0], nil
}

// Delete -.
func (r *CIRARepo) Delete(_ context.Context, configName, tenantID string) (bool, error) {
	sqlQuery, args, err := r.Builder.
		Delete("ciraconfigs").
		Where("cira_config_name = ? AND tenant_id = ?", configName, tenantID).
		ToSql()
	if err != nil {
		return false, ErrCIRARepoDatabase.Wrap("Delete", "r.Builder", err)
	}

	res, err := r.Pool.ExecContext(context.Background(), sqlQuery, args...)
	if err != nil {
		return false, ErrCIRARepoDatabase.Wrap("Delete", "r.Pool.Exec", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return false, ErrCIRARepoDatabase.Wrap("Delete", "res.RowsAffected", err)
	}

	return rowsAffected > 0, nil
}

// Update -.
func (r *CIRARepo) Update(_ context.Context, p *entity.CIRAConfig) (bool, error) {
	sqlQuery, args, err := r.Builder.
		Update("ciraconfigs").
		Set("mps_server_address", p.MPSAddress).
		Set("mps_port", p.MPSPort).
		Set("user_name", p.Username).
		Set("password", p.Password).
		Set("common_name", p.CommonName).
		Set("server_address_format", p.ServerAddressFormat).
		Set("auth_method", p.AuthMethod).
		Set("mps_root_certificate", p.MPSRootCertificate).
		Set("proxydetails", p.ProxyDetails).
		Where("cira_config_name = ? AND tenant_id = ?", p.ConfigName, p.TenantID).
		ToSql()
	if err != nil {
		return false, ErrCIRARepoDatabase.Wrap("Update", "r.Builder", err)
	}

	res, err := r.Pool.ExecContext(context.Background(), sqlQuery, args...)
	if err != nil {
		return false, ErrCIRARepoDatabase.Wrap("Update", "r.Pool.Exec", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return false, ErrCIRARepoDatabase.Wrap("Delete", "res.RowsAffected", err)
	}

	return rowsAffected > 0, nil
}

// Insert -.
func (r *CIRARepo) Insert(_ context.Context, p *entity.CIRAConfig) (string, error) {
	insertBuilder := r.Builder.
		Insert("ciraconfigs").
		Columns("cira_config_name", "mps_server_address", "mps_port", "user_name", "password", "common_name", "server_address_format", "auth_method", "mps_root_certificate", "proxydetails", "tenant_id").
		Values(p.ConfigName, p.MPSAddress, p.MPSPort, p.Username, p.Password, p.CommonName, p.ServerAddressFormat, p.AuthMethod, p.MPSRootCertificate, p.ProxyDetails, p.TenantID)

	if !r.IsEmbedded {
		insertBuilder = insertBuilder.Suffix("RETURNING xmin::text")
	}

	sqlQuery, args, err := insertBuilder.ToSql()
	if err != nil {
		return "", ErrCIRARepoDatabase.Wrap("Insert", "r.Builder", err)
	}

	version := ""

	if r.IsEmbedded {
		_, err = r.Pool.ExecContext(context.Background(), sqlQuery, args...)
	} else {
		err = r.Pool.QueryRowContext(context.Background(), sqlQuery, args...).Scan(&version)
	}

	if err != nil {
		if db.CheckNotUnique(err) {
			return "", ErrCIRARepoNotUnique.Wrap(err.Error())
		}

		return "", ErrCIRARepoDatabase.Wrap("Insert", "r.Pool.QueryRow", err)
	}

	return version, nil
}
