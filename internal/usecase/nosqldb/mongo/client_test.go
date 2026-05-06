package mongo_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	mongo "github.com/device-management-toolkit/console/internal/usecase/nosqldb/mongo"
	"github.com/device-management-toolkit/console/pkg/logger"
)

// Connect is exercised end-to-end by the api-test workflow's build-mongo job
// (real container, real ping, real ensureIndexes). Here we only cover the
// purely in-process input validation that runs before any I/O — the empty
// URI guard.
func TestConnect_RejectsEmptyURI(t *testing.T) {
	t.Parallel()

	client, db, err := mongo.Connect(context.Background(), "", logger.New("error"))
	require.Error(t, err)
	require.Nil(t, client)
	require.Nil(t, db)
}
