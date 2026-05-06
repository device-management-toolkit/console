package mongo

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/device-management-toolkit/console/internal/entity"
	"github.com/device-management-toolkit/console/internal/usecase/ciraconfigs"
)

type CIRARepo struct {
	col *mongo.Collection
}

var _ ciraconfigs.Repository = (*CIRARepo)(nil)

func NewCIRARepo(db *mongo.Database) *CIRARepo {
	return &CIRARepo{col: db.Collection(CollectionCIRAConfigs)}
}

func (r *CIRARepo) GetCount(ctx context.Context, tenantID string) (int, error) {
	if tenantID != "" && !identifierRegex.MatchString(tenantID) {
		return 0, nil
	}

	n, err := r.col.CountDocuments(ctx, bson.M{fieldTenantID: tenantID})
	if err != nil {
		return 0, errCIRADatabase.Wrap("GetCount", "CountDocuments", err)
	}

	return int(n), nil
}

func (r *CIRARepo) Get(ctx context.Context, top, skip int, tenantID string) ([]entity.CIRAConfig, error) {
	if tenantID != "" && !identifierRegex.MatchString(tenantID) {
		return []entity.CIRAConfig{}, nil
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
		options.Find().SetSort(bson.D{{Key: fieldConfigName, Value: 1}}).SetLimit(limit).SetSkip(offset))
	if err != nil {
		return nil, errCIRADatabase.Wrap("Get", "Find", err)
	}
	defer cur.Close(ctx)

	out := make([]entity.CIRAConfig, 0)
	if err := cur.All(ctx, &out); err != nil {
		return nil, errCIRADatabase.Wrap("Get", "Cursor.All", err)
	}

	return out, nil
}

func (r *CIRARepo) GetByName(ctx context.Context, configName, tenantID string) (*entity.CIRAConfig, error) {
	if !identifierRegex.MatchString(configName) {
		return nil, nil
	}

	if tenantID != "" && !identifierRegex.MatchString(tenantID) {
		return nil, nil
	}

	c := entity.CIRAConfig{}

	err := r.col.FindOne(ctx, bson.M{fieldConfigName: configName, fieldTenantID: tenantID}).Decode(&c)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}

		return nil, errCIRADatabase.Wrap("GetByName", "FindOne", err)
	}

	return &c, nil
}

func (r *CIRARepo) Delete(ctx context.Context, configName, tenantID string) (bool, error) {
	if !identifierRegex.MatchString(configName) {
		return false, nil
	}

	if tenantID != "" && !identifierRegex.MatchString(tenantID) {
		return false, nil
	}

	res, err := r.col.DeleteOne(ctx, bson.M{fieldConfigName: configName, fieldTenantID: tenantID})
	if err != nil {
		return false, errCIRADatabase.Wrap("Delete", "DeleteOne", err)
	}

	return res.DeletedCount > 0, nil
}

func (r *CIRARepo) Update(ctx context.Context, c *entity.CIRAConfig) (bool, error) {
	if !identifierRegex.MatchString(c.ConfigName) {
		return false, errCIRADatabase.Wrap("Update", "validate", nil)
	}

	if c.TenantID != "" && !identifierRegex.MatchString(c.TenantID) {
		return false, errCIRADatabase.Wrap("Update", "validate", nil)
	}

	res, err := r.col.UpdateOne(ctx,
		bson.M{fieldConfigName: c.ConfigName, fieldTenantID: c.TenantID},
		bson.M{opSet: bson.M{
			"mpsaddress":             c.MPSAddress,
			"mpsport":                c.MPSPort,
			"username":               c.Username,
			"password":               c.Password,
			"commonname":             c.CommonName,
			"serveraddressformat":    c.ServerAddressFormat,
			"authmethod":             c.AuthMethod,
			"mpsrootcertificate":     c.MPSRootCertificate,
			"proxydetails":           c.ProxyDetails,
			"generaterandompassword": c.GenerateRandomPassword,
		}},
	)
	if err != nil {
		return false, errCIRADatabase.Wrap("Update", "UpdateOne", err)
	}

	return res.MatchedCount > 0, nil
}

func (r *CIRARepo) Insert(ctx context.Context, c *entity.CIRAConfig) (string, error) {
	if !identifierRegex.MatchString(c.ConfigName) {
		return "", errCIRADatabase.Wrap("Insert", "validate", nil)
	}

	if c.TenantID != "" && !identifierRegex.MatchString(c.TenantID) {
		return "", errCIRADatabase.Wrap("Insert", "validate", nil)
	}

	if _, err := r.col.InsertOne(ctx, c); err != nil {
		if isDuplicateKey(err) {
			return "", errCIRANotUnique.Wrap(err.Error())
		}

		return "", errCIRADatabase.Wrap("Insert", "InsertOne", err)
	}

	return "", nil
}
