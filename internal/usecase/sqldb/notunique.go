package sqldb

import "github.com/device-management-toolkit/console/pkg/consoleerrors"

type NotUniqueError struct {
	Console consoleerrors.InternalError
}

func (e NotUniqueError) Error() string {
	return e.Console.Error()
}

func (e NotUniqueError) Wrap(details string) error {
	e.Console.Message = "unique constraint violation: " + details

	return e
}
