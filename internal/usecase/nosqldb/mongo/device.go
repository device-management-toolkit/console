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
	"github.com/device-management-toolkit/console/pkg/logger"
)

type DeviceRepo struct {
	col *mongo.Collection
	log logger.Interface
}

// Compile-time check: stays in lockstep with the use case interface.
var _ devices.Repository = (*DeviceRepo)(nil)

func NewDeviceRepo(db *mongo.Database, log logger.Interface) *DeviceRepo {
	return &DeviceRepo{col: db.Collection(CollectionDevices), log: log}
}

func (r *DeviceRepo) GetCount(ctx context.Context, tenantID string) (int, error) {
	n, err := r.col.CountDocuments(ctx, bson.M{"tenantid": tenantID})
	if err != nil {
		return 0, errDeviceDatabase.Wrap("GetCount", "CountDocuments", err)
	}

	return int(n), nil
}

func (r *DeviceRepo) Get(ctx context.Context, top, skip int, tenantID string) ([]entity.Device, error) {
	const defaultTop = 100

	limit := int64(defaultTop)
	if top > 0 {
		limit = int64(top)
	}

	offset := int64(0)
	if skip > 0 {
		offset = int64(skip)
	}

	cur, err := r.col.Find(ctx, bson.M{"tenantid": tenantID},
		options.Find().SetSort(bson.D{{Key: "guid", Value: 1}}).SetLimit(limit).SetSkip(offset))
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
	d := entity.Device{}

	err := r.col.FindOne(ctx, bson.M{"guid": guid, "tenantid": tenantID}).Decode(&d)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}

		return nil, errDeviceDatabase.Wrap("GetByID", "FindOne", err)
	}

	return &d, nil
}

func (r *DeviceRepo) GetDistinctTags(ctx context.Context, tenantID string) ([]string, error) {
	// SQL stores tags as a comma-joined string per row and uses SELECT DISTINCT.
	// Mirror that: pull distinct raw strings, split, de-duplicate in Go.
	var rawStrings []string
	if err := r.col.Distinct(ctx, "tags", bson.M{"tenantid": tenantID}).Decode(&rawStrings); err != nil {
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
	// Tags are stored as a single comma-joined string (SQL-compat).
	// Use regex against that string; no real array ops.
	regexes := make([]bson.M, 0, len(tags))
	for _, t := range tags {
		regexes = append(regexes, bson.M{"tags": bson.M{"$regex": "(^|,)" + regexp.QuoteMeta(t) + "(,|$)"}})
	}

	filter := bson.M{"tenantid": tenantID}

	if len(regexes) > 0 {
		if method == "AND" {
			filter["$and"] = regexes
		} else {
			filter["$or"] = regexes
		}
	}

	lim := int64(0)
	if limit > 0 {
		lim = int64(limit)
	}

	off := int64(0)
	if offset > 0 {
		off = int64(offset)
	}

	cur, err := r.col.Find(ctx, filter,
		options.Find().SetSort(bson.D{{Key: "guid", Value: 1}}).SetLimit(lim).SetSkip(off))
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
	res, err := r.col.DeleteOne(ctx, bson.M{"guid": guid, "tenantid": tenantID})
	if err != nil {
		return false, errDeviceDatabase.Wrap("Delete", "DeleteOne", err)
	}

	return res.DeletedCount > 0, nil
}

func (r *DeviceRepo) Update(ctx context.Context, d *entity.Device) (bool, error) {
	res, err := r.col.UpdateOne(ctx,
		bson.M{"guid": d.GUID, "tenantid": d.TenantID},
		bson.M{"$set": d},
	)
	if err != nil {
		return false, errDeviceDatabase.Wrap("Update", "UpdateOne", err)
	}

	return res.MatchedCount > 0, nil
}

func (r *DeviceRepo) UpdateConnectionStatus(ctx context.Context, guid string, status bool) error {
	now := time.Now()

	set := bson.M{"connectionstatus": status}
	if status {
		set["lastconnected"] = now
	} else {
		set["lastdisconnected"] = now
	}

	_, err := r.col.UpdateOne(ctx, bson.M{"guid": guid}, bson.M{"$set": set})
	if err != nil {
		return errDeviceDatabase.Wrap("UpdateConnectionStatus", "UpdateOne", err)
	}

	return nil
}

func (r *DeviceRepo) UpdateLastSeen(ctx context.Context, guid string) error {
	_, err := r.col.UpdateOne(ctx,
		bson.M{"guid": guid},
		bson.M{"$set": bson.M{"lastseen": time.Now()}},
	)
	if err != nil {
		return errDeviceDatabase.Wrap("UpdateLastSeen", "UpdateOne", err)
	}

	return nil
}

func (r *DeviceRepo) Insert(ctx context.Context, d *entity.Device) (string, error) {
	_, err := r.col.InsertOne(ctx, d)
	if err != nil {
		if isDuplicateKey(err) {
			return "", errDeviceNotUnique.Wrap(err.Error())
		}

		return "", errDeviceDatabase.Wrap("Insert", "InsertOne", err)
	}

	// SQL returns xmin (hosted) or "" (embedded). Match the embedded case.
	return "", nil
}

func (r *DeviceRepo) GetByColumn(ctx context.Context, columnName, queryValue, tenantID string) ([]entity.Device, error) {
	cur, err := r.col.Find(ctx, bson.M{columnName: queryValue, "tenantid": tenantID})
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
