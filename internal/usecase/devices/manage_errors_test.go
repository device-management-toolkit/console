package devices

import (
	"crypto/x509"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWrapCredentialError(t *testing.T) {
	t.Parallel()

	other := errors.New("something else entirely")

	tests := []struct {
		name string
		err  error
		want error
	}{
		{"nil", nil, nil},
		{"rejected password", errors.New("wsman.Client post received: 401 Unauthorized"), ErrInvalidCredentials},
		{"connection refused", errors.New("dial tcp 10.0.0.1:16992: connect: connection refused"), ErrDeviceUnreachable},
		{"timeout", errors.New("dial tcp 10.0.0.1:16992: i/o timeout"), ErrDeviceUnreachable},
		{"untrusted certificate", x509.UnknownAuthorityError{}, ErrTLSFailure},
		{"tls text", errors.New("tls: failed to verify certificate"), ErrTLSFailure},
		{"not an AMT endpoint", errors.New("wsman.Client post received: 404 Not Found"), ErrNotAMT},
		{"look-alike 401", errors.New("dial tcp 10.0.0.1:16992: i/o timeout (request 4011)"), ErrDeviceUnreachable},
		{"unrecognized", other, other},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := wrapCredentialError(tc.err)
			if tc.want == nil {
				require.NoError(t, got)

				return
			}

			require.ErrorIs(t, got, tc.want)
		})
	}
}
