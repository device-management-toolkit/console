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

func (e NotValidError) Wrap(function, call string, err error) error {
	_ = e.Console.Wrap(function, call, err)
	e.Console.Message = "Invalid input: " + err.Error()

	return e
}
