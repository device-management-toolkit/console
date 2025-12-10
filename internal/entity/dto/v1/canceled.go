package dto

import (
	"github.com/open-amt-cloud-toolkit/console/pkg/consoleerrors"
)

type CanceledError struct {
	Console consoleerrors.InternalError
}

func (e CanceledError) Error() string {
	return e.Console.Error()
}

func (e CanceledError) Wrap(function, call string, err error) error {
	_ = e.Console.Wrap(function, call, err)
	e.Console.Message = "Canceled: " + err.Error()

	return e
}
