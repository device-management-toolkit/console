package config

import (
	"os"
	"path/filepath"
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

func TestResolveConfigPath_FlagWins(t *testing.T) { //nolint:paralleltest // mutates package-global TrayMode
	orig := TrayMode
	TrayMode = true

	defer func() { TrayMode = orig }()

	got, err := resolveConfigPath("/custom/path.yml")
	assert.NoError(t, err)
	assert.Equal(t, "/custom/path.yml", got)
}

func TestResolveConfigPath_TrayUsesPerUserDir(t *testing.T) { //nolint:paralleltest // mutates package-global TrayMode
	orig := TrayMode
	TrayMode = true

	defer func() { TrayMode = orig }()

	got, err := resolveConfigPath("")
	assert.NoError(t, err)

	base, err := os.UserConfigDir()
	assert.NoError(t, err)
	assert.Equal(t, filepath.Join(base, "device-management-toolkit", "config.yml"), got)
}

func TestResolveConfigPath_NonTrayUsesBesideBinary(t *testing.T) { //nolint:paralleltest // mutates package-global TrayMode
	orig := TrayMode
	TrayMode = false

	defer func() { TrayMode = orig }()

	got, err := resolveConfigPath("")
	assert.NoError(t, err)

	want, err := besideBinaryConfigPath()
	assert.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestSeedConfig(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	src := filepath.Join(dir, "install", "config.yml")
	dst := filepath.Join(dir, "user", "config.yml")

	assert.NoError(t, os.MkdirAll(filepath.Dir(src), 0o755))
	assert.NoError(t, os.WriteFile(src, []byte("app:\n  name: seeded\n"), 0o600))

	// Copies installer config when the per-user file is missing.
	assert.NoError(t, seedConfig(src, dst))
	data, err := os.ReadFile(dst)
	assert.NoError(t, err)
	assert.Contains(t, string(data), "seeded")

	// Does not overwrite an existing per-user file.
	assert.NoError(t, os.WriteFile(dst, []byte("app:\n  name: existing\n"), 0o600))
	assert.NoError(t, seedConfig(src, dst))
	data, err = os.ReadFile(dst)
	assert.NoError(t, err)
	assert.Contains(t, string(data), "existing")

	// Missing src is a no-op, not an error.
	assert.NoError(t, seedConfig(filepath.Join(dir, "nope.yml"), filepath.Join(dir, "out.yml")))
	_, statErr := os.Stat(filepath.Join(dir, "out.yml"))
	assert.True(t, os.IsNotExist(statErr))
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
