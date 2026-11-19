package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"golang.org/x/term"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/security"

	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/pkg/secrets"
)

const (
	keyringServiceName   = "device-management-toolkit"
	keyringAdminUsername = "admin-username"
	keyringAdminPassword = "admin-password-hash"
	keyringRecoveryGuide = "https://github.com/device-management-toolkit/console/wiki/ADR-Centralize-secret-storage-and-harden-admin-credentials-Architecture"
)

var (
	errPartialAdminCredentials = errors.New("partial administrator credentials are present; delete both device-management-toolkit entries and restart; see " + keyringRecoveryGuide + " for OS-specific recovery steps")
	errInvalidAdminCredentials = errors.New("administrator credentials are incomplete or invalid; delete both device-management-toolkit entries and restart")
	errPasswordConfirmation    = errors.New("password confirmation did not match after 3 attempts")
)

const maxPasswordConfirmationAttempts = 3

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
		handleConfigAdminCredentials(cfg)
	case "keyring":
		if err := loadKeyringAdminCredentials(cfg, newKeyringStorageFunc()); err != nil {
			log.Fatalf("keyring admin credentials: %v", err)
		}
	default:
		log.Fatalf("%v", config.ErrAuthSecretBackendInvalid)
	}
}

func handleConfigAdminCredentials(cfg *config.Config) {
	if cfg.AdminPassword != "" {
		warnIfWeakAdminPassword(cfg.AdminPassword)

		return
	}

	password, err := generateRandomPassword(adminPasswordLength)
	if err != nil {
		log.Fatalf("failed to generate admin password: %v", err)
	}

	cfg.AdminPassword = password
	if err := config.SaveAdminPassword(password); err != nil {
		log.Fatalf("generated admin password but could not persist it to config: %v", err)
	}

	log.Print("Generated new admin password and persisted it to config.yml.")
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
		return bootstrapKeyringAdminCredentials(cfg, store)
	case usernameMissing || passwordMissing:
		return errPartialAdminCredentials
	case strings.TrimSpace(username) == "" || !secrets.IsPBKDF2Hash(passwordHash):
		return errInvalidAdminCredentials
	}

	cfg.AdminUsername = username
	cfg.AdminPassword = passwordHash

	return nil
}

func bootstrapKeyringAdminCredentials(cfg *config.Config, store credentialStore) error {
	reader := bufio.NewReader(os.Stdin)
	username := promptForCredential(reader, "Enter Console admin username: ")
	password := promptForSecret(reader, "Enter Console admin password: ")

	confirmed := false

	for attempt := 0; attempt < maxPasswordConfirmationAttempts; attempt++ {
		confirmation := promptForSecret(reader, "Confirm Console admin password: ")
		if password == confirmation {
			confirmed = true

			break
		}

		if attempt+1 < maxPasswordConfirmationAttempts {
			log.Print("Password confirmation did not match. Please try again.")
		}
	}

	if !confirmed {
		return errPasswordConfirmation
	}

	hash, err := secrets.GeneratePBKDF2Hash(password)
	if err != nil {
		return fmt.Errorf("hashing administrator password: %w", err)
	}

	if err := store.SetKeyValue(keyringAdminUsername, username); err != nil {
		return fmt.Errorf("storing administrator username: %w", err)
	}

	if err := store.SetKeyValue(keyringAdminPassword, hash); err != nil {
		if rollbackErr := store.DeleteKeyValue(keyringAdminUsername); rollbackErr != nil && !errors.Is(rollbackErr, security.ErrKeyNotFound) {
			return fmt.Errorf("storing administrator password hash: %w (also failed to roll back username: %w)", err, rollbackErr)
		}

		return fmt.Errorf("storing administrator password hash: %w", err)
	}

	cfg.AdminUsername = username
	cfg.AdminPassword = hash

	return nil
}

func warnIfWeakAdminPassword(password string) {
	if isStrongAdminPassword(password) {
		return
	}

	log.Printf(
		"WARNING: the configured admin password is weak. It should be at least %d characters and "+
			"contain a lowercase letter, an uppercase letter, a digit, and a symbol; longer is better. "+
			"Console is starting anyway, but set a stronger password in auth.adminPassword in "+
			"config.yml, or via AUTH_ADMIN_PASSWORD in the environment. In .env, single-quote the "+
			"value so docker compose does not expand $, and avoid # altogether: make reads .env too, "+
			"and it cuts the value at # regardless of quoting.",
		adminPasswordMinLength,
	)
}

func promptForCredential(reader *bufio.Reader, prompt string) string {
	for {
		fmt.Fprint(os.Stdout, prompt)

		input, err := reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				log.Fatal("failed to read credential from console: EOF (non-interactive keyring setup). Use AUTH_SECRET_BACKEND=config for unattended deployments.")
			}

			log.Fatalf("failed to read credential from console: %v", err)
		}

		value := removePromptLineEnding(input)
		if strings.TrimSpace(value) != "" {
			return value
		}

		log.Println("Value cannot be empty.")
	}
}

func promptForSecret(reader *bufio.Reader, prompt string) string {
	for {
		fmt.Fprint(os.Stdout, prompt)

		if term.IsTerminal(int(os.Stdin.Fd())) {
			input, err := term.ReadPassword(int(os.Stdin.Fd()))

			fmt.Fprintln(os.Stdout)

			if err != nil {
				log.Fatalf("failed to read credential from console: %v", err)
			}

			value := string(input)
			if strings.TrimSpace(value) != "" {
				return value
			}

			log.Println("Value cannot be empty.")

			continue
		}

		input, err := reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				log.Fatal("failed to read credential from console: EOF (non-interactive keyring setup). Use AUTH_SECRET_BACKEND=config for unattended deployments.")
			}

			log.Fatalf("failed to read credential from console: %v", err)
		}

		value := removePromptLineEnding(input)
		if strings.TrimSpace(value) != "" {
			return value
		}

		log.Println("Value cannot be empty.")
	}
}

func removePromptLineEnding(input string) string {
	input = strings.TrimSuffix(input, "\n")

	return strings.TrimSuffix(input, "\r")
}
