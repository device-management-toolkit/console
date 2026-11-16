package authsetup

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/security"

	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/pkg/secrets"
)

type testCredentialStore struct {
	values map[string]string
}

func (s *testCredentialStore) GetKeyValue(key string) (string, error) {
	value, ok := s.values[key]
	if !ok {
		return "", security.ErrKeyNotFound
	}

	return value, nil
}

func (s *testCredentialStore) SetKeyValue(key, value string) error {
	s.values[key] = value

	return nil
}

func (s *testCredentialStore) DeleteKeyValue(key string) error {
	delete(s.values, key)

	return nil
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
