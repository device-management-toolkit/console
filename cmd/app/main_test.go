package main

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"flag"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/security"

	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/internal/certificates"
	"github.com/device-management-toolkit/console/pkg/logger"
)

func TestMainFunction(_ *testing.T) { //nolint:paralleltest // cannot have simultaneous tests modifying env variables.
	os.Setenv("GIN_MODE", "debug")

	// Mock functions
	initializeConfigFunc = func() (*config.Config, error) {
		return &config.Config{
			HTTP: config.HTTP{Port: "8080"},
			App:  config.App{EncryptionKey: "test"},
			Log:  config.Log{Level: "info"},
			Auth: config.Auth{AdminPassword: "test"},
		}, nil
	}

	initializeAppFunc = func(_ *config.Config) error {
		return nil
	}

	runAppFunc = func(_ *config.Config, _ logger.Interface) {}

	// Mock certificate functions
	loadOrGenerateRootCertFunc = func(_ security.Storager, _ bool, _, _, _ string, _ bool) (*x509.Certificate, *rsa.PrivateKey, error) {
		return &x509.Certificate{}, &rsa.PrivateKey{}, nil
	}

	loadOrGenerateWebServerCertFunc = func(_ security.Storager, _ certificates.CertAndKeyType, _ bool, _, _, _ string, _ bool) (*x509.Certificate, *rsa.PrivateKey, error) {
		return &x509.Certificate{}, &rsa.PrivateKey{}, nil
	}

	// Call the main function
	main()
}

// TestGenerateRandomPassword tests the password generation function.
func TestGenerateRandomPassword(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		length int
	}{
		{"length 8", 8},
		{"length 16", 16},
		{"length 32", 32},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			password, err := generateRandomPassword(tc.length)
			require.NoError(t, err)
			assert.Len(t, password, tc.length)
		})
	}
}

// TestGenerateRandomPassword_SatisfiesPolicy locks the generator to the checker:
// a per-class miss would otherwise surface as a rare flake rather than a failure.
func TestGenerateRandomPassword_SatisfiesPolicy(t *testing.T) {
	t.Parallel()

	for _, length := range []int{adminPasswordMinLength, adminPasswordLength, 64} {
		for range 100 {
			password, err := generateRandomPassword(length)
			require.NoError(t, err)
			assert.True(t, isStrongAdminPassword(password), "generated %q fails the policy", password)
		}
	}
}

// TestGenerateRandomPassword_ShellSafe guards the accept-wide/generate-narrow split.
func TestGenerateRandomPassword_ShellSafe(t *testing.T) {
	t.Parallel()

	const unsafeChars = "$!#%^&<>|\"'`\\ "

	for range 100 {
		password, err := generateRandomPassword(adminPasswordLength)
		require.NoError(t, err)
		assert.NotContains(t, password, "$")
		assert.NotContains(t, password, "!")
		assert.False(t, strings.ContainsAny(password, unsafeChars), "generated %q contains an unsafe character", password)
	}
}

func TestGenerateRandomPassword_LengthTooShort(t *testing.T) {
	t.Parallel()

	_, err := generateRandomPassword(adminPasswordMinLength - 1)
	require.ErrorIs(t, err, ErrPasswordLengthTooShort)
}

// TestGenerateRandomPassword_Uniqueness ensures generated passwords are unique.
func TestGenerateRandomPassword_Uniqueness(t *testing.T) {
	t.Parallel()

	passwords := make(map[string]bool)

	for range 100 {
		password, err := generateRandomPassword(16)
		require.NoError(t, err)
		assert.False(t, passwords[password], "generated duplicate password")

		passwords[password] = true
	}
}

// TestCheckStoredEncryptionKey covers the non-fatal paths: a usable key is
// silent, a weak but correctly sized key warns and lets Console start.
func TestCheckStoredEncryptionKey(t *testing.T) { //nolint:paralleltest // rebinds the shared log output
	tests := []struct {
		name        string
		key         string
		wantWarning bool
	}{
		{"usable key", "Jf3Q2nXJ+GZzN1dbVQms0wbB4+i/5PjL", false},
		{"weak key", "aaaaaaaaaaaaaaaa", true},
	}

	for _, tc := range tests { //nolint:paralleltest // subtests rebind the shared log output
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer

			log.SetOutput(&out)

			defer log.SetOutput(os.Stderr)

			checkStoredEncryptionKey(tc.key, "local keyring")

			if tc.wantWarning {
				assert.Contains(t, out.String(), "weak")
			} else {
				assert.Empty(t, out.String())
			}
		})
	}
}

// TestHandleAdminPassword_AlreadyConfigured tests when password is already set.
func TestHandleAdminPassword_AlreadyConfigured(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Auth: config.Auth{
			AdminPassword: "already-set",
		},
	}

	handleAdminPassword(cfg)

	assert.True(t, isBcryptHash(cfg.AdminPassword))
	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(cfg.AdminPassword), []byte("already-set")))
}

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
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, isStrongAdminPassword(tc.password))
		})
	}
}

