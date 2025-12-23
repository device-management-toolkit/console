package app

import (
	"embed"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // postgres driver
	"github.com/golang-migrate/migrate/v4/source"
	_ "github.com/golang-migrate/migrate/v4/source/file" // for file source
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"github.com/device-management-toolkit/console/config"
)

const (
	_defaultAttempts     = 20
	_defaultTimeout      = time.Second
	_directoryPermission = 0o755
)

//go:embed all:migrations
var content embed.FS

var errMigrate = errors.New("migrate error")

func MigrationError(op string) error {
	return fmt.Errorf("%w: %s", errMigrate, op)
}

func Init(cfg *config.Config) error {
	databaseURL := cfg.DB.URL
	if databaseURL == "" {
		log.Printf("migrate: environment variable not declared: DB_URL -- using embedded database")
	}

	migrationsSource, err := iofs.New(content, "migrations")
	if err != nil {
		log.Fatal(err)
	}

	if strings.HasPrefix(databaseURL, "postgres://") {
		err := setupHostedDB(migrationsSource, databaseURL)
		if err != nil {
			return err
		}
	} else {
		// make sure the directory exists
		err := setupLocalDB(migrationsSource)
		if err != nil {
			return err
		}
	}

	return nil
}

func setupHostedDB(migrationsSource source.Driver, databaseURL string) error {
	databaseURL += "?sslmode=disable"

	var (
		attempts = _defaultAttempts
		err      error
		m        *migrate.Migrate
	)

	for attempts > 0 {
		m, err = migrate.NewWithSourceInstance("iofs", migrationsSource, databaseURL)
		if err == nil {
			break
		}

		log.Printf("Migrate: postgres is trying to connect, attempts left: %d", attempts)
		time.Sleep(_defaultTimeout)

		attempts--
	}

	if err != nil {
		return MigrationError(fmt.Sprintf("postgres connect error: %s", err))
	}

	err = m.Up()
	defer m.Close()

	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return MigrationError(fmt.Sprintf("up error: %s", err))
	}

	if errors.Is(err, migrate.ErrNoChange) {
		log.Printf("Migrate: no change")

		return nil
	}

	log.Printf("Migrate: up success")

	return nil
}
