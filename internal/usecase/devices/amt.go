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

type ExplorerError struct {
	Console consoleerrors.InternalError
}

func (e ExplorerError) Error() string {
	return e.Console.Error()
}

func (e ExplorerError) Wrap(call, function string, err error) error {
	_ = e.Console.Wrap(call, function, err)
	e.Console.Message = "amt explorer error"

	return e
}

type NotSupportedError struct {
	Console consoleerrors.InternalError
}

func (e NotSupportedError) Error() string {
	return e.Console.Error()
}

func (e NotSupportedError) Wrap(call, function, message string) error {
	_ = e.Console.Wrap(call, function, nil)
	e.Console.Message = message

	return e
}
