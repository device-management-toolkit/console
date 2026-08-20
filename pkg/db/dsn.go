package db

import (
	"net/url"
	"strings"
)

// PostgresPrefix is the DSN scheme that routes to the hosted Postgres path.
const PostgresPrefix = "postgres://"

// MigrationDSN defaults sslmode to "disable" when a Postgres DSN doesn't set
// one, so the migration driver (lib/pq, which defaults to "require") can reach
// the local/compose database. The runtime pool keeps pgx's own "prefer"
// default. DSNs that are non-Postgres, malformed, or carry an unparseable
// query are returned untouched.
func MigrationDSN(dsn string) string {
	if !strings.HasPrefix(dsn, PostgresPrefix) {
		return dsn
	}

	parsed, err := url.Parse(dsn)
	if err != nil {
		return dsn
	}

	query, err := url.ParseQuery(parsed.RawQuery)
	if err != nil {
		return dsn
	}

	if query.Has("sslmode") {
		return dsn
	}

	if parsed.RawQuery == "" {
		parsed.RawQuery = "sslmode=disable"
	} else {
		parsed.RawQuery += "&sslmode=disable"
	}

	return parsed.String()
}
