package sqldb

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/device-management-toolkit/console/internal/entity"
	"github.com/device-management-toolkit/console/pkg/consoleerrors"
	"github.com/device-management-toolkit/console/pkg/db"
	"github.com/device-management-toolkit/console/pkg/logger"
)

// DeviceRepo -.
type DeviceRepo struct {
	*db.SQL
	log logger.Interface
}

var (
	ErrDeviceDatabase  = DatabaseError{Console: consoleerrors.CreateConsoleError("DeviceRepo")}
	ErrDeviceNotUnique = NotUniqueError{Console: consoleerrors.CreateConsoleError("DeviceRepo")}
)

// New -.
func NewDeviceRepo(database *db.SQL, log logger.Interface) *DeviceRepo {
	return &DeviceRepo{database, log}
}

// GetCount -.
func (r *DeviceRepo) GetCount(_ context.Context, tenantID string) (int, error) {
	sqlQuery, _, err := r.Builder.
		Select("COUNT(*) OVER() AS total_count").
		From("devices").
		Where("tenantid = ?", tenantID).
		ToSql()
	if err != nil {
		return 0, ErrDeviceDatabase.Wrap("GetCount", "r.Builder: ", err)
	}

	var count int

	err = r.Pool.QueryRowContext(context.Background(), sqlQuery, tenantID).Scan(&count)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}

		return 0, ErrDeviceDatabase.Wrap("GetCount", "r.Pool.QueryRow", err)
	}

	return count, nil
}

// Get -.
func (r *DeviceRepo) Get(_ context.Context, top, skip int, tenantID string) ([]entity.Device, error) {
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
		Select("guid",
			"hostname",
			"tags",
			"mpsinstance",
			"connectionstatus",
			"mpsusername",
			"tenantid",
			"friendlyname",
			"dnssuffix",
			"deviceinfo",
			"username",
			"password",
			"usetls",
			"allowselfsigned",
			"certhash").
		From("devices").
		Where("tenantid = ?", tenantID).
		OrderBy("guid").
		Limit(limitedTop).
		Offset(limitedSkip).
		ToSql()
	if err != nil {
		return nil, ErrDeviceDatabase.Wrap("Get", "r.Builder: ", err)
	}

	rows, err := r.Pool.QueryContext(context.Background(), sqlQuery, tenantID)
	if err != nil {
		return nil, ErrDeviceDatabase.Wrap("Get", "r.Pool.Query", err)
	}

	if rows.Err() != nil {
		return nil, ErrDeviceDatabase.Wrap("Get", "rows.Err", rows.Err())
	}

	defer rows.Close()

	devices := make([]entity.Device, 0)

	for rows.Next() {
		d := entity.Device{}

		err = rows.Scan(&d.GUID, &d.Hostname, &d.Tags, &d.MPSInstance, &d.ConnectionStatus, &d.MPSUsername, &d.TenantID, &d.FriendlyName, &d.DNSSuffix, &d.DeviceInfo, &d.Username, &d.Password, &d.UseTLS, &d.AllowSelfSigned, &d.CertHash)
		if err != nil {
			return nil, ErrDeviceDatabase.Wrap("Get", "rows.Scan: ", err)
		}

		devices = append(devices, d)
	}

	return devices, nil
}

// GetByID -.
func (r *DeviceRepo) GetByID(_ context.Context, guid, tenantID string) (*entity.Device, error) {
	sqlQuery, _, err := r.Builder.
		Select(
			"guid",
			"hostname",
			"tags",
			"mpsinstance",
			"connectionstatus",
			"mpsusername",
			"tenantid",
			"friendlyname",
			"dnssuffix",
			"deviceinfo",
			"username",
			"password",
			"usetls",
			"allowselfsigned",
			"certhash").
		From("devices").
		Where("guid = ? and tenantid = ?").
		ToSql()
	if err != nil {
		return nil, ErrDeviceDatabase.Wrap("Get", "r.Builder: ", err)
	}

	rows, err := r.Pool.QueryContext(context.Background(), sqlQuery, guid, tenantID)
	if err != nil {
		return nil, ErrDeviceDatabase.Wrap("Get", "r.Pool.Query", err)
	}

	defer rows.Close()

	if rows.Err() != nil {
		return nil, ErrDeviceDatabase.Wrap("Get", "rows.Err", rows.Err())
	}

	devices := make([]*entity.Device, 0)

	for rows.Next() {
		d := &entity.Device{}

		err = rows.Scan(&d.GUID, &d.Hostname, &d.Tags, &d.MPSInstance, &d.ConnectionStatus, &d.MPSUsername, &d.TenantID, &d.FriendlyName, &d.DNSSuffix, &d.DeviceInfo, &d.Username, &d.Password, &d.UseTLS, &d.AllowSelfSigned, &d.CertHash)
		if err != nil {
			return d, ErrDeviceDatabase.Wrap("Get", "rows.Scan: ", err)
		}

		devices = append(devices, d)
	}

	if len(devices) == 0 {
		return nil, nil
	}

	return devices[0], nil
}

