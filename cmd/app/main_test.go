package main

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"flag"
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

	oldStdin := os.Stdin
	reader, writer, err := os.Pipe()
	require.NoError(t, err)
	_, err = writer.WriteString("admin\npassword\npassword\n")
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	os.Stdin = reader

	t.Cleanup(func() { os.Stdin = oldStdin; _ = reader.Close() })

	store := &mockCredentialStore{values: map[string]string{}}
	origFunc := newKeyringStorageFunc
	newKeyringStorageFunc = func() credentialStore { return store }

	t.Cleanup(func() { newKeyringStorageFunc = origFunc })

	cfg := &config.Config{Auth: config.Auth{SecretBackend: "keyring"}}
	handleAdminCredentials(cfg)

	assert.Equal(t, "admin", cfg.AdminUsername)
	assert.True(t, secrets.VerifyPBKDF2Hash(cfg.AdminPassword, "password"))
	assert.Equal(t, "admin", store.values[keyringAdminUsername])
	assert.Equal(t, cfg.AdminPassword, store.values[keyringAdminPassword])
	assert.Len(t, store.values, 2)

	configData, err := os.ReadFile(configFile)
	require.NoError(t, err)
	assert.Equal(t, initialConfig, configData, "keyring setup must not modify config.yml")
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
