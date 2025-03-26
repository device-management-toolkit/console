package app

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // postgres driver
	dbdbdb "github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source"
	_ "github.com/golang-migrate/migrate/v4/source/file" // for file source
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "modernc.org/sqlite" // sqlite3 driver

	"github.com/open-amt-cloud-toolkit/console/config"
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

func setupLocalDB(migrationsSource source.Driver) error {
	dirname, err := os.UserConfigDir()
	if err != nil {
		return err
	}

	consoleDir := filepath.Join(dirname, "device-management-toolkit")

	if _, err = os.Stat(consoleDir); os.IsNotExist(err) {
		if err1 := os.Mkdir(consoleDir, _directoryPermission); err1 != nil {
			return err1
		}
	}

	log.Printf("DB path : %s\n", filepath.Join(consoleDir, "console.db"))

	db, err := sql.Open("sqlite", filepath.Join(consoleDir, "console.db"))
	if err != nil {
		return err
	}

	defer func() {
		if err1 := db.Close(); err1 != nil {
			return
		}
	}()

	driver, err := dbdbdb.WithInstance(db, &dbdbdb.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance("iofs", migrationsSource, "console", driver)
	if err != nil {
		return err
	}

	err = m.Up()
	if err != nil {
		if !errors.Is(err, migrate.ErrNoChange) {
			return err
		}
	}

	_, err = db.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		return err
	}

	defer m.Close()

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
