package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/security"
	"gopkg.in/yaml.v2"

	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/internal/app"
	"github.com/device-management-toolkit/console/internal/certificates"
	"github.com/device-management-toolkit/console/internal/controller/openapi"
	"github.com/device-management-toolkit/console/internal/usecase"
	"github.com/device-management-toolkit/console/pkg/logger"
	secrets "github.com/device-management-toolkit/console/pkg/secrets/vault"
)

// Package-level logger for main application.
var appLog = logger.New("info")

// Sentinel errors for configuration.
var (
	ErrSecretStoreAddressNotConfigured = errors.New("secret store address not configured")
	ErrSecretStoreTokenNotConfigured   = errors.New("secret store token not configured")
)

// Function pointers for better testability.
var (
	initializeConfigFunc = config.NewConfig
	initializeAppFunc    = app.Init
	runAppFunc           = app.Run
	// NewGeneratorFunc allows tests to inject a fake OpenAPI generator.
	NewGeneratorFunc = func(u usecase.Usecases, l logger.Interface) interface {
		GenerateSpec() ([]byte, error)
		SaveSpec([]byte, string) error
	} {
		return openapi.NewGenerator(u, l)
	}
	// Certificate loading functions for testability.
	loadOrGenerateRootCertFunc      = certificates.LoadOrGenerateRootCertificateWithVault
	loadOrGenerateWebServerCertFunc = certificates.LoadOrGenerateWebServerCertificateWithVault
)

func main() {
	cfg, err := initializeConfigFunc()
	if err != nil {
		appLog.Fatal("Config error: %s", err)
	}

	// Log configuration in readable YAML format (matches config file structure)
	if cfgYAML, err := yaml.Marshal(cfg); err == nil {
		log.Printf("Configuration loaded:\n%s", string(cfgYAML))
	} else {
		log.Printf("Configuration loaded: %+v", cfg)
	}

	if err = initializeAppFunc(cfg); err != nil {
		appLog.Fatal("App init error: %s", err)
	}

	// Initialize certificate store (Vault) for MPS and domain certificates
	secretsClient, secretsErr := handleSecretsConfig(cfg)
	if secretsErr == nil {
		appLog.Info("Secrets client initialized successfully, setting as certificate store")
		app.CertStore = secretsClient
	} else {
		appLog.Warn("Secrets client initialization failed: %v (certificates will use local filesystem storage at config/)", secretsErr)
	}

	if err = setupCIRACertificates(cfg, secretsClient); err != nil {
		appLog.Fatal("CIRA certificate setup error: %s", err)
	}

	handleEncryptionKey(cfg)
	handleDebugMode(cfg)
	runAppFunc(cfg)
}

func setupCIRACertificates(cfg *config.Config, secretsClient security.Storager) error {
	appLog.Info("=== CIRA Certificate Setup ===")

	if cfg.DisableCIRA {
		appLog.Info("CIRA is disabled in configuration, skipping certificate setup")
		return nil
	}

	appLog.Info("CIRA is enabled, proceeding with certificate setup")
	if secretsClient != nil {
		appLog.Info("Vault client available for certificate storage")
	} else {
		appLog.Warn("WARNING: Vault client not available, certificates will use local filesystem storage only")
	}

	root, privateKey, err := loadOrGenerateRootCertFunc(secretsClient, true, cfg.CommonName, "US", "device-management-toolkit", true)
	if err != nil {
		return fmt.Errorf("loading or generating root certificate: %w", err)
	}

	_, _, err = loadOrGenerateWebServerCertFunc(secretsClient, certificates.CertAndKeyType{Cert: root, Key: privateKey}, false, cfg.CommonName, "US", "device-management-toolkit", true)
	if err != nil {
		return fmt.Errorf("loading or generating web server certificate: %w", err)
	}

	appLog.Info("=== CIRA Certificate Setup Complete ===")
	return nil
}

func handleDebugMode(cfg *config.Config) {
	if os.Getenv("GIN_MODE") != "debug" {
		go launchBrowser(cfg)
	} else {
		if err := handleOpenAPIGeneration(); err != nil {
			appLog.Fatal("Failed to generate OpenAPI spec: %s", err)
		}
	}
}

