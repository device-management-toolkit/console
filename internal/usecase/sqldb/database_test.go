package sqldb

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/device-management-toolkit/console/pkg/consoleerrors"
)

var (
	ErrSQLSyntax = errors.New("SQL syntax error")
	ErrOther     = errors.New("another error")
)

func TestDatabaseError_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		consoleError   consoleerrors.InternalError
		expectedResult string
	}{
		{
			name:           "Basic error message",
			consoleError:   consoleerrors.InternalError{Message: "generic database error"},
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

			err := DatabaseError{Console: tc.consoleError}
			result := err.Error()

			require.Equal(t, tc.expectedResult, result)
		})
	}
}

func TestDatabaseError_Wrap(t *testing.T) {
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
			call:           "DBQuery",
			function:       "Execute",
			err:            ErrSQLSyntax,
			expectedResult: " - Execute - DBQuery: SQL syntax error",
		},
		{
			name:           "Wrap with nil error",
			initialMessage: "another error occurred",
			call:           "DBQuery",
			function:       "Execute",
			err:            nil,
			expectedResult: " - Execute - DBQuery: ",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			internalErr := consoleerrors.InternalError{Message: tc.initialMessage}
			err := DatabaseError{Console: internalErr}

			wrappedErr := err.Wrap(tc.call, tc.function, tc.err)

			require.Equal(t, tc.expectedResult, wrappedErr.Error())
		})
	}
}
