// Package mongo implements every *.Repository interface against MongoDB.
// One file per entity; same method shape as the sqldb package.
//
// Design notes (see NOSQL.md at repo root):
//   - No BSON struct tags on entity types. The default mongo-driver codec
//     lowercases exported field names, so `Profile.TenantID` maps to the
//     BSON key `tenantid`. All query literals here use that form.
//   - Semantic errors (`DatabaseError`, `NotUniqueError`, ...) are reused
//     from `sqldb` so use cases' `errors.As` checks keep working.
//   - Unique indexes created on startup replace the SQL UNIQUE constraints.
//     Collections auto-create on first write; no migrations.
package mongo

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/device-management-toolkit/console/pkg/logger"
)

var errEmptyConnectionURI = errors.New("empty connection URI")

// Database name used by the console when running against Mongo.
const DatabaseName = "consoledb"

// Collection names — mirror the SQL table names.
const (
	CollectionDevices            = "devices"
	CollectionProfiles           = "profiles"
	CollectionCIRAConfigs        = "ciraconfigs"
	CollectionDomains            = "domains"
	CollectionIEEE8021xConfigs   = "ieee8021xconfigs"
	CollectionWirelessConfigs    = "wirelessconfigs"
	CollectionProfileWiFiConfigs = "profiles_wirelessconfigs"
)

// Connect dials MongoDB, pings it, and creates the unique indexes that
// enforce the app-level uniqueness SQL used to get from UNIQUE constraints.
// The returned *mongo.Client owns the connection pool; the caller disconnects
// it on shutdown.
func Connect(ctx context.Context, uri string, log logger.Interface) (*mongo.Client, *mongo.Database, error) {
	if uri == "" {
		return nil, nil, fmt.Errorf("mongo.Connect: %w", errEmptyConnectionURI)
	}

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, nil, fmt.Errorf("mongo.Connect: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(ctx)

		return nil, nil, fmt.Errorf("mongo.Connect: ping: %w", err)
	}

	db := client.Database(DatabaseName)

	if err := ensureIndexes(ctx, db, log); err != nil {
		_ = client.Disconnect(ctx)

		return nil, nil, fmt.Errorf("mongo.Connect: ensureIndexes: %w", err)
	}

	log.Info("mongo connected: db=%s", DatabaseName)

	return client, db, nil
}

// ensureIndexes creates the minimal set of unique indexes the SQL schema
// relies on. Safe to call repeatedly — Mongo makes CreateOne idempotent when
// the key+options match an existing index.
func ensureIndexes(ctx context.Context, db *mongo.Database, log logger.Interface) error {
	type idx struct {
		coll string
		keys bson.D
	}

	tenantScoped := []idx{
		{CollectionDevices, bson.D{{Key: "guid", Value: 1}, {Key: "tenantid", Value: 1}}},
		{CollectionProfiles, bson.D{{Key: "profilename", Value: 1}, {Key: "tenantid", Value: 1}}},
		{CollectionCIRAConfigs, bson.D{{Key: "configname", Value: 1}, {Key: "tenantid", Value: 1}}},
		{CollectionDomains, bson.D{{Key: "profilename", Value: 1}, {Key: "tenantid", Value: 1}}},
		{CollectionIEEE8021xConfigs, bson.D{{Key: "profilename", Value: 1}, {Key: "tenantid", Value: 1}}},
		{CollectionWirelessConfigs, bson.D{{Key: "profilename", Value: 1}, {Key: "tenantid", Value: 1}}},
		{CollectionProfileWiFiConfigs, bson.D{
			{Key: "profilename", Value: 1},
			{Key: "wirelessprofilename", Value: 1},
			{Key: "tenantid", Value: 1},
		}},
	}

	for _, i := range tenantScoped {
		_, err := db.Collection(i.coll).Indexes().CreateOne(ctx, mongo.IndexModel{
			Keys:    i.keys,
			Options: options.Index().SetUnique(true),
		})
		if err != nil {
			return fmt.Errorf("create unique index on %s: %w", i.coll, err)
		}
	}

	log.Info("mongo unique indexes ensured on %d collections", len(tenantScoped))

	return nil
}
