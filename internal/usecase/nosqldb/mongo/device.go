package mongo

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/device-management-toolkit/console/internal/entity"
	"github.com/device-management-toolkit/console/internal/usecase/devices"
)

type DeviceRepo struct {
	col *mongo.Collection
}

var _ devices.Repository = (*DeviceRepo)(nil)

func NewDeviceRepo(db *mongo.Database) *DeviceRepo {
	return &DeviceRepo{col: db.Collection(CollectionDevices)}
}

func (r *DeviceRepo) GetCount(ctx context.Context, tenantID string) (int, error) {
	if tenantID != "" && !identifierRegex.MatchString(tenantID) {
		return 0, nil
	}

	n, err := r.col.CountDocuments(ctx, bson.M{fieldTenantID: tenantID})
	if err != nil {
		return 0, errDeviceDatabase.Wrap("GetCount", "CountDocuments", err)
	}

	return int(n), nil
}

func (r *DeviceRepo) Get(ctx context.Context, top, skip int, tenantID string) ([]entity.Device, error) {
	if tenantID != "" && !identifierRegex.MatchString(tenantID) {
		return []entity.Device{}, nil
	}

	limit := int64(DefaultTop)
	if top > 0 {
		limit = int64(top)
	}

	offset := int64(0)
	if skip > 0 {
		offset = int64(skip)
	}

	cur, err := r.col.Find(ctx, bson.M{fieldTenantID: tenantID},
		options.Find().SetSort(bson.D{{Key: fieldGUID, Value: 1}}).SetLimit(limit).SetSkip(offset))
	if err != nil {
		return nil, errDeviceDatabase.Wrap("Get", "Find", err)
	}
	defer cur.Close(ctx)

	devs := make([]entity.Device, 0)
	if err := cur.All(ctx, &devs); err != nil {
		return nil, errDeviceDatabase.Wrap("Get", "Cursor.All", err)
	}

	return devs, nil
}

func (r *DeviceRepo) GetByID(ctx context.Context, guid, tenantID string) (*entity.Device, error) {
	if !identifierRegex.MatchString(guid) {
		return nil, nil
	}

	if tenantID != "" && !identifierRegex.MatchString(tenantID) {
		return nil, nil
	}

	d := entity.Device{}

	err := r.col.FindOne(ctx, bson.M{fieldGUID: guid, fieldTenantID: tenantID}).Decode(&d)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}

		return nil, errDeviceDatabase.Wrap("GetByID", "FindOne", err)
	}

	return &d, nil
}

func (r *DeviceRepo) GetDistinctTags(ctx context.Context, tenantID string) ([]string, error) {
	if tenantID != "" && !identifierRegex.MatchString(tenantID) {
		return []string{}, nil
	}

	// Tags are a comma-joined string (SQL-compat); split and dedupe in Go.
	var rawStrings []string
	if err := r.col.Distinct(ctx, fieldTags, bson.M{fieldTenantID: tenantID}).Decode(&rawStrings); err != nil {
		return []string{}, errDeviceDatabase.Wrap("GetDistinctTags", "Distinct", err)
	}

	seen := map[string]struct{}{}
	out := make([]string, 0, len(rawStrings))

	for _, s := range rawStrings {
		for _, t := range strings.Split(s, ",") {
			t = strings.TrimSpace(t)
			if t == "" {
				continue
			}

			if _, ok := seen[t]; ok {
				continue
			}

			seen[t] = struct{}{}
			out = append(out, t)
		}
	}

	return out, nil
}

func (r *DeviceRepo) GetByTags(ctx context.Context, tags []string, method string, limit, offset int, tenantID string) ([]entity.Device, error) {
	if tenantID != "" && !identifierRegex.MatchString(tenantID) {
		return []entity.Device{}, nil
	}

	// Tags are a comma-joined string (SQL-compat); regex against it.
	regexes := make([]bson.M, 0, len(tags))
	for _, t := range tags {
		regexes = append(regexes, bson.M{fieldTags: bson.M{opRegex: "(^|,)" + regexp.QuoteMeta(t) + "(,|$)"}})
	}

	filter := bson.M{fieldTenantID: tenantID}

	if len(regexes) > 0 {
		if method == "AND" {
			filter["$and"] = regexes
		} else {
			filter["$or"] = regexes
		}
	}

	// No DefaultTop here — limit<=0 means unbounded (matches sqldb GetByTags).
	lim := int64(0)
	if limit > 0 {
		lim = int64(limit)
	}

	off := int64(0)
	if offset > 0 {
		off = int64(offset)
	}

	cur, err := r.col.Find(ctx, filter,
		options.Find().SetSort(bson.D{{Key: fieldGUID, Value: 1}}).SetLimit(lim).SetSkip(off))
	if err != nil {
		return nil, errDeviceDatabase.Wrap("GetByTags", "Find", err)
	}
	defer cur.Close(ctx)

	devs := make([]entity.Device, 0)
	if err := cur.All(ctx, &devs); err != nil {
		return nil, errDeviceDatabase.Wrap("GetByTags", "Cursor.All", err)
	}

	return devs, nil
}

