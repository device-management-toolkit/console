package sqldb

import (
	"errors"

	"github.com/device-management-toolkit/console/pkg/consoleerrors"
)

type DatabaseError struct {
	Console consoleerrors.InternalError
}

func (e DatabaseError) Error() string {
	return e.Console.Error()
}

func (e DatabaseError) Wrap(call, function string, err error) error {
	wrapped := e.Console.Wrap(call, function, err)

	var internalErr *consoleerrors.InternalError
	if errors.As(wrapped, &internalErr) {
		e.Console = *internalErr
	}

	e.Console.Message = "database error"

	return e
}
