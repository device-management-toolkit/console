package sqldb

import (
	"context"

	"github.com/device-management-toolkit/console/internal/entity"
	"github.com/device-management-toolkit/console/pkg/consoleerrors"
	"github.com/device-management-toolkit/console/pkg/db"
	"github.com/device-management-toolkit/console/pkg/logger"
)

// ProfileWiFiConfigsRepo -.

type ProfileWiFiConfigsRepo struct {
	*db.SQL
	logger.Interface
}

var (
	ErrProfileWiFiConfigsDatabase            = DatabaseError{Console: consoleerrors.CreateConsoleError("ProfileWiFiConfigsRepo")}
	ErrProfileWiFiConfigsNotUnique           = NotUniqueError{Console: consoleerrors.CreateConsoleError("ProfileWiFiConfigsRepo")}
	ErrProfileWiFiConfigsForeignKeyViolation = ForeignKeyViolationError{Console: consoleerrors.CreateConsoleError("ProfileWiFiConfigsRepo")}
)

// New -.

func NewProfileWiFiConfigsRepo(database *db.SQL, log logger.Interface) *ProfileWiFiConfigsRepo {
	return &ProfileWiFiConfigsRepo{database, log}
}

// Get by profile name -.
func (r *ProfileWiFiConfigsRepo) GetByProfileName(_ context.Context, profileName, tenantID string) ([]entity.ProfileWiFiConfigs, error) {
	sqlQuery, args, err := r.Builder.
		Select("wireless_profile_name", "profile_name", "priority", "tenant_id").
		From("profiles_wirelessconfigs").
		Where("profile_name = ? AND tenant_id = ?", profileName, tenantID).
		OrderBy("priority").
		ToSql()
	if err != nil {
		return nil, ErrProfileWiFiConfigsDatabase.Wrap("GetByProfileName", "r.Builder", err)
	}

	rows, err := r.Pool.Query(sqlQuery, args...)
	if err != nil {
		return nil, ErrProfileWiFiConfigsDatabase.Wrap("GetByProfileName", "r.Pool.Query", err)
	}
	defer rows.Close()

	if rows.Err() != nil {
		return nil, ErrDeviceDatabase.Wrap("Get", "rows.Err", rows.Err())
	}

	var result []entity.ProfileWiFiConfigs

	for rows.Next() {
		var p entity.ProfileWiFiConfigs

		err = rows.Scan(&p.WirelessProfileName, &p.ProfileName, &p.Priority, &p.TenantID)
		if err != nil {
			return nil, ErrProfileWiFiConfigsDatabase.Wrap("GetByProfileName", "rows.Scan", err)
		}

		result = append(result, p)
	}

	return result, nil
}

// Delete -.
func (r *ProfileWiFiConfigsRepo) DeleteByProfileName(_ context.Context, profileName, tenantID string) (bool, error) {
	sqlQuery, args, err := r.Builder.
		Delete("profiles_wirelessconfigs").
		Where("profile_name = ? AND tenant_id = ?", profileName, tenantID).
		ToSql()
	if err != nil {
		return false, ErrProfileWiFiConfigsDatabase.Wrap("Delete", "r.Builder", err)
	}

	res, err := r.Pool.Exec(sqlQuery, args...)
	if err != nil {
		return false, ErrProfileWiFiConfigsDatabase.Wrap("Delete", "r.Pool.Exec", err)
	}

	result, err := res.RowsAffected()
	if err != nil {
		return false, ErrProfileWiFiConfigsDatabase.Wrap("Delete", "res.RowsAffected", err)
	}

	return result > 0, nil
}

// Insert -.
func (r *ProfileWiFiConfigsRepo) Insert(_ context.Context, p *entity.ProfileWiFiConfigs) (string, error) {
	insertBuilder := r.Builder.
		Insert("profiles_wirelessconfigs").
		Columns("wireless_profile_name", "profile_name", "priority", "tenant_id").
		Values(p.WirelessProfileName, p.ProfileName, p.Priority, p.TenantID)

	if !r.IsEmbedded {
		insertBuilder = insertBuilder.Suffix("RETURNING xmin::text")
	}

	sqlQuery, args, err := insertBuilder.ToSql()
	if err != nil {
		return "", ErrProfileWiFiConfigsDatabase.Wrap("Insert", "r.Builder", err)
	}

	version := ""

	if r.IsEmbedded {
		_, err = r.Pool.Exec(sqlQuery, args...)
	} else {
		err = r.Pool.QueryRow(sqlQuery, args...).Scan(&version)
	}

	if err != nil {
		if db.CheckNotUnique(err) {
			return "", ErrProfileWiFiConfigsNotUnique.Wrap(err.Error())
		}

		if db.CheckForeignKeyViolation(err) {
			return "", ErrProfileWiFiConfigsForeignKeyViolation.Wrap(err.Error())
		}

		return "", ErrProfileWiFiConfigsDatabase.Wrap("Insert", "r.Pool.QueryRow", err)
	}

	return version, nil
}
