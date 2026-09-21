package main

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/term"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/security"

	"github.com/device-management-toolkit/console/config"
)

const (
	keyringServiceName   = "dmt-console"
	keyringAdminUsername = "admin-username"
	keyringAdminPassword = "admin-password-hash"
	keyringAdminJWTKey   = "jwt-signing-key"
	dotEnvFile           = ".env"
	authAdminUsernameEnv = "AUTH_ADMIN_USERNAME"
	authAdminSecretEnv   = "AUTH_ADMIN_PASSWORD" // #nosec G101 -- environment variable name, not a credential value
	authAdminJWTKeyEnv   = "AUTH_JWT_KEY"
	bcryptPrefix2A       = "$2a$"
	bcryptPrefix2B       = "$2b$"
	bcryptPrefix2Y       = "$2y$"
	dotEnvSplitParts     = 2
	jwtKeyByteSize       = 32
)

var (
	errAdminCLIExclusiveFlags = errors.New("use only --clean")
	errKeystoreUnavailable    = errors.New("keystore is unavailable")
)

// credentialStore is intentionally narrow so alternate secret backends can be
// plugged in later without changing startup or CLI behavior.
type credentialStore interface {
	GetKeyValue(key string) (string, error)
	SetKeyValue(key, value string) error
	DeleteKeyValue(key string) error
}

// newKeyringStorageFunc is injectable for tests and can be replaced with other
// secret-store factories in future integrations.
var newKeyringStorageFunc = func() credentialStore { return security.NewKeyRingStorage(keyringServiceName) }

func handleAdminCLI(args []string, keyringStore credentialStore, out io.Writer) (bool, error) {
	return handleAdminCLIWithInput(args, keyringStore, out, os.Stdin)
}

func handleAdminCLIWithInput(args []string, keyringStore credentialStore, out io.Writer, input io.Reader) (bool, error) {
	command := selectedAdminCommand(args)
	if command == "" {
		return false, nil
	}

	if command == "conflict" {
		return true, errAdminCLIExclusiveFlags
	}

	configPath, err := config.ResolveConfigPathFromArgs(args)
	if err != nil {
		return true, err
	}

	if keyringStore == nil {
		return true, errKeystoreUnavailable
	}

	switch command {
	case "clean":
		return true, handleCleanCLI(keyringStore, configPath, out, input)
	default:
		return true, errAdminCLIExclusiveFlags
	}
}

func selectedAdminCommand(args []string) string {
	selected := 0
	command := ""

	if hasArg(args, "--clean") {
		selected++
		command = "clean"
	}

	if selected == 0 {
		return ""
	}

	if selected > 1 {
		return "conflict"
	}

	return command
}

func handleCleanCLI(keyringStore credentialStore, configPath string, out io.Writer, input io.Reader) error {
	fmt.Fprint(out, "Remove all standalone admin credentials and JWT key? [y/N]: ")

	reader := bufio.NewReader(input)

	response, err := reader.ReadString('\n')
	if err != nil || !strings.EqualFold(strings.TrimSpace(response), "y") {
		fmt.Fprintln(out, "Credential cleanup canceled.")

		return nil
	}

	if err := keyringStore.DeleteKeyValue(keyringAdminUsername); err != nil && !errors.Is(err, security.ErrKeyNotFound) {
		return fmt.Errorf("failed to remove admin username from keystore: %w", err)
	}

	if err := keyringStore.DeleteKeyValue(keyringAdminPassword); err != nil && !errors.Is(err, security.ErrKeyNotFound) {
		return fmt.Errorf("failed to remove admin password from keystore: %w", err)
	}

	if err := keyringStore.DeleteKeyValue(keyringAdminJWTKey); err != nil && !errors.Is(err, security.ErrKeyNotFound) {
		return fmt.Errorf("failed to remove admin JWT key from keystore: %w", err)
	}

	if err := config.ClearAdminCredentialsAndJWTKeyForPath(configPath); err != nil {
		fmt.Fprintf(out, "Warning: failed to clear admin credentials from config.yml: %v\n", err)
	}

	fmt.Fprintln(out, "Admin credentials and JWT key removed from keystore.")

	return nil
}

