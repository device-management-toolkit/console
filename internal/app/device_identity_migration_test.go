package app

import (
	"database/sql"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/stretchr/testify/require"
)

func TestDeviceIdentityMigrationFromMain(t *testing.T) {
	t.Parallel()

	pool, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = pool.Close() })

	driver, err := sqlite.WithInstance(pool, &sqlite.Config{})
	require.NoError(t, err)
	source, err := iofs.New(content, "migrations")
	require.NoError(t, err)
	m, err := migrate.NewWithInstance("iofs", source, "sqlite", driver)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = m.Close() })

	// Simulate an installation already running main's newest migration.
	require.NoError(t, m.Migrate(20260820000000))

	_, err = pool.Exec("INSERT INTO devices (guid, tenantid, connectionstatus, usetls, allowselfsigned) VALUES ('legacy', 'tenant', FALSE, FALSE, FALSE)")
	require.NoError(t, err)
	require.NoError(t, m.Up())

	var id, created, updated, deleted, product, connection string

	var isDeleted bool

	err = pool.QueryRow("SELECT id, createddate, lastupdate, isdeleted, deleteddate, producttype, connectiontype FROM devices WHERE guid = 'legacy'").
		Scan(&id, &created, &updated, &isDeleted, &deleted, &product, &connection)
	require.NoError(t, err)
	require.Empty(t, id)
	require.Empty(t, created)
	require.Empty(t, updated)
	require.False(t, isDeleted)
	require.Empty(t, deleted)
	require.Empty(t, product)
	require.Empty(t, connection)

	// Rollback preserves existing devices and can be followed by another upgrade.
	require.NoError(t, m.Steps(-1))

	var guid string
	require.NoError(t, pool.QueryRow("SELECT guid FROM devices").Scan(&guid))
	require.Equal(t, "legacy", guid)
	require.NoError(t, m.Up())
}
