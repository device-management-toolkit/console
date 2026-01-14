package dto

import (
	"errors"

	"github.com/device-management-toolkit/console/pkg/consoleerrors"
)

type NotValidError struct {
	Console consoleerrors.InternalError
}

func (e NotValidError) Error() string {
	return e.Console.Error()
}

func (e NotValidError) Wrap(function, call string, err error) error {
	wrapped := e.Console.Wrap(function, call, err)

	var internalErr *consoleerrors.InternalError
	if errors.As(wrapped, &internalErr) {
		e.Console = *internalErr
	}

	e.Console.Message = "Invalid input: " + err.Error()

	return e
}
