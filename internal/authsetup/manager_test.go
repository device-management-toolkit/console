package authsetup

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/security"

	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/pkg/secrets"
)

type testCredentialStore struct {
	values    map[string]string
	getErr    error
	setErrors map[string]error
	deleteErr error
}

func (s *testCredentialStore) GetKeyValue(key string) (string, error) {
	if s.getErr != nil {
		return "", s.getErr
	}

	value, ok := s.values[key]
	if !ok {
		return "", security.ErrKeyNotFound
	}

	return value, nil
}

func (s *testCredentialStore) SetKeyValue(key, value string) error {
	if err := s.setErrors[key]; err != nil {
		return err
	}

	s.values[key] = value

	return nil
}

func (s *testCredentialStore) DeleteKeyValue(key string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}

	delete(s.values, key)

	return nil
}

func newTestManager(t *testing.T, backend, statePath string, store *testCredentialStore) *Manager {
	t.Helper()

	manager, err := NewManagerWithStore(&config.Config{Auth: config.Auth{SecretBackend: backend}}, statePath, store)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, manager.Close()) })

	return manager
}

func TestBootstrapIsSingleUseAndSharedAcrossManagers(t *testing.T) {
	t.Parallel()

	store := &testCredentialStore{values: map[string]string{}}
	statePath := filepath.Join(t.TempDir(), "auth-state.db")
	cfg := &config.Config{Auth: config.Auth{SecretBackend: SecretBackendKeyring}}
	first, err := NewManagerWithStore(cfg, statePath, store)
	require.NoError(t, err)

	defer first.Close()

	second, err := NewManagerWithStore(&config.Config{Auth: config.Auth{SecretBackend: SecretBackendKeyring}}, statePath, store)
	require.NoError(t, err)

	defer second.Close()

	require.NotEmpty(t, first.SetupToken())
	require.Equal(t, first.SetupToken(), second.SetupToken())

	require.NoError(t, first.Bootstrap(first.SetupToken(), "admin", "Str0ng-password!", "Str0ng-password!"))
	require.True(t, secrets.IsPBKDF2Hash(store.values[keyringAdminPassword]))
	require.Equal(t, "admin", store.values[keyringAdminUsername])

	setupRequired, err := second.SetupRequired()
	require.NoError(t, err)
	require.False(t, setupRequired)
	require.ErrorIs(t, second.Bootstrap(second.SetupToken(), "other", "Str0ng-password!", "Str0ng-password!"), ErrAlreadyConfigured)
}

func TestBootstrapRejectsInvalidTokenAndWeakCredentials(t *testing.T) {
	t.Parallel()

	store := &testCredentialStore{values: map[string]string{}}
	manager, err := NewManagerWithStore(&config.Config{Auth: config.Auth{SecretBackend: SecretBackendKeyring}},
		filepath.Join(t.TempDir(), "auth-state.db"), store)
	require.NoError(t, err)

	defer manager.Close()

	require.ErrorIs(t, manager.Bootstrap("wrong-token", "admin", "Str0ng-password!", "Str0ng-password!"), ErrInvalidSetupToken)
	require.ErrorIs(t, manager.Bootstrap(manager.SetupToken(), "admin", "aaaaaaaa", "aaaaaaaa"), ErrInvalidCredentials)
	require.Empty(t, store.values)
}

func TestBootstrapRollsBackUsernameWhenPasswordWriteFails(t *testing.T) {
	t.Parallel()

	store := &testCredentialStore{
		values:    map[string]string{},
		setErrors: map[string]error{keyringAdminPassword: errors.New("keyring write failed")},
	}
	manager := newTestManager(t, SecretBackendKeyring, filepath.Join(t.TempDir(), "auth-state.db"), store)

	err := manager.Bootstrap(manager.SetupToken(), "admin", "Str0ng-password!", "Str0ng-password!")

	require.ErrorContains(t, err, "keyring write failed")
	require.Empty(t, store.values)
}

func TestCurrentCredentialsPreservesConfigValuesWhenEnvironmentIsEmpty(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yml")
	require.NoError(t, os.WriteFile(configPath, []byte("auth:\n  adminUsername: saved-user\n  adminPassword: saved-password\n"), 0o600))
	t.Setenv("AUTH_ADMIN_USERNAME", "")
	t.Setenv("AUTH_ADMIN_PASSWORD", "")

	manager := newTestManager(t, "config", configPath+".auth-state.db", nil)

	username, password, err := manager.CurrentCredentials()
	require.NoError(t, err)
	require.Equal(t, "saved-user", username)
	require.Equal(t, "saved-password", password)
}

