package main

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/security"

	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/internal/app"
	"github.com/device-management-toolkit/console/internal/certificates"
	"github.com/device-management-toolkit/console/pkg/logger"
	secrets "github.com/device-management-toolkit/console/pkg/secrets/vault"
)

// Sentinel errors for configuration.
var (
	ErrSecretStoreAddressNotConfigured = errors.New("secret store address not configured")
	ErrSecretStoreTokenNotConfigured   = errors.New("secret store token not configured")
)

// adminPasswordLength is the length of generated admin passwords.
const adminPasswordLength = 16

// Function pointers for better testability.
var (
	initializeConfigFunc = config.NewConfig
	initializeAppFunc    = app.Init
	runAppFunc           = func(cfg *config.Config, log logger.Interface) {
		app.Run(cfg, log)
	}
	// Certificate loading functions for testability.
	loadOrGenerateRootCertFunc      = certificates.LoadOrGenerateRootCertificateWithVault
	loadOrGenerateWebServerCertFunc = certificates.LoadOrGenerateWebServerCertificateWithVault
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		runHealthCheck()

		return
	}

	cfg, err := initializeConfigFunc()
	if err != nil {
		log.Fatalf("Config error: %s", err)
	}

	if err = initializeAppFunc(cfg); err != nil {
		log.Fatalf("App init error: %s", err)
	}

	// Initialize certificate store (Vault) for MPS and domain certificates
	secretsClient, secretsErr := handleSecretsConfig(cfg)
	if secretsErr == nil {
		app.CertStore = secretsClient
	}

	if err = setupCIRACertificates(cfg, secretsClient); err != nil {
		log.Fatalf("CIRA certificate setup error: %s", err)
	}

	l := logger.New(cfg.Level)

	handleEncryptionKey(cfg)
	handleAdminPassword(cfg)

	// Run with system tray (if built with tray tag and --tray flag) or standard mode
	if config.TrayMode && !trayBuildEnabled {
		log.Fatal("--tray was specified but this binary was built without tray support. Rebuild with `make build-tray` (or `go build -tags=tray`).")
	}

	if trayBuildEnabled && config.TrayMode {
		runWithTray(cfg, l)
	} else {
		handleDebugMode(cfg, l)
		runAppFunc(cfg, l)
	}
}

func setupCIRACertificates(cfg *config.Config, secretsClient security.Storager) error {
	if cfg.DisableCIRA {
		return nil
	}

	root, privateKey, err := loadOrGenerateRootCertFunc(secretsClient, true, cfg.CommonName, "US", "device-management-toolkit", true)
	if err != nil {
		return fmt.Errorf("loading or generating root certificate: %w", err)
	}

	_, _, err = loadOrGenerateWebServerCertFunc(secretsClient, certificates.CertAndKeyType{Cert: root, Key: privateKey}, false, cfg.CommonName, "US", "device-management-toolkit", true)
	if err != nil {
		return fmt.Errorf("loading or generating web server certificate: %w", err)
	}

	return nil
}

func handleDebugMode(cfg *config.Config, _ logger.Interface) {
	if os.Getenv("GIN_MODE") != "debug" {
		go launchBrowser(cfg)
	}
}

func handleSecretsConfig(cfg *config.Config) (security.Storager, error) {
	if cfg.Address == "" {
		return nil, ErrSecretStoreAddressNotConfigured
	}

	if cfg.Token == "" {
		return nil, ErrSecretStoreTokenNotConfigured
	}

	secretsClient, err := secrets.NewClient(&cfg.Secrets)
	if err != nil {
		log.Printf("Failed to connect to secret store: %v", err)

		return nil, err
	}

	log.Printf("Connected to secret store at: %s", cfg.Address)

	return secretsClient, nil
}

func handleEncryptionKey(cfg *config.Config) {
	// If encryption key is already provided via config/env, just use it
	if cfg.EncryptionKey != "" {
		log.Println("Encryption key loaded from environment")

		return
	}

	toolkitCrypto := security.Crypto{}

	// Try to initialize secret store client for encryption key retrieval
	remoteStorage, err := handleSecretsConfig(cfg)
	if err != nil {
		remoteStorage = nil
	}

	// Try remote storage first
	if done := tryRemoteStorage(cfg, remoteStorage); done {
		return
	}

	// Try local keyring storage
	localStorage := security.NewKeyRingStorage("device-management-toolkit")

	if done := tryLocalStorage(cfg, localStorage, remoteStorage); done {
		return
	}

	// Key not found anywhere, generate a new one
	cfg.EncryptionKey = handleKeyNotFound(toolkitCrypto, remoteStorage, localStorage)

	if err := saveEncryptionKey(cfg.EncryptionKey, remoteStorage, localStorage); err != nil {
		log.Printf("Warning: Failed to save encryption key: %v", err)
	}
}