func hasArg(args []string, target string) bool {
	for _, arg := range args {
		if arg == target {
			return true
		}
	}

	return false
}

func resolveAdminCredentials(reader *bufio.Reader, isFirstRun bool, username, password, jwtKey string) (
	resolvedUsername, resolvedPassword, resolvedJWTKey string, promptedUsername, promptedPassword bool,
) {
	var promptedForUsername, promptedForPassword bool

	if isFirstRun {
		username, password, jwtKey = bootstrapAdminCredentials(reader)
		promptedForUsername = true
		promptedForPassword = true
	} else {
		if username == "" {
			promptedForUsername = true
			username = promptForCredential(reader, "Enter Console admin username: ")
		}

		if password == "" {
			promptedForPassword = true
			password = promptForSecret(reader, "Enter Console admin password: ")
		}

		if jwtKey == "" {
			jwtKey = generateRandomAdminJWTKey()
		}
	}

	return username, password, jwtKey, promptedForUsername, promptedForPassword
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

	dotEnvValues := readDotEnvFile(dotEnvFile)
	keyringStore := newKeyringStorageFunc()

	username, password, jwtKey := resolveAdminCredentialsFromSources(cfg, keyringStore, dotEnvValues)

	reader := bufio.NewReader(os.Stdin)
	isFirstRun := username == "" && password == ""

	username, password, jwtKey, _, _ = resolveAdminCredentials(
		reader, isFirstRun, username, password, jwtKey,
	)

	if isFirstRun && username == "" {
		log.Print("Admin credential bootstrap canceled.")

		return
	}

	hashedPassword, _, err := normalizeAdminPasswordHash(password)
	if err != nil {
		log.Fatalf("failed to hash admin password: %v", err)
	}

	cfg.AdminUsername = username
	cfg.AdminPassword = hashedPassword
	cfg.JWTKey = jwtKey

	persistedToKeyring, usernameSaveErr, passwordSaveErr, jwtKeySaveErr := saveAdminCredentialsToKeyring(keyringStore, cfg.AdminUsername, cfg.AdminPassword, cfg.JWTKey)
	if persistedToKeyring {
		if err := config.ClearAdminCredentials(); err != nil {
			log.Fatalf("failed to clear admin credentials from config.yml after keyring storage: %v", err)
		}

		return
	}

	logKeyringSaveAndRollbackWarnings(keyringStore, usernameSaveErr, passwordSaveErr, jwtKeySaveErr)
	log.Fatal("unable to store admin credentials in the OS keyring; unlock or configure the keyring and retry")
}

func saveAdminCredentialsToKeyring(keyringStore credentialStore, username, passwordHash, jwtKey string) (persisted bool, usernameErr, passwordErr, jwtKeyErr error) {
	usernameErr = keyringStore.SetKeyValue(keyringAdminUsername, username)
	passwordErr = keyringStore.SetKeyValue(keyringAdminPassword, passwordHash)
	jwtKeyErr = keyringStore.SetKeyValue(keyringAdminJWTKey, jwtKey)

	if usernameErr == nil && passwordErr == nil && jwtKeyErr == nil {
		return true, nil, nil, nil
	}

	if usernameErr == nil {
		rollbackKeyringEntry(keyringStore, keyringAdminUsername)
	}

	if passwordErr == nil {
		rollbackKeyringEntry(keyringStore, keyringAdminPassword)
	}

	if jwtKeyErr == nil {
		rollbackKeyringEntry(keyringStore, keyringAdminJWTKey)
	}

	return false, usernameErr, passwordErr, jwtKeyErr
}

