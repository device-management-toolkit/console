package main

import (
	"bufio"
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"errors"
	"os"
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
			Auth: config.Auth{AdminUsername: "admin", AdminPassword: "test"},
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

func TestHandleAdminPassword_AlreadyConfigured(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Auth: config.Auth{AdminPassword: "already-set"},
	}

	handleAdminPassword(cfg)

	assert.Equal(t, "already-set", cfg.AdminPassword)
}

func TestNormalizeAdminPasswordHash_PlainTextInput(t *testing.T) {
	t.Parallel()

	hash, converted, err := normalizeAdminPasswordHash("plain-password")
	require.NoError(t, err)
	assert.True(t, converted)
	require.NoError(t, bcrypt.CompareHashAndPassword([]byte(hash), []byte("plain-password")))
}

func TestNormalizeAdminPasswordHash_AlreadyHashed(t *testing.T) {
	t.Parallel()

	existingHash, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	require.NoError(t, err)

	hash, converted, err := normalizeAdminPasswordHash(string(existingHash))
	require.NoError(t, err)
	assert.False(t, converted)
	assert.Equal(t, string(existingHash), hash)
}

func TestResolveAdminCredentialsFromSources_PriorityOrder(t *testing.T) {
	t.Setenv(authAdminUsernameEnv, "env-user")
	t.Setenv(authAdminSecretEnv, "env-pass")

	cfg := &config.Config{Auth: config.Auth{AdminUsername: "cfg-user", AdminPassword: "cfg-pass"}}
	store := &mockCredentialStore{values: map[string]string{
		keyringAdminUsername: "keyring-user",
		keyringAdminPassword: "keyring-pass",
	}}

	username, password := resolveAdminCredentialsFromSources(cfg, store, map[string]string{
		authAdminUsernameEnv: "dotenv-user",
		authAdminSecretEnv:   "dotenv-pass",
	})

	assert.Equal(t, "keyring-user", username)
	assert.Equal(t, "keyring-pass", password)
}

func TestResolveAdminCredentialsFromSources_FallbackToDotEnvThenConfig(t *testing.T) {
	t.Setenv(authAdminUsernameEnv, "")
	t.Setenv(authAdminSecretEnv, "")

	cfg := &config.Config{Auth: config.Auth{AdminUsername: "cfg-user", AdminPassword: "cfg-pass"}}
	store := &mockCredentialStore{errMap: map[string]error{
		keyringAdminUsername: security.ErrKeyNotFound,
		keyringAdminPassword: security.ErrKeyNotFound,
	}}

	username, password := resolveAdminCredentialsFromSources(cfg, store, map[string]string{
		authAdminUsernameEnv: "dotenv-user",
	})

	assert.Equal(t, "dotenv-user", username)
	assert.Equal(t, "cfg-pass", password)
}

func TestResolveAdminCredentialsFromSources_KeyringReadErrorFallsBack(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{Auth: config.Auth{AdminUsername: "cfg-user", AdminPassword: "cfg-pass"}}
	store := &mockCredentialStore{errMap: map[string]error{
		keyringAdminUsername: errors.New("keyring unavailable"),
		keyringAdminPassword: errors.New("keyring unavailable"),
	}}

	username, password := resolveAdminCredentialsFromSources(cfg, store, map[string]string{})

	assert.Equal(t, "cfg-user", username)
	assert.Equal(t, "cfg-pass", password)
}

func TestConfirmPersistCredentialsToConfig_Yes(t *testing.T) {
	t.Parallel()

	reader := bufio.NewReader(strings.NewReader("Y\n"))
	assert.True(t, confirmPersistCredentialsToConfig(reader))
}

func TestConfirmPersistCredentialsToConfig_No(t *testing.T) {
	t.Parallel()

	reader := bufio.NewReader(strings.NewReader("n\n"))
	assert.False(t, confirmPersistCredentialsToConfig(reader))
}

func TestHandleAdminCLI_ShowAdmin_HidesPassword(t *testing.T) {
	t.Parallel()

	store := &mockCredentialStore{values: map[string]string{
		keyringAdminUsername: "alice",
		keyringAdminPassword: "hash-value",
	}}
	buf := &bytes.Buffer{}

	handled, err := handleAdminCLI([]string{"--show-admin"}, store, bufio.NewReader(strings.NewReader("")), buf)
	require.NoError(t, err)
	assert.True(t, handled)
	assert.Contains(t, buf.String(), "admin username: alice")
	assert.NotContains(t, buf.String(), "hash-value")
}

func TestHandleAdminCLI_RemoveAdmin(t *testing.T) {
	t.Parallel()

	store := &mockCredentialStore{values: map[string]string{
		keyringAdminUsername: "alice",
		keyringAdminPassword: "super-secret",
	}}
	buf := &bytes.Buffer{}

	handled, err := handleAdminCLI([]string{"--remove-admin"}, store, bufio.NewReader(strings.NewReader("")), buf)
	require.NoError(t, err)
	assert.True(t, handled)
	assert.Contains(t, store.deletedKeys, keyringAdminUsername)
	assert.Contains(t, store.deletedKeys, keyringAdminPassword)
	assert.Contains(t, buf.String(), "Admin credentials removed from keystore.")
}

func TestHandleAdminCLI_AddAdmin(t *testing.T) {
	t.Parallel()

	store := &mockCredentialStore{}
	buf := &bytes.Buffer{}
	reader := bufio.NewReader(strings.NewReader("bob\nmy-password\n"))

	handled, err := handleAdminCLI([]string{"--add-admin"}, store, reader, buf)
	require.NoError(t, err)
	assert.True(t, handled)
	assert.Equal(t, "bob", store.values[keyringAdminUsername])
	require.NoError(t, bcrypt.CompareHashAndPassword([]byte(store.values[keyringAdminPassword]), []byte("my-password")))
	assert.Contains(t, buf.String(), "Admin credentials stored in keystore.")
}
