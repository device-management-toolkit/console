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

	md.AddResponses(deleteResponse(1))

	repo := mongo.NewWirelessRepo(db, logger.New("error"))

	ok, err := repo.Delete(context.Background(), "wifi1", "t1")
	require.NoError(t, err)
	require.True(t, ok)
}
