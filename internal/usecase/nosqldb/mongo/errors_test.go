package mongo

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// isDuplicateKey is a thin wrapper over mongo.IsDuplicateKeyError. The point
// of this test is to pin down the contract: a plain error must NOT match,
// and a real duplicate-key error from the driver MUST match.
func TestIsDuplicateKey(t *testing.T) {
	t.Parallel()

	t.Run("plain error is not a duplicate key", func(t *testing.T) {
		t.Parallel()
		require.False(t, isDuplicateKey(errors.New("boom")))
	})

	t.Run("nil is not a duplicate key", func(t *testing.T) {
		t.Parallel()
		require.False(t, isDuplicateKey(nil))
	})

	t.Run("WriteException with code 11000 is a duplicate key", func(t *testing.T) {
		t.Parallel()

		err := mongo.WriteException{
			WriteErrors: []mongo.WriteError{{Code: 11000, Message: "E11000 duplicate key"}},
		}
		require.True(t, isDuplicateKey(err))
	})

	t.Run("WriteException with non-duplicate code is not a duplicate key", func(t *testing.T) {
		t.Parallel()

		err := mongo.WriteException{
			WriteErrors: []mongo.WriteError{{Code: 121, Message: "DocumentValidationFailure"}},
		}
		require.False(t, isDuplicateKey(err))
	})
}
