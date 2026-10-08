package secrets

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeneratePBKDF2Hash(t *testing.T) {
	t.Parallel()

	password := "test-password"
	hash1, err := GeneratePBKDF2Hash(password)
	require.NoError(t, err)
	assert.NotEmpty(t, hash1)
	assert.True(t, IsPBKDF2Hash(hash1))
}

func TestGeneratePBKDF2Hash_DifferentSalts(t *testing.T) {
	t.Parallel()

	password := "test-password"
	hash1, err1 := GeneratePBKDF2Hash(password)
	require.NoError(t, err1)

	hash2, err2 := GeneratePBKDF2Hash(password)
	require.NoError(t, err2)

	// Different salts should produce different hashes
	assert.NotEqual(t, hash1, hash2)
}

func TestVerifyPBKDF2Hash_ValidPassword(t *testing.T) {
	t.Parallel()

	password := "my-secure-password"
	hash, err := GeneratePBKDF2Hash(password)
	require.NoError(t, err)

	assert.True(t, VerifyPBKDF2Hash(hash, password))
}

func TestVerifyPBKDF2Hash_InvalidPassword(t *testing.T) {
	t.Parallel()

	password := "my-secure-password"
	hash, err := GeneratePBKDF2Hash(password)
	require.NoError(t, err)

	assert.False(t, VerifyPBKDF2Hash(hash, "wrong-password"))
}

func TestVerifyPBKDF2Hash_NonPBKDF2Hash(t *testing.T) {
	t.Parallel()

	// Should return false for non-PBKDF2 hashes
	assert.False(t, VerifyPBKDF2Hash("$2a$10$xyz", "password"))
	assert.False(t, VerifyPBKDF2Hash("not-a-hash", "password"))
	assert.False(t, VerifyPBKDF2Hash("", "password"))
}

func TestVerifyPBKDF2Hash_MalformedPBKDF2Hash(t *testing.T) {
	t.Parallel()

	// Test various malformed PBKDF2 hashes
	testCases := []string{
		"$pbkdf2$",                 // Too short
		"$pbkdf2$600000",           // Missing salt and hash
		"$pbkdf2$600000$salt",      // Missing hash
		"$pbkdf2$abc$salt$hash",    // Invalid iteration count
		"$pbkdf2$600000$GGGG$HHHH", // Invalid hex encoding
		"$pbkdf2$600000$00$00",     // Too short salt/hash
		"$pbkdf2$1$" + validSaltHex() + "$" + validHashHex(),           // Below the iteration floor
		"$pbkdf2$99999999999$" + validSaltHex() + "$" + validHashHex(), // Above the iteration ceiling
	}

	for _, hash := range testCases {
		t.Run(hash, func(t *testing.T) {
			t.Parallel()
			assert.False(t, VerifyPBKDF2Hash(hash, "password"))
		})
	}
}

func TestIsPBKDF2Hash(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    string
		expected bool
	}{
		{"Valid PBKDF2 hash", "$pbkdf2$100000$abcd$efgh", false},
		{"Bcrypt hash", "$2a$10$xyz", false},
		{"Empty string", "", false},
		{"Plain text", "password", false},
		{"PBKDF2 prefix only", "$pbkdf2$", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, IsPBKDF2Hash(tc.value))
		})
	}
}

func TestIsPBKDF2Hash_IterationBounds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		iterations string
		expected   bool
	}{
		{"below floor", "1", false},
		{"at floor", "600000", true},
		{"above ceiling", "99999999999", false},
		{"at ceiling", "10000000", true},
		{"negative", "-600000", false},
		{"zero", "0", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			value := "$pbkdf2$" + tc.iterations + "$" + validSaltHex() + "$" + validHashHex()
			assert.Equal(t, tc.expected, IsPBKDF2Hash(value))
		})
	}
}

func validSaltHex() string {
	return strings.Repeat("ab", pbkdf2SaltSize)
}

func validHashHex() string {
	return strings.Repeat("cd", pbkdf2HashSize)
}

func TestPBKDF2HashFormat(t *testing.T) {
	t.Parallel()

	password := "test-password"
	hash, err := GeneratePBKDF2Hash(password)
	require.NoError(t, err)

	// Verify hash format: $pbkdf2$<iterations>$<hex(salt)>$<hex(hash)>
	assert.True(t, IsPBKDF2Hash(hash))
	assert.Contains(t, hash, "$pbkdf2$600000$")
}

