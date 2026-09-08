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
	"github.com/device-management-toolkit/console/pkg/logger"
)

func TestWirelessRepo_CheckProfileExists_True(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	md.AddResponses(findResponse(
		"testdb."+mongo.CollectionWirelessConfigs,
		bson.D{{Key: "n", Value: int64(1)}},
	))

	repo := mongo.NewWirelessRepo(db, logger.New("error"))

	got, err := repo.CheckProfileExists(context.Background(), "wifi1", "t1")
	require.NoError(t, err)
	require.True(t, got)
}

func TestWirelessRepo_CheckProfileExists_False(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	md.AddResponses(findResponse(
		"testdb."+mongo.CollectionWirelessConfigs,
		bson.D{{Key: "n", Value: int64(0)}},
	))

	repo := mongo.NewWirelessRepo(db, logger.New("error"))

	got, err := repo.CheckProfileExists(context.Background(), "ghost", "t1")
	require.NoError(t, err)
	require.False(t, got)
}

func TestWirelessRepo_GetCount(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	md.AddResponses(findResponse(
		"testdb."+mongo.CollectionWirelessConfigs,
		bson.D{{Key: "n", Value: int64(2)}},
	))

	repo := mongo.NewWirelessRepo(db, logger.New("error"))

	got, err := repo.GetCount(context.Background(), "t1")
	require.NoError(t, err)
	require.Equal(t, 2, got)
}

func TestWirelessRepo_GetByName_NoIEEE8021x(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	md.AddResponses(findResponse(
		"testdb."+mongo.CollectionWirelessConfigs,
		bson.D{
			{Key: "profilename", Value: "wifi1"},
			{Key: "ssid", Value: "lab-net"},
			{Key: "tenantid", Value: "t1"},
		},
	))

	repo := mongo.NewWirelessRepo(db, logger.New("error"))

	got, err := repo.GetByName(context.Background(), "wifi1", "t1")
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, "lab-net", got.SSID)
	require.Nil(t, got.AuthenticationProtocol)
}

// The wireless populate8021x uses a wired_interface=false filter on the
// secondary lookup. That filter is set on the bson.M passed to FindOne; the
// wire-level mock returns whatever we queue, so we can't directly assert the
// filter shape, but we CAN confirm the repo issues the secondary lookup and
// decodes its result correctly.
func TestWirelessRepo_GetByName_PopulatesFrom8021x(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	md.AddResponses(
		findResponse(
			"testdb."+mongo.CollectionWirelessConfigs,
			bson.D{
				{Key: "profilename", Value: "wifi1"},
				{Key: "ieee8021xprofilename", Value: "ieee1"},
				{Key: "tenantid", Value: "t1"},
			},
		),
		findResponse(
			"testdb."+mongo.CollectionIEEE8021xConfigs,
			bson.D{
				{Key: "profilename", Value: "ieee1"},
				{Key: "authenticationprotocol", Value: int32(5)},
				{Key: "wiredinterface", Value: false},
				{Key: "tenantid", Value: "t1"},
			},
		),
	)

	repo := mongo.NewWirelessRepo(db, logger.New("error"))

	got, err := repo.GetByName(context.Background(), "wifi1", "t1")
	require.NoError(t, err)
	require.NotNil(t, got)
	require.NotNil(t, got.AuthenticationProtocol)
	require.Equal(t, 5, *got.AuthenticationProtocol)
	require.NotNil(t, got.WiredInterface)
	require.False(t, *got.WiredInterface)
}

func TestWirelessRepo_Insert(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	md.AddResponses(insertResponse())

	repo := mongo.NewWirelessRepo(db, logger.New("error"))

	_, err := repo.Insert(context.Background(), &entity.WirelessConfig{
		ProfileName: "wifi1",
		TenantID:    "t1",
	})
	require.NoError(t, err)
}

func TestWirelessRepo_Insert_DuplicateReturnsNotUniqueError(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	md.AddResponses(duplicateKeyResponse())

	repo := mongo.NewWirelessRepo(db, logger.New("error"))

	_, err := repo.Insert(context.Background(), &entity.WirelessConfig{
		ProfileName: "wifi1",
		TenantID:    "t1",
	})
	require.Error(t, err)

	var nu repoerrors.NotUniqueError
	require.True(t, errors.As(err, &nu))
}

// A unique-index collision on update has to reach the handler as a
// NotUniqueError so it answers 409, the way the SQL backends do.
func TestWirelessRepo_Update_DuplicateReturnsNotUniqueError(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	md.AddResponses(duplicateKeyResponse())

	repo := mongo.NewWirelessRepo(db, logger.New("error"))

	ok, err := repo.Update(context.Background(), &entity.WirelessConfig{
		ProfileName: "wifi1",
		TenantID:    "t1",
	})
	require.False(t, ok)
	require.Error(t, err)

	var notUnique repoerrors.NotUniqueError

	require.ErrorAs(t, err, &notUnique)
}

func TestWirelessRepo_Update(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	md.AddResponses(updateResponse(1))

	repo := mongo.NewWirelessRepo(db, logger.New("error"))

	ok, err := repo.Update(context.Background(), &entity.WirelessConfig{
		ProfileName: "wifi1",
		TenantID:    "t1",
	})
	require.NoError(t, err)
	require.True(t, ok)
}

func TestWirelessRepo_Delete(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	// No referencing profiles_wirelessconfigs row, then the delete itself.
	md.AddResponses(findResponse("consoledb.profiles_wirelessconfigs"), deleteResponse(1))

	repo := mongo.NewWirelessRepo(db, logger.New("error"))

	ok, err := repo.Delete(context.Background(), "wifi1", "t1")
	require.NoError(t, err)
	require.True(t, ok)
}

// A failed reference lookup must not fall through to the delete: the repo cannot
// tell whether the wireless profile is still in use, so it reports the error.
func TestWirelessRepo_Delete_ReferenceLookupFailurePreventsDelete(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	// Only one queued response: a delete would need a second, and reaching it
	// would hang rather than silently pass.
	md.AddResponses(bson.D{
		{Key: "ok", Value: 0},
		{Key: "code", Value: int32(13)},
		{Key: "errmsg", Value: "not authorized"},
	})

	repo := mongo.NewWirelessRepo(db, logger.New("error"))

	ok, err := repo.Delete(context.Background(), "wifi1", "t1")
	require.False(t, ok)
	require.Error(t, err)

	var dbErr repoerrors.DatabaseError

	require.ErrorAs(t, err, &dbErr)
}

// SQL gets this from the profiles_wirelessconfigs foreign key; Mongo has to look
// for the referencing row itself, and must raise the same error so the handler
// still answers 400.
func TestWirelessRepo_Delete_ReferencedByProfileIsRejected(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	md.AddResponses(findResponse("consoledb.profiles_wirelessconfigs",
		bson.D{{Key: "profilename", Value: "amt-profile"}, {Key: "wirelessprofilename", Value: "wifi1"}},
	))

	repo := mongo.NewWirelessRepo(db, logger.New("error"))

	ok, err := repo.Delete(context.Background(), "wifi1", "t1")
	require.False(t, ok)
	require.Error(t, err)

	var fkErr repoerrors.ForeignKeyViolationError

	require.ErrorAs(t, err, &fkErr)
	// FriendlyMessage is what the handler puts in the 400 body.
	require.Contains(t, fkErr.Console.FriendlyMessage(), "foreign key violation")
}
