package main

import (
	"bufio"
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"errors"
	"flag"
	"log"
	"os"
	"path/filepath"
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

	username, password, _ := resolveAdminCredentialsFromSources(cfg, store, map[string]string{
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

	username, password, _ := resolveAdminCredentialsFromSources(cfg, store, map[string]string{
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

	username, password, _ := resolveAdminCredentialsFromSources(cfg, store, map[string]string{})

	assert.Equal(t, "cfg-user", username)
	assert.Equal(t, "cfg-pass", password)
}

func TestHandleAdminCLI_Clean(t *testing.T) {
	t.Parallel()

	store := &mockCredentialStore{values: map[string]string{
		keyringAdminUsername: "alice",
		keyringAdminPassword: "super-secret",
		keyringAdminJWTKey:   "jwt-token-key",
	}}
	buf := &bytes.Buffer{}

	handled, err := handleAdminCLIWithInput([]string{"--clean"}, store, buf, strings.NewReader("y\n"))
	require.NoError(t, err)
	assert.True(t, handled)
	assert.Contains(t, store.deletedKeys, keyringAdminUsername)
	assert.Contains(t, store.deletedKeys, keyringAdminPassword)
	assert.Contains(t, store.deletedKeys, keyringAdminJWTKey)
	assert.Contains(t, buf.String(), "Admin credentials and JWT key removed from keystore.")
}

func TestResolveAdminCredentialsFromSources_IncludesJWTKey(t *testing.T) {
	t.Setenv(authAdminUsernameEnv, "env-user")
	t.Setenv(authAdminSecretEnv, "env-pass")
	t.Setenv(authAdminJWTKeyEnv, "env-jwt-key")

	cfg := &config.Config{Auth: config.Auth{
		AdminUsername: "cfg-user",
		AdminPassword: "cfg-pass",
		JWTKey:        "cfg-jwt-key",
	}}
	store := &mockCredentialStore{values: map[string]string{
		keyringAdminUsername: "keyring-user",
		keyringAdminPassword: "keyring-pass",
		keyringAdminJWTKey:   "keyring-jwt-key",
	}}

	username, password, jwtKey := resolveAdminCredentialsFromSources(cfg, store, map[string]string{
		authAdminUsernameEnv: "dotenv-user",
		authAdminSecretEnv:   "dotenv-pass",
		authAdminJWTKeyEnv:   "dotenv-jwt-key",
	})

	assert.Equal(t, "keyring-user", username)
	assert.Equal(t, "keyring-pass", password)
	assert.Equal(t, "keyring-jwt-key", jwtKey)
}

func TestReadDotEnvFile_ParsesQuotedValuesAndIgnoresComments(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	require.NoError(t, os.WriteFile(path, []byte("# comment\nAUTH_ADMIN_USERNAME=\"quoted-user\"\nAUTH_ADMIN_PASSWORD='secret-value'\nAUTH_JWT_KEY=jwt-key\n"), 0o600))

	values := readDotEnvFile(path)
	assert.Equal(t, "quoted-user", values[authAdminUsernameEnv])
	assert.Equal(t, "secret-value", values[authAdminSecretEnv])
	assert.Equal(t, "jwt-key", values[authAdminJWTKeyEnv])
}

func TestLogKeyringSaveAndRollbackWarnings_LogsEachFailure(t *testing.T) {
	t.Parallel()

	store := &mockCredentialStore{errMap: map[string]error{
		keyringAdminUsername: errors.New("user failed"),
		keyringAdminPassword: errors.New("password failed"),
		keyringAdminJWTKey:   errors.New("jwt failed"),
	}}

	var out bytes.Buffer

	oldWriter := log.Writer()
	log.SetOutput(&out)

	defer log.SetOutput(oldWriter)

	logKeyringSaveAndRollbackWarnings(store, errors.New("user save failed"), errors.New("password save failed"), errors.New("jwt save failed"))

	assert.Contains(t, out.String(), "admin username")
	assert.Contains(t, out.String(), "admin password")
	assert.Contains(t, out.String(), "admin JWT key")
}

func TestHandleAdminCredentials_FirstRunBootstrapsAndPersistsToKeyring(t *testing.T) {
	t.Parallel()

	configFile := filepath.Join(t.TempDir(), "config.yml")
	require.NoError(t, os.WriteFile(configFile, []byte("auth:\n  jwtExpiration: 24h\n"), 0o600))

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
	_, err = writer.WriteString("\n")
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	os.Stdin = reader
	t.Cleanup(func() { os.Stdin = oldStdin; _ = reader.Close() })

	store := &mockCredentialStore{values: map[string]string{}}
	origFunc := newKeyringStorageFunc
	newKeyringStorageFunc = func() credentialStore { return store }

	t.Cleanup(func() { newKeyringStorageFunc = origFunc })

	cfg := &config.Config{}
	handleAdminCredentials(cfg)

	assert.Equal(t, "standalone", cfg.AdminUsername)
	assert.NotEmpty(t, cfg.AdminPassword)
	assert.NotEmpty(t, cfg.JWTKey)
	assert.Equal(t, "standalone", store.values[keyringAdminUsername])
	assert.NotEmpty(t, store.values[keyringAdminPassword])
	assert.NotEmpty(t, store.values[keyringAdminJWTKey])

	configData, err := os.ReadFile(configFile)
	require.NoError(t, err)
	assert.NotContains(t, string(configData), cfg.AdminUsername)
	assert.NotContains(t, string(configData), cfg.AdminPassword)
	assert.NotContains(t, string(configData), cfg.JWTKey)
	assert.Contains(t, string(configData), "adminUsername: \"\"")
	assert.Contains(t, string(configData), "jwtKey: \"\"")
}

func TestHandleAdminCredentials_OAuthConfiguredSkipsBootstrap(t *testing.T) {
	t.Parallel()

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

func TestSaveAdminCredentialsToKeyring_StoresAllThreeValues(t *testing.T) {
	t.Parallel()

	store := &mockCredentialStore{values: map[string]string{}}
	persisted, usernameErr, passwordErr, jwtKeyErr := saveAdminCredentialsToKeyring(
		store,
		"admin",
		"hashed-password",
		"jwt-key",
	)

	assert.True(t, persisted)
	assert.NoError(t, usernameErr)
	assert.NoError(t, passwordErr)
	assert.NoError(t, jwtKeyErr)
	assert.Equal(t, "admin", store.values[keyringAdminUsername])
	assert.Equal(t, "hashed-password", store.values[keyringAdminPassword])
	assert.Equal(t, "jwt-key", store.values[keyringAdminJWTKey])
}

func TestResolveAdminCredentials_FirstRunBootstrap(t *testing.T) {
	t.Parallel()

	input := "admin\npassword123\npassword123\n\n"
	reader := bufio.NewReader(strings.NewReader(input))

	username, password, jwtKey, promptedUsername, promptedPassword := resolveAdminCredentials(
		reader, true, "", "", "",
	)

	assert.Equal(t, "standalone", username)
	assert.NotEmpty(t, password)
	assert.NotEmpty(t, jwtKey)
	assert.True(t, promptedUsername)
	assert.True(t, promptedPassword)
}

func TestResolveAdminCredentials_NonFirstRunPromptsForMissing(t *testing.T) {
	t.Parallel()

	input := "entered-user\nentered-pass\n"
	reader := bufio.NewReader(strings.NewReader(input))

	username, password, jwtKey, promptedUsername, promptedPassword := resolveAdminCredentials(
		reader, false, "", "", "",
	)

	assert.Equal(t, "entered-user", username)
	assert.Equal(t, "entered-pass", password)
	assert.NotEmpty(t, jwtKey)
	assert.True(t, promptedUsername)
	assert.True(t, promptedPassword)
}

func TestResolveAdminCredentials_NonFirstRunGeneratesJWTKeyWhenMissing(t *testing.T) {
	t.Parallel()

	reader := bufio.NewReader(strings.NewReader(""))

	username, password, jwtKey, promptedUsername, promptedPassword := resolveAdminCredentials(
		reader, false, "existing-user", "existing-pass", "",
	)

	assert.Equal(t, "existing-user", username)
	assert.Equal(t, "existing-pass", password)
	assert.NotEmpty(t, jwtKey)
	assert.False(t, promptedUsername)
	assert.False(t, promptedPassword)
}

func TestRollbackKeyringEntry_SuccessfulDelete(t *testing.T) {
	t.Parallel()

	store := &mockCredentialStore{
		values:      map[string]string{"test-key": "test-value"},
		errMap:      map[string]error{},
		deletedKeys: []string{},
	}
	rollbackKeyringEntry(store, "test-key")

	require.Equal(t, 1, len(store.deletedKeys))
	assert.Equal(t, "test-key", store.deletedKeys[0])
}

func TestRollbackKeyringEntry_IgnoresNotFoundError(t *testing.T) {
	t.Parallel()

	store := &mockCredentialStore{
		errMap: map[string]error{"test-key:delete": security.ErrKeyNotFound},
	}
	// Should not panic or log an error for ErrKeyNotFound
	rollbackKeyringEntry(store, "test-key")
}
