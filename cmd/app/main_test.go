package main

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"errors"
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

func TestMainFunction(_ *testing.T) { //nolint:paralleltest // cannot have simultaneous tests modifying env variables.
	os.Setenv("GIN_MODE", "debug")

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

	loadOrGenerateRootCertFunc = func(_ security.Storager, _ bool, _, _, _ string, _ bool) (*x509.Certificate, *rsa.PrivateKey, error) {
		return &x509.Certificate{}, &rsa.PrivateKey{}, nil
	}

	loadOrGenerateWebServerCertFunc = func(_ security.Storager, _ certificates.CertAndKeyType, _ bool, _, _, _ string, _ bool) (*x509.Certificate, *rsa.PrivateKey, error) {
		return &x509.Certificate{}, &rsa.PrivateKey{}, nil
	}

	main()
}

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

// TestGenerateRandomPassword_ShellSafe guards the accept-wide/generate-narrow split.
func TestGenerateRandomPassword_ShellSafe(t *testing.T) {
	t.Parallel()

	const unsafeChars = "$!#%^&<>|\"'`\\ "

	for range 100 {
		password, err := generateRandomPassword(adminPasswordLength)
		require.NoError(t, err)
		assert.True(t, isStrongAdminPassword(password))
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

	t.Cleanup(func() { newKeyringStorageFunc = origFunc })

	cfg := &config.Config{Auth: config.Auth{
		SecretBackend: "keyring",
		StatePath:     filepath.Join(t.TempDir(), "auth-state.db"),
	}}
	handleAdminCredentials(cfg)

	assert.Empty(t, store.values, "first-run startup must not write credentials")
	assert.Empty(t, cfg.AdminUsername)
	assert.Empty(t, cfg.AdminPassword)
	assert.NotEmpty(t, cfg.SetupToken)
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
