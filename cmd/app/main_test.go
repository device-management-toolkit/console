package main

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"errors"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/security"

	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/internal/app"
	"github.com/device-management-toolkit/console/internal/certificates"
	"github.com/device-management-toolkit/console/pkg/logger"
	"github.com/device-management-toolkit/console/pkg/secrets"
)

type mockCredentialStore struct {
	values      map[string]string
	errMap      map[string]error
	deletedKeys []string
}

func TestLogSetupURLOnlyWhenUIUnavailable(t *testing.T) {
	var output bytes.Buffer

	originalOutput := log.Writer()
	originalIsTerminal := logWriterIsTerminal

	log.SetOutput(&output)
	t.Cleanup(func() {
		log.SetOutput(originalOutput)

		logWriterIsTerminal = originalIsTerminal
	})
	t.Setenv("NO_COLOR", "1")

	// Interactive terminal: same line as before, token included for copy/paste.
	logWriterIsTerminal = func() bool { return true }

	logSetupURLIfUIUnavailable("https://localhost:44283", "one-time-token", false)
	assert.Contains(t, output.String(), "Headless setup URL (one-time token expires in 10 minutes): https://localhost:44283/setup#one-time-token")

	output.Reset()
	logSetupURLIfUIUnavailable("https://localhost:44283", "one-time-token", true)
	assert.Empty(t, output.String())

	// Redirected output (file, pipe, journal): the token must not be written.
	logWriterIsTerminal = func() bool { return false }

	logSetupURLIfUIUnavailable("https://localhost:44283", "one-time-token", false)
	assert.NotContains(t, output.String(), "one-time-token")
	assert.Contains(t, output.String(), "https://localhost:44283/setup/redirect")
}

func TestSetupURLLogMessageCanBeColored(t *testing.T) {
	t.Parallel()

	message := setupURLLogMessage("https://localhost:44283", "one-time-token", true)

	assert.Contains(t, message, "\033[1;36mHeadless setup URL\033[0m")
	assert.Contains(t, message, "\033[1;33mhttps://localhost:44283/setup#one-time-token\033[0m")
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
			assert.True(t, secrets.IsStrongAdminPassword(password), "generated %q fails the policy", password)
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

//nolint:paralleltest // mutates the global keyring factory
func TestHandleAdminCredentials_FirstRunCreatesSetupTokenWithoutPrompting(t *testing.T) {
	store := &mockCredentialStore{values: map[string]string{}}
	origFunc := newKeyringStorageFunc
	newKeyringStorageFunc = func() credentialStore { return store }

	t.Cleanup(func() {
		newKeyringStorageFunc = origFunc
		app.AuthSetup = nil
	})

	cfg := &config.Config{Auth: config.Auth{SecretBackend: "keyring"}}
	handleAdminCredentials(cfg)

	assert.Empty(t, store.values, "first-run startup must not write credentials")
	assert.Empty(t, cfg.AdminUsername)
	assert.Empty(t, cfg.AdminPassword)
	assert.NotEmpty(t, cfg.SetupToken)
	require.NotNil(t, app.AuthSetup)
	assert.Equal(t, cfg.SetupToken, app.AuthSetup.SetupToken())
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

func TestPrepareSetupListener(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{}
	require.NoError(t, prepareSetupListener(cfg))
	require.Nil(t, cfg.SetupListener, "no listener unless setup is pending")

	cfg.SetupToken = "pending"
	require.NoError(t, prepareSetupListener(cfg))
	require.NotNil(t, cfg.SetupListener)

	t.Cleanup(func() { _ = cfg.SetupListener.Close() })

	addr, ok := cfg.SetupListener.Addr().(*net.TCPAddr)
	require.True(t, ok)
	require.True(t, addr.IP.IsLoopback(), "setup listener must bind to loopback only")
	require.Equal(t, "https://localhost:"+strconv.Itoa(addr.Port), cfg.SetupAPIBase)
}