func TestCurrentCredentialsReadsKeyringValues(t *testing.T) {
	t.Parallel()

	hash, err := secrets.GeneratePBKDF2Hash("Str0ng-password!")
	require.NoError(t, err)

	store := &testCredentialStore{values: map[string]string{
		keyringAdminUsername: "keyring-user",
		keyringAdminPassword: hash,
	}}
	manager := newTestManager(t, SecretBackendKeyring, filepath.Join(t.TempDir(), "auth-state.db"), store)

	username, password, err := manager.CurrentCredentials()
	require.NoError(t, err)
	require.Equal(t, "keyring-user", username)
	require.Equal(t, hash, password)
}

func TestSetupHandlerRejectsNonLoopbackAndReportsAlreadyCompleted(t *testing.T) {
	t.Parallel()

	store := &testCredentialStore{values: map[string]string{}}
	manager, err := NewManagerWithStore(&config.Config{Auth: config.Auth{SecretBackend: "keyring"}},
		filepath.Join(t.TempDir(), "auth-state.db"), store)
	require.NoError(t, err)

	defer manager.Close()

	handler := NewHandler(manager)
	req := httptest.NewRequest(http.MethodGet, setupStatusPath, http.NoBody)
	req.RemoteAddr = "192.0.2.1:1234"
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	require.Equal(t, http.StatusForbidden, response.Code)

	require.NoError(t, manager.Bootstrap(manager.SetupToken(), "admin", "Str0ng-password!", "Str0ng-password!"))

	req = httptest.NewRequest(http.MethodGet, setupStatusPath, http.NoBody)
	req.RemoteAddr = "127.0.0.1:1234"
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	require.Equal(t, http.StatusOK, response.Code)

	var body map[string]bool
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
	require.False(t, body["setupRequired"])

	require.True(t, errors.Is(manager.Bootstrap(manager.SetupToken(), "admin", "Str0ng-password!", "Str0ng-password!"), ErrAlreadyConfigured))
}

func TestSetupRedirectIsNotExposed(t *testing.T) {
	t.Parallel()

	store := &testCredentialStore{values: map[string]string{}}
	manager := newTestManager(t, SecretBackendKeyring, filepath.Join(t.TempDir(), "auth-state.db"), store)
	handler := NewHandler(manager)

	for _, remoteAddr := range []string{"192.0.2.1:1234", "127.0.0.1:1234"} {
		req := httptest.NewRequest(http.MethodGet, "/setup/redirect", http.NoBody)
		req.RemoteAddr = remoteAddr
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, req)
		require.Equal(t, http.StatusNotFound, response.Code)
		require.Empty(t, response.Header().Get("Location"))
	}
}

func TestSetupHandlerValidatesMethodsOriginsAndPayloads(t *testing.T) {
	t.Parallel()

	store := &testCredentialStore{values: map[string]string{}}
	manager := newTestManager(t, SecretBackendKeyring, filepath.Join(t.TempDir(), "auth-state.db"), store)
	handler := NewHandler(manager)

	tests := []struct {
		name       string
		method     string
		path       string
		origin     string
		body       string
		wantStatus int
		wantOrigin string
	}{
		{name: "status preflight", method: http.MethodOptions, path: setupStatusPath, wantStatus: http.StatusNoContent},
		{name: "status wrong method", method: http.MethodPost, path: setupStatusPath, wantStatus: http.StatusMethodNotAllowed},
		{name: "setup preflight", method: http.MethodOptions, path: setupPath, origin: "https://localhost:1234", wantStatus: http.StatusNoContent, wantOrigin: "https://localhost:1234"},
		{name: "setup wrong method", method: http.MethodGet, path: setupPath, wantStatus: http.StatusMethodNotAllowed},
		{name: "invalid JSON", method: http.MethodPost, path: setupPath, body: "{", wantStatus: http.StatusBadRequest},
		{name: "non-loopback origin", method: http.MethodGet, path: setupStatusPath, origin: "https://example.com", wantStatus: http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))

			req.RemoteAddr = "127.0.0.1:1234"

			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}

			response := httptest.NewRecorder()
			handler.ServeHTTP(response, req)

			require.Equal(t, tt.wantStatus, response.Code)

			if tt.wantOrigin != "" {
				require.Equal(t, tt.wantOrigin, response.Header().Get("Access-Control-Allow-Origin"))
			}
		})
	}
}

