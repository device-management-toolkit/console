package mongo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/device-management-toolkit/console/pkg/logger"
)

const (
	// CollectionMigrations records the one-time data migrations that have run.
	CollectionMigrations = "migrations"

	// migrationDiscoveredActivated mirrors the SQL migration of the same name.
	migrationDiscoveredActivated = "20261119000000_migrate_discovered_activated_to_managed"

	// migrationStatusDone marks a migration that finished.
	migrationStatusDone = "done"

	opNin       = "$nin"
	opNot       = "$not"
	opCond      = "$cond"
	opReplaceAl = "$replaceAll"
)

// MigrateDiscoveredActivatedToManaged is the Mongo counterpart of the SQL migration
// 20261119000000_migrate_discovered_activated_to_managed.
//
// A device found by discovery now stays Discovered whatever its activation state.
// Before, a discovered=true device already in ACM/CCM was listed as Managed. Without
// this step those devices would move to Discovered when Console is upgraded on Mongo,
// but not on SQLite or Postgres. Both copies of the flag are updated: the discovered
// field is what the lists filter on, and the deviceinfo JSON is what the next update
// reads back and would otherwise write over the field again.
//
// It must not run again once it has finished: after the upgrade, discovered=true with
// an activated mode is a legitimate Discovered device and must not be flipped. A marker
// document in the migrations collection is written only after the update succeeds, and
// a marker in the done state makes every later start skip the update. If the update or
// the marker write fails, no done marker exists, startup fails, and the next start runs
// the update again. That is safe, because nothing serves requests before this returns,
// so no device can yet have been left Discovered on purpose.
//
// Two instances that start at the same time may both run the update. That is harmless
// while neither has finished, since the update gives the same result each time.
func MigrateDiscoveredActivatedToManaged(ctx context.Context, db *mongo.Database, log logger.Interface) error {
	migrations := db.Collection(CollectionMigrations)
	markerID := bson.D{{Key: "_id", Value: migrationDiscoveredActivated}}

	var marker struct {
		Status string `bson:"status"`
	}

	err := migrations.FindOne(ctx, markerID).Decode(&marker)

	switch {
	case err == nil && marker.Status == migrationStatusDone:
		return nil
	case err != nil && !errors.Is(err, mongo.ErrNoDocuments):
		return fmt.Errorf("mongo.MigrateDiscoveredActivatedToManaged: read marker: %w", err)
	}

	// An activated mode is a non-empty one other than "not activated". $nin also
	// excludes a missing field.
	filter := bson.D{
		{Key: fieldDiscovered, Value: true},
		{Key: fieldCurrentMode, Value: bson.D{
			{Key: opNin, Value: bson.A{"", nil}},
			{Key: opNot, Value: bson.Regex{Pattern: "^not activated$", Options: "i"}},
		}},
	}

	// A pipeline update, so the JSON text copy can be rewritten from its own value.
	// A missing or non-string deviceinfo is left as it is.
	update := mongo.Pipeline{{{Key: opSet, Value: bson.D{
		{Key: fieldDiscovered, Value: false},
		{Key: "deviceinfo", Value: bson.D{{Key: opCond, Value: bson.D{
			{Key: "if", Value: bson.D{{Key: "$eq", Value: bson.A{bson.D{{Key: "$type", Value: "$deviceinfo"}}, "string"}}}},
			{Key: "then", Value: bson.D{{Key: opReplaceAl, Value: bson.D{
				{Key: "input", Value: "$deviceinfo"},
				{Key: "find", Value: `"discovered":true`},
				{Key: "replacement", Value: `"discovered":false`},
			}}}},
			{Key: "else", Value: "$deviceinfo"},
		}}}},
	}}}}

	res, err := db.Collection(CollectionDevices).UpdateMany(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("mongo.MigrateDiscoveredActivatedToManaged: update: %w", err)
	}

	// Recorded only now, so a failure above leaves no done marker and the next start retries.
	_, err = migrations.UpdateOne(ctx, markerID, bson.D{{Key: opSet, Value: bson.D{
		{Key: "status", Value: migrationStatusDone},
		{Key: "appliedat", Value: time.Now().UTC()},
		{Key: "modified", Value: res.ModifiedCount},
	}}}, options.UpdateOne().SetUpsert(true))
	if err != nil {
		return fmt.Errorf("mongo.MigrateDiscoveredActivatedToManaged: write marker: %w", err)
	}

	log.Info("mongo migration %s: %d activated discovered device(s) kept Managed", migrationDiscoveredActivated, res.ModifiedCount)

	return nil
}
