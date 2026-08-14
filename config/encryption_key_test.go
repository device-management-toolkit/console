package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/security"
)

func TestValidateEncryptionKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		key     string
		wantErr error
	}{
		{"aes-128 random", "Kd7wQ2mZ4tLxB9vR", nil},
		{"aes-192 random", "Kd7wQ2mZ4tLxB9vRp6HsY3jN", nil},
		{"aes-256 base64", "Jf3Q2nXJ+GZzN1dbVQms0wbB4+i/5PjL", nil},
		{"aes-256 passphrase", "MyCompanyConsoleKey2026!Prod0x7f", nil},
		{"empty", "", ErrEncryptionKeyLength},
		{"15 characters", "aaaaaaaaaaaaaaa", ErrEncryptionKeyLength},
		{"17 characters", "Kd7wQ2mZ4tLxB9vRp", ErrEncryptionKeyLength},
		{"31 characters", "Jf3Q2nXJ+GZzN1dbVQms0wbB4+i/5Pj", ErrEncryptionKeyLength},
		{"single character repeated, 16", "aaaaaaaaaaaaaaaa", ErrEncryptionKeyWeak},
		{"single character repeated, 24", "aaaaaaaaaaaaaaaaaaaaaaaa", ErrEncryptionKeyWeak},
		{"single character repeated, 32", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ErrEncryptionKeyWeak},
		{"one odd character out", "aaaaaaaaaaaaaaab", ErrEncryptionKeyWeak},
		{"two characters alternating", "abababababababab", ErrEncryptionKeyWeak},
		{"few distinct characters", "abcdabcdabcdabcdabcdabcdabcdabcd", ErrEncryptionKeyWeak},
		{"repeated block", "Kd7wQ2mZKd7wQ2mZKd7wQ2mZKd7wQ2mZ", ErrEncryptionKeyWeak},
		{"sequential run", "abcdefghijklmnop", ErrEncryptionKeyWeak},
		{"descending run", "ponmlkjihgfedcba", ErrEncryptionKeyWeak},
		{"digits then sequence", "x9!Qz3wV0123456789abKp$7mT2rLd6H", ErrEncryptionKeyWeak},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateEncryptionKey(tc.key)

			if tc.wantErr == nil {
				require.NoError(t, err)

				return
			}

			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

// TestValidateEncryptionKey_AcceptsGeneratedKey guards against the strength
// rules rejecting the keys Console generates for itself.
func TestValidateEncryptionKey_AcceptsGeneratedKey(t *testing.T) {
	t.Parallel()

	cryptor := security.Crypto{}

	for range 100 {
		key := cryptor.GenerateKey()
		require.NoError(t, ValidateEncryptionKey(key), "generated key %q rejected", key)
	}
}

// t.Setenv is incompatible with t.Parallel, so this test and its subtests run
// serially.
func TestNewConfig_RejectsInvalidEncryptionKey(t *testing.T) {
	clearEnv()

	tests := []struct {
		name string
		key  string
	}{
		{"wrong length", "aaaaaaaaaaaaaaa"},
		{"no entropy", "aaaaaaaaaaaaaaaa"},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("APP_ENCRYPTION_KEY", tc.key)

			cfg, err := NewConfig()

			require.Error(t, err)
			assert.Nil(t, cfg)
			assert.Contains(t, err.Error(), "APP_ENCRYPTION_KEY")
		})
	}
}

// t.Setenv is incompatible with t.Parallel, so this test runs serially.
func TestNewConfig_AcceptsValidEncryptionKey(t *testing.T) {
	clearEnv()
	t.Setenv("APP_ENCRYPTION_KEY", "Jf3Q2nXJ+GZzN1dbVQms0wbB4+i/5PjL")

	cfg, err := NewConfig()

	require.NoError(t, err)
	assert.Equal(t, "Jf3Q2nXJ+GZzN1dbVQms0wbB4+i/5PjL", cfg.EncryptionKey)
}
