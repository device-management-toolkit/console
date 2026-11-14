package v2

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRemoteEraseStateJSON(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		state RemoteEraseState
		value string
	}{
		{RemoteEraseStatePending, "pending"},
		{RemoteEraseStateSucceeded, "succeeded"},
		{RemoteEraseStateFailed, "failed"},
		{RemoteEraseStateUnknown, "unknown"},
	} {
		tc := tc

		t.Run(tc.value, func(t *testing.T) {
			t.Parallel()

			original := RemoteEraseStatus{Status: tc.state, Message: "BIOS status"}
			body, err := json.Marshal(original)
			require.NoError(t, err)
			require.JSONEq(t, `{"status":"`+tc.value+`","message":"BIOS status"}`, string(body))

			var decoded RemoteEraseStatus
			require.NoError(t, json.Unmarshal(body, &decoded))
			require.Equal(t, original, decoded)
		})
	}
}

func TestRemoteEraseRawWordsJSON(t *testing.T) {
	t.Parallel()

	zero := uint16(0)
	original := RemoteEraseStatus{
		Status: RemoteEraseStateSucceeded, Message: "Last boot result",
		BIOSStatus: &zero, BIOSDetail: &zero,
	}
	body, err := json.Marshal(original)
	require.NoError(t, err)
	require.JSONEq(t, `{"status":"succeeded","message":"Last boot result","biosStatus":0,"biosDetail":0}`, string(body))

	var decoded RemoteEraseStatus
	require.NoError(t, json.Unmarshal(body, &decoded))
	require.Equal(t, original, decoded)
}
