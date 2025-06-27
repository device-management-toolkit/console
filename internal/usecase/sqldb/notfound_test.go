package sqldb

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/device-management-toolkit/console/pkg/consoleerrors"
)

var ErrRecordDoesNotExist = errors.New("record does not exist")

func TestNotFoundError_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		consoleError   consoleerrors.InternalError
		expectedResult string
	}{
		{
			name:           "Basic error message",
			consoleError:   consoleerrors.InternalError{Message: "record not found"},
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

			err := NotFoundError{Console: tc.consoleError}
			result := err.Error()

			require.Equal(t, tc.expectedResult, result)
		})
	}
}

func TestNotFoundError_Wrap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		initialMessage string
		call           string
		function       string
		err            error
		expectedResult string
	}{
		{
			name:           "Wrap with valid error",
			initialMessage: "some error occurred",
			call:           "FindRecord",
			function:       "Query",
			err:            ErrRecordDoesNotExist,
			expectedResult: " - Query - FindRecord: record does not exist",
		},
		{
			name:           "Wrap with nil error",
			initialMessage: "another error occurred",
			call:           "FindRecord",
			function:       "Query",
			err:            nil,
			expectedResult: " - Query - FindRecord: ",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			internalErr := consoleerrors.InternalError{Message: tc.initialMessage}
			err := NotFoundError{Console: internalErr}

			wrappedErr := err.Wrap(tc.call, tc.function, tc.err)

			require.Equal(t, tc.expectedResult, wrappedErr.Error())
		})
	}
}
