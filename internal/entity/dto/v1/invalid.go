package dto

import (
	"github.com/device-management-toolkit/console/pkg/consoleerrors"
)

type NotValidError struct {
	Console consoleerrors.InternalError
}

func (e NotValidError) Error() string {
	return e.Console.Error()
}

func (e NotValidError) Wrap(call, function string, err error) error {
	_ = e.Console.Wrap(call, function, err)
	e.Console.Message = "Invalid input: " + err.Error()

	return e
}

// Unwrap exposes the wrapped error so errors.Is/As can traverse the chain.
func (e NotValidError) Unwrap() error {
	return e.Console.OriginalError
}
