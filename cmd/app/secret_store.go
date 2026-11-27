package main

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/security"

	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/internal/app"
	"github.com/device-management-toolkit/console/internal/authsetup"
	"github.com/device-management-toolkit/console/pkg/secrets"
)

const (
	keyringServiceName   = secrets.KeyringServiceName
	keyringAdminUsername = secrets.KeyringAdminUsernameKey
	keyringAdminPassword = secrets.KeyringAdminPasswordKey
)

var (
	errKeystoreUnavailable     = errors.New("OS keyring is unavailable; ensure a supported keyring is installed, unlocked, and accessible to this user before restarting Console")
	errPartialAdminCredentials = errors.New("partial administrator credentials are present; delete Console's administrator credentials from the OS keyring and restart Console")
	errInvalidAdminCredentials = errors.New("administrator credentials are incomplete or invalid; delete Console's administrator credentials from the OS keyring and restart Console")
)

type credentialStore interface {
	GetKeyValue(key string) (string, error)
	SetKeyValue(key, value string) error
	DeleteKeyValue(key string) error
}

var newKeyringStorageFunc = func() credentialStore {
	return security.NewKeyRingStorage(keyringServiceName)
}

func handleAdminCredentials(cfg *config.Config) {
	if cfg.Disabled {
		log.Print("Auth is disabled; skipping admin credential resolution.")

		return
	}

	if cfg.ClientID != "" {
		log.Print("OAuth2/OIDC is configured; skipping standalone admin credential resolution.")

		return
	}

	switch cfg.SecretBackend {
	case "config":
		handleAdminPassword(cfg)
	case "keyring":
		if err := loadKeyringAdminCredentials(cfg, newKeyringStorageFunc()); err != nil {
			log.Fatalf("keyring admin credentials: %v", err)
		}
	default:
		log.Fatalf("%v", config.ErrAuthSecretBackendInvalid)
	}
}

func loadKeyringAdminCredentials(cfg *config.Config, store credentialStore) error {
	if store == nil {
		return errKeystoreUnavailable
	}

	username, usernameErr := store.GetKeyValue(keyringAdminUsername)
	passwordHash, passwordErr := store.GetKeyValue(keyringAdminPassword)
	usernameMissing := errors.Is(usernameErr, security.ErrKeyNotFound)
	passwordMissing := errors.Is(passwordErr, security.ErrKeyNotFound)

	switch {
	case usernameErr != nil && !usernameMissing:
		return fmt.Errorf("reading administrator username from OS keyring; ensure the keyring is available and unlocked: %w", usernameErr)
	case passwordErr != nil && !passwordMissing:
		return fmt.Errorf("reading administrator password hash from OS keyring; ensure the keyring is available and unlocked: %w", passwordErr)
	case usernameMissing && passwordMissing:
		return startKeyringSetup(cfg, store)
	case usernameMissing || passwordMissing:
		return errPartialAdminCredentials
	case strings.TrimSpace(username) == "" || !secrets.IsPBKDF2Hash(passwordHash):
		return errInvalidAdminCredentials
	}

	cfg.AdminUsername = username
	cfg.AdminPassword = passwordHash

	return nil
}

// startKeyringSetup issues the in-memory one-time token for the loopback
// setup page. app.Run serves setup with this same Manager.
func startKeyringSetup(cfg *config.Config, store credentialStore) error {
	manager, err := authsetup.NewManager(store)
	if err != nil {
		return err
	}

	cfg.SetupToken = manager.SetupToken()
	app.AuthSetup = manager

	return nil
}
