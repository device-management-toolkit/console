package mongo_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/device-management-toolkit/console/internal/entity"
	"github.com/device-management-toolkit/console/internal/repoerrors"
	mongo "github.com/device-management-toolkit/console/internal/usecase/nosqldb/mongo"
)

func TestDeviceRepo_GetCount(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	md.AddResponses(findResponse(
		"testdb."+mongo.CollectionDevices,
		bson.D{{Key: "n", Value: int64(2)}},
	))

	repo := mongo.NewDeviceRepo(db)

	got, err := repo.GetCount(context.Background(), "t1")
	require.NoError(t, err)
	require.Equal(t, 2, got)
}

func TestDeviceRepo_GetByID_Found(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	md.AddResponses(findResponse(
		"testdb."+mongo.CollectionDevices,
		bson.D{
			{Key: "guid", Value: "g1"},
			{Key: "friendlyname", Value: "lab-host-1"},
			{Key: "tenantid", Value: "t1"},
		},
	))

	repo := mongo.NewDeviceRepo(db)

	got, err := repo.GetByID(context.Background(), "g1", "t1")
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, "g1", got.GUID)
	require.Equal(t, "lab-host-1", got.FriendlyName)
}

func TestDeviceRepo_GetByID_IdentityColumns(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	md.AddResponses(findResponse(
		"testdb."+mongo.CollectionDevices,
		bson.D{
			{Key: "guid", Value: "g1"},
			{Key: "tenantid", Value: "t1"},
			{Key: "id", Value: "11111111-2222-3333-4444-555555555555"},
			{Key: "createddate", Value: "2026-05-26T12:00:00Z"},
			{Key: "isdeleted", Value: true},
			{Key: "deleteddate", Value: "2026-05-27T08:00:00Z"},
			{Key: "producttype", Value: "vpro"},
			{Key: "connectiontype", Value: "CIRA"},
		},
	))

	repo := mongo.NewDeviceRepo(db)

	got, err := repo.GetByID(context.Background(), "g1", "t1")
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, "11111111-2222-3333-4444-555555555555", got.ID)
	require.Equal(t, "2026-05-26T12:00:00Z", got.CreatedDate)
	require.True(t, got.IsDeleted)
	require.Equal(t, "2026-05-27T08:00:00Z", got.DeletedDate)
	require.Equal(t, "vpro", got.ProductType)
	require.Equal(t, "CIRA", got.ConnectionType)
}

// updateSetKeys extracts the key set of the $set document from a captured
// `update` command (shape: {update, updates:[{q, u:{$set:{...}}}]}).
func updateSetKeys(t *testing.T, cmd bson.Raw) map[string]struct{} {
	t.Helper()

	updates, err := cmd.LookupErr("updates")
	require.NoError(t, err)

	vals, err := updates.Array().Values()
	require.NoError(t, err)
	require.NotEmpty(t, vals)

	setVal, err := vals[0].Document().LookupErr("u", "$set")
	require.NoError(t, err)

	elems, err := setVal.Document().Elements()
	require.NoError(t, err)

	keys := make(map[string]struct{}, len(elems))
	for _, e := range elems {
		keys[e.Key()] = struct{}{}
	}

	return keys
}

// TestDeviceRepo_Update_OmitsImmutableIdentityColumns mirrors the sqldb
// regression test: even when an Update carries changed id/createddate/deleteddate
// values, the Mongo $set must omit them so the immutable columns can never be mutated,
// while the mutable identity columns are still written.
func TestDeviceRepo_Update_OmitsImmutableIdentityColumns(t *testing.T) {
	t.Parallel()

	db, md, cc := newMonitoredDB(t)
	md.AddResponses(updateResponse(1))

	repo := mongo.NewDeviceRepo(db)

	ok, err := repo.Update(context.Background(), &entity.Device{
		GUID:           "g1",
		TenantID:       "t1",
		ID:             "tampered-id",
		CreatedDate:    "2099-01-01T00:00:00Z",
		DeletedDate:    "2099-01-01T00:00:00Z",
		ProductType:    "vpro",
		ConnectionType: "CIRA",
	})
	require.NoError(t, err)
	require.True(t, ok)

	cmd, found := cc.byName("update")
	require.True(t, found, "expected an update command to be sent")

	set := updateSetKeys(t, cmd)

	// Immutable: must never appear in $set.
	require.NotContains(t, set, "id", "id must not be writable via Update")
	require.NotContains(t, set, "createddate", "createddate must not be writable via Update")
	require.NotContains(t, set, "deleteddate", "deleteddate must not be writable via Update")

	// Mutable identity columns: present so they round-trip on Update.
	require.Contains(t, set, "isdeleted")
	require.Contains(t, set, "producttype")
	require.Contains(t, set, "connectiontype")
}

func TestDeviceRepo_GetByID_NotFound(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	md.AddResponses(findResponse("testdb." + mongo.CollectionDevices))

	repo := mongo.NewDeviceRepo(db)

	got, err := repo.GetByID(context.Background(), "ghost", "t1")
	require.NoError(t, err)
	require.Nil(t, got)
}

