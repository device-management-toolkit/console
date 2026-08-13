package main

import (
	"bufio"
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
	keyringServiceName   = "device-management-toolkit"
	keyringAdminUsername = "console-admin-username"
	keyringAdminPassword = "console-admin-password"
	dotEnvFile           = ".env"
	authAdminUsernameEnv = "AUTH_ADMIN_USERNAME"
	authAdminSecretEnv   = "AUTH_ADMIN_PASSWORD" // #nosec G101 -- environment variable name, not a credential value
	bcryptPrefix2A       = "$2a$"
	bcryptPrefix2B       = "$2b$"
	bcryptPrefix2Y       = "$2y$"
	dotEnvSplitParts     = 2
)

var (
	errAdminCLIExclusiveFlags = errors.New("use only one of --remove-admin or --show-admin")
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
	command := selectedAdminCommand(args)
	if command == "" {
		return false, nil
	}

	if command == "conflict" {
		return true, errAdminCLIExclusiveFlags
	}

	if keyringStore == nil {
		return true, errKeystoreUnavailable
	}

	switch command {
	case "remove":
		return true, handleRemoveAdminCLI(keyringStore, out)
	case "show":
		return true, handleShowAdminCLI(keyringStore, out)
	default:
		return true, errAdminCLIExclusiveFlags
	}
}

func selectedAdminCommand(args []string) string {
	selected := 0
	command := ""

	if hasArg(args, "--remove-admin") {
		selected++
		command = "remove"
	}

	if hasArg(args, "--show-admin") {
		selected++
		command = "show"
	}

	if selected == 0 {
		return ""
	}

	if selected > 1 {
		return "conflict"
	}

	return command
}

func handleRemoveAdminCLI(keyringStore credentialStore, out io.Writer) error {
	if err := keyringStore.DeleteKeyValue(keyringAdminUsername); err != nil && !errors.Is(err, security.ErrKeyNotFound) {
		return fmt.Errorf("failed to remove admin username from keystore: %w", err)
	}

	if err := keyringStore.DeleteKeyValue(keyringAdminPassword); err != nil && !errors.Is(err, security.ErrKeyNotFound) {
		return fmt.Errorf("failed to remove admin password from keystore: %w", err)
	}

	if err := config.ClearAdminCredentials(); err != nil {
		fmt.Fprintf(out, "Warning: failed to clear admin credentials from config.yml: %v\n", err)
	}

	fmt.Fprintln(out, "Admin credentials removed from keystore.")

	return nil
}

