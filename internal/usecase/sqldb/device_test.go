package sqldb_test

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"testing"

	"github.com/Masterminds/squirrel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/device-management-toolkit/console/internal/entity"
	"github.com/device-management-toolkit/console/internal/mocks"
	"github.com/device-management-toolkit/console/internal/repoerrors"
	"github.com/device-management-toolkit/console/internal/usecase/sqldb"
	"github.com/device-management-toolkit/console/pkg/db"
)

var (
	crthash  = "certhash"
	Certhash = &crthash
)

// setupDeviceTable creates an in-memory sqlite DB with the devices schema used in tests.
func setupDeviceTable(t *testing.T) *sql.DB {
	t.Helper()

	dbConn, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	_, err = dbConn.ExecContext(context.Background(), `
		CREATE TABLE devices (
			guid TEXT PRIMARY KEY,
			hostname TEXT NOT NULL DEFAULT '',
			tags TEXT NOT NULL DEFAULT '',
			mpsinstance TEXT NOT NULL DEFAULT '',
			connectionstatus BOOLEAN NOT NULL DEFAULT FALSE,
			mpsusername TEXT NOT NULL DEFAULT '',
			tenantid TEXT NOT NULL,
			friendlyname TEXT NOT NULL DEFAULT '',
			dnssuffix TEXT NOT NULL DEFAULT '',
			deviceinfo TEXT NOT NULL DEFAULT '',
			username TEXT NOT NULL DEFAULT '',
			password TEXT NOT NULL DEFAULT '',
			mpspassword TEXT,
			mebxpassword TEXT,
			usetls BOOLEAN NOT NULL DEFAULT FALSE,
			allowselfsigned BOOLEAN NOT NULL DEFAULT FALSE,
			certhash TEXT NOT NULL DEFAULT '',
			lastconnected TEXT,
			lastdisconnected TEXT,
			lastseen TEXT,
			islmsavailable BOOLEAN NOT NULL DEFAULT FALSE
		);
	`)
	require.NoError(t, err)

	return dbConn
}

// assertDeviceResults does a shallow check on device slice equality (len + type).
func assertDeviceResults(t *testing.T, expected, actual []entity.Device) {
	t.Helper()

	if len(actual) != len(expected) {
		t.Fatalf("Expected %d devices, got %d", len(expected), len(actual))
	}

	for i := range expected {
		assert.IsType(t, expected[i], actual[i], "Device at index %d type mismatch", i)
	}
}

