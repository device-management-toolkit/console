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

func TestDomainRepo_GetByName_Found(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	md.AddResponses(findResponse(
		"testdb."+mongo.CollectionDomains,
		bson.D{
			{Key: "profilename", Value: "Acme"},
			{Key: "domainsuffix", Value: "acme.example.com"},
			{Key: "tenantid", Value: "t1"},
		},
	))

	repo := mongo.NewDomainRepo(db)

	got, err := repo.GetByName(context.Background(), "acme", "t1")
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, "Acme", got.ProfileName)
}

func TestDomainRepo_GetByName_NotFound(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	md.AddResponses(findResponse("testdb." + mongo.CollectionDomains))

	repo := mongo.NewDomainRepo(db)

	got, err := repo.GetByName(context.Background(), "ghost", "t1")
	require.NoError(t, err)
	require.Nil(t, got)
}

func TestDomainRepo_GetDomainByDomainSuffix(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	md.AddResponses(findResponse(
		"testdb."+mongo.CollectionDomains,
		bson.D{
			{Key: "profilename", Value: "Acme"},
			{Key: "domainsuffix", Value: "acme.example.com"},
			{Key: "tenantid", Value: "t1"},
		},
	))

	repo := mongo.NewDomainRepo(db)

	got, err := repo.GetDomainByDomainSuffix(context.Background(), "acme.example.com", "t1")
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, "Acme", got.ProfileName)
}

func TestDomainRepo_GetCount(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	// CountDocuments wires through an aggregation pipeline; the response
	// shape is a cursor reply with a single { n: <count> } doc.
	md.AddResponses(findResponse(
		"testdb."+mongo.CollectionDomains,
		bson.D{{Key: "n", Value: int64(7)}},
	))

	repo := mongo.NewDomainRepo(db)

	got, err := repo.GetCount(context.Background(), "t1")
	require.NoError(t, err)
	require.Equal(t, 7, got)
}

func TestDomainRepo_Get_ReturnsRows(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	md.AddResponses(findResponse(
		"testdb."+mongo.CollectionDomains,
		bson.D{{Key: "profilename", Value: "a"}, {Key: "tenantid", Value: "t1"}},
		bson.D{{Key: "profilename", Value: "b"}, {Key: "tenantid", Value: "t1"}},
	))

	repo := mongo.NewDomainRepo(db)

	rows, err := repo.Get(context.Background(), 10, 0, "t1")
	require.NoError(t, err)
	require.Len(t, rows, 2)
	require.Equal(t, "a", rows[0].ProfileName)
	require.Equal(t, "b", rows[1].ProfileName)
}

func TestDomainRepo_Insert(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	md.AddResponses(insertResponse())

	repo := mongo.NewDomainRepo(db)

	_, err := repo.Insert(context.Background(), &entity.Domain{
		ProfileName: "Acme",
		TenantID:    "t1",
	})
	require.NoError(t, err)
}

func TestDomainRepo_Insert_DuplicateReturnsNotUniqueError(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	md.AddResponses(duplicateKeyResponse())

	repo := mongo.NewDomainRepo(db)

	_, err := repo.Insert(context.Background(), &entity.Domain{
		ProfileName: "Acme",
		TenantID:    "t1",
	})
	require.Error(t, err)

	var nu repoerrors.NotUniqueError
	require.True(t, errors.As(err, &nu))
}

func TestDomainRepo_Update_Matched(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	md.AddResponses(updateResponse(1))

	repo := mongo.NewDomainRepo(db)

	ok, err := repo.Update(context.Background(), &entity.Domain{
		ProfileName: "Acme",
		TenantID:    "t1",
	})
	require.NoError(t, err)
	require.True(t, ok)
}

func TestDomainRepo_Update_NoMatch(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	md.AddResponses(updateResponse(0))

	repo := mongo.NewDomainRepo(db)

	ok, err := repo.Update(context.Background(), &entity.Domain{
		ProfileName: "ghost",
		TenantID:    "t1",
	})
	require.NoError(t, err)
	require.False(t, ok)
}

func TestDomainRepo_Delete_Matched(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	md.AddResponses(deleteResponse(1))

	repo := mongo.NewDomainRepo(db)

	ok, err := repo.Delete(context.Background(), "Acme", "t1")
	require.NoError(t, err)
	require.True(t, ok)
}