func (r *DeviceRepo) Delete(ctx context.Context, guid, tenantID string) (bool, error) {
	if !identifierRegex.MatchString(guid) {
		return false, nil
	}

	if tenantID != "" && !identifierRegex.MatchString(tenantID) {
		return false, nil
	}

	res, err := r.col.DeleteOne(ctx, bson.M{fieldGUID: guid, fieldTenantID: tenantID})
	if err != nil {
		return false, errDeviceDatabase.Wrap("Delete", "DeleteOne", err)
	}

	return res.DeletedCount > 0, nil
}

func (r *DeviceRepo) Update(ctx context.Context, d *entity.Device) (bool, error) {
	if !identifierRegex.MatchString(d.GUID) {
		return false, errDeviceDatabase.Wrap("Update", "validate", nil)
	}

	if d.TenantID != "" && !identifierRegex.MatchString(d.TenantID) {
		return false, errDeviceDatabase.Wrap("Update", "validate", nil)
	}

	// Explicit field list mirrors sqldb/device.go:Update so a new field must be wired in intentionally.
	res, err := r.col.UpdateOne(ctx,
		bson.M{fieldGUID: d.GUID, fieldTenantID: d.TenantID},
		bson.M{opSet: bson.M{
			fieldGUID:          d.GUID,
			"hostname":         d.Hostname,
			fieldTags:          d.Tags,
			"mpsinstance":      d.MPSInstance,
			"connectionstatus": d.ConnectionStatus,
			"mpsusername":      d.MPSUsername,
			fieldTenantID:      d.TenantID,
			"friendlyname":     d.FriendlyName,
			"dnssuffix":        d.DNSSuffix,
			"deviceinfo":       d.DeviceInfo,
			"username":         d.Username,
			"password":         d.Password,
			"mpspassword":      d.MPSPassword,
			"mebxpassword":     d.MEBXPassword,
			"usetls":           d.UseTLS,
			"allowselfsigned":  d.AllowSelfSigned,
			"certhash":         d.CertHash,
		}},
	)
	if err != nil {
		return false, errDeviceDatabase.Wrap("Update", "UpdateOne", err)
	}

	return res.MatchedCount > 0, nil
}

func (r *DeviceRepo) UpdateConnectionStatus(ctx context.Context, guid string, status bool) error {
	if !identifierRegex.MatchString(guid) {
		return errDeviceDatabase.Wrap("UpdateConnectionStatus", "validate", nil)
	}

	now := time.Now()

	set := bson.M{"connectionstatus": status}
	if status {
		set["lastconnected"] = now
	} else {
		set["lastdisconnected"] = now
	}

	_, err := r.col.UpdateOne(ctx, bson.M{fieldGUID: guid}, bson.M{opSet: set})
	if err != nil {
		return errDeviceDatabase.Wrap("UpdateConnectionStatus", "UpdateOne", err)
	}

	return nil
}

func (r *DeviceRepo) UpdateLastSeen(ctx context.Context, guid string) error {
	if !identifierRegex.MatchString(guid) {
		return errDeviceDatabase.Wrap("UpdateLastSeen", "validate", nil)
	}

	_, err := r.col.UpdateOne(ctx,
		bson.M{fieldGUID: guid},
		bson.M{opSet: bson.M{"lastseen": time.Now()}},
	)
	if err != nil {
		return errDeviceDatabase.Wrap("UpdateLastSeen", "UpdateOne", err)
	}

	return nil
}

func (r *DeviceRepo) Insert(ctx context.Context, d *entity.Device) (string, error) {
	if !identifierRegex.MatchString(d.GUID) {
		return "", errDeviceDatabase.Wrap("Insert", "validate", nil)
	}

	if d.TenantID != "" && !identifierRegex.MatchString(d.TenantID) {
		return "", errDeviceDatabase.Wrap("Insert", "validate", nil)
	}

	_, err := r.col.InsertOne(ctx, d)
	if err != nil {
		if isDuplicateKey(err) {
			return "", errDeviceNotUnique.Wrap(err.Error())
		}

		return "", errDeviceDatabase.Wrap("Insert", "InsertOne", err)
	}

	return "", nil // matches embedded SQL ("" return)
}

func (r *DeviceRepo) GetByColumn(ctx context.Context, columnName, queryValue, tenantID string) ([]entity.Device, error) {
	if tenantID != "" && !identifierRegex.MatchString(tenantID) {
		return []entity.Device{}, nil
	}

	// Lowercase to match BSON codec keys (Mongo is case-sensitive; Postgres folds).
	field := strings.ToLower(columnName)

	// Allowlist — columnName becomes a bson.M key; unbounded would let callers query any field.
	switch field {
	case "hostname", "friendlyname", "tags":
	default:
		return []entity.Device{}, nil
	}

	cur, err := r.col.Find(ctx, bson.M{field: queryValue, fieldTenantID: tenantID})
	if err != nil {
		return nil, errDeviceDatabase.Wrap("GetByColumn", "Find", err)
	}
	defer cur.Close(ctx)

	devs := make([]entity.Device, 0)
	if err := cur.All(ctx, &devs); err != nil {
		return nil, errDeviceDatabase.Wrap("GetByColumn", "Cursor.All", err)
	}

	return devs, nil
}