func logKeyringSaveAndRollbackWarnings(keyringStore credentialStore, usernameSaveErr, passwordSaveErr, jwtKeySaveErr error) {
	if usernameSaveErr != nil {
		log.Printf("Warning: failed to save admin username to keyring: %v", usernameSaveErr)
	}

	if passwordSaveErr != nil {
		if usernameSaveErr == nil {
			rollbackKeyringEntry(keyringStore, keyringAdminUsername)
		}

		log.Printf("Warning: failed to save admin password to keyring: %v", passwordSaveErr)
	}

	if jwtKeySaveErr != nil {
		if usernameSaveErr == nil && passwordSaveErr == nil {
			rollbackKeyringEntry(keyringStore, keyringAdminUsername)
			rollbackKeyringEntry(keyringStore, keyringAdminPassword)
		}

		log.Printf("Warning: failed to save admin JWT key to keyring: %v", jwtKeySaveErr)
	}
}

func rollbackKeyringEntry(keyringStore credentialStore, key string) {
	if rollbackErr := keyringStore.DeleteKeyValue(key); rollbackErr != nil && !errors.Is(rollbackErr, security.ErrKeyNotFound) {
		log.Printf("Warning: failed to rollback %s from keyring: %v", key, rollbackErr)
	}
}

// resolveAdminCredentialsFromSources resolves username/password and the JWT
// key independently, each through keyring > .env/env > config.yml. They are
// resolved separately (rather than requiring all three from the same source)
// so installs upgrading from a keyring that only ever stored username+password
// keep using those values instead of falling all the way back to config.yml
// just because no JWT key has been stored yet.
func resolveAdminCredentialsFromSources(cfg *config.Config, keyringStore credentialStore, dotEnvValues map[string]string) (username, password, jwtKey string) {
	username, password = readAdminUsernamePasswordFromKeyring(keyringStore)
	jwtKey = readAdminJWTKeyFromKeyring(keyringStore)

	if username == "" {
		username = strings.TrimSpace(firstNonEmpty(dotEnvValues[authAdminUsernameEnv], os.Getenv(authAdminUsernameEnv)))
	}

	if password == "" {
		password = strings.TrimSpace(firstNonEmpty(dotEnvValues[authAdminSecretEnv], os.Getenv(authAdminSecretEnv)))
	}

	if jwtKey == "" {
		jwtKey = strings.TrimSpace(firstNonEmpty(dotEnvValues[authAdminJWTKeyEnv], os.Getenv(authAdminJWTKeyEnv)))
	}

	if username == "" {
		username = strings.TrimSpace(cfg.AdminUsername)
	}

	if password == "" {
		password = strings.TrimSpace(cfg.AdminPassword)
	}

	if jwtKey == "" {
		jwtKey = strings.TrimSpace(cfg.JWTKey)
	}

	return username, password, jwtKey
}

func readAdminUsernamePasswordFromKeyring(keyringStore credentialStore) (username, password string) {
	if keyringStore == nil {
		return "", ""
	}

	usernameVal, usernameErr := keyringStore.GetKeyValue(keyringAdminUsername)
	passwordVal, passwordErr := keyringStore.GetKeyValue(keyringAdminPassword)

	if usernameErr != nil && !errors.Is(usernameErr, security.ErrKeyNotFound) {
		log.Printf("Warning: failed to read admin username from keyring: %v", usernameErr)
	}

	if passwordErr != nil && !errors.Is(passwordErr, security.ErrKeyNotFound) {
		log.Printf("Warning: failed to read admin password from keyring: %v", passwordErr)
	}

	if usernameErr != nil || passwordErr != nil {
		if usernameErr == nil || passwordErr == nil {
			log.Print("Warning: partial admin username/password found in keyring; falling back to next source.")
		}

		return "", ""
	}

	username = strings.TrimSpace(usernameVal)
	password = strings.TrimSpace(passwordVal)

	if username == "" || password == "" {
		log.Print("Warning: incomplete admin username/password in keyring; falling back to next source.")

		return "", ""
	}

	return username, password
}

func readAdminJWTKeyFromKeyring(keyringStore credentialStore) string {
	if keyringStore == nil {
		return ""
	}

	jwtKeyVal, err := keyringStore.GetKeyValue(keyringAdminJWTKey)
	if err != nil {
		if !errors.Is(err, security.ErrKeyNotFound) {
			log.Printf("Warning: failed to read admin JWT key from keyring: %v", err)
		}

		return ""
	}

	return strings.TrimSpace(jwtKeyVal)
}