func TestDeviceRepo_GetCount(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		setup    func(dbConn *sql.DB)
		tenantID string
		expected int
		err      error
	}{
		{
			name: "Successful count",
			setup: func(dbConn *sql.DB) {
				_, err := dbConn.ExecContext(context.Background(), `INSERT INTO devices (guid, hostname, tags, mpsinstance, connectionstatus, mpsusername, tenantid, friendlyname, dnssuffix, deviceinfo, username, password, usetls, allowselfsigned) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					"guid1", "hostname1", "tag1", "mpsinstance1", true, "mpsusername1", "tenant1", "friendlyname1", "dnssuffix1", "deviceinfo1", "username1", "password1", true, false)
				require.NoError(t, err)
			},
			tenantID: "tenant1",
			expected: 1,
			err:      nil,
		},
		{
			name:     "No devices found",
			setup:    func(_ *sql.DB) {},
			tenantID: "tenant2",
			expected: 0,
			err:      nil,
		},
		{
			name:     QueryExecutionErrorTestName,
			setup:    func(_ *sql.DB) {},
			tenantID: "tenant1",
			expected: 0,
			err:      &repoerrors.DatabaseError{},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dbConn := setupDeviceTable(t)
			defer dbConn.Close()

			tc.setup(dbConn)

			sqlConfig := CreateSQLConfig(dbConn, tc.name == QueryExecutionErrorTestName)

			mockLog := mocks.NewMockLogger(nil)
			repo := sqldb.NewDeviceRepo(sqlConfig, mockLog)

			count, err := repo.GetCount(context.Background(), tc.tenantID)

			if err == nil && tc.err != nil {
				t.Errorf("Expected error of type %T, got nil", tc.err)
			} else if err != nil {
				var dbErr repoerrors.DatabaseError
				if !errors.As(err, &dbErr) {
					t.Errorf("Expected error of type %T, got %T", tc.err, err)
				}
			}

			if count != tc.expected {
				t.Errorf("Expected count %d, got %d", tc.expected, count)
			}
		})
	}
}

func checkDeviceError(t *testing.T, err, expectedErr error) {
	t.Helper()

	if err == nil && expectedErr != nil {
		t.Errorf("Expected error of type %T, got nil", expectedErr)
	} else if err != nil {
		if expectedErr == nil {
			t.Errorf("Expected no error, got %T", err)

			return
		}

		expectedErrorType := reflect.TypeOf(expectedErr)
		actualErrorType := reflect.TypeOf(err)

		if expectedErrorType != actualErrorType {
			t.Errorf("Expected error of type %T, got %T", expectedErr, err)
		}
	}
}

func TestDeviceRepo_Get(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		setup    func(dbConn *sql.DB)
		top      int
		skip     int
		tenantID string
		expected []entity.Device
		err      error
	}{
		{
			name: "Successful query",
			setup: func(dbConn *sql.DB) {
				_, err := dbConn.ExecContext(context.Background(), `INSERT INTO devices (guid, hostname, tags, mpsinstance, connectionstatus, mpsusername, tenantid, friendlyname, dnssuffix, deviceinfo, username, password, usetls, allowselfsigned, certhash) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					"guid1", "hostname1", "tag1", "mpsinstance1", true, "mpsusername1", "tenant1", "friendlyname1", "dnssuffix1", "deviceinfo1", "username1", "password1", true, false, Certhash)
				require.NoError(t, err)
			},
			top:      10,
			skip:     0,
			tenantID: "tenant1",
			expected: []entity.Device{
				{
					GUID:             "guid1",
					Hostname:         "hostname1",
					Tags:             "tag1",
					MPSInstance:      "mpsinstance1",
					ConnectionStatus: true,
					MPSUsername:      "mpsusername1",
					TenantID:         "tenant1",
					FriendlyName:     "friendlyname1",
					DNSSuffix:        "dnssuffix1",
					DeviceInfo:       "deviceinfo1",
					Username:         "username1",
					Password:         "password1",
					UseTLS:           true,
					AllowSelfSigned:  false,
					CertHash:         Certhash,
				},
			},
			err: nil,
		},
		{
			name:     "No devices found",
			setup:    func(_ *sql.DB) {},
			top:      10,
			skip:     0,
			tenantID: "tenant2",
			expected: []entity.Device{},
			err:      nil,
		},
		{
			name:     QueryExecutionErrorTestName,
			setup:    func(_ *sql.DB) {},
			top:      10,
			skip:     0,
			tenantID: "tenant1",
			expected: nil,
			err:      repoerrors.DatabaseError{},
		},
		{
			name: "Rows scan error",
			setup: func(dbConn *sql.DB) {
				_, _ = dbConn.ExecContext(context.Background(), `INSERT INTO devices (guid, hostname, tags, mpsinstance, connectionstatus, mpsusername, tenantid, friendlyname, dnssuffix, deviceinfo, username, password, usetls, allowselfsigned, certhash) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					"guid1", "hostname1", "tag1", "mpsinstance1", true, "mpsusername1", "tenant1", "friendlyname1", "dnssuffix1", "deviceinfo1", "username1", "password1", "not-a-bool", false, Certhash)
			},
			top:      10,
			skip:     0,
			tenantID: "tenant1",
			expected: nil,
			err:      repoerrors.DatabaseError{},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dbConn := setupDeviceTable(t)
			defer dbConn.Close()

			tc.setup(dbConn)

			sqlConfig := CreateSQLConfig(dbConn, tc.name == QueryExecutionErrorTestName)

			mockLog := mocks.NewMockLogger(nil)
			repo := sqldb.NewDeviceRepo(sqlConfig, mockLog)

			devices, err := repo.Get(context.Background(), tc.top, tc.skip, tc.tenantID)

			checkDeviceError(t, err, tc.err)

			assertDeviceResults(t, tc.expected, devices)
		})
	}
}

func TestDeviceRepo_GetByID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		setup    func(dbConn *sql.DB)
		guid     string
		tenantID string
		expected *entity.Device
		err      error
	}{
		{
			name: "Successful query",
			setup: func(dbConn *sql.DB) {
				_, err := dbConn.ExecContext(context.Background(), `INSERT INTO devices (guid, hostname, tags, mpsinstance, connectionstatus, mpsusername, tenantid, friendlyname, dnssuffix, deviceinfo, username, password, usetls, allowselfsigned, certhash) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					"guid1", "hostname1", "tag1", "mpsinstance1", true, "mpsusername1", "tenant1", "friendlyname1", "dnssuffix1", "deviceinfo1", "username1", "password1", true, false, Certhash)
				require.NoError(t, err)
			},
			guid:     "guid1",
			tenantID: "tenant1",
			expected: &entity.Device{
				GUID:             "guid1",
				Hostname:         "hostname1",
				Tags:             "tag1",
				MPSInstance:      "mpsinstance1",
				ConnectionStatus: true,
				MPSUsername:      "mpsusername1",
				TenantID:         "tenant1",
				FriendlyName:     "friendlyname1",
				DNSSuffix:        "dnssuffix1",
				DeviceInfo:       "deviceinfo1",
				Username:         "username1",
				Password:         "password1",
				UseTLS:           true,
				AllowSelfSigned:  false,
				CertHash:         Certhash,
			},
			err: nil,
		},
		{
			name:     "No device found",
			setup:    func(_ *sql.DB) {},
			guid:     "guid2",
			tenantID: "tenant2",
			expected: nil,
			err:      nil,
		},
		{
			name:     QueryExecutionErrorTestName,
			setup:    func(_ *sql.DB) {},
			guid:     "guid1",
			tenantID: "tenant1",
			expected: nil,
			err:      repoerrors.DatabaseError{},
		},
		{
			name: "Rows scan error",
			setup: func(dbConn *sql.DB) {
				_, _ = dbConn.ExecContext(context.Background(), `INSERT INTO devices (guid, hostname, tags, mpsinstance, connectionstatus, mpsusername, tenantid, friendlyname, dnssuffix, deviceinfo, username, password, usetls, allowselfsigned, certhash) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					"guid1", "hostname1", "tag1", "mpsinstance1", true, "mpsusername1", "tenant1", "friendlyname1", "dnssuffix1", "deviceinfo1", "username1", "password1", "not-a-bool", false, Certhash)
			},
			guid:     "guid1",
			tenantID: "tenant1",
			expected: nil,
			err:      repoerrors.DatabaseError{},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dbConn := setupDeviceTable(t)
			defer dbConn.Close()

			tc.setup(dbConn)

			sqlConfig := &db.SQL{
				Builder:    squirrel.StatementBuilder.PlaceholderFormat(squirrel.Question),
				Pool:       dbConn,
				IsEmbedded: true,
			}

			if tc.name == QueryExecutionErrorTestName {
				sqlConfig.Builder = squirrel.StatementBuilder.PlaceholderFormat(squirrel.AtP)
			}

			mockLog := mocks.NewMockLogger(nil)
			repo := sqldb.NewDeviceRepo(sqlConfig, mockLog)

			device, err := repo.GetByID(context.Background(), tc.guid, tc.tenantID)

			checkDeviceError(t, err, tc.err)

			if device == nil && tc.expected == nil {
				return
			}

			assert.IsType(t, tc.expected, device)
		})
	}
}

func TestDeviceRepo_GetDistinctTags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		setup    func(dbConn *sql.DB)
		tenantID string
		expected []string
		err      error
	}{
		{
			name: "Successful query",
			setup: func(dbConn *sql.DB) {
				_, err := dbConn.ExecContext(context.Background(), `INSERT INTO devices (guid, tags, tenantid) VALUES (?, ?, ?)`, "guid1", "tag1", "tenant1")
				require.NoError(t, err)
				_, err = dbConn.ExecContext(context.Background(), `INSERT INTO devices (guid, tags, tenantid) VALUES (?, ?, ?)`, "guid2", "tag2", "tenant1")
				require.NoError(t, err)
				_, err = dbConn.ExecContext(context.Background(), `INSERT INTO devices (guid, tags, tenantid) VALUES (?, ?, ?)`, "guid3", "tag1", "tenant1")
				require.NoError(t, err)
			},
			tenantID: "tenant1",
			expected: []string{"tag1", "tag2"},
			err:      nil,
		},
		{
			name: "No tags found",
			setup: func(dbConn *sql.DB) {
				_, _ = dbConn.ExecContext(context.Background(), "DELETE FROM devices WHERE tenantid = ?", "tenant1")
			},
			tenantID: "tenant1",
			expected: []string{},
			err:      nil,
		},
		{
			name:     QueryExecutionErrorTestName,
			setup:    func(_ *sql.DB) {},
			tenantID: "tenant1",
			expected: []string{},
			err:      repoerrors.DatabaseError{},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dbConn := setupDeviceTable(t)
			tc.setup(dbConn)

			sqlConfig := &db.SQL{
				Builder:    squirrel.StatementBuilder.PlaceholderFormat(squirrel.Question),
				Pool:       dbConn,
				IsEmbedded: true,
			}

			if tc.name == QueryExecutionErrorTestName {
				sqlConfig.Builder = squirrel.StatementBuilder.PlaceholderFormat(squirrel.AtP)
			}

			mockLog := mocks.NewMockLogger(nil)
			repo := sqldb.NewDeviceRepo(sqlConfig, mockLog)

			tags, err := repo.GetDistinctTags(context.Background(), tc.tenantID)

			checkDeviceError(t, err, tc.err)
			assert.ElementsMatch(t, tc.expected, tags)
		})
	}
}

func TestDeviceRepo_GetByTags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		setup       func(dbConn *sql.DB)
		tags        []string
		method      string
		limit       int
		offset      int
		tenantID    string
		expected    []entity.Device
		expectError bool
	}{
		{
			name: "Successful retrieval with AND operation",
			setup: func(dbConn *sql.DB) {
				_, err := dbConn.ExecContext(context.Background(), `INSERT INTO devices (guid, hostname, tags, mpsinstance, connectionstatus, mpsusername, tenantid, friendlyname, dnssuffix, deviceinfo) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					"guid1", "hostname1", ",tag1,tag2,", "mpsinstance1", true, "mpsusername1", "tenant1", "friendlyname1", "dnssuffix1", "deviceinfo1")
				require.NoError(t, err)
			},
			tags:     []string{"tag1", "tag2"},
			method:   "AND",
			limit:    10,
			offset:   0,
			tenantID: "tenant1",
			expected: []entity.Device{
				{
					GUID:             "guid1",
					Hostname:         "hostname1",
					Tags:             ",tag1,tag2,",
					MPSInstance:      "mpsinstance1",
					ConnectionStatus: true,
					MPSUsername:      "mpsusername1",
					TenantID:         "tenant1",
					FriendlyName:     "friendlyname1",
					DNSSuffix:        "dnssuffix1",
					DeviceInfo:       "deviceinfo1",
				},
			},
			expectError: false,
		},
		{
			name: "Successful retrieval with OR operation",
			setup: func(dbConn *sql.DB) {
				_, err := dbConn.ExecContext(context.Background(), `INSERT INTO devices (guid, hostname, tags, mpsinstance, connectionstatus, mpsusername, tenantid, friendlyname, dnssuffix, deviceinfo) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					"guid1", "hostname1", ",tag1,", "mpsinstance1", true, "mpsusername1", "tenant1", "friendlyname1", "dnssuffix1", "deviceinfo1")
				require.NoError(t, err)
			},
			tags:     []string{"tag1", "tag2"},
			method:   "OR",
			limit:    10,
			offset:   0,
			tenantID: "tenant1",
			expected: []entity.Device{
				{
					GUID:             "guid1",
					Hostname:         "hostname1",
					Tags:             ",tag1,",
					MPSInstance:      "mpsinstance1",
					ConnectionStatus: true,
					MPSUsername:      "mpsusername1",
					TenantID:         "tenant1",
					FriendlyName:     "friendlyname1",
					DNSSuffix:        "dnssuffix1",
					DeviceInfo:       "deviceinfo1",
				},
			},
			expectError: false,
		},
		{
			name:        "No matching tags",
			setup:       func(_ *sql.DB) {},
			tags:        []string{"nonexistent-tag"},
			method:      "OR",
			limit:       10,
			offset:      0,
			tenantID:    "tenant1",
			expected:    []entity.Device{},
			expectError: false,
		},
		{
			name:     QueryExecutionErrorTestName,
			setup:    func(_ *sql.DB) {},
			tags:     []string{"tag1"},
			method:   "AND",
			limit:    10,
			offset:   0,
			tenantID: "tenant1",
			expected: nil,
		},
		{
			name: "Row scan error",
			setup: func(dbConn *sql.DB) {
				_, err := dbConn.ExecContext(context.Background(), `INSERT INTO devices (guid, hostname, tags, mpsinstance, connectionstatus, mpsusername, tenantid, friendlyname, dnssuffix, deviceinfo) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					"guid1", "hostname1", ",tag1,", "mpsinstance1", true, "mpsusername1", "tenant1", "friendlyname1", "dnssuffix1", "deviceinfo1")
				require.NoError(t, err)
			},
			tags:     []string{"tag1"},
			method:   "AND",
			limit:    10,
			offset:   0,
			tenantID: "tenant1",
			expected: nil,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dbConn, err := sql.Open("sqlite", ":memory:")
			require.NoError(t, err)

			defer dbConn.Close()

			_, err = dbConn.ExecContext(context.Background(), `
                CREATE TABLE devices (
                    guid TEXT PRIMARY KEY,
                    hostname TEXT NOT NULL DEFAULT '',
                    tags TEXT NOT NULL DEFAULT '',
                    mpsinstance TEXT NOT NULL DEFAULT '',
                    connectionstatus BOOLEAN NOT NULL DEFAULT FALSE,
                    mpsusername TEXT NOT NULL DEFAULT '',
                    tenantid TEXT NOT NULL,
                    friendlyname TEXT NOT NULL DEFAULT '',
                    dnssuffix TEXT NOT NULL DEFAULT '',
                    deviceinfo TEXT NOT NULL DEFAULT ''
                );
            `)
			require.NoError(t, err)

			tc.setup(dbConn)

			sqlConfig := &db.SQL{
				Builder:    squirrel.StatementBuilder.PlaceholderFormat(squirrel.Question),
				Pool:       dbConn,
				IsEmbedded: true,
			}

			repo := sqldb.NewDeviceRepo(sqlConfig, mocks.NewMockLogger(nil))

			devices, err := repo.GetByTags(context.Background(), tc.tags, tc.method, tc.limit, tc.offset, tc.tenantID)

			if (err != nil) != tc.expectError {
				t.Errorf("Expected error status %v, got %v", tc.expectError, err != nil)
			}

			if devices == nil && tc.expected == nil {
				return
			}

			assert.IsType(t, tc.expected, devices)
		})
	}
}

