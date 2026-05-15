package dto

import (
	"github.com/device-management-toolkit/console/pkg/consoleerrors"
)

type CanceledError struct {
	Console consoleerrors.InternalError
}

func (e CanceledError) Error() string {
	return e.Console.Error()
}

func (e CanceledError) Wrap(call, function string, err error) error {
	_ = e.Console.Wrap(call, function, err)
	e.Console.Message = "Canceled: " + err.Error()

	return e
}
