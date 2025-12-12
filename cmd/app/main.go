/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: Apache-2.0
 **********************************************************************/

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

	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/internal/app"
	"github.com/device-management-toolkit/console/internal/certificates"
	"github.com/device-management-toolkit/console/internal/controller/openapi"
	"github.com/device-management-toolkit/console/internal/usecase"
	"github.com/device-management-toolkit/console/pkg/logger"
	secrets "github.com/device-management-toolkit/console/pkg/secrets/vault"
)

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
		log.Fatalf("Config error: %s", err)
	}

	err = initializeAppFunc(cfg)
	if err != nil {
		log.Fatalf("App init error: %s", err)
	}

	// Initialize certificate store (Vault) for MPS and domain certificates
	secretsClient, secretsErr := handleSecretsConfig(cfg)
	if secretsErr == nil {
		// Set the cert store for domain certificates (used by usecases)
		app.CertStore = secretsClient
	}

	root, privateKey, err := loadOrGenerateRootCertFunc(secretsClient, true, cfg.CommonName, "US", "device-management-toolkit", true)
	if err != nil {
		log.Fatalf("Error loading or generating root certificate: %s", err)
	}

	_, _, err = loadOrGenerateWebServerCertFunc(secretsClient, certificates.CertAndKeyType{Cert: root, Key: privateKey}, false, cfg.CommonName, "US", "device-management-toolkit", true)
	if err != nil {
		log.Fatalf("Error loading or generating web server certificate: %s", err)
	}

	handleEncryptionKey(cfg)

	if os.Getenv("GIN_MODE") != "debug" {
		go func() {
			scheme := "http"
			if cfg.TLS.Enabled {
				scheme = "https"
			}

			browserError := openBrowser(scheme+"://localhost:"+cfg.Port, runtime.GOOS)
			if browserError != nil {
				panic(browserError)
			}
		}()
	} else {
		err = handleOpenAPIGeneration()
		if err != nil {
			log.Fatalf("Failed to generate OpenAPI spec: %s", err)
		}
	}

	runAppFunc(cfg)
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

	log.Println("OpenAPI specification generated at doc/openapi.json")

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
