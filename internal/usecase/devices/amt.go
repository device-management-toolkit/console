package devices

import "github.com/open-amt-cloud-toolkit/console/pkg/consoleerrors"

type AMTError struct {
	Console consoleerrors.InternalError
}

func (e AMTError) Error() string {
	return e.Console.Error()
}

func (e AMTError) Wrap(call, function string, err error) error {
	_ = e.Console.Wrap(call, function, err)
	e.Console.Message = "amt error"

	return e
}
