package mongo

import (
	"context"
	"errors"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/device-management-toolkit/console/internal/entity"
	"github.com/device-management-toolkit/console/internal/usecase/domains"
	"github.com/device-management-toolkit/console/pkg/logger"
)

type DomainRepo struct {
	col *mongo.Collection
	log logger.Interface
}

var _ domains.Repository = (*DomainRepo)(nil)

func NewDomainRepo(db *mongo.Database, log logger.Interface) *DomainRepo {
	return &DomainRepo{col: db.Collection(CollectionDomains), log: log}
}

func (r *DomainRepo) GetCount(ctx context.Context, tenantID string) (int, error) {
	n, err := r.col.CountDocuments(ctx, bson.M{"tenantid": tenantID})
	if err != nil {
		return 0, errDomainDatabase.Wrap("GetCount", "CountDocuments", err)
	}

	return int(n), nil
}

func (r *DomainRepo) Get(ctx context.Context, top, skip int, tenantID string) ([]entity.Domain, error) {
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
		options.Find().SetSort(bson.D{{Key: "profilename", Value: 1}}).SetLimit(limit).SetSkip(offset))
	if err != nil {
		return nil, errDomainDatabase.Wrap("Get", "Find", err)
	}
	defer cur.Close(ctx)

	out := make([]entity.Domain, 0)
	if err := cur.All(ctx, &out); err != nil {
		return nil, errDomainDatabase.Wrap("Get", "Cursor.All", err)
	}

	return out, nil
}

func (r *DomainRepo) GetDomainByDomainSuffix(ctx context.Context, domainSuffix, tenantID string) (*entity.Domain, error) {
	d := entity.Domain{}

	err := r.col.FindOne(ctx, bson.M{"domainsuffix": domainSuffix, "tenantid": tenantID}).Decode(&d)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}

		return nil, errDomainDatabase.Wrap("GetDomainByDomainSuffix", "FindOne", err)
	}

	return &d, nil
}

func (r *DomainRepo) GetByName(ctx context.Context, name, tenantID string) (*entity.Domain, error) {
	// SQL uses LOWER(name) = LOWER(?); do the same with a case-insensitive regex
	// anchored to the full value.
	filter := bson.M{
		"profilename": bson.M{"$regex": "^" + regexpCaseInsensitive(name) + "$", "$options": "i"},
		"tenantid":    tenantID,
	}

	d := entity.Domain{}

	err := r.col.FindOne(ctx, filter).Decode(&d)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}

		return nil, errDomainDatabase.Wrap("GetByName", "FindOne", err)
	}

	return &d, nil
}

func (r *DomainRepo) Delete(ctx context.Context, name, tenantID string) (bool, error) {
	res, err := r.col.DeleteOne(ctx, bson.M{
		"profilename": bson.M{"$regex": "^" + regexpCaseInsensitive(name) + "$", "$options": "i"},
		"tenantid":    tenantID,
	})
	if err != nil {
		return false, errDomainDatabase.Wrap("Delete", "DeleteOne", err)
	}

	return res.DeletedCount > 0, nil
}

func (r *DomainRepo) Update(ctx context.Context, d *entity.Domain) (bool, error) {
	res, err := r.col.UpdateOne(ctx,
		bson.M{"profilename": d.ProfileName, "tenantid": d.TenantID},
		bson.M{"$set": bson.M{
			"profilename":                   d.ProfileName,
			"domainsuffix":                  d.DomainSuffix,
			"provisioningcert":              d.ProvisioningCert,
			"provisioningcertstorageformat": d.ProvisioningCertStorageFormat,
			"provisioningcertpassword":      d.ProvisioningCertPassword,
			"expirationdate":                d.ExpirationDate,
		}},
	)
	if err != nil {
		if isDuplicateKey(err) {
			return false, errDomainNotUnique.Wrap(err.Error())
		}

		return false, errDomainDatabase.Wrap("Update", "UpdateOne", err)
	}

	return res.MatchedCount > 0, nil
}

func (r *DomainRepo) Insert(ctx context.Context, d *entity.Domain) (string, error) {
	if _, err := r.col.InsertOne(ctx, d); err != nil {
		if isDuplicateKey(err) {
			return "", errDomainNotUnique.Wrap(err.Error())
		}

		return "", errDomainDatabase.Wrap("Insert", "InsertOne", err)
	}

	return "", nil
}

// regexpCaseInsensitive escapes regex metacharacters in name for safe use in
// a $regex filter (same purpose as regexp.QuoteMeta — re-implemented here
// without pulling in the regexp package since this file only needs escaping).
func regexpCaseInsensitive(s string) string {
	const meta = `\.+*?()|[]{}^$`

	var b strings.Builder

	b.Grow(len(s))

	for _, r := range s {
		if strings.ContainsRune(meta, r) {
			b.WriteByte('\\')
		}

		b.WriteRune(r)
	}

	return b.String()
}
