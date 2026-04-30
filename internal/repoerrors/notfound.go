package repoerrors

import "github.com/device-management-toolkit/console/pkg/consoleerrors"

type NotFoundError struct {
	Console consoleerrors.InternalError
}

func (e NotFoundError) Error() string {
	return e.Console.Error()
}

// Wrap records call/function/err on e.Console (mutated in place via pointer
// receiver on InternalError.Wrap) and returns e by value. The discarded return
// is intentional: only the side effect on e.Console is needed.
func (e NotFoundError) Wrap(call, function string, err error) error {
	_ = e.Console.Wrap(call, function, err)
	e.Console.Message = "Error not found"

	return e
}

func (e NotFoundError) WrapWithMessage(call, function, message string) error {
	_ = e.Console.Wrap(call, function, nil)
	e.Console.Message = message

	return e
}
