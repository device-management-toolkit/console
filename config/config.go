package config

import (
	"errors"
	"flag"
	"os"
	"path/filepath"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"gopkg.in/yaml.v2"
)

var ConsoleConfig *Config

type (
	// Config -.
	Config struct {
		App  `yaml:"app"`
		HTTP `yaml:"http"`
		Log  `yaml:"logger"`
		DB   `yaml:"postgres"`
		EA   `yaml:"ea"`
		Auth `yaml:"auth"`
	}

	// App -.
	App struct {
		Name                 string `env-required:"true" yaml:"name" env:"APP_NAME"`
		Repo                 string `env-required:"true" yaml:"repo" env:"APP_REPO"`
		Version              string `env-required:"true"`
		EncryptionKey        string `yaml:"encryption_key" env:"APP_ENCRYPTION_KEY"`
		AllowInsecureCiphers bool   `yaml:"allow_insecure_ciphers" env:"APP_ALLOW_INSECURE_CIPHERS"`
	}

	// HTTP -.
	HTTP struct {
		Host           string   `env-required:"true" yaml:"host" env:"HTTP_HOST"`
		Port           string   `env-required:"true" yaml:"port" env:"HTTP_PORT"`
		AllowedOrigins []string `env-required:"true" yaml:"allowed_origins" env:"HTTP_ALLOWED_ORIGINS"`
		AllowedHeaders []string `env-required:"true" yaml:"allowed_headers" env:"HTTP_ALLOWED_HEADERS"`
		WSCompression  bool     `yaml:"ws_compression" env:"WS_COMPRESSION"`
	}

	// Log -.
	Log struct {
		Level string `env-required:"true" yaml:"log_level"   env:"LOG_LEVEL"`
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
)

// NewConfig returns app config.
func NewConfig() (*Config, error) {
	// set defaults
	ConsoleConfig = &Config{
		App: App{
			Name:                 "console",
			Repo:                 "device-management-toolkit/console",
			Version:              "DEVELOPMENT",
			EncryptionKey:        "",
			AllowInsecureCiphers: false,
		},
		HTTP: HTTP{
			Host:           "localhost",
			Port:           "8181",
			AllowedOrigins: []string{"*"},
			AllowedHeaders: []string{"*"},
			WSCompression:  true,
		},
		Log: Log{
			Level: "info",
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
	}

	// Define a command line flag for the config path
	var configPathFlag string
	if flag.Lookup("config") == nil {
		flag.StringVar(&configPathFlag, "config", "", "path to config file")
	}

	flag.Parse()

	// Determine the config path
	var configPath string
	if configPathFlag != "" {
		configPath = configPathFlag
	} else {
		ex, err := os.Executable()
		if err != nil {
			panic(err)
		}

		exPath := filepath.Dir(ex)

		configPath = filepath.Join(exPath, "config", "config.yml")
	}

	err := cleanenv.ReadConfig(configPath, ConsoleConfig)

	var pathErr *os.PathError

	if errors.As(err, &pathErr) {
		// Write config file out to disk
		configDir := filepath.Dir(configPath)
		if err := os.MkdirAll(configDir, os.ModePerm); err != nil {
			return nil, err
		}

		file, err := os.Create(configPath)
		if err != nil {
			return nil, err
		}
		defer file.Close()

		encoder := yaml.NewEncoder(file)
		defer encoder.Close()

		if err := encoder.Encode(ConsoleConfig); err != nil {
			return nil, err
		}
	}

	err = cleanenv.ReadEnv(ConsoleConfig)
	if err != nil {
		return nil, err
	}

	return ConsoleConfig, nil
}
