package secrets

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsStrongAdminPassword(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		password string
		want     bool
	}{
		{"meets every rule", "P@ssw0rdd", true},
		{"exactly min length", "P@ssw0rd", true},
		{"empty", "", false},
		{"one under min length", "P@ss0rd", false},
		{"no lowercase", "P@SSW0RD", false},
		{"no uppercase", "p@ssw0rd", false},
		{"no digit", "P@ssword", false},
		{"no symbol", "Passw0rdd", false},
		// AMT requires the special to come from !@#$%^&*; these do not, but the
		// admin password never reaches AMT, so they count as a symbol here.
		{"symbol AMT would not accept as its special", "Passw0rd(", true},
		{"hyphen counts as a symbol", "Passw0rd-x", true},
		{"underscore counts as a symbol", "Passw0rd_x", true},
		// No upper length bound: a long passphrase must not be reported as weak.
		{"long passphrase", "P@ssw0rd" + strings.Repeat("a", 40), true},
		{"length counts runes not bytes", "P@ssw0ré", true},
		{"non-ASCII letters only", strings.Repeat("é", 8), false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, IsStrongAdminPassword(tc.password))
		})
	}
}
