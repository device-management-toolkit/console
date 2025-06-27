package sqldb

import "github.com/device-management-toolkit/console/pkg/consoleerrors"

type ForeignKeyViolationError struct {
	Console consoleerrors.InternalError
}

func (e ForeignKeyViolationError) Error() string {
	return e.Console.Error()
}

func (e ForeignKeyViolationError) Wrap(details string) error {
	e.Console.Message = "foreign key violation: " + details

	return e
}
