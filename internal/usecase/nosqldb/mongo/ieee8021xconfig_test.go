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

func TestIEEE8021xRepo_CheckProfileExists_True(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	md.AddResponses(findResponse(
		"testdb."+mongo.CollectionIEEE8021xConfigs,
		bson.D{{Key: "n", Value: int64(1)}},
	))

	repo := mongo.NewIEEE8021xRepo(db)

	got, err := repo.CheckProfileExists(context.Background(), "ieee1", "t1")
	require.NoError(t, err)
	require.True(t, got)
}

func TestIEEE8021xRepo_CheckProfileExists_False(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	md.AddResponses(findResponse(
		"testdb."+mongo.CollectionIEEE8021xConfigs,
		bson.D{{Key: "n", Value: int64(0)}},
	))

	repo := mongo.NewIEEE8021xRepo(db)

	got, err := repo.CheckProfileExists(context.Background(), "ghost", "t1")
	require.NoError(t, err)
	require.False(t, got)
}

func TestIEEE8021xRepo_GetCount(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	md.AddResponses(findResponse(
		"testdb."+mongo.CollectionIEEE8021xConfigs,
		bson.D{{Key: "n", Value: int64(2)}},
	))

	repo := mongo.NewIEEE8021xRepo(db)

	got, err := repo.GetCount(context.Background(), "t1")
	require.NoError(t, err)
	require.Equal(t, 2, got)
}

func TestIEEE8021xRepo_GetByName(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	md.AddResponses(findResponse(
		"testdb."+mongo.CollectionIEEE8021xConfigs,
		bson.D{
			{Key: "profilename", Value: "ieee1"},
			{Key: "authenticationprotocol", Value: int32(4)},
			{Key: "wiredinterface", Value: true},
			{Key: "tenantid", Value: "t1"},
		},
	))

	repo := mongo.NewIEEE8021xRepo(db)

	got, err := repo.GetByName(context.Background(), "ieee1", "t1")
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, "ieee1", got.ProfileName)
	require.Equal(t, 4, got.AuthenticationProtocol)
	require.True(t, got.WiredInterface)
}

func TestIEEE8021xRepo_GetByName_NotFound(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	md.AddResponses(findResponse("testdb." + mongo.CollectionIEEE8021xConfigs))

	repo := mongo.NewIEEE8021xRepo(db)

	got, err := repo.GetByName(context.Background(), "ghost", "t1")
	require.NoError(t, err)
	require.Nil(t, got)
}

func TestIEEE8021xRepo_Get(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	md.AddResponses(findResponse(
		"testdb."+mongo.CollectionIEEE8021xConfigs,
		bson.D{{Key: "profilename", Value: "ieee1"}, {Key: "tenantid", Value: "t1"}},
		bson.D{{Key: "profilename", Value: "ieee2"}, {Key: "tenantid", Value: "t1"}},
	))

	repo := mongo.NewIEEE8021xRepo(db)

	rows, err := repo.Get(context.Background(), 10, 0, "t1")
	require.NoError(t, err)
	require.Len(t, rows, 2)
}

func TestIEEE8021xRepo_Insert(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	md.AddResponses(insertResponse())

	repo := mongo.NewIEEE8021xRepo(db)

	_, err := repo.Insert(context.Background(), &entity.IEEE8021xConfig{
		ProfileName: "ieee1",
		TenantID:    "t1",
	})
	require.NoError(t, err)
}

func TestIEEE8021xRepo_Insert_DuplicateReturnsNotUniqueError(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	md.AddResponses(duplicateKeyResponse())

	repo := mongo.NewIEEE8021xRepo(db)

	_, err := repo.Insert(context.Background(), &entity.IEEE8021xConfig{
		ProfileName: "ieee1",
		TenantID:    "t1",
	})
	require.Error(t, err)

	var nu repoerrors.NotUniqueError
	require.True(t, errors.As(err, &nu))
}

func TestIEEE8021xRepo_Update(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	md.AddResponses(updateResponse(1))

	repo := mongo.NewIEEE8021xRepo(db)

	ok, err := repo.Update(context.Background(), &entity.IEEE8021xConfig{
		ProfileName: "ieee1",
		TenantID:    "t1",
	})
	require.NoError(t, err)
	require.True(t, ok)
}

func TestIEEE8021xRepo_Delete(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	md.AddResponses(deleteResponse(1))

	repo := mongo.NewIEEE8021xRepo(db)

	ok, err := repo.Delete(context.Background(), "ieee1", "t1")
	require.NoError(t, err)
	require.True(t, ok)
}
