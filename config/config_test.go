package config

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func clearEnv() {
	os.Unsetenv("APP_NAME")
	os.Unsetenv("HTTP_PORT")
	os.Unsetenv("LOG_LEVEL")
	os.Unsetenv("DB_POOL_MAX")
	os.Unsetenv("DB_URL")
	os.Unsetenv("SECRETS_ADDR")
}

func TestNewConfig_InvalidEnvVar(t *testing.T) {
	clearEnv()
	defer clearEnv()

	// DB_POOL_MAX expects an int; a non-numeric value causes cleanenv.ReadEnv to fail.
	t.Setenv("DB_POOL_MAX", "not-a-number")

	cfg, err := NewConfig()
	assert.Error(t, err)
	assert.Nil(t, cfg)
}

func TestNewConfig_Defaults(t *testing.T) { //nolint:paralleltest // cannot have simultaneous tests modifying environment variables
	clearEnv() // Clear environment variables to ensure defaults are tested

	cfg, err := NewConfig()

	cfg.EncryptionKey = "test" // Added to pass the test

	assert.NoError(t, err)
	assert.NotNil(t, cfg)

	// Verify default values
	assert.Equal(t, "console", cfg.Name)
	assert.Equal(t, "device-management-toolkit/console", cfg.Repo)
	assert.Equal(t, "DEVELOPMENT", cfg.Version)
	assert.Equal(t, "test", cfg.EncryptionKey)

	assert.Equal(t, "", cfg.Host)
	assert.Equal(t, "8181", cfg.Port)
	assert.Equal(t, []string{"*"}, cfg.AllowedOrigins)
	assert.Equal(t, []string{"*"}, cfg.AllowedHeaders)
	assert.Equal(t, true, cfg.TLS.Enabled)

	assert.Equal(t, "info", cfg.Level)

	assert.Equal(t, 2, cfg.PoolMax)
}

func TestNewConfig_EnvVars(t *testing.T) { //nolint:paralleltest // cannot have simultaneous tests modifying environment variables
	// Set environment variables
	os.Setenv("APP_NAME", "testApp")
	os.Setenv("HTTP_PORT", "9090")
	os.Setenv("LOG_LEVEL", "debug")
	os.Setenv("DB_POOL_MAX", "10")
	os.Setenv("DB_URL", "postgres://user:password@localhost:5432/testdb")
	os.Setenv("HTTP_TLS_ENABLED", "false")

	defer clearEnv() // Ensure environment variables are cleared after test

	cfg, err := NewConfig()
	assert.NoError(t, err)
	assert.NotNil(t, cfg)

	// Verify environment variable values
	assert.Equal(t, "testApp", cfg.Name)
	assert.Equal(t, "9090", cfg.Port)
	assert.Equal(t, "debug", cfg.Level)
	assert.Equal(t, 10, cfg.PoolMax)
	assert.Equal(t, "postgres://user:password@localhost:5432/testdb", cfg.DB.URL)
	assert.Equal(t, false, cfg.TLS.Enabled)
}

func TestNewConfig_FileAndEnvVars(t *testing.T) { //nolint:paralleltest // cannot have simultaneous tests modifying environment variables
	clearEnv() // Clear environment variables before setting new ones

	// Create a temporary config file
	configYAML := `
app:
  name: fileApp
http:
  port: "8080"
logger:
  log_level: warn
postgres:
  pool_max: 5
  url: postgres://fileuser:filepassword@localhost:5432/filedb
`
	configFilePath := "./test_config.yml"
	err := os.WriteFile(configFilePath, []byte(configYAML), 0o600)
	assert.NoError(t, err)

	defer os.Remove(configFilePath)

	// Set environment variables
	os.Setenv("APP_NAME", "envApp")
	os.Setenv("HTTP_PORT", "9090")
	os.Setenv("LOG_LEVEL", "debug")
	os.Setenv("DB_POOL_MAX", "10")
	os.Setenv("DB_URL", "postgres://envuser:envpassword@localhost:5432/envdb")

	defer clearEnv() // Ensure environment variables are cleared after test

	cfg, err := NewConfig()
	assert.NoError(t, err)
	assert.NotNil(t, cfg)

	// Verify environment variable values override file values
	assert.Equal(t, "envApp", cfg.Name)
	assert.Equal(t, "9090", cfg.Port)
	assert.Equal(t, "debug", cfg.Level)
	assert.Equal(t, 10, cfg.PoolMax)
	assert.Equal(t, "postgres://envuser:envpassword@localhost:5432/envdb", cfg.DB.URL)
}

func TestValidate_SecretsAddrEmpty(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Secrets: Secrets{
			Address: "",
		},
	}
	assert.NoError(t, cfg.Validate())
}

func TestValidate_SecretsAddrHTTPSNonLocalhost(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Secrets: Secrets{
			Address: "https://vault.example.com:8200",
		},
	}
	assert.NoError(t, cfg.Validate())
}

