//go:build nosqlite

package app

import (
	"errors"

	"github.com/golang-migrate/migrate/v4/source"
)

func setupLocalDB(migrationsSource source.Driver) error {
	return errors.New("SQLite support not included in this build - use PostgreSQL with DB_URL environment variable")
}