func TestDeviceRepo_Delete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		setup    func(dbConn *sql.DB)
		guid     string
		tenantID string
		expected bool
		err      error
	}{
		{
			name: "Successful delete",
			setup: func(dbConn *sql.DB) {
				_, err := dbConn.ExecContext(context.Background(), `INSERT INTO devices (guid, hostname, tags, mpsinstance, connectionstatus, mpsusername, tenantid, friendlyname, dnssuffix, deviceinfo, username, password, usetls, allowselfsigned) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					"guid1", "hostname1", "tag1", "mpsinstance1", true, "mpsusername1", "tenant1", "friendlyname1", "dnssuffix1", "deviceinfo1", "username1", "password1", true, false)
				require.NoError(t, err)
			},
			guid:     "guid1",
			tenantID: "tenant1",
			expected: true,
			err:      nil,
		},
		{
			name:     "No matching device",
			setup:    func(_ *sql.DB) {},
			guid:     "guid2",
			tenantID: "tenant2",
			expected: false,
			err:      nil,
		},
		{
			name:     QueryExecutionErrorTestName,
			setup:    func(_ *sql.DB) {},
			guid:     "guid1",
			tenantID: "tenant1",
			expected: false,
			err:      &repoerrors.DatabaseError{},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dbConn, err := sql.Open("sqlite", ":memory:")
			require.NoError(t, err)

			defer dbConn.Close()

			_, err = dbConn.ExecContext(context.Background(), `
				CREATE TABLE devices (
					guid TEXT PRIMARY KEY,
					hostname TEXT NOT NULL DEFAULT '',
					tags TEXT NOT NULL DEFAULT '',
					mpsinstance TEXT NOT NULL DEFAULT '',
					connectionstatus BOOLEAN NOT NULL DEFAULT FALSE,
					mpsusername TEXT NOT NULL DEFAULT '',
					tenantid TEXT NOT NULL,
					friendlyname TEXT NOT NULL DEFAULT '',
					dnssuffix TEXT NOT NULL DEFAULT '',
					deviceinfo TEXT NOT NULL DEFAULT '',
					username TEXT NOT NULL DEFAULT '',
					password TEXT NOT NULL DEFAULT '',
					mpspassword TEXT,
					mebxpassword TEXT,
					usetls BOOLEAN NOT NULL DEFAULT FALSE,
					allowselfsigned BOOLEAN NOT NULL DEFAULT FALSE,
					islmsavailable BOOLEAN NOT NULL DEFAULT FALSE
				);
			`)
			require.NoError(t, err)

			tc.setup(dbConn)

			sqlConfig := &db.SQL{
				Builder:    squirrel.StatementBuilder.PlaceholderFormat(squirrel.Question),
				Pool:       dbConn,
				IsEmbedded: true,
			}

			if tc.name == QueryExecutionErrorTestName {
				sqlConfig.Builder = squirrel.StatementBuilder.PlaceholderFormat(squirrel.AtP)
			}

			mockLog := mocks.NewMockLogger(nil)
			repo := sqldb.NewDeviceRepo(sqlConfig, mockLog)

			deleted, err := repo.Delete(context.Background(), tc.guid, tc.tenantID)

			if err == nil && tc.err != nil {
				t.Errorf("Expected error of type %T, got nil", tc.err)
			} else if err != nil {
				var dbErr repoerrors.DatabaseError
				if !errors.As(err, &dbErr) {
					t.Errorf("Expected error of type %T, got %T", tc.err, err)
				}
			}

			if deleted != tc.expected {
				t.Errorf("Expected deleted status %v, got %v", tc.expected, deleted)
			}
		})
	}
}

func TestDeviceRepo_Update(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		setup    func(dbConn *sql.DB)
		device   *entity.Device
		expected bool
		err      error
	}{
		{
			name: "Successful update",
			setup: func(dbConn *sql.DB) {
				_, err := dbConn.ExecContext(context.Background(), `INSERT INTO devices (guid, hostname, tags, mpsinstance, connectionstatus, mpsusername, tenantid, friendlyname, dnssuffix, deviceinfo, username, password, usetls, allowselfsigned, certhash) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					"guid1", "hostname1", "tag1", "mpsinstance1", true, "mpsusername1", "tenant1", "friendlyname1", "dnssuffix1", "deviceinfo1", "username1", "password1", true, false, Certhash)
				require.NoError(t, err)
			},
			device: &entity.Device{
				GUID:             "guid1",
				Hostname:         "updated_hostname",
				Tags:             "updated_tags",
				MPSInstance:      "updated_mpsinstance",
				ConnectionStatus: false,
				MPSUsername:      "updated_mpsusername",
				TenantID:         "tenant1",
				FriendlyName:     "updated_friendlyname",
				DNSSuffix:        "updated_dnssuffix",
				DeviceInfo:       "updated_deviceinfo",
				Username:         "updated_username",
				Password:         "updated_password",
				UseTLS:           false,
				AllowSelfSigned:  true,
				CertHash:         Certhash,
			},
			expected: true,
			err:      nil,
		},
		{
			name:  "Update non-existent device",
			setup: func(_ *sql.DB) {},
			device: &entity.Device{
				GUID:             "nonexistent_guid",
				Hostname:         "hostname",
				Tags:             "tags",
				MPSInstance:      "mpsinstance",
				ConnectionStatus: true,
				MPSUsername:      "mpsusername",
				TenantID:         "tenant",
				FriendlyName:     "friendlyname",
				DNSSuffix:        "dnssuffix",
				DeviceInfo:       "deviceinfo",
				Username:         "username",
				Password:         "password",
				UseTLS:           true,
				AllowSelfSigned:  false,
				CertHash:         Certhash,
			},
			expected: false,
			err:      nil,
		},
		{
			name:  QueryExecutionErrorTestName,
			setup: func(_ *sql.DB) {},
			device: &entity.Device{
				GUID:             "guid1",
				Hostname:         "hostname",
				Tags:             "tags",
				MPSInstance:      "mpsinstance",
				ConnectionStatus: true,
				MPSUsername:      "mpsusername",
				TenantID:         "tenant1",
				FriendlyName:     "friendlyname",
				DNSSuffix:        "dnssuffix",
				DeviceInfo:       "deviceinfo",
				Username:         "username",
				Password:         "password",
				UseTLS:           true,
				AllowSelfSigned:  false,
				CertHash:         Certhash,
			},
			expected: false,
			err:      &repoerrors.DatabaseError{},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dbConn, err := sql.Open("sqlite", ":memory:")
			require.NoError(t, err)

			defer dbConn.Close()

			_, err = dbConn.ExecContext(context.Background(), `
				CREATE TABLE devices (
					guid TEXT PRIMARY KEY,
					hostname TEXT NOT NULL DEFAULT '',
					tags TEXT NOT NULL DEFAULT '',
					mpsinstance TEXT NOT NULL DEFAULT '',
					connectionstatus BOOLEAN NOT NULL DEFAULT FALSE,
					mpsusername TEXT NOT NULL DEFAULT '',
					tenantid TEXT NOT NULL,
					friendlyname TEXT NOT NULL DEFAULT '',
					dnssuffix TEXT NOT NULL DEFAULT '',
					deviceinfo TEXT NOT NULL DEFAULT '',
					username TEXT NOT NULL DEFAULT '',
					password TEXT NOT NULL DEFAULT '',
					mpspassword TEXT,
					mebxpassword TEXT,
					usetls BOOLEAN NOT NULL DEFAULT FALSE,
					allowselfsigned BOOLEAN NOT NULL DEFAULT FALSE,
					certhash TEXT NOT NULL DEFAULT '',
					islmsavailable BOOLEAN NOT NULL DEFAULT FALSE
				);
			`)
			require.NoError(t, err)

			tc.setup(dbConn)

			sqlConfig := &db.SQL{
				Builder:    squirrel.StatementBuilder.PlaceholderFormat(squirrel.Question),
				Pool:       dbConn,
				IsEmbedded: true,
			}

			if tc.name == QueryExecutionErrorTestName {
				sqlConfig.Builder = squirrel.StatementBuilder.PlaceholderFormat(squirrel.AtP)
			}

			mockLog := mocks.NewMockLogger(nil)
			repo := sqldb.NewDeviceRepo(sqlConfig, mockLog)

			updated, err := repo.Update(context.Background(), tc.device)

			if err == nil && tc.err != nil {
				t.Errorf("Expected error of type %T, got nil", tc.err)
			} else if err != nil {
				var dbErr repoerrors.DatabaseError
				if !errors.As(err, &dbErr) {
					t.Errorf("Expected error of type %T, got %T", tc.err, err)
				}
			}

			if updated != tc.expected {
				t.Errorf("Expected update status %v, got %v", tc.expected, updated)
			}
		})
	}
}

