package domains

import (
	"errors"

	"github.com/device-management-toolkit/console/pkg/consoleerrors"
)

const certExpired = "certificate has expired"

type CertExpirationError struct {
	Console consoleerrors.InternalError
}

func (e CertExpirationError) Error() string {
	return certExpired
}

func (e CertExpirationError) Wrap(call, function string, err error) error {
	wrapped := e.Console.Wrap(call, function, err)

	var internalErr *consoleerrors.InternalError
	if errors.As(wrapped, &internalErr) {
		e.Console = *internalErr
	}

	e.Console.Message = certExpired

	return e
}
