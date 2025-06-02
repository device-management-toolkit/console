package amtexplorer

import "github.com/device-management-toolkit/console/pkg/consoleerrors"

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