func TestDeviceRepo_Insert(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		setup    func(dbConn *sql.DB)
		device   *entity.Device
		expected string
		err      error
	}{
		{
			name:  "Successful insert",
			setup: func(_ *sql.DB) {},
			device: &entity.Device{
				GUID:             "guid1",
				Hostname:         "hostname",
				Tags:             "tags",
				MPSInstance:      "mpsinstance",
				ConnectionStatus: true,
				MPSUsername:      "mpsusername",
				TenantID:         "tenant1",
				FriendlyName:     "friendlyname",
				DNSSuffix:        "dnssuffix",
				DeviceInfo:       "deviceinfo",
				Username:         "username",
				Password:         "password",
				UseTLS:           true,
				AllowSelfSigned:  false,
				CertHash:         Certhash,
			},
			expected: "",
			err:      nil,
		},
		{
			name: "Insert with not unique error",
			setup: func(dbConn *sql.DB) {
				_, err := dbConn.ExecContext(context.Background(), `INSERT INTO devices (guid, hostname, tags, mpsinstance, connectionstatus, mpsusername, tenantid, friendlyname, dnssuffix, deviceinfo, username, password, usetls, allowselfsigned, certhash) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					"guid1", "hostname1", "tag1", "mpsinstance1", true, "mpsusername1", "tenant1", "friendlyname1", "dnssuffix1", "deviceinfo1", "username1", "password1", true, false, "certhash")
				require.NoError(t, err)
			},
			device: &entity.Device{
				GUID:             "guid1",
				Hostname:         "hostname",
				Tags:             "tags",
				MPSInstance:      "mpsinstance",
				ConnectionStatus: true,
				MPSUsername:      "mpsusername",
				TenantID:         "tenant1",
				FriendlyName:     "friendlyname",
				DNSSuffix:        "dnssuffix",
				DeviceInfo:       "deviceinfo",
				Username:         "username",
				Password:         "password",
				UseTLS:           true,
				AllowSelfSigned:  false,
				CertHash:         Certhash,
			},
			expected: "",
			err:      repoerrors.NotUniqueError{},
		},
		{
			name:  QueryExecutionErrorTestName,
			setup: func(_ *sql.DB) {},
			device: &entity.Device{
				GUID:             "guid1",
				Hostname:         "hostname",
				Tags:             "tags",
				MPSInstance:      "mpsinstance",
				ConnectionStatus: true,
				MPSUsername:      "mpsusername",
				TenantID:         "tenant1",
				FriendlyName:     "friendlyname",
				DNSSuffix:        "dnssuffix",
				DeviceInfo:       "deviceinfo",
				Username:         "username",
				Password:         "password",
				UseTLS:           true,
				AllowSelfSigned:  false,
				CertHash:         Certhash,
			},
			expected: "",
			err:      repoerrors.DatabaseError{},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dbConn := setupDeviceTable(t)
			tc.setup(dbConn)

			sqlConfig := &db.SQL{
				Builder:    squirrel.StatementBuilder.PlaceholderFormat(squirrel.Question),
				Pool:       dbConn,
				IsEmbedded: true,
			}

			if tc.name == QueryExecutionErrorTestName {
				sqlConfig.Builder = squirrel.StatementBuilder.PlaceholderFormat(squirrel.AtP)
			}

			mockLog := mocks.NewMockLogger(nil)
			repo := sqldb.NewDeviceRepo(sqlConfig, mockLog)

			version, err := repo.Insert(context.Background(), tc.device)

			checkDeviceError(t, err, tc.err)

			if version != tc.expected {
				t.Errorf("Expected version %v, got %v", tc.expected, version)
			}
		})
	}
}

func TestDeviceRepo_GetByColumn(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		setup       func(dbConn *sql.DB)
		columnName  string
		queryValue  string
		tenantID    string
		expected    []entity.Device
		expectError bool
	}{
		{
			name: "Successful retrieval",
			setup: func(dbConn *sql.DB) {
				_, err := dbConn.ExecContext(context.Background(), `INSERT INTO devices (guid, hostname, tags, mpsinstance, connectionstatus, mpsusername, tenantid, friendlyname, dnssuffix, deviceinfo, username, password, usetls, allowselfsigned, certhash) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					"guid1", "hostname1", "tag1", "mpsinstance1", true, "mpsusername1", "tenant1", "friendlyname1", "dnssuffix1", "deviceinfo1", "username1", "password1", true, false, Certhash)
				require.NoError(t, err)
			},
			columnName: "guid",
			queryValue: "guid1",
			tenantID:   "tenant1",
			expected: []entity.Device{
				{
					GUID:             "guid1",
					Hostname:         "hostname1",
					Tags:             "tag1",
					MPSInstance:      "mpsinstance1",
					ConnectionStatus: true,
					MPSUsername:      "mpsusername1",
					TenantID:         "tenant1",
					FriendlyName:     "friendlyname1",
					DNSSuffix:        "dnssuffix1",
					DeviceInfo:       "deviceinfo1",
					Username:         "username1",
					Password:         "password1",
					UseTLS:           true,
					AllowSelfSigned:  false,
					CertHash:         Certhash,
				},
			},
			expectError: false,
		},
		{
			name:        "No devices found",
			setup:       func(_ *sql.DB) {},
			columnName:  "guid",
			queryValue:  "nonexistent-guid",
			tenantID:    "tenant1",
			expected:    []entity.Device{},
			expectError: false,
		},
		{
			name:       QueryExecutionErrorTestName,
			setup:      func(_ *sql.DB) {},
			columnName: "guid",
			queryValue: "guid1",
			tenantID:   "tenant1",
			expected:   nil,
		},
		{
			name: "Row scan error",
			setup: func(dbConn *sql.DB) {
				_, err := dbConn.ExecContext(context.Background(), `INSERT INTO devices (guid, hostname, tags, mpsinstance, connectionstatus, mpsusername, tenantid, friendlyname, dnssuffix, deviceinfo, username, password, usetls, allowselfsigned, certhash) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					"guid1", "hostname1", "tag1", "mpsinstance1", true, "mpsusername1", "tenant1", "friendlyname1", "dnssuffix1", "deviceinfo1", "username1", "password1", true, false, Certhash)
				require.NoError(t, err)
			},
			columnName: "guid",
			queryValue: "guid1",
			tenantID:   "tenant1",
			expected:   nil,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dbConn, err := sql.Open("sqlite", ":memory:")
			require.NoError(t, err)

			defer dbConn.Close()

			_, err = dbConn.ExecContext(context.Background(), `
                CREATE TABLE devices (
                    guid TEXT PRIMARY KEY,
                    hostname TEXT NOT NULL DEFAULT '',
                    tags TEXT NOT NULL DEFAULT '',
                    mpsinstance TEXT NOT NULL DEFAULT '',
                    connectionstatus BOOLEAN NOT NULL DEFAULT FALSE,
                    mpsusername TEXT NOT NULL DEFAULT '',
                    tenantid TEXT NOT NULL,
                    friendlyname TEXT NOT NULL DEFAULT '',
                    dnssuffix TEXT NOT NULL DEFAULT '',
                    deviceinfo TEXT NOT NULL DEFAULT '',
                    username TEXT NOT NULL DEFAULT '',
                    password TEXT NOT NULL DEFAULT '',
                    usetls BOOLEAN NOT NULL DEFAULT FALSE,
                    allowselfsigned BOOLEAN NOT NULL DEFAULT FALSE,
					certhash TEXT NOT NULL DEFAULT '',
					islmsavailable BOOLEAN NOT NULL DEFAULT FALSE
                );
            `)
			require.NoError(t, err)

			tc.setup(dbConn)

			sqlConfig := &db.SQL{
				Builder:    squirrel.StatementBuilder.PlaceholderFormat(squirrel.Question),
				Pool:       dbConn,
				IsEmbedded: true,
			}

			repo := sqldb.NewDeviceRepo(sqlConfig, mocks.NewMockLogger(nil))

			devices, err := repo.GetByColumn(context.Background(), tc.columnName, tc.queryValue, tc.tenantID)

			if (err != nil) != tc.expectError {
				t.Errorf("Expected error status %v, got %v", tc.expectError, err != nil)
			}

			if devices == nil && tc.expected == nil {
				return
			}

			assert.IsType(t, tc.expected, devices)
		})
	}
}

func TestDeviceRepo_UpdateConnectionStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		setup  func(dbConn *sql.DB)
		guid   string
		status bool
		err    error
		verify func(t *testing.T, dbConn *sql.DB)
	}{
		{
			name: "Set connected status to true",
			setup: func(dbConn *sql.DB) {
				_, err := dbConn.ExecContext(context.Background(),
					`INSERT INTO devices (guid, hostname, tags, mpsinstance, connectionstatus, mpsusername, tenantid, friendlyname, dnssuffix, deviceinfo, username, password, usetls, allowselfsigned) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					"guid1", "hostname1", "tag1", "mps1", false, "mpsuser1", "tenant1", "friendly1", "dns1", "info1", "user1", "pass1", true, false)
				require.NoError(t, err)
			},
			guid:   "guid1",
			status: true,
			err:    nil,
			verify: func(t *testing.T, dbConn *sql.DB) {
				t.Helper()

				var (
					connStatus    bool
					lastConnected sql.NullString
				)

				err := dbConn.QueryRowContext(context.Background(),
					"SELECT connectionstatus, lastconnected FROM devices WHERE guid = ?", "guid1").
					Scan(&connStatus, &lastConnected)
				require.NoError(t, err)
				assert.True(t, connStatus)
				assert.True(t, lastConnected.Valid, "lastconnected should be set")
			},
		},
		{
			name: "Set connected status to false",
			setup: func(dbConn *sql.DB) {
				_, err := dbConn.ExecContext(context.Background(),
					`INSERT INTO devices (guid, hostname, tags, mpsinstance, connectionstatus, mpsusername, tenantid, friendlyname, dnssuffix, deviceinfo, username, password, usetls, allowselfsigned) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					"guid2", "hostname2", "tag2", "mps2", true, "mpsuser2", "tenant1", "friendly2", "dns2", "info2", "user2", "pass2", true, false)
				require.NoError(t, err)
			},
			guid:   "guid2",
			status: false,
			err:    nil,
			verify: func(t *testing.T, dbConn *sql.DB) {
				t.Helper()

				var (
					connStatus       bool
					lastDisconnected sql.NullString
				)

				err := dbConn.QueryRowContext(context.Background(),
					"SELECT connectionstatus, lastdisconnected FROM devices WHERE guid = ?", "guid2").
					Scan(&connStatus, &lastDisconnected)
				require.NoError(t, err)
				assert.False(t, connStatus)
				assert.True(t, lastDisconnected.Valid, "lastdisconnected should be set")
			},
		},
		{
			name:   "Update non-existent device - no error",
			setup:  func(_ *sql.DB) {},
			guid:   "nonexistent",
			status: true,
			err:    nil,
			verify: func(_ *testing.T, _ *sql.DB) {},
		},
		{
			name:   QueryExecutionErrorTestName,
			setup:  func(_ *sql.DB) {},
			guid:   "guid1",
			status: true,
			err:    &repoerrors.DatabaseError{},
			verify: func(_ *testing.T, _ *sql.DB) {},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dbConn := setupDeviceTable(t)
			defer dbConn.Close()

			tc.setup(dbConn)

			sqlConfig := CreateSQLConfig(dbConn, tc.name == QueryExecutionErrorTestName)

			mockLog := mocks.NewMockLogger(nil)
			repo := sqldb.NewDeviceRepo(sqlConfig, mockLog)

			err := repo.UpdateConnectionStatus(context.Background(), tc.guid, tc.status)

			if tc.err == nil {
				require.NoError(t, err)
			} else {
				require.Error(t, err)

				var dbErr repoerrors.DatabaseError
				assert.True(t, errors.As(err, &dbErr))
			}

			tc.verify(t, dbConn)
		})
	}
}

func TestDeviceRepo_UpdateLastSeen(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		setup  func(dbConn *sql.DB)
		guid   string
		err    error
		verify func(t *testing.T, dbConn *sql.DB)
	}{
		{
			name: "Successfully updates lastseen",
			setup: func(dbConn *sql.DB) {
				_, err := dbConn.ExecContext(context.Background(),
					`INSERT INTO devices (guid, hostname, tags, mpsinstance, connectionstatus, mpsusername, tenantid, friendlyname, dnssuffix, deviceinfo, username, password, usetls, allowselfsigned) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					"guid1", "hostname1", "tag1", "mps1", false, "mpsuser1", "tenant1", "friendly1", "dns1", "info1", "user1", "pass1", true, false)
				require.NoError(t, err)
			},
			guid: "guid1",
			err:  nil,
			verify: func(t *testing.T, dbConn *sql.DB) {
				t.Helper()

				var lastSeen sql.NullString

				err := dbConn.QueryRowContext(context.Background(),
					"SELECT lastseen FROM devices WHERE guid = ?", "guid1").
					Scan(&lastSeen)
				require.NoError(t, err)
				assert.True(t, lastSeen.Valid, "lastseen should be set")
			},
		},
		{
			name:   "Update non-existent device - no error",
			setup:  func(_ *sql.DB) {},
			guid:   "nonexistent",
			err:    nil,
			verify: func(_ *testing.T, _ *sql.DB) {},
		},
		{
			name:   QueryExecutionErrorTestName,
			setup:  func(_ *sql.DB) {},
			guid:   "guid1",
			err:    &repoerrors.DatabaseError{},
			verify: func(_ *testing.T, _ *sql.DB) {},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dbConn := setupDeviceTable(t)
			defer dbConn.Close()

			tc.setup(dbConn)

			sqlConfig := CreateSQLConfig(dbConn, tc.name == QueryExecutionErrorTestName)

			mockLog := mocks.NewMockLogger(nil)
			repo := sqldb.NewDeviceRepo(sqlConfig, mockLog)

			err := repo.UpdateLastSeen(context.Background(), tc.guid)

			if tc.err == nil {
				require.NoError(t, err)
			} else {
				require.Error(t, err)

				var dbErr repoerrors.DatabaseError
				assert.True(t, errors.As(err, &dbErr))
			}

			tc.verify(t, dbConn)
		})
	}
}

