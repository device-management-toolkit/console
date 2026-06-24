package config

import (
	"errors"
	"flag"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"gopkg.in/yaml.v2"
)

var ConsoleConfig *Config

// TrayMode indicates whether to run with system tray UI.
var TrayMode bool

const defaultHost = "localhost"

const (
	// configFilePerm restricts the config file to its owner — it can hold
	// secrets (auth.adminPassword, auth.jwtKey), so it must not be world-readable.
	configFilePerm = 0o600
	// configDirPerm restricts the config directory to its owner, matching the
	// confidentiality of the file it contains.
	configDirPerm = 0o700
	// goosWindows is runtime.GOOS's value on Windows, named to avoid repeating
	// the bare string literal across path resolution and tests.
	goosWindows = "windows"
)

type (
	// Config -.
	Config struct {
		App     `yaml:"app"`
		HTTP    `yaml:"http"`
		Log     `yaml:"logger"`
		Secrets `yaml:"secrets"`
		DB      `yaml:"postgres"`
		EA      `yaml:"ea"`
		Auth    `yaml:"auth"`
		UI      `yaml:"ui"`
	}

	// App -.
	App struct {
		Name                 string `env-required:"true" yaml:"name" env:"APP_NAME"`
		Repo                 string `env-required:"true" yaml:"repo" env:"APP_REPO"`
		Version              string `env-required:"true"`
		CommonName           string `env-required:"true" yaml:"common_name" env:"APP_COMMON_NAME"`
		EncryptionKey        string `yaml:"encryption_key" env:"APP_ENCRYPTION_KEY"`
		AllowInsecureCiphers bool   `yaml:"allow_insecure_ciphers" env:"APP_ALLOW_INSECURE_CIPHERS"`
		DisableCIRA          bool   `yaml:"disable_cira" env:"APP_DISABLE_CIRA"`
	}

	// HTTP -.
	HTTP struct {
		Host           string   `yaml:"host" env:"HTTP_HOST"`
		Port           string   `env-required:"true" yaml:"port" env:"HTTP_PORT"`
		AllowedOrigins []string `env-required:"true" yaml:"allowed_origins" env:"HTTP_ALLOWED_ORIGINS"`
		AllowedHeaders []string `env-required:"true" yaml:"allowed_headers" env:"HTTP_ALLOWED_HEADERS"`
		WSCompression  bool     `yaml:"ws_compression" env:"WS_COMPRESSION"`
		TLS            TLS      `yaml:"tls"`
	}

	// TLS -.
	TLS struct {
		Enabled  bool   `yaml:"enabled" env:"HTTP_TLS_ENABLED"`
		CertFile string `yaml:"certFile" env:"HTTP_TLS_CERT_FILE"`
		KeyFile  string `yaml:"keyFile" env:"HTTP_TLS_KEY_FILE"`
	}

	// Log -.
	Log struct {
		Level string `env-required:"true" yaml:"log_level"   env:"LOG_LEVEL"`
	}

	// Secrets -.
	Secrets struct {
		Address string `yaml:"address" env:"SECRETS_ADDR"`
		Token   string `yaml:"token" env:"SECRETS_TOKEN"`
		Path    string `yaml:"path" env:"SECRETS_PATH"`
	}

	// DB -.
	//
	// Provider selects the backend: "postgres", "sqlite" (default), or "mongo".
	// See internal/app/repos.go for the per-provider rules around DB_URL.
	DB struct {
		Provider string `yaml:"provider" env:"DB_PROVIDER"`
		PoolMax  int    `env-required:"true" yaml:"pool_max" env:"DB_POOL_MAX"`
		URL      string `env:"DB_URL"`
	}

	// EA -.
	EA struct {
		URL      string `yaml:"url" env:"EA_URL"`
		Username string `yaml:"username" env:"EA_USERNAME"`
		Password string `yaml:"password" env:"EA_PASSWORD"`
	}

	// Auth -.
	Auth struct {
		Disabled                 bool          `yaml:"disabled" env:"AUTH_DISABLED"`
		AdminUsername            string        `yaml:"adminUsername" env:"AUTH_ADMIN_USERNAME"`
		AdminPassword            string        `yaml:"adminPassword" env:"AUTH_ADMIN_PASSWORD"`
		JWTKey                   string        `env-required:"true" yaml:"jwtKey" env:"AUTH_JWT_KEY"`
		JWTExpiration            time.Duration `yaml:"jwtExpiration" env:"AUTH_JWT_EXPIRATION"`
		RedirectionJWTExpiration time.Duration `yaml:"redirectionJWTExpiration" env:"AUTH_REDIRECTION_JWT_EXPIRATION"`
		ClientID                 string        `yaml:"clientId" env:"AUTH_CLIENT_ID"`
		Issuer                   string        `yaml:"issuer" env:"AUTH_ISSUER"`
		TLSSkipVerify            bool          `yaml:"tlsSkipVerify" env:"AUTH_TLS_SKIP_VERIFY"`
		UI                       UIAuthConfig  `yaml:"ui"`
	}

	// UIAuthConfig -.
	UIAuthConfig struct {
		ClientID                          string `yaml:"clientId" env:"AUTH_UI_CLIENT_ID"`
		Issuer                            string `yaml:"issuer" env:"AUTH_UI_ISSUER"`
		RedirectURI                       string `yaml:"redirectUri" env:"AUTH_UI_REDIRECT_URI"`
		Scope                             string `yaml:"scope" env:"AUTH_UI_SCOPE"`
		ResponseType                      string `yaml:"responseType" env:"AUTH_UI_RESPONSE_TYPE"`
		RequireHTTPS                      bool   `yaml:"requireHttps" env:"AUTH_UI_REQUIRE_HTTPS"`
		StrictDiscoveryDocumentValidation bool   `yaml:"strictDiscoveryDocumentValidation" env:"AUTH_UI_STRICT_DISCOVERY"`
	}

	// UI -.
	UI struct {
		ExternalURL string `yaml:"externalUrl" env:"UI_EXTERNAL_URL"`
	}
)

