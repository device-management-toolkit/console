package main

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"errors"
	"flag"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/security"

	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/internal/certificates"
	"github.com/device-management-toolkit/console/pkg/logger"
	"github.com/device-management-toolkit/console/pkg/secrets"
)

type mockCredentialStore struct {
	values      map[string]string
	errMap      map[string]error
	deletedKeys []string
}

func (m *mockCredentialStore) GetKeyValue(key string) (string, error) {
	if err, ok := m.errMap[key]; ok {
		return "", err
	}

	if v, ok := m.values[key]; ok {
		return v, nil
	}

	return "", security.ErrKeyNotFound
}

func (m *mockCredentialStore) SetKeyValue(key, value string) error {
	if err, ok := m.errMap[key+":set"]; ok {
		return err
	}

	if m.values == nil {
		m.values = map[string]string{}
	}

	m.values[key] = value

	return nil
}

func (m *mockCredentialStore) DeleteKeyValue(key string) error {
	if err, ok := m.errMap[key+":delete"]; ok {
		return err
	}

	m.deletedKeys = append(m.deletedKeys, key)
	delete(m.values, key)

	return nil
}

func withStdio(t *testing.T, input string) func() string {
	t.Helper()

	stdinReader, stdinWriter, err := os.Pipe()
	require.NoError(t, err)
	_, err = stdinWriter.WriteString(input)
	require.NoError(t, err)
	require.NoError(t, stdinWriter.Close())

	stdoutReader, stdoutWriter, err := os.Pipe()
	require.NoError(t, err)

	oldStdin, oldStdout := os.Stdin, os.Stdout
	os.Stdin, os.Stdout = stdinReader, stdoutWriter

	finished := false
	finish := func() string {
		if finished {
			return ""
		}

		finished = true
		os.Stdin, os.Stdout = oldStdin, oldStdout
		_ = stdoutWriter.Close()
		output, err := io.ReadAll(stdoutReader)
		require.NoError(t, err)

		_ = stdinReader.Close()
		_ = stdoutReader.Close()

		return string(output)
	}

	t.Cleanup(func() {
		if !finished {
			finish()
		}
	})

	return finish
}

func TestMainFunction(_ *testing.T) { //nolint:paralleltest // cannot have simultaneous tests modifying env variables.
	os.Setenv("GIN_MODE", "debug")

	// Mock functions
	initializeConfigFunc = func() (*config.Config, error) {
		return &config.Config{
			HTTP: config.HTTP{Port: "8080"},
			App:  config.App{EncryptionKey: "test"},
			Log:  config.Log{Level: "info"},
			Auth: config.Auth{Disabled: true},
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

	assert.Equal(t, "already-set", cfg.AdminPassword)
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

	assert.Equal(t, "weak", cfg.AdminPassword)
	assert.Contains(t, buf.String(), "Console is starting anyway")
}

// TestHandleAdminPassword_AuthDisabled verifies no password is generated (and
// nothing is persisted to config.yml) when authentication is turned off.
func TestHandleAdminPassword_AuthDisabled(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Auth: config.Auth{
			Disabled:      true,
			AdminPassword: "",
		},
	}

	handleAdminPassword(cfg)

	assert.Empty(t, cfg.AdminPassword)
}

func TestLoadKeyringAdminCredentialsUsesOnlyKeyring(t *testing.T) {
	t.Parallel()

	hash, err := secrets.GeneratePBKDF2Hash("keyring-password")
	require.NoError(t, err)

	cfg := &config.Config{Auth: config.Auth{
		AdminUsername: "config-user",
		AdminPassword: "config-password",
	}}
	store := &mockCredentialStore{values: map[string]string{
		keyringAdminUsername: "keyring-user",
		keyringAdminPassword: hash,
	}}

	require.NoError(t, loadKeyringAdminCredentials(cfg, store))
	assert.Equal(t, "keyring-user", cfg.AdminUsername)
	assert.Equal(t, hash, cfg.AdminPassword)
}

func TestLoadKeyringAdminCredentialsUnavailableStoreHasRecoveryGuidance(t *testing.T) {
	t.Parallel()

	err := loadKeyringAdminCredentials(&config.Config{}, nil)
	require.ErrorIs(t, err, errKeystoreUnavailable)
	assert.Contains(t, err.Error(), "installed, unlocked, and accessible")
}

func TestLoadKeyringAdminCredentialsReadErrorHasRecoveryGuidance(t *testing.T) {
	t.Parallel()

	const lockedError = "keyring locked"

	hash, err := secrets.GeneratePBKDF2Hash("stored-password")
	require.NoError(t, err)

	initialValues := map[string]string{
		keyringAdminUsername: "keyring-user",
		keyringAdminPassword: hash,
	}

	tests := []struct {
		name        string
		key         string
		wantMessage string
	}{
		{
			name:        "username",
			key:         keyringAdminUsername,
			wantMessage: "reading administrator username from OS keyring",
		},
		{
			name:        "password hash",
			key:         keyringAdminPassword,
			wantMessage: "reading administrator password hash from OS keyring",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := &mockCredentialStore{
				values: map[string]string{
					keyringAdminUsername: initialValues[keyringAdminUsername],
					keyringAdminPassword: initialValues[keyringAdminPassword],
				},
				errMap: map[string]error{
					tt.key: errors.New(lockedError),
				},
			}

			cfg := &config.Config{Auth: config.Auth{
				AdminUsername: "config-user",
				AdminPassword: "config-password",
			}}
			err := loadKeyringAdminCredentials(cfg, store)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantMessage)
			assert.Contains(t, err.Error(), "available and unlocked")
			assert.ErrorContains(t, err, lockedError)
			assert.Equal(t, "config-user", cfg.AdminUsername)
			assert.Equal(t, "config-password", cfg.AdminPassword)
			assert.Equal(t, initialValues, store.values)
		})
	}
}

