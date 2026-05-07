package app

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/pkg/logger"
)

// Unit tests cover the pure-logic error paths in buildRepos — provider typos
// and empty DB_URL — so a regression in the dispatch table or error wrapping
// fails at `go test`, not at a CI integration run. Real-backend happy paths
// are covered end-to-end by .github/workflows/api-test.yml.

func TestBuildRepos_UnsupportedProvider(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{}
	cfg.Provider = "redis"

	_, err := buildRepos(cfg, logger.New("error"))
	require.Error(t, err)
	require.True(t, errors.Is(err, errUnsupportedProvider),
		"want errUnsupportedProvider in chain, got %v", err)
}

func TestBuildRepos_EmptyURL(t *testing.T) {
	t.Parallel()

	for _, provider := range []string{ProviderPostgres, ProviderMongo} {
		provider := provider

		t.Run(provider, func(t *testing.T) {
			t.Parallel()

			cfg := &config.Config{}
			cfg.Provider = provider

			_, err := buildRepos(cfg, logger.New("error"))
			require.Error(t, err)
			require.True(t, errors.Is(err, errDBURLRequired),
				"provider %q: want errDBURLRequired in chain, got %v", provider, err)
		})
	}
}
