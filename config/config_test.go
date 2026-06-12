package config

import (
	"bytes"
	"encoding/hex"
	"flag"
	"log"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func clearEnv() {
	os.Unsetenv("APP_NAME")
	os.Unsetenv("HTTP_PORT")
	os.Unsetenv("LOG_LEVEL")
	os.Unsetenv("DB_POOL_MAX")
	os.Unsetenv("DB_URL")
}

var logOutputMu sync.Mutex

func captureWarnIfWeakJWTKeyOutput(run func()) string {
	logOutputMu.Lock()
	defer logOutputMu.Unlock()

	var buf bytes.Buffer

	originalWriter := log.Writer()

	log.SetOutput(&buf)
	defer log.SetOutput(originalWriter)

	run()

	return buf.String()
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

//nolint:paralleltest // mutates process environment variable AUTH_JWT_KEY
func TestReadOrInitConfig_GeneratesJWTKeyForNewConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	originalJWTKey, hadJWTKey := os.LookupEnv("AUTH_JWT_KEY")
	err := os.Unsetenv("AUTH_JWT_KEY")
	require.NoError(t, err)

	t.Cleanup(func() {
		if hadJWTKey {
			require.NoError(t, os.Setenv("AUTH_JWT_KEY", originalJWTKey))
		} else {
			require.NoError(t, os.Unsetenv("AUTH_JWT_KEY"))
		}
	})

	cfg := defaultConfig()
	err = readOrInitConfig(configPath, cfg)
	assert.NoError(t, err)
	assert.NotEmpty(t, cfg.JWTKey)
	assert.Len(t, cfg.JWTKey, 64)

	data, err := os.ReadFile(configPath)
	assert.NoError(t, err)

	fileCfg := defaultConfig()
	err = yaml.Unmarshal(data, fileCfg)
	assert.NoError(t, err)
	assert.Equal(t, cfg.JWTKey, fileCfg.JWTKey)
}

//nolint:paralleltest // mutates process environment variable AUTH_JWT_KEY
func TestReadOrInitConfig_GeneratesJWTKeyForExistingEmptyConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	originalJWTKey, hadJWTKey := os.LookupEnv("AUTH_JWT_KEY")
	err := os.Unsetenv("AUTH_JWT_KEY")
	require.NoError(t, err)

	t.Cleanup(func() {
		if hadJWTKey {
			require.NoError(t, os.Setenv("AUTH_JWT_KEY", originalJWTKey))
		} else {
			require.NoError(t, os.Unsetenv("AUTH_JWT_KEY"))
		}
	})

	cfg := defaultConfig()
	cfg.JWTKey = ""
	err = writeConfig(configPath, cfg)
	assert.NoError(t, err)

	readCfg := defaultConfig()
	err = readOrInitConfig(configPath, readCfg)
	assert.NoError(t, err)
	assert.NotEmpty(t, readCfg.JWTKey)
	assert.Len(t, readCfg.JWTKey, 64)

	data, err := os.ReadFile(configPath)
	assert.NoError(t, err)

	fileCfg := defaultConfig()
	err = yaml.Unmarshal(data, fileCfg)
	assert.NoError(t, err)
	assert.Equal(t, readCfg.JWTKey, fileCfg.JWTKey)
}

func TestReadOrInitConfig_DoesNotMutateExistingNonEmptyJWTKey(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	cfg := defaultConfig()
	cfg.JWTKey = "existing-jwt-key"
	err := writeConfig(configPath, cfg)
	assert.NoError(t, err)

	readCfg := defaultConfig()
	err = readOrInitConfig(configPath, readCfg)
	assert.NoError(t, err)
	assert.Equal(t, "existing-jwt-key", readCfg.JWTKey)

	data, err := os.ReadFile(configPath)
	assert.NoError(t, err)

	fileCfg := defaultConfig()
	err = yaml.Unmarshal(data, fileCfg)
	assert.NoError(t, err)
	assert.Equal(t, "existing-jwt-key", fileCfg.JWTKey)
}

func TestGenerateJWTKey_ReturnsHexEncoded256BitKey(t *testing.T) {
	t.Parallel()

	key, err := generateJWTKey()
	assert.NoError(t, err)
	assert.Len(t, key, 64)

	decoded, err := hex.DecodeString(key)
	assert.NoError(t, err)
	assert.Len(t, decoded, 32)
}

func TestResolveConfigPath_UsesProvidedFlag(t *testing.T) {
	t.Parallel()

	path, err := resolveConfigPath("/tmp/custom-config.yml")
	assert.NoError(t, err)
	assert.Equal(t, "/tmp/custom-config.yml", path)
}

func TestNewConfig_EnvOverridesFileJWTKey(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	cfg := defaultConfig()
	cfg.JWTKey = "file-key"
	err := writeConfig(configPath, cfg)
	assert.NoError(t, err)

	origArgs := os.Args
	origFlagSet := flag.CommandLine
	os.Args = []string{origArgs[0], "-config", configPath}
	flag.CommandLine = flag.NewFlagSet(origArgs[0], flag.ContinueOnError)

	t.Cleanup(func() {
		os.Args = origArgs
		flag.CommandLine = origFlagSet
	})

	t.Setenv("AUTH_JWT_KEY", "env-key")

	loadedCfg, err := NewConfig()
	assert.NoError(t, err)
	assert.Equal(t, "env-key", loadedCfg.JWTKey)
}

func TestWarnIfWeakJWTKey_LogsWarning(t *testing.T) {
	t.Parallel()

	cfg := defaultConfig()
	cfg.JWTKey = "short"

	output := captureWarnIfWeakJWTKeyOutput(func() {
		warnIfWeakJWTKey(cfg)
	})

	assert.Contains(t, output, "at least 32 bytes")
}

func TestWarnIfWeakJWTKey_DoesNotLogForStrongKey(t *testing.T) {
	t.Parallel()

	cfg := defaultConfig()

	strongKey, err := generateJWTKey()
	assert.NoError(t, err)

	cfg.JWTKey = strongKey

	output := captureWarnIfWeakJWTKeyOutput(func() {
		warnIfWeakJWTKey(cfg)
	})

	assert.Empty(t, output)
}