func TestDeviceRepo_Get(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	md.AddResponses(findResponse(
		"testdb."+mongo.CollectionDevices,
		bson.D{{Key: "guid", Value: "g1"}, {Key: "tenantid", Value: "t1"}},
		bson.D{{Key: "guid", Value: "g2"}, {Key: "tenantid", Value: "t1"}},
	))

	repo := mongo.NewDeviceRepo(db)

	rows, err := repo.Get(context.Background(), 10, 0, "t1")
	require.NoError(t, err)
	require.Len(t, rows, 2)
}

// GetDistinctTags issues the `distinct` command, then de-duplicates and
// trims tags in Go. The test asserts the post-driver in-process logic.
func TestDeviceRepo_GetDistinctTags_DeduplicatesAcrossRows(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	md.AddResponses(distinctResponse("lab,gpu", "lab,cpu", "  gpu , cpu  ", ""))

	repo := mongo.NewDeviceRepo(db)

	tags, err := repo.GetDistinctTags(context.Background(), "t1")
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"lab", "gpu", "cpu"}, tags)
}

// HTTP layer passes column names like "HostName", "FriendlyName" matching the
// entity field name. Mongo's default codec lowercases those into BSON keys
// ("hostname", "friendlyname"), and BSON keys are case-sensitive — so the
// repo must lowercase the column before building the filter. SQL gets away
// without this because Postgres folds unquoted identifiers to lowercase.
func TestDeviceRepo_GetByColumn_LowercasesColumnName(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	md.AddResponses(findResponse(
		"testdb."+mongo.CollectionDevices,
		bson.D{
			{Key: "guid", Value: "g1"},
			{Key: "friendlyname", Value: "lab-host-1"},
			{Key: "tenantid", Value: "t1"},
		},
	))

	repo := mongo.NewDeviceRepo(db)

	rows, err := repo.GetByColumn(context.Background(), "FriendlyName", "lab-host-1", "t1")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, "g1", rows[0].GUID)
}

func TestDeviceRepo_GetByTags_OR(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	md.AddResponses(findResponse(
		"testdb."+mongo.CollectionDevices,
		bson.D{{Key: "guid", Value: "g1"}, {Key: "tags", Value: "lab,gpu"}, {Key: "tenantid", Value: "t1"}},
	))

	repo := mongo.NewDeviceRepo(db)

	rows, err := repo.GetByTags(context.Background(), []string{"lab"}, "OR", 10, 0, "t1")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, "g1", rows[0].GUID)
}

func TestDeviceRepo_GetByTags_AND(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	// Mock returns whatever — the repo just decodes; we only assert the
	// caller wiring produces a working call.
	md.AddResponses(findResponse(
		"testdb."+mongo.CollectionDevices,
		bson.D{{Key: "guid", Value: "g1"}, {Key: "tenantid", Value: "t1"}},
	))

	repo := mongo.NewDeviceRepo(db)

	rows, err := repo.GetByTags(context.Background(), []string{"lab", "gpu"}, "AND", 10, 0, "t1")
	require.NoError(t, err)
	require.Len(t, rows, 1)
}

func TestDeviceRepo_Insert(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	md.AddResponses(insertResponse())

	repo := mongo.NewDeviceRepo(db)

	_, err := repo.Insert(context.Background(), &entity.Device{
		GUID:     "g1",
		TenantID: "t1",
	})
	require.NoError(t, err)
}

func TestDeviceRepo_Insert_DuplicateReturnsNotUniqueError(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	md.AddResponses(duplicateKeyResponse())

	repo := mongo.NewDeviceRepo(db)

	_, err := repo.Insert(context.Background(), &entity.Device{
		GUID:     "g1",
		TenantID: "t1",
	})
	require.Error(t, err)

	var nu repoerrors.NotUniqueError
	require.True(t, errors.As(err, &nu))
}

func TestDeviceRepo_Update_Matched(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	md.AddResponses(updateResponse(1))

	repo := mongo.NewDeviceRepo(db)

	ok, err := repo.Update(context.Background(), &entity.Device{
		GUID:     "g1",
		TenantID: "t1",
	})
	require.NoError(t, err)
	require.True(t, ok)
}

func TestDeviceRepo_Update_NoMatch(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	md.AddResponses(updateResponse(0))

	repo := mongo.NewDeviceRepo(db)

	ok, err := repo.Update(context.Background(), &entity.Device{
		GUID:     "ghost",
		TenantID: "t1",
	})
	require.NoError(t, err)
	require.False(t, ok)
}

func TestDeviceRepo_Delete_Matched(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	md.AddResponses(deleteResponse(1))

	repo := mongo.NewDeviceRepo(db)

	ok, err := repo.Delete(context.Background(), "g1", "t1")
	require.NoError(t, err)
	require.True(t, ok)
}

func TestDeviceRepo_Delete_NoMatch(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	md.AddResponses(deleteResponse(0))

	repo := mongo.NewDeviceRepo(db)

	ok, err := repo.Delete(context.Background(), "ghost", "t1")
	require.NoError(t, err)
	require.False(t, ok)
}