func TestLoadKeyringAdminCredentialsUnavailableServiceDoesNotBootstrap(t *testing.T) {
	t.Parallel()

	serviceErr := errors.New("secret service unavailable: D-Bus session is not available")
	hash, err := secrets.GeneratePBKDF2Hash("stored-password")
	require.NoError(t, err)

	initialValues := map[string]string{
		keyringAdminUsername: "stored-user",
		keyringAdminPassword: hash,
	}
	store := &mockCredentialStore{
		values: map[string]string{
			keyringAdminUsername: initialValues[keyringAdminUsername],
			keyringAdminPassword: initialValues[keyringAdminPassword],
		},
		errMap: map[string]error{
			keyringAdminUsername: serviceErr,
			keyringAdminPassword: serviceErr,
		},
	}
	cfg := &config.Config{Auth: config.Auth{
		AdminUsername: "config-user",
		AdminPassword: "config-password",
	}}

	err = loadKeyringAdminCredentials(cfg, store)
	require.ErrorIs(t, err, serviceErr)
	assert.Contains(t, err.Error(), "ensure the keyring is available and unlocked")
	assert.Equal(t, "config-user", cfg.AdminUsername)
	assert.Equal(t, "config-password", cfg.AdminPassword)
	assert.Equal(t, initialValues, store.values)
}

func TestLoadKeyringAdminCredentialsRejectsPartialState(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{Auth: config.Auth{
		AdminUsername: "config-user",
		AdminPassword: "config-password",
	}}
	store := &mockCredentialStore{values: map[string]string{
		keyringAdminUsername: "keyring-user",
	}}

	err := loadKeyringAdminCredentials(cfg, store)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "partial")
	assert.Equal(t, "config-user", cfg.AdminUsername)
	assert.Equal(t, "config-password", cfg.AdminPassword)
}

//nolint:paralleltest // temporarily replaces process stdin and stdout
func TestBootstrapKeyringAdminCredentialsRejectsPasswordMismatch(t *testing.T) {
	finishStdio := withStdio(t, "admin\nfirst-password\nsecond-password\nsecond-password\nsecond-password\n")

	cfg := &config.Config{}
	store := &mockCredentialStore{values: map[string]string{}}

	err := bootstrapKeyringAdminCredentials(cfg, store)
	require.ErrorIs(t, err, errPasswordConfirmation)
	assert.Empty(t, store.values, "mismatched input must not write credentials")
	assert.Empty(t, cfg.AdminUsername)
	assert.Empty(t, cfg.AdminPassword)
	finishStdio()
}

//nolint:paralleltest // temporarily replaces process stdin and stdout
func TestBootstrapKeyringAdminCredentialsRejectsWeakPassword(t *testing.T) {
	for _, password := range []string{"weak", "aaaaaaaa", "éñøü"} {
		t.Run(password, func(t *testing.T) {
			finishStdio := withStdio(t, "admin\n"+strings.Repeat(password+"\n", maxPasswordAttempts))

			cfg := &config.Config{}
			store := &mockCredentialStore{values: map[string]string{}}

			err := bootstrapKeyringAdminCredentials(cfg, store)
			require.ErrorIs(t, err, errWeakAdminPassword)
			assert.Empty(t, store.values, "weak input must not write credentials")
			assert.Empty(t, cfg.AdminUsername)
			assert.Empty(t, cfg.AdminPassword)
			finishStdio()
		})
	}
}