func TestWarnOnWeakAdminPassword(t *testing.T) { //nolint:paralleltest // rebinds the global log output.
	tests := []struct {
		name     string
		password string
		wantLog  string
	}{
		{
			name:     "weak password warns",
			password: "weak",
			wantLog:  "WARNING: the configured admin password is weak",
		},
		{
			name:     "long but not complex still warns",
			password: "alllowercaseletters",
			wantLog:  "WARNING: the configured admin password is weak",
		},
		{
			name:     "compliant password is silent",
			password: "P@ssw0rdd",
			wantLog:  "",
		},
	}

	for _, tc := range tests { //nolint:paralleltest // rebinds the global log output.
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer

			orig := log.Writer()

			log.SetOutput(&buf)
			t.Cleanup(func() { log.SetOutput(orig) })

			warnOnWeakAdminPassword(tc.password)

			if tc.wantLog == "" {
				assert.Empty(t, buf.String())

				return
			}

			assert.Contains(t, buf.String(), tc.wantLog)
		})
	}
}

func TestHandleAdminPassword_WeakConfiguredPasswordStillStarts(t *testing.T) { //nolint:paralleltest // rebinds the global log output.
	var buf bytes.Buffer

	orig := log.Writer()

	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(orig) })

	cfg := &config.Config{
		Auth: config.Auth{
			AdminPassword: "weak",
		},
	}

	handleAdminPassword(cfg)

	assert.True(t, isBcryptHash(cfg.AdminPassword))
	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(cfg.AdminPassword), []byte("weak")))
	assert.Contains(t, buf.String(), "Console is starting anyway")
}

func TestHandleAdminPassword_GeneratesAndPersistsWhenUnset(t *testing.T) { //nolint:paralleltest // rebinds the global log output and config flag
	var buf bytes.Buffer

	orig := log.Writer()

	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(orig) })

	configPath := filepath.Join(t.TempDir(), "config.yml")

	if flag.Lookup("config") == nil {
		flag.String("config", "", "path to config file")
	}

	prev := flag.Lookup("config").Value.String()

	require.NoError(t, flag.Set("config", configPath))
	t.Cleanup(func() { _ = flag.Set("config", prev) })

	cfg := &config.Config{}

	handleAdminPassword(cfg)

	assert.True(t, isBcryptHash(cfg.AdminPassword), "generated password must be stored as a hash")

	// The operator can only ever learn the generated password from this output,
	// so it must be printed and must match the stored hash.
	shown := regexp.MustCompile(`\n\n {4}(\S+)\n\n`).FindStringSubmatch(buf.String())
	require.Len(t, shown, 2, "generated password must be shown once: %s", buf.String())
	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(cfg.AdminPassword), []byte(shown[1])))

	saved, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Contains(t, string(saved), cfg.AdminPassword, "hash must be persisted so it survives restart")
	assert.NotContains(t, string(saved), shown[1], "plaintext must never be written to config")
}

func TestHandleAdminPassword_KeepsExistingHashUnchanged(t *testing.T) {
	t.Parallel()

	hash, err := bcrypt.GenerateFromPassword([]byte("P@ssw0rdd"), bcrypt.DefaultCost)
	require.NoError(t, err)

	cfg := &config.Config{Auth: config.Auth{AdminPassword: string(hash)}}

	handleAdminPassword(cfg)

	assert.Equal(t, string(hash), cfg.AdminPassword, "an already-hashed password must not be re-hashed")
}

func TestHandleAdminPassword_StartsWhenConfigIsNotWritable(t *testing.T) { //nolint:paralleltest // rebinds the global log output and config flag
	var buf bytes.Buffer

	orig := log.Writer()

	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(orig) })

	// A path under a regular file cannot be written, standing in for a read-only
	// config or a password supplied entirely via the environment.
	blocker := filepath.Join(t.TempDir(), "not-a-dir")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0o600))

	if flag.Lookup("config") == nil {
		flag.String("config", "", "path to config file")
	}

	prev := flag.Lookup("config").Value.String()

	require.NoError(t, flag.Set("config", filepath.Join(blocker, "config.yml")))
	t.Cleanup(func() { _ = flag.Set("config", prev) })

	cfg := &config.Config{Auth: config.Auth{AdminPassword: "P@ssw0rdd"}}

	handleAdminPassword(cfg)

	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(cfg.AdminPassword), []byte("P@ssw0rdd")),
		"startup must continue with the in-memory hash")
	assert.Contains(t, buf.String(), "could not persist the hashed admin password")
}

func TestNormalizeAdminPasswordHash(t *testing.T) {
	t.Parallel()

	t.Run("empty stays empty", func(t *testing.T) {
		t.Parallel()

		got, converted, err := normalizeAdminPasswordHash("")

		require.NoError(t, err)
		assert.False(t, converted)
		assert.Empty(t, got)
	})

	t.Run("plaintext beginning with a bcrypt prefix is still hashed", func(t *testing.T) {
		t.Parallel()

		plaintext := "$2a$notarealhash"

		got, converted, err := normalizeAdminPasswordHash(plaintext)

		require.NoError(t, err)
		assert.True(t, converted, "a prefix alone must not be treated as a hash")
		assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(got), []byte(plaintext)))
	})

	t.Run("plaintext is hashed", func(t *testing.T) {
		t.Parallel()

		got, converted, err := normalizeAdminPasswordHash("P@ssw0rdd")

		require.NoError(t, err)
		assert.True(t, converted)
		assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(got), []byte("P@ssw0rdd")))
	})

	t.Run("existing hash is returned as is", func(t *testing.T) {
		t.Parallel()

		hash, err := bcrypt.GenerateFromPassword([]byte("P@ssw0rdd"), bcrypt.DefaultCost)
		require.NoError(t, err)

		got, converted, err := normalizeAdminPasswordHash(string(hash))

		require.NoError(t, err)
		assert.False(t, converted)
		assert.Equal(t, string(hash), got)
	})
}
