package authsetup

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/security"

	"github.com/device-management-toolkit/console/pkg/secrets"
)

const (
	testUser     = "admin"
	testPassword = "New-Secret-123!"
)

type memoryStore struct {
	values map[string]string
	setErr map[string]error
	getErr error
}

func newMemoryStore() *memoryStore {
	return &memoryStore{values: map[string]string{}, setErr: map[string]error{}}
}

func (s *memoryStore) GetKeyValue(key string) (string, error) {
	if s.getErr != nil {
		return "", s.getErr
	}

	value, ok := s.values[key]
	if !ok {
		return "", security.ErrKeyNotFound
	}

	return value, nil
}

func (s *memoryStore) SetKeyValue(key, value string) error {
	if err := s.setErr[key]; err != nil {
		return err
	}

	s.values[key] = value

	return nil
}

func (s *memoryStore) DeleteKeyValue(key string) error {
	delete(s.values, key)

	return nil
}

func newTestManager(t *testing.T) (*Manager, *memoryStore) {
	t.Helper()

	store := newMemoryStore()
	manager, err := NewManager(store)
	require.NoError(t, err)
	require.NotEmpty(t, manager.SetupToken())

	return manager, store
}

func TestBootstrapStoresHashAndConsumesToken(t *testing.T) {
	t.Parallel()

	manager, store := newTestManager(t)
	token := manager.SetupToken()

	required, err := manager.SetupRequired()
	require.NoError(t, err)
	require.True(t, required)

	require.NoError(t, manager.Bootstrap(token, " "+testUser+" ", testPassword, testPassword))

	require.Equal(t, testUser, store.values[secrets.KeyringAdminUsernameKey])
	require.True(t, secrets.VerifyPBKDF2Hash(store.values[secrets.KeyringAdminPasswordKey], testPassword))
	require.Empty(t, manager.SetupToken(), "token is single-use")

	required, err = manager.SetupRequired()
	require.NoError(t, err)
	require.False(t, required)

	require.ErrorIs(t, manager.Bootstrap(token, testUser, testPassword, testPassword), ErrAlreadyConfigured)
}

func TestBootstrapRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	manager, store := newTestManager(t)
	token := manager.SetupToken()

	tests := []struct {
		name                         string
		token, user, password, again string
		want                         error
	}{
		{"empty username", token, " ", testPassword, testPassword, ErrInvalidCredentials},
		{"confirmation mismatch", token, testUser, testPassword, testPassword + "x", ErrInvalidCredentials},
		{"weak password", token, testUser, "weakpass", "weakpass", ErrInvalidCredentials},
		{"wrong token", "nope", testUser, testPassword, testPassword, ErrInvalidSetupToken},
	}

	for _, tt := range tests {
		require.ErrorIs(t, manager.Bootstrap(tt.token, tt.user, tt.password, tt.again), tt.want, tt.name)
	}

	require.Empty(t, store.values, "nothing is stored on failure")
	require.Equal(t, token, manager.SetupToken(), "failed attempts do not consume the token")
}

func TestBootstrapRejectsExpiredToken(t *testing.T) {
	t.Parallel()

	manager, _ := newTestManager(t)
	manager.now = func() time.Time { return time.Now().Add(setupTokenLifetime + time.Second) }

	require.ErrorIs(t, manager.Bootstrap(manager.SetupToken(), testUser, testPassword, testPassword), ErrInvalidSetupToken)
}

func TestBootstrapRollsBackUsernameWhenHashWriteFails(t *testing.T) {
	t.Parallel()

	manager, store := newTestManager(t)
	store.setErr[secrets.KeyringAdminPasswordKey] = errors.New("keyring locked")

	require.Error(t, manager.Bootstrap(manager.SetupToken(), testUser, testPassword, testPassword))
	require.Empty(t, store.values, "a failed write must not leave a partial keyring")
	require.NotEmpty(t, manager.SetupToken(), "setup can be retried")
}

func TestKeyringReadErrorsAreReported(t *testing.T) {
	t.Parallel()

	for _, key := range []string{secrets.KeyringAdminUsernameKey, secrets.KeyringAdminPasswordKey} {
		manager, store := newTestManager(t)
		store.values[key] = "x"
		store.getErr = nil

		failing := &keyErrorStore{memoryStore: store, key: key}
		manager.store = failing

		_, err := manager.SetupRequired()
		require.ErrorContains(t, err, "keyring locked", key)
		require.ErrorContains(t, manager.Bootstrap(manager.SetupToken(), testUser, testPassword, testPassword), "keyring locked", key)
	}
}

// keyErrorStore fails reads of one key with an error other than "not found".
type keyErrorStore struct {
	*memoryStore
	key string
}

func (s *keyErrorStore) GetKeyValue(key string) (string, error) {
	if key == s.key {
		return "", errors.New("keyring locked")
	}

	return s.memoryStore.GetKeyValue(key)
}

func TestBootstrapUsernameWriteFailure(t *testing.T) {
	t.Parallel()

	manager, store := newTestManager(t)
	store.setErr[secrets.KeyringAdminUsernameKey] = errors.New("keyring locked")

	require.ErrorContains(t, manager.Bootstrap(manager.SetupToken(), testUser, testPassword, testPassword), "store administrator username")
	require.Empty(t, store.values)
}
