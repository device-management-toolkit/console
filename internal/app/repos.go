package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/internal/usecase"
	mongodb "github.com/device-management-toolkit/console/internal/usecase/nosqldb/mongo"
	"github.com/device-management-toolkit/console/pkg/db"
	"github.com/device-management-toolkit/console/pkg/logger"
)

// Supported values for config.DB.Provider.
// Empty defaults to sqlite so `go run` with no config still works.
const (
	ProviderPostgres = "postgres"
	ProviderSQLite   = "sqlite"
	ProviderMongo    = "mongo"
)

var (
	errUnsupportedProvider = errors.New("unsupported db.provider")
	errDBURLRequired       = errors.New("DB_URL required")
)

// buildRepos dials the configured database and returns the repo bundle the
// use cases consume. Unknown providers are rejected up front so typos fail
// at startup instead of silently falling back.
func buildRepos(cfg *config.Config, log logger.Interface) (*usecase.Repos, error) {
	switch cfg.Provider {
	case ProviderPostgres:
		return buildPostgresRepos(cfg, log)
	case ProviderMongo:
		return buildMongoRepos(cfg, log)
	case ProviderSQLite, "":
		return buildSQLiteRepos(cfg, log)
	default:
		return nil, fmt.Errorf("app.buildRepos: %w %q (want %q, %q, or %q)",
			errUnsupportedProvider, cfg.Provider, ProviderPostgres, ProviderSQLite, ProviderMongo)
	}
}

func buildPostgresRepos(cfg *config.Config, log logger.Interface) (*usecase.Repos, error) {
	if cfg.DB.URL == "" {
		return nil, fmt.Errorf("app.buildPostgresRepos: %w for provider %q", errDBURLRequired, ProviderPostgres)
	}

	database, err := db.New(cfg.DB.URL, sql.Open, db.MaxPoolSize(cfg.PoolMax), db.EnableForeignKeys(true))
	if err != nil {
		return nil, fmt.Errorf("app.buildPostgresRepos: %w", err)
	}

	return usecase.NewSQLRepos(database, log), nil
}

func buildSQLiteRepos(cfg *config.Config, log logger.Interface) (*usecase.Repos, error) {
	// Embedded SQLite ignores any URL — db.New routes to the on-disk path
	// when the URL is empty.
	database, err := db.New("", sql.Open, db.MaxPoolSize(cfg.PoolMax), db.EnableForeignKeys(true))
	if err != nil {
		return nil, fmt.Errorf("app.buildSQLiteRepos: %w", err)
	}

	return usecase.NewSQLRepos(database, log), nil
}

func buildMongoRepos(cfg *config.Config, log logger.Interface) (*usecase.Repos, error) {
	if cfg.DB.URL == "" {
		return nil, fmt.Errorf("app.buildMongoRepos: %w for provider %q", errDBURLRequired, ProviderMongo)
	}

	client, database, err := mongodb.Connect(context.Background(), cfg.DB.URL, log)
	if err != nil {
		return nil, fmt.Errorf("app.buildMongoRepos: %w", err)
	}

	return usecase.NewMongoRepos(client, database, log), nil
}
