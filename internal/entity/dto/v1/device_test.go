package dto

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestDevice_JSONContract locks two intentional serialization decisions for the
// identity/lifecycle columns on the /api/v1 device shape:
//   - isDeleted has NO omitempty, so it is always emitted and callers can
//     distinguish a false value from an absent field.
//   - the other new fields ARE omitempty, so they stay absent on empty/legacy
//     payloads and don't change the wire shape existing v1 consumers expect.
func TestDevice_JSONContract(t *testing.T) {
	t.Parallel()

	t.Run("zero value emits isDeleted but omits optional identity fields", func(t *testing.T) {
		t.Parallel()

		out := deviceJSONFields(t, Device{})

		require.Contains(t, out, "isDeleted", "isDeleted must always be present")
		require.JSONEq(t, `false`, string(out["isDeleted"]))

		for _, k := range []string{"id", "createdDate", "deletedDate", "productType", "connectionType"} {
			require.NotContains(t, out, k, "%s must be omitempty on an empty device", k)
		}
	})

	t.Run("populated values serialize under the expected keys", func(t *testing.T) {
		t.Parallel()

		out := deviceJSONFields(t, Device{
			ID:             "id-1",
			CreatedDate:    "2026-05-26T12:00:00Z",
			IsDeleted:      true,
			DeletedDate:    "2026-05-27T08:00:00Z",
			ProductType:    "vpro",
			ConnectionType: "CIRA",
		})

		require.JSONEq(t, `"id-1"`, string(out["id"]))
		require.JSONEq(t, `"2026-05-26T12:00:00Z"`, string(out["createdDate"]))
		require.JSONEq(t, `true`, string(out["isDeleted"]))
		require.JSONEq(t, `"2026-05-27T08:00:00Z"`, string(out["deletedDate"]))
		require.JSONEq(t, `"vpro"`, string(out["productType"]))
		require.JSONEq(t, `"CIRA"`, string(out["connectionType"]))
	})
}

// deviceJSONFields marshals d and returns its top-level JSON object keyed by
// field name, so a test can assert on key presence/absence and values.
func deviceJSONFields(t *testing.T, d Device) map[string]json.RawMessage {
	t.Helper()

	b, err := json.Marshal(d)
	require.NoError(t, err)

	var m map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(b, &m))

	return m
}