// getPreferredIPAddress detects the most likely candidate IP address for this machine.
// It prefers non-loopback IPv4 addresses and excludes link-local addresses.
func getPreferredIPAddress() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return defaultHost
	}

	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				// Exclude link-local addresses (169.254.x.x)
				if !ipNet.IP.IsLinkLocalUnicast() {
					return ipNet.IP.String()
				}
			}
		}
	}

	return defaultHost
}

// defaultConfig constructs the in-memory default configuration.
func defaultConfig() *Config {
	return &Config{
		App: App{
			Name:                 "console",
			Repo:                 "device-management-toolkit/console",
			Version:              "DEVELOPMENT",
			CommonName:           getPreferredIPAddress(),
			EncryptionKey:        "",
			AllowInsecureCiphers: false,
			DisableCIRA:          true,
		},
		HTTP: HTTP{
			Host:           "",
			Port:           "8181",
			AllowedOrigins: []string{"*"},
			AllowedHeaders: []string{"*"},
			WSCompression:  true,
			TLS: TLS{
				Enabled:  true,
				CertFile: "",
				KeyFile:  "",
			},
		},
		Log: Log{
			Level: "info",
		},
		Secrets: Secrets{
			Address: "http://localhost:8200",
			Token:   "",
			Path:    "secret/data/console",
		},
		DB: DB{
			Provider: "sqlite",
			PoolMax:  2,
			URL:      "",
		},
		EA: EA{
			URL:      "http://localhost:8000",
			Username: "",
			Password: "",
		},
		Auth: Auth{
			AdminUsername:            "standalone",
			AdminPassword:            "", // Generated and stored in config on first run if not provided
			JWTKey:                   "your_secret_jwt_key",
			JWTExpiration:            24 * time.Hour,
			RedirectionJWTExpiration: 5 * time.Minute,
			// OAUTH CONFIG, if provided will not use basic auth
			ClientID: "",
			Issuer:   "",
			UI: UIAuthConfig{
				ClientID:                          "",
				Issuer:                            "",
				RedirectURI:                       "",
				Scope:                             "",
				ResponseType:                      "",
				RequireHTTPS:                      false,
				StrictDiscoveryDocumentValidation: true,
			},
		},
		UI: UI{
			ExternalURL: "",
		},
	}
}