func (r *DeviceRepo) GetDistinctTags(_ context.Context, tenantID string) ([]string, error) {
	sqlQuery, _, err := r.Builder.
		Select("DISTINCT tags as tag").
		From("devices").
		Where("tenantid = ?", tenantID).
		ToSql()
	if err != nil {
		return []string{}, ErrDeviceDatabase.Wrap("GetDistinctTags", "r.Builder: ", err)
	}

	rows, err := r.Pool.QueryContext(context.Background(), sqlQuery, tenantID)
	if err != nil {
		return []string{}, ErrDeviceDatabase.Wrap("GetDistinctTags", "r.Pool.Query", err)
	}

	defer rows.Close()

	if rows.Err() != nil {
		return nil, ErrDeviceDatabase.Wrap("Get", "rows.Err", rows.Err())
	}

	tags := make([]string, 0)

	for rows.Next() {
		var tag string

		err = rows.Scan(&tag)
		if err != nil {
			return []string{tag}, ErrDeviceDatabase.Wrap("GetDistinctTags", "rows.Scan: ", err)
		}

		tags = append(tags, tag)
	}

	if len(tags) == 0 {
		return []string{}, nil
	}

	return tags, nil
}

func (r *DeviceRepo) GetByTags(_ context.Context, tags []string, method string, limit, offset int, tenantID string) ([]entity.Device, error) {
	builder := r.Builder.
		Select("guid",
			"hostname",
			"tags",
			"mpsinstance",
			"connectionstatus",
			"mpsusername",
			"tenantid",
			"friendlyname",
			"dnssuffix",
			"deviceinfo").
		From("devices")

	var params []interface{}

	if method == "AND" {
		// All tags must be present (simulating an 'AND' operation)
		for _, tag := range tags {
			builder = builder.Where("(',' || tags || ',') LIKE ? AND tenantId = ?", "%,"+tag+",%", tenantID)
			params = append(params, "%,"+tag+",%", tenantID)
		}
	} else {
		// Any tag is present (simulating an 'OR' operation)
		var conditions []string
		for _, tag := range tags {
			conditions = append(conditions, "(',' || tags || ',') LIKE ?")
			params = append(params, "%,"+tag+",%")
		}

		tagsCondition := strings.Join(conditions, " OR ")

		builder = builder.Where("("+tagsCondition+") AND tenantId = ?", append(params, tenantID)...)
	}

	limitedLimit := uint64(0)
	if limit > 0 {
		limitedLimit = uint64(limit)
	}

	limitedOffset := uint64(0)
	if offset > 0 {
		limitedOffset = uint64(offset)
	}

	sqlQuery, args, err := builder.OrderBy("guid").
		Limit(limitedLimit).
		Offset(limitedOffset).
		ToSql()
	if err != nil {
		return nil, ErrDeviceDatabase.Wrap("GetByTags", "r.Builder: ", err)
	}

	rows, err := r.Pool.QueryContext(context.Background(), sqlQuery, args...)
	if err != nil {
		return nil, ErrDeviceDatabase.Wrap("GetByTags", "r.Pool.QueryContext", err)
	}
	defer rows.Close()

	if rows.Err() != nil {
		return nil, ErrDeviceDatabase.Wrap("GetByTags", "rows.Err", rows.Err())
	}

	devices := make([]entity.Device, 0)

	for rows.Next() {
		var d entity.Device
		if err := rows.Scan(&d.GUID, &d.Hostname, &d.Tags, &d.MPSInstance, &d.ConnectionStatus, &d.MPSUsername, &d.TenantID, &d.FriendlyName, &d.DNSSuffix, &d.DeviceInfo); err != nil {
			return nil, ErrDeviceDatabase.Wrap("GetByTags", "rows.Scan", err)
		}

		devices = append(devices, d)
	}

	return devices, nil
}

