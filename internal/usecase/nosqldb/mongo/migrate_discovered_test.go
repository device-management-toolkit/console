package mongo_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
	mongodriver "go.mongodb.org/mongo-driver/v2/mongo"

	mongo "github.com/device-management-toolkit/console/internal/usecase/nosqldb/mongo"
	"github.com/device-management-toolkit/console/pkg/logger"
)

const migrationID = "20261119000000_migrate_discovered_activated_to_managed"

// commandErrorResponse mimics a server error reply, which the driver returns as a command error.
func commandErrorResponse() bson.D {
	return bson.D{
		{Key: "ok", Value: 0},
		{Key: "errmsg", Value: "boom"},
		{Key: "code", Value: int32(1)},
	}
}

func markerResponse(status string) bson.D {
	return findResponse("testdb."+mongo.CollectionMigrations, bson.D{
		{Key: "_id", Value: migrationID},
		{Key: "status", Value: status},
	})
}

func TestMigrateDiscoveredActivatedToManaged(t *testing.T) {
	t.Parallel()

	run := func(db *mongodriver.Database) error {
		return mongo.MigrateDiscoveredActivatedToManaged(context.Background(), db, logger.New("error"))
	}

	t.Run("updates the devices, then records the marker", func(t *testing.T) {
		t.Parallel()

		db, md := newMockedDB(t)
		md.AddResponses(
			findResponse("testdb."+mongo.CollectionMigrations),
			updateResponse(2),
			updateResponse(1),
		)

		require.NoError(t, run(db))
	})

	t.Run("does nothing once the marker is done", func(t *testing.T) {
		t.Parallel()

		db, md := newMockedDB(t)
		// Only the marker read is answered. An update after it would find no queued reply and fail.
		md.AddResponses(markerResponse("done"))

		require.NoError(t, run(db))
	})

	t.Run("runs again when the marker exists but is not done", func(t *testing.T) {
		t.Parallel()

		db, md := newMockedDB(t)
		md.AddResponses(markerResponse("running"), updateResponse(0), updateResponse(1))

		require.NoError(t, run(db))
	})

	t.Run("fails and writes no marker when the update fails", func(t *testing.T) {
		t.Parallel()

		db, md := newMockedDB(t)
		// No reply is queued for a marker write, so one would fail the test with a different error.
		md.AddResponses(findResponse("testdb."+mongo.CollectionMigrations), commandErrorResponse())

		err := run(db)
		require.Error(t, err)
		require.ErrorContains(t, err, "update:")
	})

	t.Run("fails when the marker cannot be written", func(t *testing.T) {
		t.Parallel()

		db, md := newMockedDB(t)
		md.AddResponses(findResponse("testdb."+mongo.CollectionMigrations), updateResponse(2), commandErrorResponse())

		err := run(db)
		require.Error(t, err)
		require.ErrorContains(t, err, "write marker")
	})

	t.Run("fails when the marker cannot be read", func(t *testing.T) {
		t.Parallel()

		db, md := newMockedDB(t)
		md.AddResponses(commandErrorResponse())

		err := run(db)
		require.Error(t, err)
		require.ErrorContains(t, err, "read marker")
	})
}
