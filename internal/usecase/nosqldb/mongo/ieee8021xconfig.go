package mongo

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/device-management-toolkit/console/internal/entity"
	"github.com/device-management-toolkit/console/internal/usecase/ieee8021xconfigs"
)

type IEEE8021xRepo struct {
	col *mongo.Collection
}

var _ ieee8021xconfigs.Repository = (*IEEE8021xRepo)(nil)

func NewIEEE8021xRepo(db *mongo.Database) *IEEE8021xRepo {
	return &IEEE8021xRepo{col: db.Collection(CollectionIEEE8021xConfigs)}
}

func (r *IEEE8021xRepo) CheckProfileExists(ctx context.Context, profileName, tenantID string) (bool, error) {
	if !identifierRegex.MatchString(profileName) {
		return false, nil
	}

	if tenantID != "" && !identifierRegex.MatchString(tenantID) {
		return false, nil
	}

	n, err := r.col.CountDocuments(ctx, bson.M{fieldProfileName: profileName, fieldTenantID: tenantID},
		options.Count().SetLimit(1))
	if err != nil {
		return false, errIEEEDatabase.Wrap("CheckProfileExists", "CountDocuments", err)
	}

	return n > 0, nil
}

func (r *IEEE8021xRepo) GetCount(ctx context.Context, tenantID string) (int, error) {
	if tenantID != "" && !identifierRegex.MatchString(tenantID) {
		return 0, nil
	}

	n, err := r.col.CountDocuments(ctx, bson.M{fieldTenantID: tenantID})
	if err != nil {
		return 0, errIEEEDatabase.Wrap("GetCount", "CountDocuments", err)
	}

	return int(n), nil
}

func (r *IEEE8021xRepo) Get(ctx context.Context, top, skip int, tenantID string) ([]entity.IEEE8021xConfig, error) {
	if tenantID != "" && !identifierRegex.MatchString(tenantID) {
		return []entity.IEEE8021xConfig{}, nil
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
		return nil, errIEEEDatabase.Wrap("Get", "Find", err)
	}
	defer cur.Close(ctx)

	out := make([]entity.IEEE8021xConfig, 0)
	if err := cur.All(ctx, &out); err != nil {
		return nil, errIEEEDatabase.Wrap("Get", "Cursor.All", err)
	}

	return out, nil
}

func (r *IEEE8021xRepo) GetByName(ctx context.Context, profileName, tenantID string) (*entity.IEEE8021xConfig, error) {
	if !identifierRegex.MatchString(profileName) {
		return nil, nil
	}

	if tenantID != "" && !identifierRegex.MatchString(tenantID) {
		return nil, nil
	}

	c := entity.IEEE8021xConfig{}

	err := r.col.FindOne(ctx, bson.M{fieldProfileName: profileName, fieldTenantID: tenantID}).Decode(&c)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}

		return nil, errIEEEDatabase.Wrap("GetByName", "FindOne", err)
	}

	return &c, nil
}

func (r *IEEE8021xRepo) Delete(ctx context.Context, profileName, tenantID string) (bool, error) {
	if !identifierRegex.MatchString(profileName) {
		return false, nil
	}

	if tenantID != "" && !identifierRegex.MatchString(tenantID) {
		return false, nil
	}

	res, err := r.col.DeleteOne(ctx, bson.M{fieldProfileName: profileName, fieldTenantID: tenantID})
	if err != nil {
		return false, errIEEEDatabase.Wrap("Delete", "DeleteOne", err)
	}

	return res.DeletedCount > 0, nil
}

func (r *IEEE8021xRepo) Update(ctx context.Context, c *entity.IEEE8021xConfig) (bool, error) {
	if !identifierRegex.MatchString(c.ProfileName) {
		return false, errIEEEDatabase.Wrap("Update", "validate", nil)
	}

	if c.TenantID != "" && !identifierRegex.MatchString(c.TenantID) {
		return false, errIEEEDatabase.Wrap("Update", "validate", nil)
	}

	res, err := r.col.UpdateOne(ctx,
		bson.M{fieldProfileName: c.ProfileName, fieldTenantID: c.TenantID},
		bson.M{opSet: bson.M{
			"authenticationprotocol": c.AuthenticationProtocol,
			"pxetimeout":             c.PXETimeout,
			"wiredinterface":         c.WiredInterface,
		}},
	)
	if err != nil {
		return false, errIEEEDatabase.Wrap("Update", "UpdateOne", err)
	}

	return res.MatchedCount > 0, nil
}

func (r *IEEE8021xRepo) Insert(ctx context.Context, c *entity.IEEE8021xConfig) (string, error) {
	if !identifierRegex.MatchString(c.ProfileName) {
		return "", errIEEEDatabase.Wrap("Insert", "validate", nil)
	}

	if c.TenantID != "" && !identifierRegex.MatchString(c.TenantID) {
		return "", errIEEEDatabase.Wrap("Insert", "validate", nil)
	}

	if _, err := r.col.InsertOne(ctx, c); err != nil {
		if isDuplicateKey(err) {
			return "", errIEEENotUnique.Wrap(err.Error())
		}

		return "", errIEEEDatabase.Wrap("Insert", "InsertOne", err)
	}

	return "", nil
}