// tryRemoteStorage attempts to store/retrieve the encryption key from remote storage.
func tryRemoteStorage(cfg *config.Config, remoteStorage security.Storager) bool {
	if remoteStorage == nil {
		return false
	}

	if cfg.EncryptionKey != "" {
		// Store static key in secret store (not recommended)
		if err := remoteStorage.SetKeyValue("default-security-key", cfg.EncryptionKey); err == nil {
			log.Println("Encryption key stored in secret store")

			return true
		}
	} else {
		// Retrieve from secret store
		key, err := remoteStorage.GetKeyValue("default-security-key")
		if err == nil {
			cfg.EncryptionKey = key

			log.Println("Encryption key loaded from secret store")

			return true
		}
	}

	return false
}

// tryLocalStorage attempts to store/retrieve the encryption key from local keyring.
func tryLocalStorage(cfg *config.Config, localStorage, remoteStorage security.Storager) bool {
	var err error

	if cfg.EncryptionKey != "" {
		err = localStorage.SetKeyValue("default-security-key", cfg.EncryptionKey)
		if err == nil {
			log.Println("Encryption key stored in local keyring")

			return true
		}
	} else {
		cfg.EncryptionKey, err = localStorage.GetKeyValue("default-security-key")
		if err == nil {
			log.Println("Encryption key loaded from local keyring")
			syncKeyToRemote(cfg.EncryptionKey, remoteStorage)

			return true
		}
	}

	// Check for unexpected errors
	if err != nil && !errors.Is(err, security.ErrKeyNotFound) {
		log.Fatal(err)
	}

	return false
}

// syncKeyToRemote syncs an encryption key to the remote storage if available.
func syncKeyToRemote(key string, remoteStorage security.Storager) {
	if remoteStorage == nil {
		return
	}

	if err := remoteStorage.SetKeyValue("default-security-key", key); err != nil {
		log.Printf("Warning: Failed to sync key to secret store: %v", err)
	} else {
		log.Println("Encryption key synced to secret store")
	}
}

func saveEncryptionKey(key string, remoteStorage, localStorage security.Storager) error {
	if remoteStorage != nil {
		err := remoteStorage.SetKeyValue("default-security-key", key)
		if err == nil {
			log.Println("Encryption key saved to secret store")

			return nil
		}

		return err
	}

	err := localStorage.SetKeyValue("default-security-key", key)
	if err == nil {
		log.Println("Encryption key saved to local keyring")

		return nil
	}

	return err
}

func handleKeyNotFound(toolkitCrypto security.Crypto, _, _ security.Storager) string {
	log.Print("\033[31mWarning: Key Not Found, Generate new key? -- This will prevent access to existing data? Y/N: \033[0m")

	var response string

	_, err := fmt.Scanln(&response)
	if err != nil {
		log.Fatal(err)

		return ""
	}

	if response != "Y" && response != "y" {
		log.Fatal("Exiting without generating a new key.")

		return ""
	}

	return toolkitCrypto.GenerateKey()
}

// generateRandomPassword creates a cryptographically secure random password.
func generateRandomPassword(length int) (string, error) {
	bytes := make([]byte, length)

	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}

// handleAdminPassword ensures cfg.AdminPassword is set, generating one and
// persisting it to config.yml on first run if nothing was provided via config
// or environment.
func handleAdminPassword(cfg *config.Config) {
	if cfg.AdminPassword != "" {
		return
	}

	password, err := generateRandomPassword(adminPasswordLength)
	if err != nil {
		log.Fatalf("Failed to generate admin password: %v", err)
	}

	cfg.AdminPassword = password

	if err := config.SaveAdminPassword(cfg.AdminPassword); err != nil {
		log.Fatalf(
			"Generated admin password but failed to persist it to config (%v).\n"+
				"Refusing to start with an unsaved credential that would vanish on restart.\n"+
				"Set AUTH_ADMIN_PASSWORD in the environment (or auth.adminPassword in config) "+
				"to provide the admin password directly.",
			err,
		)
	}

	log.Printf("Generated new admin password and persisted to config; see auth.adminPassword in config.yml.")
}
