package sqldb

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
		name           string
		initialMessage string
		details        string
		expectedResult string
	}{
		{
			name:           "Wrap with details",
			initialMessage: "error occurred",
			details:        "unique constraint",
			expectedResult: " -  - : ",
		},
		{
			name:           "Wrap with empty details",
			initialMessage: "error occurred",
			details:        "",
			expectedResult: " -  - : ",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			internalErr := consoleerrors.InternalError{Message: tc.initialMessage}
			err := NotUniqueError{Console: internalErr}

			wrappedErr := err.Wrap(tc.details)

			require.Equal(t, tc.expectedResult, wrappedErr.Error())
		})
	}
}
