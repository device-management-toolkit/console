package mongo

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/device-management-toolkit/console/internal/entity"
	"github.com/device-management-toolkit/console/internal/usecase/ieee8021xconfigs"
	"github.com/device-management-toolkit/console/pkg/logger"
)

type IEEE8021xRepo struct {
	col *mongo.Collection
	log logger.Interface
}

var _ ieee8021xconfigs.Repository = (*IEEE8021xRepo)(nil)

func NewIEEE8021xRepo(db *mongo.Database, log logger.Interface) *IEEE8021xRepo {
	return &IEEE8021xRepo{col: db.Collection(CollectionIEEE8021xConfigs), log: log}
}

func (r *IEEE8021xRepo) CheckProfileExists(ctx context.Context, profileName, tenantID string) (bool, error) {
	n, err := r.col.CountDocuments(ctx, bson.M{"profilename": profileName, "tenantid": tenantID},
		options.Count().SetLimit(1))
	if err != nil {
		return false, errIEEEDatabase.Wrap("CheckProfileExists", "CountDocuments", err)
	}

	return n > 0, nil
}

func (r *IEEE8021xRepo) GetCount(ctx context.Context, tenantID string) (int, error) {
	n, err := r.col.CountDocuments(ctx, bson.M{"tenantid": tenantID})
	if err != nil {
		return 0, errIEEEDatabase.Wrap("GetCount", "CountDocuments", err)
	}

	return int(n), nil
}

func (r *IEEE8021xRepo) Get(ctx context.Context, top, skip int, tenantID string) ([]entity.IEEE8021xConfig, error) {
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
		options.Find().SetLimit(limit).SetSkip(offset))
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
	c := entity.IEEE8021xConfig{}

	err := r.col.FindOne(ctx, bson.M{"profilename": profileName, "tenantid": tenantID}).Decode(&c)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}

		return nil, errIEEEDatabase.Wrap("GetByName", "FindOne", err)
	}

	return &c, nil
}

func (r *IEEE8021xRepo) Delete(ctx context.Context, profileName, tenantID string) (bool, error) {
	res, err := r.col.DeleteOne(ctx, bson.M{"profilename": profileName, "tenantid": tenantID})
	if err != nil {
		return false, errIEEEDatabase.Wrap("Delete", "DeleteOne", err)
	}

	return res.DeletedCount > 0, nil
}

func (r *IEEE8021xRepo) Update(ctx context.Context, c *entity.IEEE8021xConfig) (bool, error) {
	res, err := r.col.UpdateOne(ctx,
		bson.M{"profilename": c.ProfileName, "tenantid": c.TenantID},
		bson.M{"$set": bson.M{
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
	if _, err := r.col.InsertOne(ctx, c); err != nil {
		if isDuplicateKey(err) {
			return "", errIEEENotUnique.Wrap(err.Error())
		}

		return "", errIEEEDatabase.Wrap("Insert", "InsertOne", err)
	}

	return "", nil
}
