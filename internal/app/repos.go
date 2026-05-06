package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/internal/usecase"
	mongodb "github.com/device-management-toolkit/console/internal/usecase/nosqldb/mongo"
	"github.com/device-management-toolkit/console/pkg/db"
	"github.com/device-management-toolkit/console/pkg/logger"
)

// mongoStartupTimeout caps connect+ping+index-ensure so an unreachable Mongo
// fails fast instead of waiting on the driver's default server-selection timeout.
const mongoStartupTimeout = 30 * time.Second

// mongoShutdownTimeout caps how long the Mongo client gets to disconnect
// gracefully. Without it, an unreachable server would block the process
// from exiting indefinitely.
const mongoShutdownTimeout = 5 * time.Second

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

// buildMongoRepos lives in the app layer (next to its driver dialer) so the
// usecase package stays free of concrete storage-driver imports.
func buildMongoRepos(cfg *config.Config, log logger.Interface) (*usecase.Repos, error) {
	if cfg.DB.URL == "" {
		return nil, fmt.Errorf("app.buildMongoRepos: %w for provider %q", errDBURLRequired, ProviderMongo)
	}

	startupCtx, cancel := context.WithTimeout(context.Background(), mongoStartupTimeout)
	defer cancel()

	client, database, err := mongodb.Connect(startupCtx, cfg.DB.URL, log)
	if err != nil {
		return nil, fmt.Errorf("app.buildMongoRepos: %w", err)
	}

	return &usecase.Repos{
		Devices:            mongodb.NewDeviceRepo(database),
		Domains:            mongodb.NewDomainRepo(database),
		Profiles:           mongodb.NewProfileRepo(database, log),
		ProfileWiFiConfigs: mongodb.NewProfileWiFiConfigsRepo(database),
		IEEE8021xConfigs:   mongodb.NewIEEE8021xRepo(database),
		CIRAConfigs:        mongodb.NewCIRARepo(database),
		WirelessConfigs:    mongodb.NewWirelessRepo(database, log),
		Closer: usecase.CloserFunc(func() error {
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), mongoShutdownTimeout)
			defer shutdownCancel()

			return client.Disconnect(shutdownCtx)
		}),
	}, nil
}
