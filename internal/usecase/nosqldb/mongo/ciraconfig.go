package mongo

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/device-management-toolkit/console/internal/entity"
	"github.com/device-management-toolkit/console/internal/usecase/ciraconfigs"
	"github.com/device-management-toolkit/console/pkg/logger"
)

type CIRARepo struct {
	col *mongo.Collection
	log logger.Interface
}

var _ ciraconfigs.Repository = (*CIRARepo)(nil)

func NewCIRARepo(db *mongo.Database, log logger.Interface) *CIRARepo {
	return &CIRARepo{col: db.Collection(CollectionCIRAConfigs), log: log}
}

func (r *CIRARepo) GetCount(ctx context.Context, tenantID string) (int, error) {
	n, err := r.col.CountDocuments(ctx, bson.M{"tenantid": tenantID})
	if err != nil {
		return 0, errCIRADatabase.Wrap("GetCount", "CountDocuments", err)
	}

	return int(n), nil
}

func (r *CIRARepo) Get(ctx context.Context, top, skip int, tenantID string) ([]entity.CIRAConfig, error) {
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
		options.Find().SetSort(bson.D{{Key: "configname", Value: 1}}).SetLimit(limit).SetSkip(offset))
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
	c := entity.CIRAConfig{}

	err := r.col.FindOne(ctx, bson.M{"configname": configName, "tenantid": tenantID}).Decode(&c)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}

		return nil, errCIRADatabase.Wrap("GetByName", "FindOne", err)
	}

	return &c, nil
}

func (r *CIRARepo) Delete(ctx context.Context, configName, tenantID string) (bool, error) {
	res, err := r.col.DeleteOne(ctx, bson.M{"configname": configName, "tenantid": tenantID})
	if err != nil {
		return false, errCIRADatabase.Wrap("Delete", "DeleteOne", err)
	}

	return res.DeletedCount > 0, nil
}

func (r *CIRARepo) Update(ctx context.Context, c *entity.CIRAConfig) (bool, error) {
	res, err := r.col.UpdateOne(ctx,
		bson.M{"configname": c.ConfigName, "tenantid": c.TenantID},
		bson.M{"$set": bson.M{
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
	if _, err := r.col.InsertOne(ctx, c); err != nil {
		if isDuplicateKey(err) {
			return "", errCIRANotUnique.Wrap(err.Error())
		}

		return "", errCIRADatabase.Wrap("Insert", "InsertOne", err)
	}

	return "", nil
}
