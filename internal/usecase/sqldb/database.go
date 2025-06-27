package sqldb

import "github.com/device-management-toolkit/console/pkg/consoleerrors"

type DatabaseError struct {
	Console consoleerrors.InternalError
}

func (e DatabaseError) Error() string {
	return e.Console.Error()
}

func (e DatabaseError) Wrap(call, function string, err error) error {
	_ = e.Console.Wrap(call, function, err)
	e.Console.Message = "database error"

	return e
}
