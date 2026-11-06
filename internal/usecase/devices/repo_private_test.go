package devices

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestDescribeProbeFailure: the AMT UUID lookup error names the likely cause
// first so a user can act on it, and keeps the raw error for support.
func TestDescribeProbeFailure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  string
		want string
	}{
		{"refused means AMT is not listening", "Post \"http://10.0.0.5:16992/wsman\": dial tcp 10.0.0.5:16992: connectex: No connection could be made because the target machine actively refused it.", "configured for CIRA"},
		{"401 means bad credentials", "wsman: 401 Unauthorized", "rejected the credentials"},
		{"tls failure points at the TLS settings", "tls: failed to verify certificate: x509: certificate signed by unknown authority", "Use TLS and Allow Self-Signed"},
		{"timeout means unreachable", "dial tcp 10.0.0.5:16992: i/o timeout", "unreachable from Console"},
		{"unknown host means unreachable", "dial tcp: lookup amt15.lab: no such host", "unreachable from Console"},
		{"anything else gets the generic guidance", "unexpected EOF", "could not read the AMT UUID from 10.0.0.5"},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := describeProbeFailure("10.0.0.5", errors.New(tt.err))
			require.Contains(t, got, tt.want)
			require.Contains(t, got, "("+tt.err+")", "the raw error must be preserved")
		})
	}
}