func TestDeviceRepo_IsLMSAvailable_RoundTrip(t *testing.T) {
	t.Parallel()

	dbConn := setupDeviceTable(t)
	defer dbConn.Close()

	sqlConfig := &db.SQL{
		Builder:    squirrel.StatementBuilder.PlaceholderFormat(squirrel.Question),
		Pool:       dbConn,
		IsEmbedded: true,
	}

	mockLog := mocks.NewMockLogger(nil)
	repo := sqldb.NewDeviceRepo(sqlConfig, mockLog)

	// Insert a device with IsLMSAvailable = true
	dev := &entity.Device{
		GUID:           "guid-lms-true",
		Hostname:       "host-lms",
		TenantID:       "tenant1",
		Username:       "admin",
		Password:       "pass",
		CertHash:       Certhash,
		IsLMSAvailable: true,
	}

	_, err := repo.Insert(context.Background(), dev)
	require.NoError(t, err)

	// Retrieve and verify the field round-trips
	got, err := repo.GetByID(context.Background(), "guid-lms-true", "tenant1")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.True(t, got.IsLMSAvailable, "IsLMSAvailable should be true after insert")

	// Update the device with IsLMSAvailable = false
	dev.IsLMSAvailable = false

	_, err = repo.Update(context.Background(), dev)
	require.NoError(t, err)

	got, err = repo.GetByID(context.Background(), "guid-lms-true", "tenant1")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.False(t, got.IsLMSAvailable, "IsLMSAvailable should be false after update")
}
