package config

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"gopkg.in/yaml.v2"
)

var ConsoleConfig *Config

// TrayMode indicates whether to run with system tray UI.
var TrayMode bool

var (
	ErrJWTExpirationInvalid            = errors.New("config: auth.jwtExpiration must be at least 1 minute (e.g. 24h) — very short expirations render tokens unusable")
	ErrRedirectionJWTExpirationInvalid = errors.New("config: auth.redirectionJWTExpiration must be at least 1 minute (e.g. 5m) — very short expirations render redirection tokens unusable")
)

const defaultHost = "localhost"

// DefaultSessionCookieName names the HttpOnly cookie holding the session JWT.
const DefaultSessionCookieName = "console_session"

// File modes for the config directory and file (owner-only for the file since
// it can carry sensitive settings).
const (
	configFilePerm os.FileMode = 0o600
	configDirPerm  os.FileMode = 0o700

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
		Host             string   `yaml:"host" env:"HTTP_HOST"`
		Port             string   `env-required:"true" yaml:"port" env:"HTTP_PORT"`
		AllowedOrigins   []string `yaml:"allowed_origins" env:"HTTP_ALLOWED_ORIGINS"`
		AllowedHeaders   []string `env-required:"true" yaml:"allowed_headers" env:"HTTP_ALLOWED_HEADERS"`
		AllowCredentials bool     `yaml:"allow_credentials" env:"HTTP_ALLOW_CREDENTIALS"`
		WSCompression    bool     `yaml:"ws_compression" env:"WS_COMPRESSION"`
		TLS              TLS      `yaml:"tls"`
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
	//
	// The Cookie* fields govern the HttpOnly session cookie the browser uses in
	// place of Web Storage. Additive only: /authorize still returns the token in
	// the body and the Authorization header still wins, so REST clients are
	// unaffected.
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
		CookieEnabled            bool          `yaml:"cookieEnabled" env:"AUTH_COOKIE_ENABLED"`
		CookieName               string        `yaml:"cookieName" env:"AUTH_COOKIE_NAME"`
		CookieSecure             bool          `yaml:"cookieSecure" env:"AUTH_COOKIE_SECURE"`
		CookieSameSite           string        `yaml:"cookieSameSite" env:"AUTH_COOKIE_SAME_SITE"`
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

// CookieAuthEnabled reports whether the HttpOnly session cookie is in use. Off
// under OIDC, where the IdP owns the token. Read by the middleware and the spec.
func (a Auth) CookieAuthEnabled() bool {
	return a.CookieEnabled && a.ClientID == ""
}

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
			Host:             "",
			Port:             "8181",
			AllowedOrigins:   []string{},
			AllowedHeaders:   []string{"*"},
			AllowCredentials: false,
			WSCompression:    true,
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
			CookieEnabled:            true,
			CookieName:               DefaultSessionCookieName,
			CookieSecure:             true,
			CookieSameSite:           "strict",
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

	if TrayMode {
		if perUser, err := perUserConfigPath(); err == nil {
			return perUser, nil
		}
	}

	machine, err := machineConfigPath()
	if err != nil {
		return "", err
	}

	if _, statErr := os.Stat(machine); statErr == nil {
		return machine, nil
	}

	return besideBinaryConfigPath()
}

func perUserConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, "device-management-toolkit", "config", "config.yml"), nil
}

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

func seedConfig(src, dst string) error {
	_, err := os.Stat(dst)
	if err == nil {
		return nil
	}

	if !os.IsNotExist(err) {
		return err
	}

	data, err := os.ReadFile(src)
	if err != nil {
		return nil //nolint:nilerr // no installer config to migrate (e.g. dev run); init proceeds normally
	}

	if mkErr := os.MkdirAll(filepath.Dir(dst), configDirPerm); mkErr != nil {
		return mkErr
	}

	// #nosec G703 -- dst derives from resolveConfigPath, not external input.
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
func writeConfig(configPath string, cfg *Config) error {
	configDir := filepath.Dir(configPath)
	if mkErr := os.MkdirAll(configDir, configDirPerm); mkErr != nil {
		return mkErr
	}

	file, err := os.OpenFile(configPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, configFilePerm)
	if err != nil {
		return err
	}
	defer file.Close()

	// Tighten pre-existing files too; OpenFile's mode only applies at creation.
	if err := os.Chmod(configPath, configFilePerm); err != nil {
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

// validate checks that all Config values are sane.
// It returns an error for any setting that would cause a runtime failure or
// deny all service to legitimate users (e.g. zero/negative JWT expiration).
func (c *Config) validate() error {
	if c.JWTExpiration < time.Minute {
		return ErrJWTExpirationInvalid
	}

	if c.RedirectionJWTExpiration < time.Minute {
		return ErrRedirectionJWTExpirationInvalid
	}

	return nil
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
		ConsoleConfig = nil

		return nil, err
	}

	if TrayMode && configPathFlag == "" {
		if src, srcErr := machineConfigPath(); srcErr == nil {
			if seedErr := seedConfig(src, configPath); seedErr != nil {
				return nil, seedErr
			}
		}
	}

	if err := readOrInitConfig(configPath, ConsoleConfig); err != nil {
		ConsoleConfig = nil

		return nil, err
	}

	if err := cleanenv.ReadEnv(ConsoleConfig); err != nil {
		ConsoleConfig = nil

		return nil, err
	}

	if err := ConsoleConfig.validate(); err != nil {
		ConsoleConfig = nil

		return nil, err
	}

	if err := validatePort(ConsoleConfig.Port); err != nil {
		return nil, err
	}

	if err := validateAndSetEncryptionKey(ConsoleConfig.EncryptionKey); err != nil {
		return nil, err
	}

	return ConsoleConfig, nil
}

// Sentinel errors for port validation.
var (
	ErrPortNotNumeric = errors.New("HTTP port (HTTP_PORT) must be a decimal integer")
	ErrPortOutOfRange = errors.New("HTTP port (HTTP_PORT) must be in range 1-65535")
)

// validatePort returns an error if port is not a decimal integer in the range 1–65535.
func validatePort(port string) error {
	n, err := strconv.Atoi(port)
	if err != nil {
		return ErrPortNotNumeric
	}

	if n < 1 || n > 65535 {
		return ErrPortOutOfRange
	}

	return nil
}

func validateAndSetEncryptionKey(encryptionKey string) error {
	if encryptionKey != "" {
		if err := ValidateEncryptionKey(encryptionKey); err != nil {
			return fmt.Errorf(
				"invalid APP_ENCRYPTION_KEY (app.encryption_key in config.yml): %w.\n"+
					"Generate one with `openssl rand -base64 24` (32 characters), "+
					"or leave it unset and let Console generate and store a key for you",
				err,
			)
		}
	}

	return nil
}