//nolint:paralleltest // temporarily replaces process stdin
func TestBootstrapKeyringAdminCredentialsRetriesWeakPassword(t *testing.T) {
	finishStdio := withStdio(t, "admin\nweak\naaaaaaaa\nvalid-password\nvalid-password\n")

	cfg := &config.Config{}
	store := &mockCredentialStore{values: map[string]string{}}
	require.NoError(t, bootstrapKeyringAdminCredentials(cfg, store))
	assert.Equal(t, "admin", cfg.AdminUsername)
	assert.True(t, secrets.VerifyPBKDF2Hash(cfg.AdminPassword, "valid-password"))
	assert.Equal(t, cfg.AdminPassword, store.values[keyringAdminPassword])
	finishStdio()
}

//nolint:paralleltest // temporarily replaces process stdin and stdout
func TestBootstrapKeyringAdminCredentialsRetriesInvalidUsername(t *testing.T) {
	finishStdio := withStdio(t, "\nbad name\nvalid_user\nvalid-password\nvalid-password\n")
	cfg := &config.Config{}
	store := &mockCredentialStore{values: map[string]string{}}

	require.NoError(t, bootstrapKeyringAdminCredentials(cfg, store))
	assert.Equal(t, "valid_user", cfg.AdminUsername)
	finishStdio()
}

func TestIsValidAdminUsername(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		username string
		want     bool
	}{
		{name: "valid", username: "admin_user-1.test", want: true},
		{name: "empty", username: "", want: false},
		{name: "too long", username: strings.Repeat("a", 65), want: false},
		{name: "whitespace", username: "admin user", want: false},
		{name: "non-ASCII", username: "管理员", want: false},
		{name: "invalid leading character", username: ".admin", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, isValidAdminUsername(tt.username))
		})
	}
}

//nolint:paralleltest // temporarily replaces process stdin and stdout
func TestBootstrapKeyringAdminCredentialsRejectsPasswordWhitespace(t *testing.T) {
	finishStdio := withStdio(t, "admin\n valid-password\nvalid-password \nvalid-password\nvalid-password\n")
	cfg := &config.Config{}
	store := &mockCredentialStore{values: map[string]string{}}

	require.NoError(t, bootstrapKeyringAdminCredentials(cfg, store))
	assert.True(t, secrets.VerifyPBKDF2Hash(cfg.AdminPassword, "valid-password"))
	finishStdio()
}

//nolint:paralleltest // temporarily replaces process stdin and stdout
func TestBootstrapKeyringAdminCredentialsRollsBackUsernameOnPasswordWriteFailure(t *testing.T) {
	finishStdio := withStdio(t, "admin\nvalid-password\nvalid-password\n")
	storeErr := errors.New("keyring write failed")
	store := &mockCredentialStore{
		values: map[string]string{},
		errMap: map[string]error{keyringAdminPassword + ":set": storeErr},
	}
	cfg := &config.Config{}

	err := bootstrapKeyringAdminCredentials(cfg, store)
	require.ErrorIs(t, err, storeErr)
	assert.Empty(t, store.values)
	assert.Equal(t, []string{keyringAdminUsername}, store.deletedKeys)
	assert.Empty(t, cfg.AdminUsername)
	assert.Empty(t, cfg.AdminPassword)
	finishStdio()
}

//nolint:paralleltest // temporarily replaces process stdin and stdout
func TestLoadKeyringAdminCredentialsReusesStoredCredentialsWithoutPrompting(t *testing.T) {
	hash, err := secrets.GeneratePBKDF2Hash("valid-password")
	require.NoError(t, err)
	store := &mockCredentialStore{values: map[string]string{
		keyringAdminUsername: "valid_user",
		keyringAdminPassword: hash,
	}}
	cfg := &config.Config{}
	finishStdio := withStdio(t, "")

	require.NoError(t, loadKeyringAdminCredentials(cfg, store))
	assert.Equal(t, "valid_user", cfg.AdminUsername)
	assert.Equal(t, hash, cfg.AdminPassword)
	assert.Empty(t, finishStdio(), "reusing keyring credentials must not prompt")
}

