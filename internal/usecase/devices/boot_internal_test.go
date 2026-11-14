package devices

import (
	"testing"

	"github.com/stretchr/testify/require"

	dtov2 "github.com/device-management-toolkit/console/internal/entity/dto/v2"
)

func TestRemoteEraseStatusFromBIOS(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		words   []uint16
		state   dtov2.RemoteEraseState
		message string
	}{
		{"missing", nil, dtov2.RemoteEraseStateUnknown, "BIOS erase outcome is unconfirmed."},
		{"truncated", []uint16{0}, dtov2.RemoteEraseStateUnknown, "BIOS erase outcome is unconfirmed."},
		{"success", []uint16{0, 0}, dtov2.RemoteEraseStateSucceeded, "BIOS reported the last boot completed without error. This does not confirm completion of the current erase request."},
		{"in progress", []uint16{1, 0}, dtov2.RemoteEraseStatePending, "Waiting for the BIOS erase result."},
		{"not updated", []uint16{2, 0}, dtov2.RemoteEraseStateUnknown, "BIOS status has not been updated. Completion of the requested erase is unconfirmed."},
		{"password failure", []uint16{0xffff, 2}, dtov2.RemoteEraseStateFailed, "BIOS erase failed: drive password/authentication failure."},
		{"drive failure", []uint16{0xffff, 1}, dtov2.RemoteEraseStateFailed, "BIOS erase failed: general drive failure."},
		{"unsupported", []uint16{0xffff, 3}, dtov2.RemoteEraseStateFailed, "BIOS erase failed: feature is not supported."},
		{"other failure", []uint16{0xffff, 99}, dtov2.RemoteEraseStateFailed, "BIOS reported that the erase failed."},
		{"unknown", []uint16{4, 0}, dtov2.RemoteEraseStateUnknown, "BIOS erase outcome is unconfirmed."},
		{"inconsistent success", []uint16{0, 2}, dtov2.RemoteEraseStateUnknown, "BIOS erase outcome is unconfirmed."},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			expected := dtov2.RemoteEraseStatus{
				Status:  tc.state,
				Message: tc.message,
			}
			if len(tc.words) > 0 {
				expected.BIOSStatus = &tc.words[0]
			}

			if len(tc.words) > 1 {
				expected.BIOSDetail = &tc.words[1]
			}

			require.Equal(t, expected, remoteEraseStatusFromBIOS(tc.words))
		})
	}
}
