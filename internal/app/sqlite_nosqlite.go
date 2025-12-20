//go:build nosqlite

package app

// SQLite driver and migration support are NOT imported when building with nosqlite tag
// Only PostgreSQL is supported in this build
