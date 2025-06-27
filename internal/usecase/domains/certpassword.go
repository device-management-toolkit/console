package domains

import "github.com/device-management-toolkit/console/pkg/consoleerrors"

type CertPasswordError struct {
	Console consoleerrors.InternalError
}

func (e CertPasswordError) Error() string {
	return e.Console.Error()
}

func (e CertPasswordError) Wrap(call, function string, err error) error {
	_ = e.Console.Wrap(call, function, err)
	e.Console.Message = "unable to decrypt certificate, incorrect password"

	return e
}