func handleShowAdminCLI(keyringStore credentialStore, out io.Writer) error {
	username, err := keyringStore.GetKeyValue(keyringAdminUsername)
	if err != nil {
		return fmt.Errorf("failed to read admin username from keystore: %w", err)
	}

	fmt.Fprintf(out, "admin username: %s\n", username)

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

func handleAdminCredentials(cfg *config.Config) {
	if cfg.Disabled {
		log.Print("Auth is disabled; skipping admin credential resolution.")

		return
	}

	dotEnvValues := readDotEnvFile(dotEnvFile)
	keyringStore := newKeyringStorageFunc()

	username, password := resolveAdminCredentialsFromSources(cfg, keyringStore, dotEnvValues)

	reader := bufio.NewReader(os.Stdin)
	promptedForUsername := false
	promptedForPassword := false

	if username == "" {
		promptedForUsername = true
		username = promptForCredential(reader, "Enter Console admin username: ")
	}

	if password == "" {
		promptedForPassword = true
		password = promptForSecret(reader, "Enter Console admin password: ")
	}

	hashedPassword, _, err := normalizeAdminPasswordHash(password)
	if err != nil {
		log.Fatalf("failed to hash admin password: %v", err)
	}

	cfg.AdminUsername = username
	cfg.AdminPassword = hashedPassword

	persistedToKeyring, usernameSaveErr, passwordSaveErr := saveAdminCredentialsToKeyring(keyringStore, cfg.AdminUsername, cfg.AdminPassword)
	if persistedToKeyring {
		clearAdminCredentialsInConfigAfterKeyringStore()

		return
	}

	logKeyringSaveAndRollbackWarnings(keyringStore, usernameSaveErr, passwordSaveErr)

	if !promptedForUsername && !promptedForPassword {
		return
	}

	log.Print("Warning: keyring storage is unavailable; entered admin credentials may be lost after restart.")

	if !confirmPersistCredentialsToConfig(reader) {
		log.Print("Admin credentials were not written to config.yml. Provide them again on next startup or configure keyring/.env.")

		return
	}

	if err := config.SaveAdminCredentials(cfg.AdminUsername, cfg.AdminPassword); err != nil {
		log.Printf("Warning: failed to persist admin credentials to config.yml: %v", err)
	} else {
		log.Print("Admin credentials persisted to config.yml.")
	}
}

func saveAdminCredentialsToKeyring(keyringStore credentialStore, username, passwordHash string) (persisted bool, usernameErr, passwordErr error) {
	usernameErr = keyringStore.SetKeyValue(keyringAdminUsername, username)
	passwordErr = keyringStore.SetKeyValue(keyringAdminPassword, passwordHash)

	if usernameErr == nil && passwordErr == nil {
		return true, nil, nil
	}

	return false, usernameErr, passwordErr
}

func clearAdminCredentialsInConfigAfterKeyringStore() {
	if err := config.ClearAdminCredentials(); err != nil {
		log.Printf("Warning: failed to clear admin credentials from config.yml: %v", err)
	} else {
		log.Print("Admin credentials cleared from config.yml (stored in keystore).")
	}
}

func logKeyringSaveAndRollbackWarnings(keyringStore credentialStore, usernameSaveErr, passwordSaveErr error) {
	if usernameSaveErr != nil {
		log.Printf("Warning: failed to save admin username to keyring: %v", usernameSaveErr)
	}

	if passwordSaveErr != nil {
		if usernameSaveErr == nil {
			if rollbackErr := keyringStore.DeleteKeyValue(keyringAdminUsername); rollbackErr != nil && !errors.Is(rollbackErr, security.ErrKeyNotFound) {
				log.Printf("Warning: failed to rollback admin username from keyring: %v", rollbackErr)
			}
		}

		log.Printf("Warning: failed to save admin password to keyring: %v", passwordSaveErr)
	}

	if usernameSaveErr == nil || passwordSaveErr != nil {
		return
	}

	if rollbackErr := keyringStore.DeleteKeyValue(keyringAdminPassword); rollbackErr != nil && !errors.Is(rollbackErr, security.ErrKeyNotFound) {
		log.Printf("Warning: failed to rollback admin password from keyring: %v", rollbackErr)
	}
}

func resolveAdminCredentialsFromSources(cfg *config.Config, keyringStore credentialStore, dotEnvValues map[string]string) (username, password string) {
	username = ""
	password = ""

	if keyringStore != nil {
		username, password, ok := readAdminCredentialsFromKeyring(keyringStore)
		if ok {
			return username, password
		}
	}

	if username == "" {
		username = strings.TrimSpace(firstNonEmpty(dotEnvValues[authAdminUsernameEnv], os.Getenv(authAdminUsernameEnv)))
	}

	if password == "" {
		password = strings.TrimSpace(firstNonEmpty(dotEnvValues[authAdminSecretEnv], os.Getenv(authAdminSecretEnv)))
	}

	if username == "" {
		username = strings.TrimSpace(cfg.AdminUsername)
	}

	if password == "" {
		password = strings.TrimSpace(cfg.AdminPassword)
	}

	return username, password
}

func readAdminCredentialsFromKeyring(keyringStore credentialStore) (username, password string, ok bool) {
	usernameVal, usernameErr := keyringStore.GetKeyValue(keyringAdminUsername)
	passwordVal, passwordErr := keyringStore.GetKeyValue(keyringAdminPassword)

	if usernameErr == nil && passwordErr == nil {
		username = strings.TrimSpace(usernameVal)
		password = strings.TrimSpace(passwordVal)

		if username == "" || password == "" {
			log.Print("Warning: incomplete admin credentials in keyring; falling back to next source.")

			return "", "", false
		}

		return username, password, true
	}

	if usernameErr != nil && !errors.Is(usernameErr, security.ErrKeyNotFound) {
		log.Printf("Warning: failed to read admin username from keyring: %v", usernameErr)
	}

	if passwordErr != nil && !errors.Is(passwordErr, security.ErrKeyNotFound) {
		log.Printf("Warning: failed to read admin password from keyring: %v", passwordErr)
	}

	if (usernameErr == nil && errors.Is(passwordErr, security.ErrKeyNotFound)) ||
		(passwordErr == nil && errors.Is(usernameErr, security.ErrKeyNotFound)) {
		log.Print("Warning: partial admin credentials found in keyring; falling back to next source.")
	}

	return "", "", false
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

func confirmPersistCredentialsToConfig(reader *bufio.Reader) bool {
	log.Print("Store admin username/password in config.yml as fallback? Y/N: ")

	input, err := reader.ReadString('\n')
	if err != nil {
		log.Fatalf("failed to read confirmation from console: %v", err)
	}

	response := strings.TrimSpace(input)

	return response == "Y" || response == "y"
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