func launchBrowser(cfg *config.Config) {
	scheme := "http"
	if cfg.TLS.Enabled {
		scheme = "https"
	}

	if err := openBrowser(scheme+"://localhost:"+cfg.Port, runtime.GOOS); err != nil {
		panic(err)
	}
}

func handleOpenAPIGeneration() error {
	l := logger.New("info")
	usecases := usecase.Usecases{}

	// Create OpenAPI generator
	generator := NewGeneratorFunc(usecases, l)

	// Generate specification
	spec, err := generator.GenerateSpec()
	if err != nil {
		return err
	}

	// Save to file
	if err := generator.SaveSpec(spec, "doc/openapi.json"); err != nil {
		return err
	}

	appLog.Info("OpenAPI specification generated at doc/openapi.json")

	return nil
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
		appLog.Warn("Failed to connect to secret store: %v", err)

		return nil, err
	}

	appLog.Info("Connected to secret store at: %s", cfg.Address)

	return secretsClient, nil
}

func handleEncryptionKey(cfg *config.Config) {
	// If encryption key is already provided via config/env, just use it
	if cfg.EncryptionKey != "" {
		appLog.Info("Encryption key loaded from environment")

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
		appLog.Warn("Warning: Failed to save encryption key: %v", err)
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
			appLog.Info("Encryption key stored in secret store")

			return true
		}
	} else {
		// Retrieve from secret store
		key, err := remoteStorage.GetKeyValue("default-security-key")
		if err == nil {
			cfg.EncryptionKey = key

			appLog.Info("Encryption key loaded from secret store")

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
			appLog.Info("Encryption key stored in local keyring")

			return true
		}
	} else {
		cfg.EncryptionKey, err = localStorage.GetKeyValue("default-security-key")
		if err == nil {
			appLog.Info("Encryption key loaded from local keyring")
			syncKeyToRemote(cfg.EncryptionKey, remoteStorage)

			return true
		}
	}

	// Check for unexpected errors
	if err != nil && !errors.Is(err, security.ErrKeyNotFound) {
		appLog.Fatal(err)
	}

	return false
}

// syncKeyToRemote syncs an encryption key to the remote storage if available.
func syncKeyToRemote(key string, remoteStorage security.Storager) {
	if remoteStorage == nil {
		return
	}

	if err := remoteStorage.SetKeyValue("default-security-key", key); err != nil {
		appLog.Warn("Warning: Failed to sync key to secret store: %v", err)
	} else {
		appLog.Info("Encryption key synced to secret store")
	}
}

func saveEncryptionKey(key string, remoteStorage, localStorage security.Storager) error {
	if remoteStorage != nil {
		err := remoteStorage.SetKeyValue("default-security-key", key)
		if err == nil {
			appLog.Info("Encryption key saved to secret store")

			return nil
		}

		return err
	}

	err := localStorage.SetKeyValue("default-security-key", key)
	if err == nil {
		appLog.Info("Encryption key saved to local keyring")

		return nil
	}

	return err
}

func handleKeyNotFound(toolkitCrypto security.Crypto, _, _ security.Storager) string {
	log.Print("\033[31mWarning: Key Not Found, Generate new key? -- This will prevent access to existing data? Y/N: \033[0m")

	var response string

	_, err := fmt.Scanln(&response)
	if err != nil {
		appLog.Fatal(err)

		return ""
	}

	if response != "Y" && response != "y" {
		appLog.Fatal("Exiting without generating a new key.")

		return ""
	}

	return toolkitCrypto.GenerateKey()
}

// CommandExecutor is an interface to allow for mocking exec.Command in tests.
type CommandExecutor interface {
	Execute(name string, arg ...string) error
}

// RealCommandExecutor is a real implementation of CommandExecutor.
type RealCommandExecutor struct{}

func (e *RealCommandExecutor) Execute(name string, arg ...string) error {
	return exec.CommandContext(context.Background(), name, arg...).Start()
}

// Global command executor, can be replaced in tests.
var cmdExecutor CommandExecutor = &RealCommandExecutor{}

func openBrowser(url, currentOS string) error {
	var cmd string

	var args []string

	switch currentOS {
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", url}
	default:
		cmd = "xdg-open"
		args = []string{url}
	}

	return cmdExecutor.Execute(cmd, args...)
}
