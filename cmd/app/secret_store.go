package main

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/security"

	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/internal/authsetup"
	"github.com/device-management-toolkit/console/pkg/secrets"
)

const (
	keyringAdminUsername = "admin-username"
	keyringAdminPassword = "admin-password-hash"
)

var (
	errKeystoreUnavailable     = errors.New("OS keyring is unavailable; ensure a supported keyring is installed, unlocked, and accessible to this user before restarting Console")
	errPartialAdminCredentials = authsetup.ErrIncompleteStore
	errInvalidAdminCredentials = authsetup.ErrIncompleteStore
)

type credentialStore interface {
	GetKeyValue(key string) (string, error)
	SetKeyValue(key, value string) error
	DeleteKeyValue(key string) error
}

var newKeyringStorageFunc = func() credentialStore {
	return security.NewKeyRingStorage(authsetup.KeyringServiceName)
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
	case authsetup.SecretBackendKeyring:
		manager, err := authsetup.NewManagerWithStore(cfg, cfg.StatePath, newKeyringStorageFunc())
		if err != nil {
			log.Fatalf("keyring admin credentials: %v", err)
		}

		cfg.SetupToken = manager.SetupToken()
		if err := manager.Close(); err != nil {
			log.Fatalf("close keyring setup state: %v", err)
		}
	default:
		log.Fatalf("%v", config.ErrAuthSecretBackendInvalid)
	}
}

// loadKeyringAdminCredentials remains a read-only helper for startup diagnostics.
// First-run credential creation is handled through the local setup API.
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
		return fmt.Errorf("reading administrator username from OS keyring: %w (ensure the keyring is available and unlocked)", usernameErr)
	case passwordErr != nil && !passwordMissing:
		return fmt.Errorf("reading administrator password hash from OS keyring: %w (ensure the keyring is available and unlocked)", passwordErr)
	case usernameMissing && passwordMissing:
		return authsetup.ErrSetupRequired
	case usernameMissing || passwordMissing:
		return errPartialAdminCredentials
	case strings.TrimSpace(username) == "" || !secrets.IsPBKDF2Hash(passwordHash):
		return errInvalidAdminCredentials
	}

	cfg.AdminUsername = username
	cfg.AdminPassword = passwordHash

	return nil
}
