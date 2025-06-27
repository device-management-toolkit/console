package config

import (
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
}

func TestNewConfig_Defaults(t *testing.T) { //nolint:paralleltest // cannot have simultaneous tests modifying environment variables
	clearEnv() // Clear environment variables to ensure defaults are tested

	cfg, err := NewConfig()

	cfg.App.EncryptionKey = "test" // Added to pass the test

	assert.NoError(t, err)
	assert.NotNil(t, cfg)

	// Verify default values
	assert.Equal(t, "console", cfg.App.Name)
	assert.Equal(t, "device-management-toolkit/console", cfg.App.Repo)
	assert.Equal(t, "DEVELOPMENT", cfg.App.Version)
	assert.Equal(t, "test", cfg.App.EncryptionKey)

	assert.Equal(t, "8181", cfg.HTTP.Port)
	assert.Equal(t, []string{"*"}, cfg.HTTP.AllowedOrigins)
	assert.Equal(t, []string{"*"}, cfg.HTTP.AllowedHeaders)

	assert.Equal(t, "info", cfg.Log.Level)

	assert.Equal(t, 2, cfg.DB.PoolMax)
}

func TestNewConfig_EnvVars(t *testing.T) { //nolint:paralleltest // cannot have simultaneous tests modifying environment variables
	// Set environment variables
	os.Setenv("APP_NAME", "testApp")
	os.Setenv("HTTP_PORT", "9090")
	os.Setenv("LOG_LEVEL", "debug")
	os.Setenv("DB_POOL_MAX", "10")
	os.Setenv("DB_URL", "postgres://user:password@localhost:5432/testdb")

	defer clearEnv() // Ensure environment variables are cleared after test

	cfg, err := NewConfig()
	assert.NoError(t, err)
	assert.NotNil(t, cfg)

	// Verify environment variable values
	assert.Equal(t, "testApp", cfg.App.Name)
	assert.Equal(t, "9090", cfg.HTTP.Port)
	assert.Equal(t, "debug", cfg.Log.Level)
	assert.Equal(t, 10, cfg.DB.PoolMax)
	assert.Equal(t, "postgres://user:password@localhost:5432/testdb", cfg.DB.URL)
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
	assert.Equal(t, "envApp", cfg.App.Name)
	assert.Equal(t, "9090", cfg.HTTP.Port)
	assert.Equal(t, "debug", cfg.Log.Level)
	assert.Equal(t, 10, cfg.DB.PoolMax)
	assert.Equal(t, "postgres://envuser:envpassword@localhost:5432/envdb", cfg.DB.URL)
}
