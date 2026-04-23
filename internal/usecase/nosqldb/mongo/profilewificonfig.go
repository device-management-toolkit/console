package mongo

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/device-management-toolkit/console/internal/entity"
	"github.com/device-management-toolkit/console/internal/usecase/profilewificonfigs"
	"github.com/device-management-toolkit/console/pkg/logger"
)

type ProfileWiFiConfigsRepo struct {
	col *mongo.Collection
	log logger.Interface
}

var _ profilewificonfigs.Repository = (*ProfileWiFiConfigsRepo)(nil)

func NewProfileWiFiConfigsRepo(db *mongo.Database, log logger.Interface) *ProfileWiFiConfigsRepo {
	return &ProfileWiFiConfigsRepo{col: db.Collection(CollectionProfileWiFiConfigs), log: log}
}

func (r *ProfileWiFiConfigsRepo) GetByProfileName(ctx context.Context, profileName, tenantID string) ([]entity.ProfileWiFiConfigs, error) {
	cur, err := r.col.Find(ctx, bson.M{"profilename": profileName, "tenantid": tenantID},
		options.Find().SetSort(bson.D{{Key: "priority", Value: 1}}))
	if err != nil {
		return nil, errProfileWiFiConfigsDatabase.Wrap("GetByProfileName", "Find", err)
	}
	defer cur.Close(ctx)

	out := make([]entity.ProfileWiFiConfigs, 0)
	if err := cur.All(ctx, &out); err != nil {
		return nil, errProfileWiFiConfigsDatabase.Wrap("GetByProfileName", "Cursor.All", err)
	}

	return out, nil
}

func (r *ProfileWiFiConfigsRepo) DeleteByProfileName(ctx context.Context, profileName, tenantID string) (bool, error) {
	res, err := r.col.DeleteMany(ctx, bson.M{"profilename": profileName, "tenantid": tenantID})
	if err != nil {
		return false, errProfileWiFiConfigsDatabase.Wrap("DeleteByProfileName", "DeleteMany", err)
	}

	return res.DeletedCount > 0, nil
}

func (r *ProfileWiFiConfigsRepo) Insert(ctx context.Context, p *entity.ProfileWiFiConfigs) (string, error) {
	if _, err := r.col.InsertOne(ctx, p); err != nil {
		if isDuplicateKey(err) {
			return "", errProfileWiFiConfigsNotUnique.Wrap(err.Error())
		}

		return "", errProfileWiFiConfigsDatabase.Wrap("Insert", "InsertOne", err)
	}

	return "", nil
}
