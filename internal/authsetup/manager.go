package authsetup

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/security"

	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/pkg/secrets"
)

const (
	keyringServiceName   = "dmt-console"
	keyringAdminUsername = "admin-username"
	keyringAdminPassword = "admin-password-hash"
	SecretBackendKeyring = "keyring"
	setupTokenFileMode   = 0o600
	setupTokenDirMode    = 0o700
)

var (
	ErrAlreadyConfigured            = errors.New("administrator credentials are already configured")
	ErrIncompleteStore              = errors.New("partial or invalid administrator credentials are present; delete both dmt-console entries and restart")
	ErrSetupRequired                = errors.New("administrator setup required; use the local setup page")
	errCredentialVerificationFailed = errors.New("stored administrator credentials do not match the requested values")
	errAuthVersionCannotDecrease    = errors.New("auth version cannot decrease")
)

type CredentialStore interface {
	GetKeyValue(key string) (string, error)
	SetKeyValue(key, value string) error
	DeleteKeyValue(key string) error
}

type Manager struct {
	cfg       *config.Config
	store     CredentialStore
	state     *stateStore
	token     string
	tokenFile string
	statePath string
}

func NewManager(cfg *config.Config, statePath string) (*Manager, error) {
	return NewManagerWithStore(cfg, statePath, security.NewKeyRingStorage(keyringServiceName))
}

func NewManagerWithStore(cfg *config.Config, statePath string, store CredentialStore) (*Manager, error) {
	state, err := openStateStore(statePath)
	if err != nil {
		return nil, err
	}

	manager := &Manager{
		cfg:       cfg,
		store:     store,
		state:     state,
		tokenFile: statePath + ".setup-token",
		statePath: statePath,
	}
	if cfg.SecretBackend != SecretBackendKeyring || cfg.Disabled || cfg.ClientID != "" {
		return manager, nil
	}

	if err := manager.loadKeyringCredentials(); err != nil {
		if !errors.Is(err, ErrSetupRequired) {
			_ = state.close()

			return nil, err
		}

		manager.token, err = state.setupToken(time.Now())
		if err != nil {
			_ = state.close()

			return nil, err
		}

		if err := persistSetupToken(manager.tokenFile, manager.token); err != nil {
			_ = state.close()

			return nil, err
		}
	} else {
		_ = os.Remove(manager.tokenFile)
	}

	return manager, nil
}

func (m *Manager) SetupToken() string {
	return m.token
}

func (m *Manager) SetupRequired() (bool, error) {
	if m.cfg.SecretBackend != SecretBackendKeyring || m.cfg.Disabled || m.cfg.ClientID != "" {
		return false, nil
	}

	err := m.loadKeyringCredentials()
	if errors.Is(err, ErrSetupRequired) {
		return true, nil
	}

	if err != nil {
		return false, err
	}

	return false, nil
}

func (m *Manager) Close() error {
	return m.state.close()
}

func (m *Manager) AuthVersion() (uint64, error) {
	return m.state.authVersion()
}

func (m *Manager) WithAuthVersion(fn func(uint64) (uint64, error)) error {
	return m.state.withAuthVersion(fn)
}

func (m *Manager) CurrentCredentials() (username, password string, err error) {
	if m.cfg.SecretBackend == SecretBackendKeyring {
		username, err := m.store.GetKeyValue(keyringAdminUsername)
		if err != nil {
			return "", "", fmt.Errorf("read administrator username from keyring: %w", err)
		}

		passwordHash, err := m.store.GetKeyValue(keyringAdminPassword)
		if err != nil {
			return "", "", fmt.Errorf("read administrator password hash from keyring: %w", err)
		}

		if strings.TrimSpace(username) == "" || !secrets.IsPBKDF2Hash(passwordHash) {
			return "", "", ErrIncompleteStore
		}

		return username, passwordHash, nil
	}

	configPath := strings.TrimSuffix(m.statePath, ".auth-state.db")

	username, password, err = config.ReadAdminCredentialsFromFile(configPath)
	if err != nil {
		return "", "", err
	}

	if value, ok := os.LookupEnv("AUTH_ADMIN_USERNAME"); ok && value != "" {
		username = value
	}

	if value, ok := os.LookupEnv("AUTH_ADMIN_PASSWORD"); ok && value != "" {
		password = value
	}

	if username == "" {
		username = m.cfg.AdminUsername
	}

	if password == "" {
		password = m.cfg.AdminPassword
	}

	return username, password, nil
}

