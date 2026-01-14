package devices

import (
	"errors"

	"github.com/device-management-toolkit/console/pkg/consoleerrors"
)

type AMTError struct {
	Console consoleerrors.InternalError
}

func (e AMTError) Error() string {
	return e.Console.Error()
}

func (e AMTError) Wrap(call, function string, err error) error {
	wrapped := e.Console.Wrap(call, function, err)

	var internalErr *consoleerrors.InternalError
	if errors.As(wrapped, &internalErr) {
		e.Console = *internalErr
	}

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
	wrapped := e.Console.Wrap(call, function, err)

	var internalErr *consoleerrors.InternalError
	if errors.As(wrapped, &internalErr) {
		e.Console = *internalErr
	}

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
	wrapped := e.Console.Wrap(call, function, nil)

	var internalErr *consoleerrors.InternalError
	if errors.As(wrapped, &internalErr) {
		e.Console = *internalErr
	}

	e.Console.Message = message

	return e
}

type ValidationError struct {
	Console consoleerrors.InternalError
}

func (e ValidationError) Error() string {
	return e.Console.Error()
}

func (e ValidationError) Wrap(call, function, message string) error {
	wrapped := e.Console.Wrap(call, function, nil)

	var internalErr *consoleerrors.InternalError
	if errors.As(wrapped, &internalErr) {
		e.Console = *internalErr
	}

	e.Console.Message = message

	return e
}