//nolint:paralleltest // mutates global config and flag state
func TestHandleAdminCredentials_FirstRunBootstrapsAndPersistsToKeyring(t *testing.T) {
	configFile := filepath.Join(t.TempDir(), "config.yml")
	initialConfig := []byte("auth:\n  jwtExpiration: 24h\n")
	require.NoError(t, os.WriteFile(configFile, initialConfig, 0o600))

	flagValue := flag.Lookup("config")
	if flagValue == nil {
		flag.String("config", "", "path to config file")

		flagValue = flag.Lookup("config")
	}

	oldConfigValue := flagValue.Value.String()

	require.NoError(t, flag.Set("config", configFile))

	t.Cleanup(func() {
		_ = flag.Set("config", oldConfigValue)
	})

	const plaintextPassword = "secret"
	finishStdio := withStdio(t, "admin\n\n"+plaintextPassword+"\nwrong\n"+plaintextPassword+"\n")

	var logOutput bytes.Buffer

	oldLogOutput := log.Writer()

	log.SetOutput(&logOutput)

	t.Cleanup(func() { log.SetOutput(oldLogOutput) })

	store := &mockCredentialStore{values: map[string]string{}}
	origFunc := newKeyringStorageFunc
	newKeyringStorageFunc = func() credentialStore { return store }

	t.Cleanup(func() { newKeyringStorageFunc = origFunc })

	cfg := &config.Config{Auth: config.Auth{SecretBackend: "keyring"}}
	handleAdminCredentials(cfg)

	stdout := finishStdio()

	assert.Equal(t, "admin", cfg.AdminUsername)
	assert.True(t, secrets.VerifyPBKDF2Hash(cfg.AdminPassword, plaintextPassword))
	assert.Equal(t, "admin", store.values[keyringAdminUsername])
	assert.Equal(t, cfg.AdminPassword, store.values[keyringAdminPassword])
	assert.NotEqual(t, plaintextPassword, store.values[keyringAdminPassword])
	assert.Len(t, store.values, 2)
	assert.False(t, bytes.Contains(stdout, []byte(plaintextPassword)))
	assert.NotContains(t, logOutput.String(), plaintextPassword)

	configData, err := os.ReadFile(configFile)
	require.NoError(t, err)
	assert.Equal(t, initialConfig, configData, "keyring setup must not modify config.yml")
	assert.NotContains(t, string(configData), plaintextPassword)
}

//nolint:paralleltest // mutates os.Stdin
func TestBootstrapKeyringAdminCredentialsFailsAfterConfirmationRetries(t *testing.T) {
	oldStdin := os.Stdin
	reader, writer, err := os.Pipe()
	require.NoError(t, err)
	_, err = writer.WriteString("admin\npassword\nwrong-1\nwrong-2\nwrong-3\n")
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	os.Stdin = reader

	t.Cleanup(func() { os.Stdin = oldStdin; _ = reader.Close() })

	store := &mockCredentialStore{values: map[string]string{}}
	err = bootstrapKeyringAdminCredentials(&config.Config{}, store)
	require.ErrorIs(t, err, errPasswordConfirmation)
	assert.Empty(t, store.values, "credentials must not be stored after failed confirmation")
}

//nolint:paralleltest // mutates the global newKeyringStorageFunc
func TestHandleAdminCredentials_OAuthConfiguredSkipsBootstrap(t *testing.T) {
	cfg := &config.Config{
		Auth: config.Auth{
			ClientID: "oauth-client-id",
		},
	}

	var callCount int

	origFunc := newKeyringStorageFunc
	newKeyringStorageFunc = func() credentialStore {
		callCount++

		return nil
	}

	t.Cleanup(func() { newKeyringStorageFunc = origFunc })

	handleAdminCredentials(cfg)
	assert.Equal(t, 0, callCount, "keyring storage should not be accessed when OAuth2 is configured")
}

//nolint:paralleltest // mutates the global newKeyringStorageFunc
func TestHandleAdminCredentials_ConfigBackendDoesNotUseKeyring(t *testing.T) {
	cfg := &config.Config{Auth: config.Auth{
		SecretBackend: "config",
		AdminUsername: "config-user",
		AdminPassword: "config-password",
	}}

	var callCount int

	origFunc := newKeyringStorageFunc
	newKeyringStorageFunc = func() credentialStore {
		callCount++

		return nil
	}

	t.Cleanup(func() { newKeyringStorageFunc = origFunc })

	handleAdminCredentials(cfg)

	assert.Zero(t, callCount)
	assert.Equal(t, "config-user", cfg.AdminUsername)
	assert.Equal(t, "config-password", cfg.AdminPassword)
}