func persistSetupToken(path, token string) error {
	if err := os.MkdirAll(filepath.Dir(path), setupTokenDirMode); err != nil {
		return fmt.Errorf("create setup token directory: %w", err)
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, setupTokenFileMode)
	if err != nil {
		return fmt.Errorf("create setup token file: %w", err)
	}

	if err := file.Chmod(setupTokenFileMode); err != nil {
		_ = file.Close()

		return fmt.Errorf("restrict setup token file permissions: %w", err)
	}

	if _, err := file.WriteString(token); err != nil {
		_ = file.Close()

		return fmt.Errorf("write setup token file: %w", err)
	}

	if err := file.Sync(); err != nil {
		_ = file.Close()

		return fmt.Errorf("persist setup token file: %w", err)
	}

	if err := file.Close(); err != nil {
		return fmt.Errorf("close setup token file: %w", err)
	}

	return nil
}

func (m *Manager) loadKeyringCredentials() error {
	username, usernameErr := m.store.GetKeyValue(keyringAdminUsername)
	passwordHash, passwordErr := m.store.GetKeyValue(keyringAdminPassword)
	usernameMissing := errors.Is(usernameErr, security.ErrKeyNotFound)
	passwordMissing := errors.Is(passwordErr, security.ErrKeyNotFound)

	switch {
	case usernameErr != nil && !usernameMissing:
		return fmt.Errorf("read administrator username from keyring: %w", usernameErr)
	case passwordErr != nil && !passwordMissing:
		return fmt.Errorf("read administrator password hash from keyring: %w", passwordErr)
	case usernameMissing && passwordMissing:
		return ErrSetupRequired
	case usernameMissing || passwordMissing:
		return ErrIncompleteStore
	case strings.TrimSpace(username) == "" || !secrets.IsPBKDF2Hash(passwordHash):
		return ErrIncompleteStore
	}

	m.cfg.AdminUsername = username
	m.cfg.AdminPassword = passwordHash

	return nil
}

func (m *Manager) Bootstrap(token, username, password, confirmation string) error {
	if m.cfg.SecretBackend != SecretBackendKeyring || m.cfg.Disabled || m.cfg.ClientID != "" {
		return ErrAlreadyConfigured
	}

	if strings.TrimSpace(username) == "" || password == "" || password != confirmation ||
		!secrets.IsAdminPasswordStrong(password) {
		return ErrInvalidCredentials
	}

	err := m.state.consumeSetupToken(token, time.Now(), m.credentialsConfigured, func() error {
		return m.storeBootstrapCredentials(strings.TrimSpace(username), password)
	})
	if err == nil {
		_ = os.Remove(m.tokenFile)
	}

	return err
}

func (m *Manager) credentialsConfigured() (bool, error) {
	err := m.loadKeyringCredentials()
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, ErrSetupRequired):
		return false, nil
	default:
		return false, err
	}
}

func (m *Manager) storeBootstrapCredentials(username, password string) error {
	if err := m.loadKeyringCredentials(); err == nil {
		return ErrAlreadyConfigured
	} else if !errors.Is(err, ErrSetupRequired) {
		return err
	}

	hash, err := secrets.GeneratePBKDF2Hash(password)
	if err != nil {
		return fmt.Errorf("hash administrator password: %w", err)
	}

	if err := m.store.SetKeyValue(keyringAdminUsername, username); err != nil {
		return fmt.Errorf("store administrator username: %w", err)
	}

	if err := m.store.SetKeyValue(keyringAdminPassword, hash); err != nil {
		return rollbackUsername(m.store, err)
	}

	savedUsername, usernameErr := m.store.GetKeyValue(keyringAdminUsername)
	savedHash, hashErr := m.store.GetKeyValue(keyringAdminPassword)

	if usernameErr != nil {
		return fmt.Errorf("verify stored administrator username: %w", usernameErr)
	}

	if hashErr != nil {
		return fmt.Errorf("verify stored administrator password hash: %w", hashErr)
	}

	if savedUsername != username || savedHash != hash {
		return errors.Join(errCredentialVerificationFailed, rollbackCredentials(m.store))
	}

	m.cfg.AdminUsername = savedUsername
	m.cfg.AdminPassword = savedHash
	m.token = ""

	return nil
}

func rollbackUsername(store CredentialStore, storeErr error) error {
	rollbackErr := store.DeleteKeyValue(keyringAdminUsername)
	if rollbackErr != nil && !errors.Is(rollbackErr, security.ErrKeyNotFound) {
		return fmt.Errorf("store administrator password hash: %w (also failed to roll back username: %w)", storeErr, rollbackErr)
	}

	return fmt.Errorf("store administrator password hash: %w", storeErr)
}

func rollbackCredentials(store CredentialStore) error {
	usernameErr := store.DeleteKeyValue(keyringAdminUsername)
	passwordErr := store.DeleteKeyValue(keyringAdminPassword)

	if errors.Is(usernameErr, security.ErrKeyNotFound) {
		usernameErr = nil
	}

	if errors.Is(passwordErr, security.ErrKeyNotFound) {
		passwordErr = nil
	}

	return errors.Join(usernameErr, passwordErr)
}

var ErrInvalidCredentials = errors.New("invalid administrator credentials")
