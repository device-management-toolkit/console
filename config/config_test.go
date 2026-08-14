package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func clearEnv() {
	os.Unsetenv("APP_NAME")
	os.Unsetenv("HTTP_PORT")
	os.Unsetenv("LOG_LEVEL")
	os.Unsetenv("DB_POOL_MAX")
	os.Unsetenv("DB_URL")
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

func TestValidatePort(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		port    string
		wantErr error
	}{
		{name: "valid port", port: "8181", wantErr: nil},
		{name: "port 1", port: "1", wantErr: nil},
		{name: "port 65535", port: "65535", wantErr: nil},
		{name: "port 0", port: "0", wantErr: ErrPortOutOfRange},
		{name: "port 65536", port: "65536", wantErr: ErrPortOutOfRange},
		{name: "negative", port: "-1", wantErr: ErrPortOutOfRange},
		{name: "non-numeric", port: "abc", wantErr: ErrPortNotNumeric},
		{name: "with metacharacter", port: "8181&calc", wantErr: ErrPortNotNumeric},
		{name: "empty string", port: "", wantErr: ErrPortNotNumeric},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validatePort(tt.port)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestNewConfig_InvalidPort(t *testing.T) { //nolint:paralleltest // cannot have simultaneous tests modifying environment variables
	clearEnv()
	os.Setenv("HTTP_PORT", "not-a-port")

	defer clearEnv()

	_, err := NewConfig()
	require.ErrorIs(t, err, ErrPortNotNumeric)
}

func TestNewConfig_PortOutOfRange(t *testing.T) { //nolint:paralleltest // cannot have simultaneous tests modifying environment variables
	clearEnv()
	os.Setenv("HTTP_PORT", "70000")

	defer clearEnv()

	_, err := NewConfig()
	require.ErrorIs(t, err, ErrPortOutOfRange)
}
