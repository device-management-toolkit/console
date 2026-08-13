package db

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeDSN(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "no query string defaults to disable",
			url:      "postgres://user:pass@localhost:5432/rpsdb",
			expected: "postgres://user:pass@localhost:5432/rpsdb?sslmode=disable",
		},
		{
			name:     "explicit sslmode is preserved",
			url:      "postgres://user:pass@localhost:5432/rpsdb?sslmode=verify-full",
			expected: "postgres://user:pass@localhost:5432/rpsdb?sslmode=verify-full",
		},
		{
			name:     "explicit sslmode with sslrootcert is preserved",
			url:      "postgres://user:pass@db.example.com:5432/rpsdb?sslmode=verify-full&sslrootcert=%2Fetc%2Fssl%2Fca.pem",
			expected: "postgres://user:pass@db.example.com:5432/rpsdb?sslmode=verify-full&sslrootcert=%2Fetc%2Fssl%2Fca.pem",
		},
		{
			name:     "existing query without sslmode keeps its params",
			url:      "postgres://user:pass@localhost:5432/rpsdb?connect_timeout=5",
			expected: "postgres://user:pass@localhost:5432/rpsdb?connect_timeout=5&sslmode=disable",
		},
		{
			name:     "malformed url is passed through untouched",
			url:      "postgres://user:pass@local host:5432/rpsdb",
			expected: "postgres://user:pass@local host:5432/rpsdb",
		},
		{
			name:     "non-postgres dsn is passed through untouched",
			url:      "mongodb://mongoadmin:admin123@localhost:27017/?authSource=admin",
			expected: "mongodb://mongoadmin:admin123@localhost:27017/?authSource=admin",
		},
		{
			name:     "empty dsn is passed through untouched",
			url:      "",
			expected: "",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.expected, NormalizeDSN(tt.url))
		})
	}
}
