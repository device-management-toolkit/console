package app

import (
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite" // sqlite3 driver
)

func TestMigrateDiscoveredActivatedToManaged(t *testing.T) {
	t.Parallel()

	script, err := content.ReadFile("migrations/20261119000000_migrate_discovered_activated_to_managed.up.sql")
	require.NoError(t, err)

	sqlDB, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	defer sqlDB.Close()

	_, err = sqlDB.Exec(`CREATE TABLE devices (guid TEXT, deviceinfo TEXT, currentmode TEXT, discovered BOOLEAN)`)
	require.NoError(t, err)

	rows := []struct {
		guid, info, mode string
		discovered       any
	}{
		{"acm", `{"currentMode":"admin control mode","discovered":true}`, "admin control mode", true},
		{"ccm", `{"currentMode":"client control mode","discovered":true}`, "client control mode", true},
		{"preprov", `{"currentMode":"Not Activated","discovered":true}`, "Not Activated", true},
		{"nomode", `{"discovered":true}`, "", true},
		{"managed", `{"currentMode":"admin control mode","discovered":false}`, "admin control mode", false},
		{"legacy", `{"currentMode":"admin control mode"}`, "admin control mode", nil},
	}
	for _, r := range rows {
		_, err = sqlDB.Exec(`INSERT INTO devices VALUES (?, ?, ?, ?)`, r.guid, r.info, r.mode, r.discovered)
		require.NoError(t, err)
	}

	_, err = sqlDB.Exec(string(script))
	require.NoError(t, err)

	want := map[string]bool{"acm": false, "ccm": false, "preprov": true, "nomode": true, "managed": false}

	for guid, discovered := range want {
		var (
			col  bool
			blob string
		)

		require.NoError(t, sqlDB.QueryRow(`SELECT discovered, deviceinfo FROM devices WHERE guid = ?`, guid).Scan(&col, &blob), guid)
		require.Equal(t, discovered, col, guid)

		var info struct {
			Discovered *bool `json:"discovered"`
		}

		require.NoError(t, json.Unmarshal([]byte(blob), &info), guid)
		require.NotNil(t, info.Discovered, guid)
		require.Equal(t, discovered, *info.Discovered, guid)
	}

	var legacy sql.NullBool

	require.NoError(t, sqlDB.QueryRow(`SELECT discovered FROM devices WHERE guid = 'legacy'`).Scan(&legacy))
	require.False(t, legacy.Valid)
}
