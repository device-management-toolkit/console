package mongo_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/drivertest"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/xoptions"
)

// Tests in this directory unit-test the repo implementations against the
// MongoDB Go driver's official wire-level mock (drivertest.NewMockDeployment,
// see GODRIVER-3241). The mock intercepts driver operations at the OP_MSG
// level and returns pre-queued bson responses, so the repo code under test
// is the real production code — no interface refactor or hand-rolled fakes.
//
// End-to-end behavior (real Mongo bson encoding, real index conflicts, real
// duplicate-key wire errors) is covered by the api-test workflow's
// build-mongo job; these tests cover the in-process logic each repo runs
// before/after the driver call.
//
// Stability note: the drivertest package lives under mongo-driver/v2/x/, the
// driver team's experimental namespace. Its API has no formal stability
// guarantee, so go.mod pins mongo-driver to a known-good version and the
// import is intentionally confined to this one helper file. Any future
// breaking change in drivertest is contained to mongo_test.go, never to
// production code.

// newMockedDB returns a *mongo.Database backed by drivertest.MockDeployment.
// Each test gets its own client + deployment so queued responses don't leak
// across tests. Caller queues responses via md.AddResponses(...) before
// invoking the code under test.
func newMockedDB(t *testing.T) (*mongo.Database, *drivertest.MockDeployment) {
	t.Helper()

	md := drivertest.NewMockDeployment()

	opts := options.Client()
	require.NoError(t, xoptions.SetInternalClientOptions(opts, "deployment", md))

	client, err := mongo.Connect(opts)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = client.Disconnect(context.Background())
	})

	return client.Database("testdb"), md
}

// findResponse builds the bson reply shape expected for a `find` command:
//
//	{ ok: 1, cursor: { id: 0, ns: "<ns>", firstBatch: [...] } }
//
// id=0 signals "no more batches" so the driver does not issue a getMore.
func findResponse(ns string, docs ...bson.D) bson.D {
	batch := make(bson.A, len(docs))
	for i, d := range docs {
		batch[i] = d
	}

	return bson.D{
		{Key: "cursor", Value: bson.D{
			{Key: "id", Value: int64(0)},
			{Key: "ns", Value: ns},
			{Key: "firstBatch", Value: batch},
		}},
		{Key: "ok", Value: 1},
	}
}

// insertResponse is the success reply for an `insert` command (one doc).
func insertResponse() bson.D {
	return bson.D{
		{Key: "n", Value: int32(1)},
		{Key: "ok", Value: 1},
	}
}

// duplicateKeyResponse mimics a Mongo E11000 duplicate-key error response.
// The driver lifts this into a *mongo.WriteException whose first WriteError
// has code 11000, which is what isDuplicateKey detects.
func duplicateKeyResponse() bson.D {
	return bson.D{
		{Key: "ok", Value: 1},
		{Key: "n", Value: int32(0)},
		{Key: "writeErrors", Value: bson.A{
			bson.D{
				{Key: "index", Value: int32(0)},
				{Key: "code", Value: int32(11000)},
				{Key: "errmsg", Value: "E11000 duplicate key error"},
			},
		}},
	}
}

// updateResponse is the success reply for an `update` command (one match).
func updateResponse(matched int32) bson.D {
	return bson.D{
		{Key: "n", Value: matched},
		{Key: "nModified", Value: matched},
		{Key: "ok", Value: 1},
	}
}

// deleteResponse is the success reply for a `delete` command.
func deleteResponse(deleted int32) bson.D {
	return bson.D{
		{Key: "n", Value: deleted},
		{Key: "ok", Value: 1},
	}
}

// distinctResponse is the success reply for the `distinct` command.
func distinctResponse(values ...any) bson.D {
	out := make(bson.A, len(values))
	copy(out, values)

	return bson.D{
		{Key: "values", Value: out},
		{Key: "ok", Value: 1},
	}
}
