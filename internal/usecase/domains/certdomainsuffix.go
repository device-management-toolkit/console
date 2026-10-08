package domains

import "github.com/device-management-toolkit/console/pkg/consoleerrors"

const certDomainSuffixMismatch = "FQDN not associated with provisioning certificate"

// CertDomainSuffixError is returned when the domain suffix does not belong to
// the domain named by the provisioning certificate's Common Name.
type CertDomainSuffixError struct {
	Console consoleerrors.InternalError
}

func (e CertDomainSuffixError) Error() string {
	return certDomainSuffixMismatch
}

func (e CertDomainSuffixError) Wrap(call, function string, err error) error {
	_ = e.Console.Wrap(call, function, err)
	e.Console.Message = certDomainSuffixMismatch

	return e
}
