package domains

import "github.com/device-management-toolkit/console/pkg/consoleerrors"

const certExpired = "certificate has expired"

type CertExpirationError struct {
	Console consoleerrors.InternalError
}

func (e CertExpirationError) Error() string {
	return certExpired
}

func (e CertExpirationError) Wrap(call, function string, err error) error {
	_ = e.Console.Wrap(call, function, err)
	e.Console.Message = certExpired

	return e
}
