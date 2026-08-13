package db

import (
	"net/url"
	"strings"
)

// PostgresPrefix is the DSN scheme that routes to the hosted Postgres path.
const PostgresPrefix = "postgres://"

// NormalizeDSN defaults sslmode to "disable" only when a Postgres DSN doesn't
// set one, so the migration driver (lib/pq, which defaults to require) and the
// runtime pool (pgx, which defaults to prefer) resolve the same URL to the same
// TLS behavior. Non-Postgres DSNs are returned untouched.
func NormalizeDSN(dsn string) string {
	if !strings.HasPrefix(dsn, PostgresPrefix) {
		return dsn
	}

	parsed, err := url.Parse(dsn)
	if err != nil {
		return dsn // let the driver report the malformed DSN
	}

	query := parsed.Query()
	if query.Has("sslmode") {
		return dsn
	}

	query.Set("sslmode", "disable")
	parsed.RawQuery = query.Encode()

	return parsed.String()
}
