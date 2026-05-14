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

func TestProfileWiFiConfigsRepo_GetByProfileName(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	// The repo sorts by priority ascending; the mock returns whatever we
	// queue, which lets us verify the entity decoding.
	md.AddResponses(findResponse(
		"testdb."+mongo.CollectionProfileWiFiConfigs,
		bson.D{
			{Key: "profilename", Value: "p1"},
			{Key: "wirelessprofilename", Value: "w-first"},
			{Key: "priority", Value: int32(1)},
			{Key: "tenantid", Value: "t1"},
		},
		bson.D{
			{Key: "profilename", Value: "p1"},
			{Key: "wirelessprofilename", Value: "w-second"},
			{Key: "priority", Value: int32(2)},
			{Key: "tenantid", Value: "t1"},
		},
	))

	repo := mongo.NewProfileWiFiConfigsRepo(db)

	rows, err := repo.GetByProfileName(context.Background(), "p1", "t1")
	require.NoError(t, err)
	require.Len(t, rows, 2)
	require.Equal(t, "w-first", rows[0].WirelessProfileName)
	require.Equal(t, 1, rows[0].Priority)
	require.Equal(t, "w-second", rows[1].WirelessProfileName)
}

func TestProfileWiFiConfigsRepo_GetByProfileName_Empty(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	md.AddResponses(findResponse("testdb." + mongo.CollectionProfileWiFiConfigs))

	repo := mongo.NewProfileWiFiConfigsRepo(db)

	rows, err := repo.GetByProfileName(context.Background(), "p1", "t1")
	require.NoError(t, err)
	require.Empty(t, rows)
}

func TestProfileWiFiConfigsRepo_DeleteByProfileName_RemovesRows(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	md.AddResponses(deleteResponse(3))

	repo := mongo.NewProfileWiFiConfigsRepo(db)

	ok, err := repo.DeleteByProfileName(context.Background(), "p1", "t1")
	require.NoError(t, err)
	require.True(t, ok)
}

func TestProfileWiFiConfigsRepo_DeleteByProfileName_NoMatch(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	md.AddResponses(deleteResponse(0))

	repo := mongo.NewProfileWiFiConfigsRepo(db)

	ok, err := repo.DeleteByProfileName(context.Background(), "ghost", "t1")
	require.NoError(t, err)
	require.False(t, ok)
}

func TestProfileWiFiConfigsRepo_Insert(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	md.AddResponses(insertResponse())

	repo := mongo.NewProfileWiFiConfigsRepo(db)

	_, err := repo.Insert(context.Background(), &entity.ProfileWiFiConfigs{
		ProfileName:         "p1",
		WirelessProfileName: "w1",
		Priority:            1,
		TenantID:            "t1",
	})
	require.NoError(t, err)
}

func TestProfileWiFiConfigsRepo_Insert_DuplicateReturnsNotUniqueError(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	md.AddResponses(duplicateKeyResponse())

	repo := mongo.NewProfileWiFiConfigsRepo(db)

	_, err := repo.Insert(context.Background(), &entity.ProfileWiFiConfigs{
		ProfileName:         "p1",
		WirelessProfileName: "w1",
		TenantID:            "t1",
	})
	require.Error(t, err)

	var nu repoerrors.NotUniqueError
	require.True(t, errors.As(err, &nu))
}
