package tenant_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/device-management-toolkit/console/internal/tenant"
)

func TestValidRejects(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
	}{
		{"single space", " "},
		{"embedded space", "tenant a"},
		{"leading space", " tenant-a"},
		{"trailing space", "tenant-a "},
		{"tab", "tenant\tb"},
		{"newline", "tenant\nb"},
		{"carriage return", "tenant\rb"},
		{"null byte", "tenant\x00b"},
		{"forward slash", "tenant/a"},
		{"backslash", "tenant\\a"},
		{"colon", "tenant:a"},
		{"at sign", "tenant@a"},
		{"percent encoding", "tenant%20a"},
		{"sql quote", "tenant'a"},
		{"sql injection", "a' OR '1'='1"},
		{"wildcard", "tenant*"},
		{"comma", "tenant,a"},
		{"path traversal", "../tenant"},
		{"cyrillic homoglyph", "tenant-\u0430"},
		{"zero width space", "tenant\u200ba"},
		{"emoji", "tenant-\U0001F600"},
		{"one over max length", strings.Repeat("a", tenant.MaxLength+1)},
		{"far over max length", strings.Repeat("a", 4096)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.False(t, tenant.Valid(tt.value))
		})
	}
}

func TestValidAccepts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
	}{
		{"empty is the default tenant", ""},
		{"single character", "a"},
		{"single digit", "1"},
		{"lower case", "tenant"},
		{"upper case", "TENANT"},
		{"mixed case", "TenantA"},
		{"hyphen", "tenant-a"},
		{"underscore", "tenant_a"},
		{"dot", "tenant.a"},
		{"all separators", "a-b_c.d"},
		{"uuid", "3a1b0c8e-4f2d-4a6b-9c1e-7d5f8a0b2c3d"},
		{"at max length", strings.Repeat("a", tenant.MaxLength)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.True(t, tenant.Valid(tt.value))
		})
	}
}

func TestContextRoundTrip(t *testing.T) {
	t.Parallel()

	ctx := tenant.WithContext(context.Background(), "tenant-a")
	require.Equal(t, "tenant-a", tenant.FromContext(ctx))
}

func TestFromContextWithoutTenant(t *testing.T) {
	t.Parallel()

	require.Empty(t, tenant.FromContext(context.Background()))
}
