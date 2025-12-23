//go:build !nosqlite

package app

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"os"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	dbdbdb "github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source"
)

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

	_, err = db.ExecContext(context.Background(), "PRAGMA foreign_keys = ON")
	if err != nil {
		return err
	}

	defer m.Close()

	return nil
}
