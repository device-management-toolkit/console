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

func TestProfileRepo_GetCount(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	md.AddResponses(findResponse(
		"testdb."+mongo.CollectionProfiles,
		bson.D{{Key: "n", Value: int64(4)}},
	))

	repo := mongo.NewProfileRepo(db, logger.New("error"))

	got, err := repo.GetCount(context.Background(), "t1")
	require.NoError(t, err)
	require.Equal(t, 4, got)
}

func TestProfileRepo_GetByName_NoIEEE8021x(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	// Profile has no IEEE8021xProfileName, so populate8021x is skipped — only
	// one mock response needed.
	md.AddResponses(findResponse(
		"testdb."+mongo.CollectionProfiles,
		bson.D{
			{Key: "profilename", Value: "p1"},
			{Key: "activation", Value: "ccmactivate"},
			{Key: "tenantid", Value: "t1"},
		},
	))

	repo := mongo.NewProfileRepo(db, logger.New("error"))

	got, err := repo.GetByName(context.Background(), "p1", "t1")
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, "p1", got.ProfileName)
	require.Nil(t, got.AuthenticationProtocol, "no 8021x ref → no populate")
}

// When the Profile references an IEEE8021x config, GetByName issues a second
// FindOne against the ieee8021xconfigs collection to populate the join fields.
func TestProfileRepo_GetByName_PopulatesFrom8021x(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	md.AddResponses(
		findResponse(
			"testdb."+mongo.CollectionProfiles,
			bson.D{
				{Key: "profilename", Value: "p1"},
				{Key: "ieee8021xprofilename", Value: "ieee1"},
				{Key: "tenantid", Value: "t1"},
			},
		),
		findResponse(
			"testdb."+mongo.CollectionIEEE8021xConfigs,
			bson.D{
				{Key: "profilename", Value: "ieee1"},
				{Key: "authenticationprotocol", Value: int32(4)},
				{Key: "pxetimeout", Value: int32(60)},
				{Key: "wiredinterface", Value: true},
				{Key: "tenantid", Value: "t1"},
			},
		),
	)

	repo := mongo.NewProfileRepo(db, logger.New("error"))

	got, err := repo.GetByName(context.Background(), "p1", "t1")
	require.NoError(t, err)
	require.NotNil(t, got)
	require.NotNil(t, got.AuthenticationProtocol)
	require.Equal(t, 4, *got.AuthenticationProtocol)
	require.NotNil(t, got.PXETimeout)
	require.Equal(t, 60, *got.PXETimeout)
	require.NotNil(t, got.WiredInterface)
	require.True(t, *got.WiredInterface)
}

func TestProfileRepo_GetByName_NotFound(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	md.AddResponses(findResponse("testdb." + mongo.CollectionProfiles))

	repo := mongo.NewProfileRepo(db, logger.New("error"))

	got, err := repo.GetByName(context.Background(), "ghost", "t1")
	require.NoError(t, err)
	require.Nil(t, got)
}

func TestProfileRepo_Insert(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	md.AddResponses(insertResponse())

	repo := mongo.NewProfileRepo(db, logger.New("error"))

	_, err := repo.Insert(context.Background(), &entity.Profile{
		ProfileName: "p1",
		TenantID:    "t1",
	})
	require.NoError(t, err)
}

func TestProfileRepo_Insert_DuplicateReturnsNotUniqueError(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	md.AddResponses(duplicateKeyResponse())

	repo := mongo.NewProfileRepo(db, logger.New("error"))

	_, err := repo.Insert(context.Background(), &entity.Profile{
		ProfileName: "p1",
		TenantID:    "t1",
	})
	require.Error(t, err)

	var nu repoerrors.NotUniqueError
	require.True(t, errors.As(err, &nu))
}

func TestProfileRepo_Update(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	md.AddResponses(updateResponse(1))

	repo := mongo.NewProfileRepo(db, logger.New("error"))

	ok, err := repo.Update(context.Background(), &entity.Profile{
		ProfileName: "p1",
		TenantID:    "t1",
	})
	require.NoError(t, err)
	require.True(t, ok)
}

func TestProfileRepo_Delete(t *testing.T) {
	t.Parallel()

	db, md := newMockedDB(t)

	md.AddResponses(deleteResponse(1))

	repo := mongo.NewProfileRepo(db, logger.New("error"))

	ok, err := repo.Delete(context.Background(), "p1", "t1")
	require.NoError(t, err)
	require.True(t, ok)
}