// Delete -.
func (r *DeviceRepo) Delete(_ context.Context, guid, tenantID string) (bool, error) {
	sqlQuery, _, err := r.Builder.
		Delete("devices").
		Where("guid = ? AND tenantid = ?", guid, tenantID).
		ToSql()
	if err != nil {
		return false, ErrDeviceDatabase.Wrap("Delete", "r.Builder", err)
	}

	res, err := r.Pool.ExecContext(context.Background(), sqlQuery, guid, tenantID)
	if err != nil {
		return false, ErrDeviceDatabase.Wrap("Delete", "r.Pool.Exec", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return false, ErrDeviceDatabase.Wrap("Delete", "res.RowsAffected", err)
	}

	return rowsAffected > 0, nil
}

// Update -.
func (r *DeviceRepo) Update(_ context.Context, d *entity.Device) (bool, error) {
	sqlQuery, args, err := r.Builder.
		Update("devices").
		Set("guid", d.GUID).
		Set("hostname", d.Hostname).
		Set("tags", d.Tags).
		Set("mpsinstance", d.MPSInstance).
		Set("connectionstatus", d.ConnectionStatus).
		Set("mpsusername", d.MPSUsername).
		Set("tenantid", d.TenantID).
		Set("friendlyname", d.FriendlyName).
		Set("dnssuffix", d.DNSSuffix).
		Set("deviceinfo", d.DeviceInfo).
		Set("username", d.Username).
		Set("password", d.Password).
		Set("useTLS", d.UseTLS).
		Set("allowSelfSigned", d.AllowSelfSigned).
		Set("certhash", d.CertHash).
		Where("guid = ? AND tenantid = ?", d.GUID, d.TenantID).
		ToSql()
	if err != nil {
		return false, ErrDeviceDatabase.Wrap("Update", "r.Builder", err)
	}

	res, err := r.Pool.ExecContext(context.Background(), sqlQuery, args...)
	if err != nil {
		return false, ErrDeviceDatabase.Wrap("Update", "r.Pool.Exec", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return false, ErrDeviceDatabase.Wrap("Delete", "res.RowsAffected", err)
	}

	return rowsAffected > 0, nil
}

// Insert -.
func (r *DeviceRepo) Insert(_ context.Context, d *entity.Device) (string, error) {
	insertBuilder := r.Builder.
		Insert("devices").
		Columns("guid", "hostname", "tags", "mpsinstance", "connectionstatus", "mpsusername", "tenantid", "friendlyname", "dnssuffix", "deviceinfo", "username", "password", "usetls", "allowselfsigned", "certhash").
		Values(d.GUID, d.Hostname, d.Tags, d.MPSInstance, d.ConnectionStatus, d.MPSUsername, d.TenantID, d.FriendlyName, d.DNSSuffix, d.DeviceInfo, d.Username, d.Password, d.UseTLS, d.AllowSelfSigned, d.CertHash)

	if !r.IsEmbedded {
		insertBuilder = insertBuilder.Suffix("RETURNING xmin::text")
	}

	sqlQuery, args, err := insertBuilder.ToSql()
	if err != nil {
		return "", ErrDeviceDatabase.Wrap("Insert", "r.Builder", err)
	}

	version := ""

	if r.IsEmbedded {
		_, err = r.Pool.ExecContext(context.Background(), sqlQuery, args...)
	} else {
		err = r.Pool.QueryRowContext(context.Background(), sqlQuery, args...).Scan(&version)
	}

	if err != nil {
		if db.CheckNotUnique(err) {
			return "", ErrDeviceNotUnique
		}

		return "", ErrDeviceDatabase.Wrap("Insert", "r.Pool.QueryRow", err)
	}

	return version, nil
}

func (r *DeviceRepo) GetByColumn(_ context.Context, columnName, queryValue, tenantID string) ([]entity.Device, error) {
	sqlQuery, _, err := r.Builder.
		Select(
			"guid",
			"hostname",
			"tags",
			"mpsinstance",
			"connectionstatus",
			"mpsusername",
			"tenantid",
			"friendlyname",
			"dnssuffix",
			"deviceinfo",
			"username",
			"password",
			"usetls",
			"allowselfsigned",
			"certhash").
		From("devices").
		Where(columnName+" = ? AND tenantid = ?", queryValue, tenantID).
		ToSql()
	if err != nil {
		return nil, ErrDeviceDatabase.Wrap("Get", "r.Builder: ", err)
	}

	rows, err := r.Pool.QueryContext(context.Background(), sqlQuery, queryValue, tenantID)
	if err != nil {
		return nil, ErrDeviceDatabase.Wrap("Get", "r.Pool.Query", err)
	}

	if rows.Err() != nil {
		return nil, ErrDeviceDatabase.Wrap("Get", "rows.Err", rows.Err())
	}

	defer rows.Close()

	devices := make([]entity.Device, 0)

	for rows.Next() {
		d := entity.Device{}

		err = rows.Scan(&d.GUID, &d.Hostname, &d.Tags, &d.MPSInstance, &d.ConnectionStatus, &d.MPSUsername, &d.TenantID, &d.FriendlyName, &d.DNSSuffix, &d.DeviceInfo, &d.Username, &d.Password, &d.UseTLS, &d.AllowSelfSigned, &d.CertHash)
		if err != nil {
			return nil, ErrDeviceDatabase.Wrap("Get", "rows.Scan: ", err)
		}

		devices = append(devices, d)
	}

	return devices, nil
}