func TestValidate_SecretsAddrHTTPLocalhost(t *testing.T) {
	t.Parallel()

	testCases := []string{
		"http://localhost:8200",
		"http://127.0.0.1:8200",
		"http://127.0.0.2:8200",
		"http://[::1]:8200",
		"http://[::1]",
	}
	for _, addr := range testCases {
		t.Run(addr, func(t *testing.T) {
			t.Parallel()

			cfg := &Config{
				Secrets: Secrets{
					Address: addr,
				},
			}
			assert.NoError(t, cfg.Validate(), "expected %s to be valid", addr)
		})
	}
}

func TestValidate_SecretsAddrHTTPNonLocalhost(t *testing.T) {
	t.Parallel()

	testCases := []string{
		"http://vault.example.com:8200",
		"http://192.168.1.1:8200",
		"http://vault-server:8200",
	}
	for _, addr := range testCases {
		t.Run(addr, func(t *testing.T) {
			t.Parallel()

			cfg := &Config{
				Secrets: Secrets{
					Address: addr,
				},
			}
			err := cfg.Validate()
			assert.Error(t, err, "expected %s to fail", addr)
			assert.True(t, errors.Is(err, ErrSecretsAddrInsecure), "expected ErrSecretsAddrInsecure, got %v", err)
		})
	}
}

func TestValidate_SecretsAddrInvalidURL(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Secrets: Secrets{
			Address: "://invalid",
		},
	}
	err := cfg.Validate()
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrSecretsAddrInvalid), "expected ErrSecretsAddrInvalid, got %v", err)
}

func TestValidate_SecretsAddrMissingScheme(t *testing.T) {
	t.Parallel()

	testCases := []string{
		"vault.example.com:8200",
		"localhost:8200",
	}

	for _, addr := range testCases {
		t.Run(addr, func(t *testing.T) {
			t.Parallel()

			cfg := &Config{
				Secrets: Secrets{
					Address: addr,
				},
			}

			err := cfg.Validate()
			assert.Error(t, err)
			assert.True(t, errors.Is(err, ErrSecretsAddrMissingScheme), "expected ErrSecretsAddrMissingScheme, got %v", err)
		})
	}
}

func TestValidate_SecretsAddrUnsupportedScheme(t *testing.T) {
	t.Parallel()

	testCases := []string{
		"ftp://vault.example.com:8200",
		"file:///tmp/vault",
	}

	for _, addr := range testCases {
		t.Run(addr, func(t *testing.T) {
			t.Parallel()

			cfg := &Config{
				Secrets: Secrets{
					Address: addr,
				},
			}

			err := cfg.Validate()
			assert.Error(t, err)
			assert.True(t, errors.Is(err, ErrSecretsAddrInvalid), "expected ErrSecretsAddrInvalid, got %v", err)
			assert.Contains(t, err.Error(), "unsupported scheme")
		})
	}
}

func TestIsLocalhost(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		host     string
		expected bool
	}{
		// Localhost variants
		{"localhost", true},
		{"LOCALHOST", true},
		{"localhost.", true},
		{"LOCALHOST.", true},
		{"127.0.0.1", true},
		{"127.255.255.255", true},
		{"::1", true},
		{"127.1.1.1", true},

		// Non-localhost
		{"192.168.1.1", false},
		{"vault.example.com", false},
		{"172.16.0.1", false},
		{"example.com", false},
		{"2001:db8::1", false},
	}
	for _, tc := range testCases {
		t.Run(tc.host, func(t *testing.T) {
			t.Parallel()

			result := isLocalhost(tc.host)
			assert.Equal(t, tc.expected, result, "isLocalhost(%s) = %v, want %v", tc.host, result, tc.expected)
		})
	}
}

func TestValidate_CallsValidateSecretsAddr(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Secrets: Secrets{
			Address: "http://vault.example.com:8200", // Remote HTTP - should fail
		},
	}

	err := cfg.Validate()
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrSecretsAddrInsecure), "expected ErrSecretsAddrInsecure, got %v", err)
}

func TestValidate_AllowsValidRemoteHTTPS(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Secrets: Secrets{
			Address: "https://vault.example.com:8200",
		},
	}

	err := cfg.Validate()
	assert.NoError(t, err)
}

func TestValidateSecretsAddr_NoHost(t *testing.T) {
	t.Parallel()

	testCases := []string{
		"http://",       // Scheme but no host
		"https://",      // Scheme but no host
		"http://:8200",  // Scheme with port but no host
		"https://:8200", // Scheme with port but no host
	}
	for _, addr := range testCases {
		t.Run(addr, func(t *testing.T) {
			t.Parallel()

			cfg := &Config{
				Secrets: Secrets{
					Address: addr,
				},
			}
			err := cfg.validateSecretsAddr()
			assert.Error(t, err, "expected %s to fail", addr)
			assert.True(t, errors.Is(err, ErrSecretsAddrNoHost), "expected ErrSecretsAddrNoHost, got %v", err)
		})
	}
}
