//go:build !nosqlite

package app

import (
	_ "github.com/golang-migrate/migrate/v4/database/sqlite" // sqlite migration driver
	_ "modernc.org/sqlite"                                   // sqlite3 driver
)

// SQLite driver and migration support is imported for database/sql
