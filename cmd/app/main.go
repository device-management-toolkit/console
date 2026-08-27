package main

import (
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"math/big"
	"os"
	"regexp"
	"strings"
	"unicode/utf8"

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
	ErrPasswordLengthTooShort          = errors.New("requested password length is below the minimum")
)

// adminPasswordLength is the length of generated admin passwords.
const adminPasswordLength = 16

// adminPasswordMinLength is the floor, in runes, for the warning and the generator.
// No ceiling: this password is only compared against the login request in httpapi/v1.
const adminPasswordMinLength = 8

// RE2 has no lookahead, so complexity is one regex per required class. The last is
// "not a letter or digit", broader than the !@#$%^&* AMT profiles require.
var adminPasswordComplexity = []*regexp.Regexp{
	regexp.MustCompile(`[a-z]`),
	regexp.MustCompile(`[A-Z]`),
	regexp.MustCompile(`\d`),
	regexp.MustCompile(`[^a-zA-Z0-9]`),
}

// Classes used to generate a password. Only @ and * survive being pasted verbatim:
// $ ! expand in sh, # truncates in make (-include .env), % ^ & break cmd.exe's set.
const (
	lowerChars      = "abcdefghijklmnopqrstuvwxyz"
	upperChars      = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	digitChars      = "0123456789"
	genSpecialChars = "@*"
)

var (
	genPasswordClasses = []string{lowerChars, upperChars, digitChars, genSpecialChars}
	allGenChars        = strings.Join(genPasswordClasses, "")
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
		checkStoredEncryptionKey(cfg.EncryptionKey, "secret store")

		return
	}

	// Try local keyring storage
	localStorage := security.NewKeyRingStorage("device-management-toolkit")

	if done := tryLocalStorage(cfg, localStorage, remoteStorage); done {
		checkStoredEncryptionKey(cfg.EncryptionKey, "local keyring")

		return
	}

	// Key not found anywhere, generate a new one
	cfg.EncryptionKey = handleKeyNotFound(toolkitCrypto, remoteStorage, localStorage)

	if err := saveEncryptionKey(cfg.EncryptionKey, remoteStorage, localStorage); err != nil {
		log.Printf("Warning: Failed to save encryption key: %v", err)
	}
}

// checkStoredEncryptionKey validates a key that came out of the secret store or
// the local keyring. Keys supplied through config/env are already rejected by
// config.NewConfig, so a failure here means the store holds a key written by an
// older build that did not validate.
//
// A wrong-sized key can never encrypt anything (crypto/aes rejects it), so
// Console refuses to start. A weak but usable key only warns: existing device
// credentials are encrypted with it, and exiting would leave the operator unable
// to start Console and read their own data.
func checkStoredEncryptionKey(key, source string) {
	err := config.ValidateEncryptionKey(key)
	if err == nil {
		return
	}

	if errors.Is(err, config.ErrEncryptionKeyLength) {
		log.Fatalf(
			"Encryption key from the %s is unusable: %v.\n"+
				"Device credentials cannot be encrypted with it. Replace the stored "+
				"`default-security-key`, or set APP_ENCRYPTION_KEY to a 16, 24 or 32 "+
				"character key (note that credentials encrypted with a different key "+
				"become unreadable).",
			source, err,
		)
	}

	log.Printf(
		"Warning: encryption key from the %s is weak: %v. "+
			"Rotating it requires re-entering device credentials, so plan the change.",
		source, err,
	)
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
		log.Fatalf(
			"Local keyring unavailable (%v).\n"+
				"Set APP_ENCRYPTION_KEY in the environment (or encryption_key in config) "+
				"to provide the encryption key directly, or configure a remote secret store.",
			err,
		)
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

// generateRandomPassword returns a cryptographically secure password of exactly
// length characters that satisfies isStrongAdminPassword.
func generateRandomPassword(length int) (string, error) {
	if length < adminPasswordMinLength {
		return "", ErrPasswordLengthTooShort
	}

	password := make([]byte, 0, length)

	for _, class := range genPasswordClasses {
		char, err := randomChar(class)
		if err != nil {
			return "", err
		}

		password = append(password, char)
	}

	for len(password) < length {
		char, err := randomChar(allGenChars)
		if err != nil {
			return "", err
		}

		password = append(password, char)
	}

	if err := shufflePassword(password); err != nil {
		return "", err
	}

	return string(password), nil
}

// randomChar avoids the modulo bias of reducing a random byte.
func randomChar(set string) (byte, error) {
	index, err := rand.Int(rand.Reader, big.NewInt(int64(len(set))))
	if err != nil {
		return 0, fmt.Errorf("randomChar: %w", err)
	}

	return set[index.Int64()], nil
}

// shufflePassword removes the fixed class ordering of the first characters.
func shufflePassword(password []byte) error {
	for i := len(password) - 1; i > 0; i-- {
		j, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			return fmt.Errorf("shufflePassword: %w", err)
		}

		swap := j.Int64()
		password[i], password[swap] = password[swap], password[i]
	}

	return nil
}

// handleAdminPassword ensures cfg.AdminPassword is set, generating one and
// persisting it to config.yml on first run if nothing was provided via config
// or environment.
func handleAdminPassword(cfg *config.Config) {
	if cfg.AdminPassword != "" {
		warnOnWeakAdminPassword(cfg.AdminPassword)

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

// warnOnWeakAdminPassword warns but does not stop startup: migrated MPS/RPS
// credentials predate this policy, and refusing to boot would lock operators out.
func warnOnWeakAdminPassword(password string) {
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

func isStrongAdminPassword(password string) bool {
	if utf8.RuneCountInString(password) < adminPasswordMinLength {
		return false
	}

	for _, rule := range adminPasswordComplexity {
		if !rule.MatchString(password) {
			return false
		}
	}

	return true
}