// resolveConfigPath determines the effective config file path based on a flag value or default location.
func resolveConfigPath(configPathFlag string) (string, error) {
	if configPathFlag != "" {
		return configPathFlag, nil
	}

	// The tray runs unelevated as the logged-in user, and its config holds
	// credentials (auth.adminPassword, auth.jwtKey). Keep it in the per-user
	// config dir — the same base the SQLite DB uses — instead of world-readable
	// beside the system-wide binary (Program Files / /usr/local / /opt).
	if TrayMode {
		if perUser, err := perUserConfigPath(); err == nil {
			return perUser, nil
		}
		// Fall through to the headless resolution below if the per-user dir is unavailable.
	}

	// Headless uses the installer-provisioned machine-wide config when one is
	// present. Without an installer (a plain `go run`, a dev build, or the test
	// binary) that path is an unwritable system dir (/etc, /Library, ProgramData),
	// so fall back to the writable beside-binary path — keeping zero-infra startup.
	machine, err := machineConfigPath()
	if err != nil {
		return "", err
	}

	if _, statErr := os.Stat(machine); statErr == nil {
		return machine, nil
	}

	return besideBinaryConfigPath()
}

// perUserConfigPath returns <user config dir>/device-management-toolkit/config/config.yml.
func perUserConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, "device-management-toolkit", "config", "config.yml"), nil
}

// besideBinaryConfigPath returns the config path next to the executable. It is
// the writable fallback for headless runs with no installer-provisioned machine
// config (dev builds, `go run`, tests) and for an unrecognized GOOS.
func besideBinaryConfigPath() (string, error) {
	ex, err := os.Executable()
	if err != nil {
		return "", err
	}

	// Resolve symlinks so invocation via a wrapper symlink (e.g. /usr/local/bin/dmt-console
	// → /usr/local/device-management-toolkit/console) anchors config beside the real binary
	// rather than beside the symlink.
	if resolved, evalErr := filepath.EvalSymlinks(ex); evalErr == nil {
		ex = resolved
	}

	return filepath.Join(filepath.Dir(ex), "config", "config.yml"), nil
}

// machineConfigPath returns the machine-wide, installer-provisioned config path,
// kept out of the program directory and readable by the unelevated user:
//   - Windows: %ProgramData%\device-management-toolkit\config.yml
//   - macOS:   /Library/Application Support/device-management-toolkit/config.yml
//   - Linux:   /etc/dmt-console/config/config.yml
//
// Anything else falls back to the beside-binary path.
func machineConfigPath() (string, error) {
	switch runtime.GOOS {
	case goosWindows:
		if dir := os.Getenv("ProgramData"); dir != "" {
			return filepath.Join(dir, "device-management-toolkit", "config.yml"), nil
		}
	case "darwin":
		return "/Library/Application Support/device-management-toolkit/config.yml", nil
	case "linux":
		return "/etc/dmt-console/config/config.yml", nil
	}

	return besideBinaryConfigPath()
}

// seedConfig copies an installer-provisioned config from src to dst when dst does
// not yet exist, carrying the installer's credentials (admin password, jwtKey) into
// the per-user location instead of generating a fresh config. A missing src or an
// already-present dst are both no-ops. On a shared machine each OS user seeds from
// the same src, inheriting one credential set until rotated — intentional.
func seedConfig(src, dst string) error {
	_, err := os.Stat(dst)
	if err == nil {
		return nil // dst already present — never overwrite an existing per-user config
	}

	if !os.IsNotExist(err) {
		return err // unexpected stat error (permissions/IO) — surface it rather than silently re-seeding
	}

	data, err := os.ReadFile(src)
	if err != nil {
		return nil //nolint:nilerr // no installer config to migrate (e.g. dev run); init proceeds normally
	}

	if mkErr := os.MkdirAll(filepath.Dir(dst), configDirPerm); mkErr != nil {
		return mkErr
	}

	// #nosec G703 -- dst is derived from os.UserConfigDir()/os.Executable() (see resolveConfigPath), not external input.
	return os.WriteFile(dst, data, configFilePerm)
}