func promptForCredential(reader *bufio.Reader, prompt string) string {
	for {
		fmt.Fprint(os.Stdout, prompt)

		input, err := reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				log.Fatal("failed to read credential from console: EOF (non-interactive startup). Set AUTH_ADMIN_USERNAME and AUTH_ADMIN_PASSWORD, or enable AUTH_DISABLED=true.")
			}

			log.Fatalf("failed to read credential from console: %v", err)
		}

		value := strings.TrimSpace(input)
		if value != "" {
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

			value := strings.TrimSpace(string(input))
			if value != "" {
				return value
			}

			log.Println("Value cannot be empty.")

			continue
		}

		input, err := reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				log.Fatal("failed to read credential from console: EOF (non-interactive startup). Set AUTH_ADMIN_USERNAME and AUTH_ADMIN_PASSWORD, or enable AUTH_DISABLED=true.")
			}

			log.Fatalf("failed to read credential from console: %v", err)
		}

		value := strings.TrimSpace(input)
		if value != "" {
			return value
		}

		log.Println("Value cannot be empty.")
	}
}

func readDotEnvFile(path string) map[string]string {
	values := map[string]string{}

	data, err := os.ReadFile(path)
	if err != nil {
		return values
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		parts := strings.SplitN(trimmed, "=", dotEnvSplitParts)
		if len(parts) != dotEnvSplitParts {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		value = strings.Trim(value, "\"'")

		if key != "" {
			values[key] = value
		}
	}

	return values
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}

	return ""
}

func normalizeAdminPasswordHash(password string) (hash string, converted bool, err error) {
	if isBcryptHash(password) {
		return password, false, nil
	}

	bcryptHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", false, err
	}

	return string(bcryptHash), true, nil
}

func isBcryptHash(value string) bool {
	return strings.HasPrefix(value, bcryptPrefix2A) || strings.HasPrefix(value, bcryptPrefix2B) || strings.HasPrefix(value, bcryptPrefix2Y)
}

func bootstrapAdminCredentials(reader *bufio.Reader) (username, password, jwtKey string) {
	generatedPassword, err := generateRandomPassword(adminPasswordLength)
	if err != nil {
		log.Fatalf("Failed to generate random password: %v", err)
	}

	password = generatedPassword
	jwtKey = generateRandomAdminJWTKey()

	fmt.Fprintln(os.Stdout, "")
	fmt.Fprintln(os.Stdout, "\033[31m===============================================\033[0m")
	fmt.Fprintln(os.Stdout, "\033[31mCONSOLE ADMIN CREDENTIALS - FIRST-RUN BOOTSTRAP\033[0m")
	fmt.Fprintln(os.Stdout, "\033[31m===============================================\033[0m")
	fmt.Fprintln(os.Stdout, "\033[31mRecord these values now; they will NOT be shown again.\033[0m")
	fmt.Fprintln(os.Stdout, "")
	fmt.Fprintf(os.Stdout, "  Admin Username: %s\n", "standalone")
	fmt.Fprintf(os.Stdout, "  Admin Password: %s\n", password)
	fmt.Fprintln(os.Stdout, "")
	fmt.Fprintf(os.Stdout, "  JWT Signing Key: %s\n", jwtKey)
	fmt.Fprintln(os.Stdout, "")
	fmt.Fprintln(os.Stdout, "\033[31mThese will be stored in the OS keyring for security.\033[0m")
	fmt.Fprintln(os.Stdout, "\033[31m===============================================\033[0m")
	fmt.Fprintln(os.Stdout, "")

	fmt.Fprint(os.Stdout, "Press Enter to continue: ")

	_, _ = reader.ReadString('\n')

	return "standalone", password, jwtKey
}

func generateRandomAdminJWTKey() string {
	keyBytes := make([]byte, jwtKeyByteSize)

	_, err := rand.Read(keyBytes)
	if err != nil {
		log.Fatalf("Failed to generate JWT key: %v", err)
	}

	return base64.RawURLEncoding.EncodeToString(keyBytes)
}
