package devices

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"strings"
)

var (
	// ErrInvalidCredentials is returned when the device rejects the AMT credentials.
	ErrInvalidCredentials = errors.New("the device rejected the AMT username or password")
	// ErrDeviceUnreachable is returned when the device could not be reached (refused, timeout, no route).
	ErrDeviceUnreachable = errors.New("the device could not be reached")
	// ErrTLSFailure is returned when the TLS handshake with the device failed, e.g. an untrusted certificate.
	ErrTLSFailure = errors.New("the TLS connection to the device failed")
	// ErrNotAMT is returned when the endpoint answered but is not an Intel AMT service.
	ErrNotAMT = errors.New("the device did not respond as an Intel AMT device")
)

// wrapCredentialError classifies a failure from verifying credentials so the caller
// can tell the user what went wrong. Unrecognized errors are returned untouched.
// Order matters: an HTTP 401 means AMT answered and the password is wrong, TLS
// failures are checked before generic network errors because they also satisfy
// net.Error in some cases.
func wrapCredentialError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case strings.Contains(err.Error(), "received: 401"):
		return fmt.Errorf("%w: %w", ErrInvalidCredentials, err)
	case isTLSError(err):
		return fmt.Errorf("%w: %w", ErrTLSFailure, err)
	case isUnreachableError(err):
		return fmt.Errorf("%w: %w", ErrDeviceUnreachable, err)
	case isNotAMTError(err):
		return fmt.Errorf("%w: %w", ErrNotAMT, err)
	}

	return err
}

func isTLSError(err error) bool {
	var (
		unknownAuthority x509.UnknownAuthorityError
		certInvalid      x509.CertificateInvalidError
		hostname         x509.HostnameError
		verification     *tls.CertificateVerificationError
		recordHeader     tls.RecordHeaderError
	)

	if errors.As(err, &unknownAuthority) || errors.As(err, &certInvalid) || errors.As(err, &hostname) ||
		errors.As(err, &verification) || errors.As(err, &recordHeader) {
		return true
	}

	msg := err.Error()

	return strings.Contains(msg, "x509:") || strings.Contains(msg, "tls:")
}

func isUnreachableError(err error) bool {
	var netErr net.Error

	if errors.As(err, &netErr) {
		return true
	}

	msg := err.Error()

	return strings.Contains(msg, "connection refused") || strings.Contains(msg, "i/o timeout") ||
		strings.Contains(msg, "no route to host") || strings.Contains(msg, "network is unreachable")
}

func isNotAMTError(err error) bool {
	msg := err.Error()

	return strings.Contains(msg, "received: 404") || strings.Contains(msg, "malformed HTTP")
}
