package secrets

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsAdminPasswordStrong(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		password string
		want     bool
	}{
		{name: "strong password", password: "example-Password-8", want: true},
		{name: "unicode characters count as characters", password: "abéçdefg", want: true},
		{name: "too short", password: "abc123", want: false},
		{name: "too few distinct characters", password: "aaaaaaaa", want: false},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, testCase.want, IsAdminPasswordStrong(testCase.password))
		})
	}
}
