package sqldb

import (
	"errors"

	"github.com/device-management-toolkit/console/pkg/consoleerrors"
)

type NotFoundError struct {
	Console consoleerrors.InternalError
}

func (e NotFoundError) Error() string {
	return e.Console.Error()
}

func (e NotFoundError) Wrap(call, function string, err error) error {
	wrapped := e.Console.Wrap(call, function, err)

	var internalErr *consoleerrors.InternalError
	if errors.As(wrapped, &internalErr) {
		e.Console = *internalErr
	}

	e.Console.Message = "Error not found"

	return e
}

func (e NotFoundError) WrapWithMessage(call, function, message string) error {
	wrapped := e.Console.Wrap(call, function, nil)

	var internalErr *consoleerrors.InternalError
	if errors.As(wrapped, &internalErr) {
		e.Console = *internalErr
	}

	e.Console.Message = message

	return e
}
