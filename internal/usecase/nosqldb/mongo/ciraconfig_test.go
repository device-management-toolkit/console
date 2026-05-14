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

func TestCIRARepo_GetCount(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	md.AddResponses(
		findResponse("testdb."+mongo.CollectionCIRAConfigs, bson.D{{Key: "n", Value: int64(3)}}),
	)

	repo := mongo.NewCIRARepo(db)

	got, err := repo.GetCount(context.Background(), "t1")
	require.NoError(t, err)
	require.Equal(t, 3, got)
}

func TestCIRARepo_GetByName(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	md.AddResponses(findResponse(
		"testdb."+mongo.CollectionCIRAConfigs,
		bson.D{
			{Key: "configname", Value: "cira1"},
			{Key: "mpsaddress", Value: "mps.example.com"},
			{Key: "mpsport", Value: int32(4433)},
			{Key: "tenantid", Value: "t1"},
		},
	))

	repo := mongo.NewCIRARepo(db)

	got, err := repo.GetByName(context.Background(), "cira1", "t1")
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, "cira1", got.ConfigName)
	require.Equal(t, "mps.example.com", got.MPSAddress)
	require.Equal(t, 4433, got.MPSPort)
}

func TestCIRARepo_GetByName_NotFound(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	// Empty firstBatch: driver decodes nothing, FindOne returns ErrNoDocuments,
	// which the repo translates into (nil, nil).
	md.AddResponses(findResponse("testdb." + mongo.CollectionCIRAConfigs))

	repo := mongo.NewCIRARepo(db)

	got, err := repo.GetByName(context.Background(), "ghost", "t1")
	require.NoError(t, err)
	require.Nil(t, got)
}

func TestCIRARepo_Insert(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	md.AddResponses(insertResponse())

	repo := mongo.NewCIRARepo(db)

	_, err := repo.Insert(context.Background(), &entity.CIRAConfig{
		ConfigName: "cira1",
		TenantID:   "t1",
	})
	require.NoError(t, err)
}

func TestCIRARepo_Insert_DuplicateReturnsNotUniqueError(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	md.AddResponses(duplicateKeyResponse())

	repo := mongo.NewCIRARepo(db)

	_, err := repo.Insert(context.Background(), &entity.CIRAConfig{
		ConfigName: "cira1",
		TenantID:   "t1",
	})
	require.Error(t, err)

	var nu repoerrors.NotUniqueError
	require.True(t, errors.As(err, &nu), "expected NotUniqueError, got %T: %v", err, err)
}

func TestCIRARepo_Update_Matched(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	md.AddResponses(updateResponse(1))

	repo := mongo.NewCIRARepo(db)

	ok, err := repo.Update(context.Background(), &entity.CIRAConfig{
		ConfigName: "cira1",
		TenantID:   "t1",
	})
	require.NoError(t, err)
	require.True(t, ok)
}

func TestCIRARepo_Update_NoMatch(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	md.AddResponses(updateResponse(0))

	repo := mongo.NewCIRARepo(db)

	ok, err := repo.Update(context.Background(), &entity.CIRAConfig{
		ConfigName: "ghost",
		TenantID:   "t1",
	})
	require.NoError(t, err)
	require.False(t, ok)
}

func TestCIRARepo_Delete_Matched(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	md.AddResponses(deleteResponse(1))

	repo := mongo.NewCIRARepo(db)

	ok, err := repo.Delete(context.Background(), "cira1", "t1")
	require.NoError(t, err)
	require.True(t, ok)
}

func TestCIRARepo_Delete_NoMatch(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	md.AddResponses(deleteResponse(0))

	repo := mongo.NewCIRARepo(db)

	ok, err := repo.Delete(context.Background(), "ghost", "t1")
	require.NoError(t, err)
	require.False(t, ok)
}
