package config

import (
	"os"
	"path/filepath"
	"runtime"
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

	assert.NoError(t, err)
	// require (not assert) so the dereferences below can't run on a nil cfg.
	require.NotNil(t, cfg)

	cfg.EncryptionKey = "test" // Added to pass the test

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
	assert.Equal(t, filepath.Join(base, "device-management-toolkit", "config", "config.yml"), got)
}

func TestResolveConfigPath_NonTrayPrefersMachineThenBesideBinary(t *testing.T) { //nolint:paralleltest // mutates package-global TrayMode
	orig := TrayMode
	TrayMode = false

	defer func() { TrayMode = orig }()

	got, err := resolveConfigPath("")
	assert.NoError(t, err)

	// With an installer-provisioned machine config present, that path wins;
	// without one (the usual dev/test case) it falls back to beside-binary.
	machine, err := machineConfigPath()
	assert.NoError(t, err)

	if _, statErr := os.Stat(machine); statErr == nil {
		assert.Equal(t, machine, got)
	} else {
		want, beErr := besideBinaryConfigPath()
		assert.NoError(t, beErr)
		assert.Equal(t, want, got)
	}
}

func TestMachineConfigPath_PerGOOS(t *testing.T) { //nolint:paralleltest // reads ProgramData env on Windows
	got, err := machineConfigPath()
	assert.NoError(t, err)

	switch runtime.GOOS {
	case goosWindows:
		if pd := os.Getenv("ProgramData"); pd != "" {
			assert.Equal(t, filepath.Join(pd, "device-management-toolkit", "config.yml"), got)
		}
	case "darwin":
		assert.Equal(t, "/Library/Application Support/device-management-toolkit/config.yml", got)
	case "linux":
		assert.Equal(t, "/etc/dmt-console/config/config.yml", got)
	default:
		want, beErr := besideBinaryConfigPath()
		assert.NoError(t, beErr)
		assert.Equal(t, want, got)
	}
}

func TestMachineConfigPath_WindowsProgramDataUnsetFallsBack(t *testing.T) { //nolint:paralleltest // mutates ProgramData env
	if runtime.GOOS != goosWindows {
		t.Skip("ProgramData fallback only applies on Windows")
	}

	if orig, had := os.LookupEnv("ProgramData"); had {
		os.Unsetenv("ProgramData")

		defer os.Setenv("ProgramData", orig)
	}

	got, err := machineConfigPath()
	assert.NoError(t, err)

	want, err := besideBinaryConfigPath()
	assert.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestResolveConfigPath_TrayFallsBackToHeadlessResolution(t *testing.T) { //nolint:paralleltest // mutates TrayMode and user-config env
	orig := TrayMode
	TrayMode = true

	defer func() { TrayMode = orig }()

	// Force os.UserConfigDir to fail so perUserConfigPath errors and we fall through.
	saved := map[string]string{}

	for _, key := range []string{"XDG_CONFIG_HOME", "HOME", "AppData"} {
		if v, ok := os.LookupEnv(key); ok {
			saved[key] = v

			os.Unsetenv(key)
		}
	}

	defer func() {
		for key, v := range saved {
			os.Setenv(key, v)
		}
	}()

	if _, err := os.UserConfigDir(); err == nil {
		t.Skip("os.UserConfigDir still resolves; cannot exercise the fallback on this platform")
	}

	got, err := resolveConfigPath("")
	assert.NoError(t, err)

	// Fallthrough lands in the headless resolution: machine config when present,
	// otherwise the writable beside-binary path.
	machine, err := machineConfigPath()
	assert.NoError(t, err)

	if _, statErr := os.Stat(machine); statErr == nil {
		assert.Equal(t, machine, got)
	} else {
		want, beErr := besideBinaryConfigPath()
		assert.NoError(t, beErr)
		assert.Equal(t, want, got)
	}
}

func TestSeedConfig_StatErrorIsSurfaced(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == goosWindows {
		t.Skip("stat error semantics for a non-directory parent differ on Windows")
	}

	dir := t.TempDir()

	// A regular file as dst's parent makes os.Stat(dst) fail with ENOTDIR — not IsNotExist.
	notDir := filepath.Join(dir, "file")
	assert.NoError(t, os.WriteFile(notDir, []byte("x"), 0o600))
	dst := filepath.Join(notDir, "config.yml")

	src := filepath.Join(dir, "src.yml")
	assert.NoError(t, os.WriteFile(src, []byte("app:\n  name: seeded\n"), 0o600))

	err := seedConfig(src, dst)
	assert.Error(t, err)
	assert.False(t, os.IsNotExist(err))
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

	// The seeded file holds credentials — it must be owner-only, not world-readable.
	if runtime.GOOS != goosWindows {
		info, statErr := os.Stat(dst)
		assert.NoError(t, statErr)
		assert.Equal(t, os.FileMode(configFilePerm), info.Mode().Perm())
	}

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

func TestWriteConfig_OwnerOnlyPermissions(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == goosWindows {
		t.Skip("POSIX permission bits are not meaningful on Windows")
	}

	dir := t.TempDir()
	configPath := filepath.Join(dir, "device-management-toolkit", "config.yml")

	// A freshly generated config carries secrets (generated jwtKey/adminPassword),
	// so the generate path must write owner-only just like the seed path.
	assert.NoError(t, writeConfig(configPath, defaultConfig()))

	fileInfo, err := os.Stat(configPath)
	assert.NoError(t, err)
	assert.Equal(t, os.FileMode(configFilePerm), fileInfo.Mode().Perm())

	dirInfo, err := os.Stat(filepath.Dir(configPath))
	assert.NoError(t, err)
	assert.Equal(t, os.FileMode(configDirPerm), dirInfo.Mode().Perm())
}

func TestWriteConfig_TightensExistingLooseFile(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == goosWindows {
		t.Skip("POSIX permission bits are not meaningful on Windows")
	}

	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yml")

	// Simulate a config left world-readable by an older build. O_CREATE alone
	// would not retighten it, so writeConfig must chmod the existing file.
	assert.NoError(t, os.WriteFile(configPath, []byte("app:\n  name: old\n"), 0o644))

	assert.NoError(t, writeConfig(configPath, defaultConfig()))

	fileInfo, err := os.Stat(configPath)
	assert.NoError(t, err)
	assert.Equal(t, os.FileMode(configFilePerm), fileInfo.Mode().Perm())
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