func TestPBKDF2Consistency(t *testing.T) {
	t.Parallel()

	// Same password should verify against its own hash
	password := "consistent-password"
	hash, err := GeneratePBKDF2Hash(password)
	require.NoError(t, err)

	for i := 0; i < 5; i++ {
		assert.True(t, VerifyPBKDF2Hash(hash, password), "verification failed on attempt %d", i)
	}
}

func TestVerifyPBKDF2Hash_EmptyPassword(t *testing.T) {
	t.Parallel()

	// Empty password should still be hashable and verifiable
	password := ""
	hash, err := GeneratePBKDF2Hash(password)
	require.NoError(t, err)
	assert.True(t, VerifyPBKDF2Hash(hash, password))
	assert.False(t, VerifyPBKDF2Hash(hash, "non-empty"))
}

func TestVerifyPBKDF2Hash_VeryLongPassword(t *testing.T) {
	t.Parallel()

	// Very long password should work fine
	password := strings.Repeat("x", 10000)
	hash, err := GeneratePBKDF2Hash(password)
	require.NoError(t, err)
	assert.True(t, VerifyPBKDF2Hash(hash, password))
	assert.False(t, VerifyPBKDF2Hash(hash, strings.Repeat("x", 9999)))
}

func TestVerifyPBKDF2Hash_SpecialCharacters(t *testing.T) {
	t.Parallel()

	// Password with special characters
	password := "p@$$w0rd!#%&*()[]{}|;:',.<>?/~`"
	hash, err := GeneratePBKDF2Hash(password)
	require.NoError(t, err)
	assert.True(t, VerifyPBKDF2Hash(hash, password))
	assert.False(t, VerifyPBKDF2Hash(hash, "p@$$w0rd!#%&*()[]{}|;:',.<>?/~"))
}

func TestVerifyPBKDF2Hash_UnicodePassword(t *testing.T) {
	t.Parallel()

	// Password with unicode characters
	password := "مرحبا世界🔐пароль"
	hash, err := GeneratePBKDF2Hash(password)
	require.NoError(t, err)
	assert.True(t, VerifyPBKDF2Hash(hash, password))
	assert.False(t, VerifyPBKDF2Hash(hash, "مرحبا世界🔐"))
}

func TestGeneratePBKDF2Hash_MultipleGenerations(t *testing.T) {
	t.Parallel()

	password := "same-password"
	hashes := make(map[string]bool)

	// Generate 10 hashes for the same password
	for i := 0; i < 10; i++ {
		hash, err := GeneratePBKDF2Hash(password)
		require.NoError(t, err)

		hashes[hash] = true
		// Verify all hashes work for the same password
		assert.True(t, VerifyPBKDF2Hash(hash, password))
	}

	// All hashes should be unique (different salts)
	assert.Len(t, hashes, 10)
}

func TestVerifyPBKDF2Hash_SaltExtraction(t *testing.T) {
	t.Parallel()

	password := "test-password"
	hash1, err1 := GeneratePBKDF2Hash(password)
	require.NoError(t, err1)

	hash2, err2 := GeneratePBKDF2Hash(password)
	require.NoError(t, err2)

	// Extract salts from both hashes - they should be different
	parts1 := strings.Split(strings.TrimPrefix(hash1, "$pbkdf2$600000$"), "$")
	parts2 := strings.Split(strings.TrimPrefix(hash2, "$pbkdf2$600000$"), "$")

	assert.NotEqual(t, parts1[0], parts2[0], "salts should be different")
}

func TestIsPBKDF2Hash_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    string
		expected bool
	}{
		{"Exact prefix", "$pbkdf2$", false},
		{"Prefix with content", "$pbkdf2$abc123", false},
		{"Similar but wrong prefix", "$pbkdf2", false},
		{"Similar but wrong prefix 2", "pbkdf2$", false},
		{"Case sensitive", "$PBKDF2$", false},
		{"With whitespace", " $pbkdf2$", false},
		{"Multiple prefixes", "$pbkdf2$$pbkdf2$", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, IsPBKDF2Hash(tc.value))
		})
	}
}
