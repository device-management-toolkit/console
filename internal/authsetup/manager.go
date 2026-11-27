// Package authsetup implements first-run administrator setup for
// AUTH_SECRET_BACKEND=keyring: a one-time token, held only in memory, that
// authorizes writing the first administrator credentials to the OS keyring.
package authsetup

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/security"

	"github.com/device-management-toolkit/console/pkg/secrets"
)

const (
	setupTokenLifetime = 10 * time.Minute
	setupTokenSize     = 32
)

var (
	ErrAlreadyConfigured  = errors.New("administrator credentials are already configured")
	ErrInvalidCredentials = errors.New("invalid administrator credentials")
	ErrInvalidSetupToken  = errors.New("invalid or expired setup token")
)

// Manager owns the one-time setup token. The token is never written to disk,
// so the CLI, browser launch, tray, and setup listener share one Manager.
type Manager struct {
	store security.Storager
	now   func() time.Time

	mu        sync.Mutex // guards token and expiresAt, and serializes Bootstrap
	token     string
	expiresAt time.Time
}

// NewManager issues a setup token. Call it only when both keyring entries are
// absent.
func NewManager(store security.Storager) (*Manager, error) {
	raw := make([]byte, setupTokenSize)
	if _, err := rand.Read(raw); err != nil {
		return nil, fmt.Errorf("generate setup token: %w", err)
	}

	m := &Manager{store: store, now: time.Now, token: base64.RawURLEncoding.EncodeToString(raw)}
	m.expiresAt = m.now().Add(setupTokenLifetime)

	return m, nil
}

// SetupToken returns the pending token, or "" once setup has completed.
func (m *Manager) SetupToken() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.token
}

// SetupRequired reports whether both keyring entries are still absent.
func (m *Manager) SetupRequired() (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.keyringEmpty()
}

// Bootstrap stores the first administrator credentials if token is valid and
// the keyring is still empty. The token is single-use.
func (m *Manager) Bootstrap(token, username, password, confirmation string) error {
	username = strings.TrimSpace(username)
	if username == "" || password != confirmation || !secrets.IsStrongAdminPassword(password) {
		return ErrInvalidCredentials
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	empty, err := m.keyringEmpty()
	if err != nil {
		return err
	}

	if !empty {
		return ErrAlreadyConfigured
	}

	if m.token == "" || !m.now().Before(m.expiresAt) ||
		subtle.ConstantTimeCompare([]byte(token), []byte(m.token)) != 1 {
		return ErrInvalidSetupToken
	}

	hash, err := secrets.GeneratePBKDF2Hash(password)
	if err != nil {
		return fmt.Errorf("hash administrator password: %w", err)
	}

	if err := m.store.SetKeyValue(secrets.KeyringAdminUsernameKey, username); err != nil {
		return fmt.Errorf("store administrator username: %w", err)
	}

	if err := m.store.SetKeyValue(secrets.KeyringAdminPasswordKey, hash); err != nil {
		_ = m.store.DeleteKeyValue(secrets.KeyringAdminUsernameKey)

		return fmt.Errorf("store administrator password hash: %w", err)
	}

	m.token = ""

	return nil
}

func (m *Manager) keyringEmpty() (bool, error) {
	_, usernameErr := m.store.GetKeyValue(secrets.KeyringAdminUsernameKey)
	_, passwordErr := m.store.GetKeyValue(secrets.KeyringAdminPasswordKey)
	usernameMissing := errors.Is(usernameErr, security.ErrKeyNotFound)
	passwordMissing := errors.Is(passwordErr, security.ErrKeyNotFound)

	switch {
	case usernameErr != nil && !usernameMissing:
		return false, fmt.Errorf("read administrator username from keyring: %w", usernameErr)
	case passwordErr != nil && !passwordMissing:
		return false, fmt.Errorf("read administrator password hash from keyring: %w", passwordErr)
	}

	return usernameMissing && passwordMissing, nil
}
