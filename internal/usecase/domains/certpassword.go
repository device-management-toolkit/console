package domains

import "github.com/device-management-toolkit/console/pkg/consoleerrors"

const invalidCertificate = "invalid provisioning certificate"

type CertFormatError struct {
	Console consoleerrors.InternalError
}

func (e CertFormatError) Error() string {
	return invalidCertificate
}

func (e CertFormatError) Wrap(call, function string, err error) error {
	_ = e.Console.Wrap(call, function, err)
	e.Console.Message = invalidCertificate

	return e
}

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
