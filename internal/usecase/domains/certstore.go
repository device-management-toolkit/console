package domains

import (
	"errors"

	"github.com/device-management-toolkit/console/pkg/consoleerrors"
)

type CertStoreError struct {
	Console consoleerrors.InternalError
}

func (e CertStoreError) Error() string {
	return e.Console.Error()
}

func (e CertStoreError) Wrap(call, function string, err error) error {
	wrapped := e.Console.Wrap(call, function, err)

	var internalErr *consoleerrors.InternalError
	if errors.As(wrapped, &internalErr) {
		e.Console = *internalErr
	}

	e.Console.Message = "certificate store operation failed"

	return e
}
