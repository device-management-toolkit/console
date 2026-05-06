package mongo

import (
	"context"
	"errors"
	"regexp"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/device-management-toolkit/console/internal/entity"
	"github.com/device-management-toolkit/console/internal/usecase/domains"
)

type DomainRepo struct {
	col *mongo.Collection
}

var _ domains.Repository = (*DomainRepo)(nil)

func NewDomainRepo(db *mongo.Database) *DomainRepo {
	return &DomainRepo{col: db.Collection(CollectionDomains)}
}

func (r *DomainRepo) GetCount(ctx context.Context, tenantID string) (int, error) {
	if tenantID != "" && !identifierRegex.MatchString(tenantID) {
		return 0, nil
	}

	n, err := r.col.CountDocuments(ctx, bson.M{fieldTenantID: tenantID})
	if err != nil {
		return 0, errDomainDatabase.Wrap("GetCount", "CountDocuments", err)
	}

	return int(n), nil
}

func (r *DomainRepo) Get(ctx context.Context, top, skip int, tenantID string) ([]entity.Domain, error) {
	if tenantID != "" && !identifierRegex.MatchString(tenantID) {
		return []entity.Domain{}, nil
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
		options.Find().SetSort(bson.D{{Key: fieldProfileName, Value: 1}}).SetLimit(limit).SetSkip(offset))
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
	if !domainSuffixRegex.MatchString(domainSuffix) {
		return nil, nil
	}

	if tenantID != "" && !identifierRegex.MatchString(tenantID) {
		return nil, nil
	}

	d := entity.Domain{}

	err := r.col.FindOne(ctx, bson.M{fieldDomainSuffix: domainSuffix, fieldTenantID: tenantID}).Decode(&d)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}

		return nil, errDomainDatabase.Wrap("GetDomainByDomainSuffix", "FindOne", err)
	}

	return &d, nil
}

func (r *DomainRepo) GetByName(ctx context.Context, name, tenantID string) (*entity.Domain, error) {
	if !identifierRegex.MatchString(name) {
		return nil, nil
	}

	if tenantID != "" && !identifierRegex.MatchString(tenantID) {
		return nil, nil
	}

	// Case-insensitive match (mirrors SQL LOWER(name) = LOWER(?)).
	filter := bson.M{
		fieldProfileName: bson.M{opRegex: "^" + regexp.QuoteMeta(name) + "$", "$options": "i"},
		fieldTenantID:    tenantID,
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
	if !identifierRegex.MatchString(name) {
		return false, nil
	}

	if tenantID != "" && !identifierRegex.MatchString(tenantID) {
		return false, nil
	}

	res, err := r.col.DeleteOne(ctx, bson.M{
		fieldProfileName: bson.M{opRegex: "^" + regexp.QuoteMeta(name) + "$", "$options": "i"},
		fieldTenantID:    tenantID,
	})
	if err != nil {
		return false, errDomainDatabase.Wrap("Delete", "DeleteOne", err)
	}

	return res.DeletedCount > 0, nil
}

func (r *DomainRepo) Update(ctx context.Context, d *entity.Domain) (bool, error) {
	if !identifierRegex.MatchString(d.ProfileName) {
		return false, errDomainDatabase.Wrap("Update", "validate", nil)
	}

	if d.TenantID != "" && !identifierRegex.MatchString(d.TenantID) {
		return false, errDomainDatabase.Wrap("Update", "validate", nil)
	}

	// profilename intentionally not in $set — it's the filter key, immutable per SQL semantics.
	res, err := r.col.UpdateOne(ctx,
		bson.M{fieldProfileName: d.ProfileName, fieldTenantID: d.TenantID},
		bson.M{opSet: bson.M{
			fieldDomainSuffix:               d.DomainSuffix,
			"provisioningcert":              d.ProvisioningCert,
			"provisioningcertstorageformat": d.ProvisioningCertStorageFormat,
			"provisioningcertpassword":      d.ProvisioningCertPassword,
			"expirationdate":                d.ExpirationDate,
		}},
	)
	if err != nil {
		return false, errDomainDatabase.Wrap("Update", "UpdateOne", err)
	}

	return res.MatchedCount > 0, nil
}

func (r *DomainRepo) Insert(ctx context.Context, d *entity.Domain) (string, error) {
	if !identifierRegex.MatchString(d.ProfileName) {
		return "", errDomainDatabase.Wrap("Insert", "validate", nil)
	}

	if d.TenantID != "" && !identifierRegex.MatchString(d.TenantID) {
		return "", errDomainDatabase.Wrap("Insert", "validate", nil)
	}

	if _, err := r.col.InsertOne(ctx, d); err != nil {
		if isDuplicateKey(err) {
			return "", errDomainNotUnique.Wrap(err.Error())
		}

		return "", errDomainDatabase.Wrap("Insert", "InsertOne", err)
	}

	return "", nil
}
