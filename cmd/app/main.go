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
	"github.com/device-management-toolkit/console/pkg/secrets"
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

	root, privateKey, err := certificates.CheckAndLoadOrGenerateRootCertificate(true, cfg.CommonName, "US", "device-management-toolkit", true)
	if err != nil {
		log.Fatalf("Error loading or generating root certificate: %s", err)
	}

	_, _, err = certificates.CheckAndLoadOrGenerateWebServerCertificate(certificates.CertAndKeyType{Cert: root, Key: privateKey}, false, cfg.CommonName, "US", "device-management-toolkit", true)
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
	secretsClient, err := secrets.NewClient(&cfg.Secrets)
	if err != nil {
		return nil, err
	}

	return secretsClient, nil
}

func handleEncryptionKey(cfg *config.Config) {
	toolkitCrypto := security.Crypto{}

	if cfg.EncryptionKey != "" {
		return
	}

	var remoteStorage security.Storager

	// Try to initialize Vault client and get key
	remoteStorage, err := handleSecretsConfig(cfg)
	if err == nil {
		cfg.EncryptionKey, err = remoteStorage.GetKeyValue("default-security-key")
		if err == nil {
			log.Println("Encryption key loaded from Vault")

			return
		}
	} else {
		remoteStorage = nil
	}

	// Try local keyring storage (simple key-value API)
	localStorage := security.NewKeyRingStorage("device-management-toolkit")

	cfg.EncryptionKey, err = localStorage.GetKeyValue("default-security-key")
	if err == nil {
		log.Println("Encryption key loaded from local keyring")

		if remoteStorage != nil {
			syncErr := remoteStorage.SetKeyValue("default-security-key", cfg.EncryptionKey)
			if syncErr != nil {
				log.Printf("Warning: Failed to sync key to Vault: %v", syncErr)
			}
		}

		return
	}

	if !errors.Is(err, security.ErrKeyNotFound) {
		log.Fatal(err)
		return
	}

	// Key not found anywhere, generate a new one
	handleKeyNotFound(cfg, toolkitCrypto, remoteStorage, localStorage)
}

func saveEncryptionKey(key string, remoteStorage, localStorage security.Storager) error {
	if remoteStorage != nil {
		err := remoteStorage.SetKeyValue("default-security-key", key)
		if err == nil {
			log.Println("Encryption key saved to Vault")

			return nil
		}

		return err
	}

	err := localStorage.SetKeyValue("default-security-key", key)
	if err == nil {
		log.Println("Encryption key saved to local keyring")

		return nil
	}

	return nil
}

func handleKeyNotFound(cfg *config.Config, toolkitCrypto security.Crypto, remoteStorage, localStorage security.Storager) {
	log.Print("\033[31mWarning: Key Not Found, Generate new key? -- This will prevent access to existing data? Y/N: \033[0m")

	var response string

	_, err := fmt.Scanln(&response)
	if err != nil {
		log.Fatal(err)

		return
	}

	if response != "Y" && response != "y" {
		log.Fatal("Exiting without generating a new key.")

		return
	}

	cfg.EncryptionKey = toolkitCrypto.GenerateKey()

	saveEncryptionKey(cfg.EncryptionKey, remoteStorage, localStorage)
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
