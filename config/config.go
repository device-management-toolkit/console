package config

import (
	"errors"
	"flag"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"gopkg.in/yaml.v2"
)

var ConsoleConfig *Config

const defaultHost = "localhost"

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
		Redfish `yaml:"redfish"`
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
		Host           string   `env-required:"true" yaml:"host" env:"HTTP_HOST"`
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
	DB struct {
		PoolMax int    `env-required:"true" yaml:"pool_max" env:"DB_POOL_MAX"`
		URL     string `env:"DB_URL"`
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
		UI                       UIAuthConfig  `yaml:"ui"`
	}

	// UIAuthConfig -.
	UIAuthConfig struct {
		ClientID                          string `yaml:"clientId"`
		Issuer                            string `yaml:"issuer"`
		RedirectURI                       string `yaml:"redirectUri"`
		Scope                             string `yaml:"scope"`
		ResponseType                      string `yaml:"responseType"`
		RequireHTTPS                      bool   `yaml:"requireHttps"`
		StrictDiscoveryDocumentValidation bool   `yaml:"strictDiscoveryDocumentValidation"`
	}

	// UI -.
	UI struct {
		ExternalURL string `yaml:"externalUrl" env:"UI_EXTERNAL_URL"`
	}
	// Redfish -.
	Redfish struct {
		EnvironmentUUID string `yaml:"environment_uuid" env:"REDFISH_ENV_UUID"`
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
			Host:           "localhost",
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
			PoolMax: 2,
			URL:     "",
		},
		EA: EA{
			URL:      "http://localhost:8000",
			Username: "",
			Password: "",
		},
		Auth: Auth{
			AdminUsername:            "standalone",
			AdminPassword:            "G@ppm0ym",
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
		Redfish: Redfish{
			EnvironmentUUID: "",
		},
	}
}

// resolveConfigPath determines the effective config file path based on a flag value or default location.
func resolveConfigPath(configPathFlag string) (string, error) {
	if configPathFlag != "" {
		return configPathFlag, nil
	}

	ex, err := os.Executable()
	if err != nil {
		return "", err
	}

	exPath := filepath.Dir(ex)

	return filepath.Join(exPath, "config", "config.yml"), nil
}

// readOrInitConfig attempts to read the config file; if it doesn't exist, writes the provided cfg to disk.
func readOrInitConfig(configPath string, cfg *Config) error {
	err := cleanenv.ReadConfig(configPath, cfg)
	if err == nil {
		return nil
	}

	var pathErr *os.PathError
	if errors.As(err, &pathErr) {
		// Write config file out to disk
		configDir := filepath.Dir(configPath)
		if mkErr := os.MkdirAll(configDir, os.ModePerm); mkErr != nil {
			return mkErr
		}

		file, cErr := os.Create(configPath)
		if cErr != nil {
			return cErr
		}
		defer file.Close()

		encoder := yaml.NewEncoder(file)
		defer encoder.Close()

		if encErr := encoder.Encode(cfg); encErr != nil {
			return encErr
		}

		return nil
	}

	return err
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

	if !flag.Parsed() {
		flag.Parse()
	}

	// Determine the config path
	configPath, err := resolveConfigPath(configPathFlag)
	if err != nil {
		return nil, err
	}

	if err := readOrInitConfig(configPath, ConsoleConfig); err != nil {
		return nil, err
	}

	if err := cleanenv.ReadEnv(ConsoleConfig); err != nil {
		return nil, err
	}

	return ConsoleConfig, nil
}
