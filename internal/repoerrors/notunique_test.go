package repoerrors

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/device-management-toolkit/console/pkg/consoleerrors"
)

func TestNotUniqueError_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		consoleError   consoleerrors.InternalError
		expectedResult string
	}{
		{
			name:           "Basic error message",
			consoleError:   consoleerrors.InternalError{Message: "unique constraint violation"},
			expectedResult: " -  - : ",
		},
		{
			name:           "Empty error message",
			consoleError:   consoleerrors.InternalError{Message: ""},
			expectedResult: " -  - : ",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := NotUniqueError{Console: tc.consoleError}
			result := err.Error()

			require.Equal(t, tc.expectedResult, result)
		})
	}
}

func TestNotUniqueError_Wrap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		details          string
		expectedFriendly string
	}{
		{
			name:             "Wrap with details",
			details:          "unique constraint",
			expectedFriendly: "unique constraint violation: unique constraint",
		},
		{
			name:             "Wrap with empty details",
			details:          "",
			expectedFriendly: "unique constraint violation: ",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := NotUniqueError{Console: consoleerrors.InternalError{}}

			wrappedErr := err.Wrap(tc.details)

			var nuErr NotUniqueError
			require.ErrorAs(t, wrappedErr, &nuErr)
			require.Equal(t, tc.expectedFriendly, nuErr.Console.FriendlyMessage())
		})
	}
}