func TestSetupHandlerMapsBootstrapResults(t *testing.T) {
	t.Parallel()

	store := &testCredentialStore{values: map[string]string{}}
	manager := newTestManager(t, SecretBackendKeyring, filepath.Join(t.TempDir(), "auth-state.db"), store)
	handler := NewHandler(manager)

	postSetup := func(token, password string) *httptest.ResponseRecorder {
		body, err := json.Marshal(setupRequest{
			Token:           token,
			Username:        "admin",
			Password:        password,
			ConfirmPassword: password,
		})
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, setupPath, bytes.NewReader(body))
		req.RemoteAddr = "127.0.0.1:1234"
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, req)

		return response
	}

	require.Equal(t, http.StatusUnauthorized, postSetup("invalid", "Str0ng-password!").Code)
	require.Equal(t, http.StatusBadRequest, postSetup(manager.SetupToken(), "aaaaaaaa").Code)
	require.Equal(t, http.StatusCreated, postSetup(manager.SetupToken(), "Str0ng-password!").Code)
	require.Equal(t, http.StatusConflict, postSetup(manager.SetupToken(), "Str0ng-password!").Code)
}

func TestSetupHandlerReportsStorageErrors(t *testing.T) {
	t.Parallel()

	t.Run("status read fails", func(t *testing.T) {
		t.Parallel()

		store := &testCredentialStore{values: map[string]string{}}
		manager := newTestManager(t, SecretBackendKeyring, filepath.Join(t.TempDir(), "auth-state.db"), store)
		store.getErr = errors.New("keyring unavailable")

		req := httptest.NewRequest(http.MethodGet, setupStatusPath, http.NoBody)
		req.RemoteAddr = "127.0.0.1:1234"
		response := httptest.NewRecorder()
		NewHandler(manager).ServeHTTP(response, req)

		require.Equal(t, http.StatusInternalServerError, response.Code)
	})

	t.Run("credential write fails", func(t *testing.T) {
		t.Parallel()

		store := &testCredentialStore{
			values:    map[string]string{},
			setErrors: map[string]error{keyringAdminUsername: errors.New("keyring unavailable")},
		}
		manager := newTestManager(t, SecretBackendKeyring, filepath.Join(t.TempDir(), "auth-state.db"), store)
		body, err := json.Marshal(setupRequest{
			Token:           manager.SetupToken(),
			Username:        "admin",
			Password:        "Str0ng-password!",
			ConfirmPassword: "Str0ng-password!",
		})
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, setupPath, bytes.NewReader(body))
		req.RemoteAddr = "127.0.0.1:1234"
		response := httptest.NewRecorder()
		NewHandler(manager).ServeHTTP(response, req)

		require.Equal(t, http.StatusInternalServerError, response.Code)
		require.Empty(t, store.values)
	})
}

func TestAuthVersionIsDurableAndSerializesConcurrentUpdates(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "auth-state.db")
	first, err := NewManagerWithStore(&config.Config{Auth: config.Auth{SecretBackend: "config"}}, path, nil)
	require.NoError(t, err)
	second, err := NewManagerWithStore(&config.Config{Auth: config.Auth{SecretBackend: "config"}}, path, nil)
	require.NoError(t, err)

	var workers sync.WaitGroup

	errs := make(chan error, 2)

	for _, manager := range []*Manager{first, second} {
		manager := manager

		workers.Go(func() {
			errs <- manager.WithAuthVersion(func(version uint64) (uint64, error) {
				return version + 1, nil
			})
		})
	}

	workers.Wait()
	close(errs)

	for err := range errs {
		require.NoError(t, err)
	}

	version, err := first.AuthVersion()
	require.NoError(t, err)
	require.Equal(t, uint64(2), version)
	require.NoError(t, first.Close())
	require.NoError(t, second.Close())

	restarted, err := NewManagerWithStore(&config.Config{Auth: config.Auth{SecretBackend: "config"}}, path, nil)
	require.NoError(t, err)

	defer restarted.Close()

	version, err = restarted.AuthVersion()
	require.NoError(t, err)
	require.Equal(t, uint64(2), version)
}
