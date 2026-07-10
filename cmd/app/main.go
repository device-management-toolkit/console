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
	"github.com/device-management-toolkit/console/internal/controller/httpapi"
	"github.com/device-management-toolkit/console/pkg/logger"
	secrets "github.com/device-management-toolkit/console/pkg/secrets/vault"
)

// Sentinel errors for configuration.
var (
	ErrSecretStoreAddressNotConfigured = errors.New("secret store address not configured")
	ErrSecretStoreTokenNotConfigured   = errors.New("secret store token not configured")
	ErrJWTKeyMissing                   = errors.New("JWT signing key is empty")
	ErrJWTKeyInsecure                  = errors.New("JWT signing key is the known-insecure default")
)

// adminPasswordLength is the length of generated admin passwords.
const adminPasswordLength = 16

const (
	insecureDefaultJWTKey   = "your_secret_jwt_key"  // rejected at startup
	jwtKeyStorageKey        = "jwt-signing-key"      // secret store / keyring key
	encryptionKeyStorageKey = "default-security-key" // secret store / keyring key
	jwtKeyBytes             = 32                     // 256-bit, matches HS256
	keyringServiceName      = "device-management-toolkit"
)

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
	if len(os.Args) > 1 && (os.Args[1] == "--health" || os.Args[1] == "-health") {
		runHealthCheck()
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
	handleJWTKey(cfg)

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

func handleDebugMode(cfg *config.Config, l logger.Interface) {
	if !httpapi.HasUI() {
		l.Info("UI assets not embedded; skipping browser launch")

		return
	}

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
	if cfg.EncryptionKey != "" {
		log.Println("Encryption key loaded from environment")

		return
	}

	store := newSecretStore(cfg)

	key, err := store.get(encryptionKeyStorageKey)
	if err != nil {
		log.Fatalf(
			"Local keyring unavailable (%v).\n"+
				"Set APP_ENCRYPTION_KEY in the environment (or encryption_key in config) "+
				"to provide the encryption key directly, or configure a remote secret store.",
			err,
		)
	}

	if key != "" {
		cfg.EncryptionKey = key

		log.Println("Encryption key loaded from secure storage")

		return
	}

	// Not found anywhere: prompt (losing this key prevents access to existing data), then store.
	cfg.EncryptionKey = handleKeyNotFound()

	if err := store.set(encryptionKeyStorageKey, cfg.EncryptionKey); err != nil {
		log.Printf("Warning: Failed to save encryption key: %v", err)
	}
}

func handleKeyNotFound() string {
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

	return security.Crypto{}.GenerateKey()
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

// handleJWTKey resolves the JWT key: env/config, else secret store / keyring, else generate.
// Rejects the insecure default; never writes the key to config.yml.
func handleJWTKey(cfg *config.Config) {
	if cfg.JWTKey == insecureDefaultJWTKey {
		log.Fatalf("insecure default JWT key; unset auth.jwtKey or set AUTH_JWT_KEY to a strong value")
	}

	if cfg.JWTKey != "" {
		log.Println("JWT signing key loaded from environment/config")

		return
	}

	store := newSecretStore(cfg)

	// Losing the JWT key is harmless (tokens re-issue), so warn on storage errors rather than fatal.
	key, err := store.get(jwtKeyStorageKey)
	if err != nil {
		log.Printf("Warning: keyring unavailable for JWT signing key: %v", err)
	}

	if key != "" {
		cfg.JWTKey = key

		log.Println("JWT signing key loaded from secure storage")

		return
	}

	generated, genErr := generateJWTKey()
	if genErr != nil {
		log.Fatalf("Failed to generate JWT signing key: %v", genErr)
	}

	cfg.JWTKey = generated

	if saveErr := store.set(jwtKeyStorageKey, cfg.JWTKey); saveErr != nil {
		log.Printf("Warning: generated JWT signing key but could not persist it (%v); "+
			"a new key will be generated on restart", saveErr)
	} else {
		log.Println("Generated and stored new JWT signing key")
	}

	if valErr := validateJWTKey(cfg.JWTKey); valErr != nil {
		log.Fatalf("JWT signing key unusable: %v", valErr)
	}
}

// validateJWTKey rejects an empty or known-insecure signing key.
func validateJWTKey(key string) error {
	switch key {
	case "":
		return ErrJWTKeyMissing
	case insecureDefaultJWTKey:
		return ErrJWTKeyInsecure
	default:
		return nil
	}
}

// generateJWTKey returns a base64-encoded 256-bit random key.
func generateJWTKey() (string, error) {
	b := make([]byte, jwtKeyBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(b), nil
}
