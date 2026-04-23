package consul

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/device-management-toolkit/console/config"
)

type fakeService struct {
	healthErr error
	values    map[string][]byte
	getErr    error
	seedErr   error
	seeded    *config.Config
	seedCalls int
}

func (f *fakeService) Health(_ context.Context, _ string) error {
	return f.healthErr
}

func (f *fakeService) Get(_ context.Context, _ string) (map[string][]byte, error) {
	return f.values, f.getErr
}

func (f *fakeService) Seed(_ context.Context, _ string, cfg *config.Config) error {
	f.seedCalls++
	f.seeded = cfg

	return f.seedErr
}

func (f *fakeService) Process(values map[string][]byte, cfg *config.Config) error {
	live := &Service{}

	return live.Process(values, cfg)
}

func baseConfig() *config.Config {
	cfg := &config.Config{}
	cfg.Consul.KeyPrefix = "console"
	cfg.Host = "localhost"
	cfg.Port = "8181"
	cfg.Level = "info"

	return cfg
}

func TestProcessServiceConfigs_SeedsWhenKVEmpty(t *testing.T) {
	t.Parallel()

	fake := &fakeService{values: nil}
	cfg := baseConfig()

	err := ProcessServiceConfigs(context.Background(), fake, cfg)
	require.NoError(t, err)
	require.Equal(t, 1, fake.seedCalls)
	require.Equal(t, cfg, fake.seeded)
}

func TestProcessServiceConfigs_OverlaysWhenKVPresent(t *testing.T) {
	t.Parallel()

	stored := &config.Config{}
	stored.Host = "overlay-host"
	stored.Port = "9999"
	stored.Level = "debug"

	blob, err := json.Marshal(stored)
	require.NoError(t, err)

	fake := &fakeService{values: map[string][]byte{"console/config": blob}}
	cfg := baseConfig()

	err = ProcessServiceConfigs(context.Background(), fake, cfg)
	require.NoError(t, err)
	require.Equal(t, 0, fake.seedCalls)
	require.Equal(t, "overlay-host", cfg.Host)
	require.Equal(t, "9999", cfg.Port)
	require.Equal(t, "debug", cfg.Level)
}

func TestProcessServiceConfigs_WrapsGetError(t *testing.T) {
	t.Parallel()

	fake := &fakeService{getErr: errors.New("boom")}
	cfg := baseConfig()

	err := ProcessServiceConfigs(context.Background(), fake, cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "boom")
}

func TestService_Process_IgnoresEmptyValues(t *testing.T) {
	t.Parallel()

	cfg := baseConfig()
	originalHost := cfg.Host

	s := &Service{}

	err := s.Process(map[string][]byte{"console/empty": {}}, cfg)
	require.NoError(t, err)
	require.Equal(t, originalHost, cfg.Host)
}

func TestService_Process_ReturnsUnmarshalError(t *testing.T) {
	t.Parallel()

	cfg := baseConfig()
	s := &Service{}

	err := s.Process(map[string][]byte{"console/bad": []byte("not-json")}, cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unmarshal config")
}

func TestSeedBlob_ExcludesSecrets(t *testing.T) {
	t.Parallel()

	cfg := baseConfig()
	cfg.App.EncryptionKey = "encryption-secret"
	cfg.Secrets.Token = "vault-token-secret"
	cfg.Auth.AdminPassword = "admin-password-secret"
	cfg.Auth.JWTKey = "jwt-key-secret"
	cfg.DB.URL = "postgres://user:password@db:5432/x"
	cfg.EA.Password = "ea-password-secret"

	blob, err := json.Marshal(cfg)
	require.NoError(t, err)

	body := string(blob)
	require.NotContains(t, body, "encryption-secret")
	require.NotContains(t, body, "vault-token-secret")
	require.NotContains(t, body, "admin-password-secret")
	require.NotContains(t, body, "jwt-key-secret")
	require.NotContains(t, body, "password@db")
	require.NotContains(t, body, "ea-password-secret")
}

func TestNewService_RejectsEmptyHostOrPort(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name, host, port string
	}{
		{"empty host", "", "8500"},
		{"empty port", "consul", ""},
		{"both empty", "", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc, err := NewService(tc.host, tc.port)
			require.Nil(t, svc)
			require.Error(t, err)
			require.Contains(t, err.Error(), "host and port are required")
		})
	}
}

func TestOverlay_DoesNotClobberSecrets(t *testing.T) {
	t.Parallel()

	cfg := baseConfig()
	cfg.App.EncryptionKey = "from-env"
	cfg.Auth.JWTKey = "from-env"
	cfg.Secrets.Token = "from-env"

	stored := baseConfig()
	stored.Host = "overlay-host"

	blob, err := json.Marshal(stored)
	require.NoError(t, err)

	s := &Service{}
	err = s.Process(map[string][]byte{"console/config": blob}, cfg)
	require.NoError(t, err)

	require.Equal(t, "overlay-host", cfg.Host)
	require.Equal(t, "from-env", cfg.App.EncryptionKey)
	require.Equal(t, "from-env", cfg.Auth.JWTKey)
	require.Equal(t, "from-env", cfg.Secrets.Token)
}
