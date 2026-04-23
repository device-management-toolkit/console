package mongo

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/device-management-toolkit/console/internal/entity"
	"github.com/device-management-toolkit/console/internal/usecase/wificonfigs"
	"github.com/device-management-toolkit/console/pkg/logger"
)

type WirelessRepo struct {
	col          *mongo.Collection
	ieee8021xCol *mongo.Collection
	log          logger.Interface
}

var _ wificonfigs.Repository = (*WirelessRepo)(nil)

func NewWirelessRepo(db *mongo.Database, log logger.Interface) *WirelessRepo {
	return &WirelessRepo{
		col:          db.Collection(CollectionWirelessConfigs),
		ieee8021xCol: db.Collection(CollectionIEEE8021xConfigs),
		log:          log,
	}
}

func (r *WirelessRepo) CheckProfileExists(ctx context.Context, profileName, tenantID string) (bool, error) {
	n, err := r.col.CountDocuments(ctx, bson.M{"profilename": profileName, "tenantid": tenantID},
		options.Count().SetLimit(1))
	if err != nil {
		return false, errWiFiDatabase.Wrap("CheckProfileExists", "CountDocuments", err)
	}

	return n > 0, nil
}

func (r *WirelessRepo) GetCount(ctx context.Context, tenantID string) (int, error) {
	n, err := r.col.CountDocuments(ctx, bson.M{"tenantid": tenantID})
	if err != nil {
		return 0, errWiFiDatabase.Wrap("GetCount", "CountDocuments", err)
	}

	return int(n), nil
}

func (r *WirelessRepo) Get(ctx context.Context, top, skip int, tenantID string) ([]entity.WirelessConfig, error) {
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
		return nil, errWiFiDatabase.Wrap("Get", "Find", err)
	}
	defer cur.Close(ctx)

	out := make([]entity.WirelessConfig, 0)
	if err := cur.All(ctx, &out); err != nil {
		return nil, errWiFiDatabase.Wrap("Get", "Cursor.All", err)
	}

	for i := range out {
		r.populate8021x(ctx, &out[i])
	}

	return out, nil
}

func (r *WirelessRepo) GetByName(ctx context.Context, profileName, tenantID string) (*entity.WirelessConfig, error) {
	w := entity.WirelessConfig{}

	err := r.col.FindOne(ctx, bson.M{"profilename": profileName, "tenantid": tenantID}).Decode(&w)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}

		return nil, errWiFiDatabase.Wrap("GetByName", "FindOne", err)
	}

	r.populate8021x(ctx, &w)

	return &w, nil
}

// populate8021x mirrors the LEFT JOIN onto ieee8021xconfigs (wired_interface=false).
func (r *WirelessRepo) populate8021x(ctx context.Context, w *entity.WirelessConfig) {
	if w.IEEE8021xProfileName == nil || *w.IEEE8021xProfileName == "" {
		return
	}

	var ieee entity.IEEE8021xConfig
	if err := r.ieee8021xCol.FindOne(ctx, bson.M{
		"profilename":    *w.IEEE8021xProfileName,
		"tenantid":       w.TenantID,
		"wiredinterface": false,
	}).Decode(&ieee); err != nil {
		if !errors.Is(err, mongo.ErrNoDocuments) {
			r.log.Warn("populate8021x: %v", err)
		}

		return
	}

	ap := ieee.AuthenticationProtocol
	w.AuthenticationProtocol = &ap
	w.PXETimeout = ieee.PXETimeout

	wi := ieee.WiredInterface
	w.WiredInterface = &wi
}

func (r *WirelessRepo) Delete(ctx context.Context, profileName, tenantID string) (bool, error) {
	res, err := r.col.DeleteOne(ctx, bson.M{"profilename": profileName, "tenantid": tenantID})
	if err != nil {
		return false, errWiFiDatabase.Wrap("Delete", "DeleteOne", err)
	}

	return res.DeletedCount > 0, nil
}

func (r *WirelessRepo) Update(ctx context.Context, w *entity.WirelessConfig) (bool, error) {
	res, err := r.col.UpdateOne(ctx,
		bson.M{"profilename": w.ProfileName, "tenantid": w.TenantID},
		bson.M{"$set": bson.M{
			"authenticationmethod": w.AuthenticationMethod,
			"encryptionmethod":     w.EncryptionMethod,
			"ssid":                 w.SSID,
			"pskvalue":             w.PSKValue,
			"pskpassphrase":        w.PSKPassphrase,
			"linkpolicy":           w.LinkPolicy,
			"ieee8021xprofilename": nullIfEmptyPtr(w.IEEE8021xProfileName),
		}},
	)
	if err != nil {
		return false, errWiFiDatabase.Wrap("Update", "UpdateOne", err)
	}

	return res.MatchedCount > 0, nil
}

func (r *WirelessRepo) Insert(ctx context.Context, w *entity.WirelessConfig) (string, error) {
	ieee := nullIfEmptyPtr(w.IEEE8021xProfileName)

	doc := bson.M{
		"profilename":          w.ProfileName,
		"authenticationmethod": w.AuthenticationMethod,
		"encryptionmethod":     w.EncryptionMethod,
		"ssid":                 w.SSID,
		"pskvalue":             w.PSKValue,
		"pskpassphrase":        w.PSKPassphrase,
		"linkpolicy":           w.LinkPolicy,
		"creationdate":         time.Now().Format("2006-01-02 15:04:05"),
		"tenantid":             w.TenantID,
		"ieee8021xprofilename": ieee,
	}

	if _, err := r.col.InsertOne(ctx, doc); err != nil {
		if isDuplicateKey(err) {
			return "", errWiFiNotUnique.Wrap(err.Error())
		}

		return "", errWiFiDatabase.Wrap("Insert", "InsertOne", err)
	}

	return "", nil
}