// readOrInitConfig attempts to read the config file; if it doesn't exist, writes the provided cfg to disk.
func readOrInitConfig(configPath string, cfg *Config) error {
	err := cleanenv.ReadConfig(configPath, cfg)
	if err == nil {
		return nil
	}

	var pathErr *os.PathError
	if errors.As(err, &pathErr) {
		return writeConfig(configPath, cfg)
	}

	return err
}

// writeConfig serializes cfg to configPath, creating the parent directory if needed.
// The file and directory are owner-only (configFilePerm/configDirPerm): a generated
// config carries secrets (a freshly generated jwtKey/adminPassword) just like a seeded
// one, so the freshly-generated path must not write them world-readable.
func writeConfig(configPath string, cfg *Config) error {
	configDir := filepath.Dir(configPath)
	if mkErr := os.MkdirAll(configDir, configDirPerm); mkErr != nil {
		return mkErr
	}

	file, err := os.OpenFile(configPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, configFilePerm)
	if err != nil {
		return err
	}
	defer file.Close()

	// O_CREATE only applies configFilePerm when the file is newly created; an
	// existing config left world-readable by an older build keeps its old mode.
	// Chmod explicitly so upgrades are tightened to owner-only too.
	if err := file.Chmod(configFilePerm); err != nil {
		return err
	}

	encoder := yaml.NewEncoder(file)
	defer encoder.Close()

	return encoder.Encode(cfg)
}

// SaveAdminPassword persists adminPassword to auth.adminPassword in config.yml
// without touching any other field. It re-reads the file directly (bypassing the
// env-var overlay applied by cleanenv) so env-only secrets like APP_ENCRYPTION_KEY,
// SECRETS_TOKEN, DB_URL, EA_PASSWORD, and AUTH_JWT_KEY cannot leak to disk.
func SaveAdminPassword(adminPassword string) error {
	var configPathFlag string
	if f := flag.Lookup("config"); f != nil {
		configPathFlag = f.Value.String()
	}

	configPath, err := resolveConfigPath(configPathFlag)
	if err != nil {
		return err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	fileCfg := defaultConfig()
	if err := yaml.Unmarshal(data, fileCfg); err != nil {
		return err
	}

	fileCfg.AdminPassword = adminPassword

	return writeConfig(configPath, fileCfg)
}

// NewConfig returns app config.
func NewConfig() (*Config, error) {
	// set defaults
	ConsoleConfig = defaultConfig()

	// Define a command line flag for the config path
	var configPathFlag string
	if flag.Lookup("config") == nil {
		flag.StringVar(&configPathFlag, "config", "", "path to config file")
	}

	if flag.Lookup("tray") == nil {
		flag.BoolVar(&TrayMode, "tray", false, "run with system tray icon")
	}

	if !flag.Parsed() {
		flag.Parse()
	}

	// Determine the config path
	configPath, err := resolveConfigPath(configPathFlag)
	if err != nil {
		return nil, err
	}

	// First tray launch: seed the per-user config from the machine-wide installer
	// config so its credentials carry over instead of being regenerated.
	if TrayMode && configPathFlag == "" {
		if src, srcErr := machineConfigPath(); srcErr == nil {
			if seedErr := seedConfig(src, configPath); seedErr != nil {
				return nil, seedErr
			}
		}
	}

	if err := readOrInitConfig(configPath, ConsoleConfig); err != nil {
		return nil, err
	}

	if err := cleanenv.ReadEnv(ConsoleConfig); err != nil {
		return nil, err
	}

	return ConsoleConfig, nil
}
