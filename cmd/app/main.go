package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/security"

	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/internal/app"
	"github.com/device-management-toolkit/console/internal/controller/openapi"
	"github.com/device-management-toolkit/console/internal/usecase"
	"github.com/device-management-toolkit/console/pkg/logger"
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

	handleEncryptionKey(cfg)

	if os.Getenv("GIN_MODE") != "debug" {
		go func() {
			browserError := openBrowser("http://localhost:"+cfg.Port, runtime.GOOS)
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

func handleEncryptionKey(cfg *config.Config) {
	toolkitCrypto := security.Crypto{}

	if cfg.EncryptionKey != "" {
		return
	}

	secureStorage := security.NewKeyRingStorage("device-management-toolkit")

	var err error

	cfg.EncryptionKey, err = secureStorage.GetKeyValue("default-security-key")
	if err == nil {
		return
	}

	if err.Error() != "secret not found in keyring" {
		log.Fatal(err)

		return
	}

	handleKeyNotFound(cfg, toolkitCrypto, secureStorage)
}

func handleKeyNotFound(cfg *config.Config, toolkitCrypto security.Crypto, secureStorage security.Storage) {
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

	err = secureStorage.SetKeyValue("default-security-key", cfg.EncryptionKey)
	if err != nil {
		log.Fatal(err)
	}
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
