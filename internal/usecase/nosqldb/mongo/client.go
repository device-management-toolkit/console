// Package mongo implements every *.Repository interface against MongoDB.
// Entities carry explicit `bson:"<lowercase>"` tags; constants in fields.go
// mirror them. Unique indexes here stand in for the SQL UNIQUE constraints.
// Errors are reused from `repoerrors` so use cases' `errors.As` checks work.
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

const DatabaseName = "consoledb"

// DefaultTop is the page size used when a caller passes top<=0 (mirrors sqldb).
const DefaultTop = 100

const (
	CollectionDevices            = "devices"
	CollectionProfiles           = "profiles"
	CollectionCIRAConfigs        = "ciraconfigs"
	CollectionDomains            = "domains"
	CollectionIEEE8021xConfigs   = "ieee8021xconfigs"
	CollectionWirelessConfigs    = "wirelessconfigs"
	CollectionProfileWiFiConfigs = "profiles_wirelessconfigs"
)

// Connect dials Mongo, pings, and creates the unique indexes that stand in
// for the SQL UNIQUE constraints. Caller disconnects the returned client.
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

// ensureIndexes creates the unique indexes the SQL schema relies on.
// Idempotent — safe to call on every startup.
func ensureIndexes(ctx context.Context, db *mongo.Database, log logger.Interface) error {
	type idx struct {
		coll string
		keys bson.D
	}

	tenantScoped := []idx{
		{CollectionDevices, bson.D{{Key: fieldGUID, Value: 1}, {Key: fieldTenantID, Value: 1}}},
		{CollectionProfiles, bson.D{{Key: fieldProfileName, Value: 1}, {Key: fieldTenantID, Value: 1}}},
		{CollectionCIRAConfigs, bson.D{{Key: fieldConfigName, Value: 1}, {Key: fieldTenantID, Value: 1}}},
		{CollectionDomains, bson.D{{Key: fieldProfileName, Value: 1}, {Key: fieldTenantID, Value: 1}}},
		// SQL also enforces UNIQUE(domain_suffix, tenant_id).
		{CollectionDomains, bson.D{{Key: fieldDomainSuffix, Value: 1}, {Key: fieldTenantID, Value: 1}}},
		{CollectionIEEE8021xConfigs, bson.D{{Key: fieldProfileName, Value: 1}, {Key: fieldTenantID, Value: 1}}},
		{CollectionWirelessConfigs, bson.D{{Key: fieldProfileName, Value: 1}, {Key: fieldTenantID, Value: 1}}},
		// SQL PK includes priority — multiple link rows per (profile, wifi, tenant) at different priorities are valid.
		{CollectionProfileWiFiConfigs, bson.D{
			{Key: fieldProfileName, Value: 1},
			{Key: "wirelessprofilename", Value: 1},
			{Key: "priority", Value: 1},
			{Key: fieldTenantID, Value: 1},
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

	// Mirrors SQL's lower_name_suffix_idx — global, case-insensitive
	// (collation strength 2 = LOWER() equivalent).
	if _, err := db.Collection(CollectionDomains).Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: fieldProfileName, Value: 1}, {Key: fieldDomainSuffix, Value: 1}},
		Options: options.Index().
			SetUnique(true).
			SetCollation(&options.Collation{Locale: "en", Strength: 2}).
			SetName("lower_name_suffix_idx"),
	}); err != nil {
		return fmt.Errorf("create case-insensitive unique index on %s: %w", CollectionDomains, err)
	}

	log.Info("mongo unique indexes ensured (%d total)", len(tenantScoped)+1)

	return nil
}
