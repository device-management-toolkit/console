package sqldb_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/Masterminds/squirrel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/device-management-toolkit/console/internal/entity"
	"github.com/device-management-toolkit/console/internal/mocks"
	"github.com/device-management-toolkit/console/internal/usecase/sqldb"
	"github.com/device-management-toolkit/console/pkg/db"
)

var (
	ErrGeneral        = errors.New("general error")
	ErrDeviceDatabase = errors.New("device database error")
)

var ErrCIRARepoDatabase = errors.New("CIRARepo database error")

const QueryExecutionErrorTestName = "Query execution error"

func CreateSQLConfig(dbConn *sql.DB, isExecutionError bool) *db.SQL {
	builder := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Question)
	if isExecutionError {
		builder = squirrel.StatementBuilder.PlaceholderFormat(squirrel.AtP)
	}

	return &db.SQL{
		Builder:    builder,
		Pool:       dbConn,
		IsEmbedded: true,
	}
}

func assertTestResult(t *testing.T, expected, actual *entity.CIRAConfig, expectedErr, actualErr error) {
	t.Helper()

	if expected == nil && actual == nil {
		return
	}

	if actualErr == nil && expectedErr != nil {
		t.Errorf("Expected error of type %T, got nil", expectedErr)
	} else if actualErr != nil {
		var dbErr sqldb.DatabaseError
		if !errors.As(actualErr, &dbErr) {
			t.Errorf("Expected error of type %T, got %T", expectedErr, actualErr)
		}
	}

	if expected == nil && actual == nil {
		return
	}

	assert.IsType(t, expected, actual)
}

func TestCIRARepo_GetCount(t *testing.T) {
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
				_, err := dbConn.ExecContext(context.Background(), `INSERT INTO ciraconfigs (cira_config_name, tenant_id) VALUES (?,?)`, "cira1", "tenant1")
				require.NoError(t, err)
			},
			tenantID: "tenant1",
			expected: 1,
			err:      nil,
		},
		{
			name:     "No configurations found",
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
			err:      &sqldb.DatabaseError{},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dbConn := setupDatabase(t)
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
			repo := sqldb.NewCIRARepo(sqlConfig, mockLog)

			count, err := repo.GetCount(context.Background(), tc.tenantID)

			if err == nil && tc.err != nil {
				t.Errorf("Expected error of type %T, got nil", tc.err)
			} else if err != nil {
				var dbErr sqldb.DatabaseError
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

func setupDatabase(t *testing.T) *sql.DB {
	t.Helper()

	dbConn, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	_, err = dbConn.ExecContext(context.Background(), schema)
	require.NoError(t, err)

	return dbConn
}

func TestCIRARepo_Get(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		setup    func(dbConn *sql.DB)
		top      int
		skip     int
		tenantID string
		expected []entity.CIRAConfig
		err      error
	}{
		{
			name: "Successful query",
			setup: func(dbConn *sql.DB) {
				_, err := dbConn.ExecContext(context.Background(), `INSERT INTO ciraconfigs (cira_config_name, mps_server_address, mps_port, user_name, password, common_name, server_address_format, auth_method, mps_root_certificate, proxydetails, tenant_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					"config1", "mpsaddress1", 1234, "user1", "pass1", "common1", 1, 1, "rootcert1", "proxydetail1", "tenant1")
				require.NoError(t, err)
			},
			top:      10,
			skip:     0,
			tenantID: "tenant1",
			expected: []entity.CIRAConfig{
				{
					ConfigName:          "config1",
					MPSAddress:          "mpsaddress1",
					MPSPort:             1234,
					Username:            "user1",
					Password:            "pass1",
					CommonName:          "common1",
					ServerAddressFormat: 1,
					AuthMethod:          1,
					MPSRootCertificate:  "rootcert1",
					ProxyDetails:        "proxydetail1",
					TenantID:            "tenant1",
				},
			},
			err: nil,
		},
		{
			name:     "No configs found",
			setup:    func(_ *sql.DB) {},
			top:      10,
			skip:     0,
			tenantID: "tenant2",
			expected: []entity.CIRAConfig{},
			err:      nil,
		},
		{
			name:     QueryExecutionErrorTestName,
			setup:    func(_ *sql.DB) {},
			top:      10,
			skip:     0,
			tenantID: "tenant1",
			expected: nil,
			err:      &sqldb.DatabaseError{},
		},
		{
			name: "Rows scan error",
			setup: func(dbConn *sql.DB) {
				_, _ = dbConn.ExecContext(context.Background(), `INSERT INTO ciraconfigs (cira_config_name, mps_server_address, mps_port, user_name, password, common_name, server_address_format, auth_method, mps_root_certificate, proxydetails, tenant_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					"config1", "mpsaddress1", "not-a-number", "user1", "pass1", "common1", 1, 1, "rootcert1", "proxydetail1", "tenant1")
			},
			top:      10,
			skip:     0,
			tenantID: "tenant1",
			expected: nil,
			err:      &sqldb.DatabaseError{},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dbConn := setupDatabase(t)
			defer dbConn.Close()

			tc.setup(dbConn)

			sqlConfig := CreateSQLConfig(dbConn, tc.name == QueryExecutionErrorTestName)

			mockLog := mocks.NewMockLogger(nil)
			repo := sqldb.NewCIRARepo(sqlConfig, mockLog)

			configs, err := repo.Get(context.Background(), tc.top, tc.skip, tc.tenantID)

			var expectedConfig *entity.CIRAConfig
			if len(tc.expected) > 0 {
				expectedConfig = &tc.expected[0]
			}

			var actualConfig *entity.CIRAConfig
			if len(configs) > 0 {
				actualConfig = &configs[0]
			}

			assertTestResult(t, expectedConfig, actualConfig, tc.err, err)
		})
	}
}

func TestCIRARepo_GetByName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setup      func(dbConn *sql.DB)
		configName string
		tenantID   string
		expected   *entity.CIRAConfig
		err        error
	}{
		{
			name: "Successful query",
			setup: func(dbConn *sql.DB) {
				_, err := dbConn.ExecContext(context.Background(), `INSERT INTO ciraconfigs (
					cira_config_name, mps_server_address, mps_port, user_name, password, common_name,
					server_address_format, auth_method, mps_root_certificate, proxydetails, tenant_id
				) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					"config1", "mps_address1", 8080, "username1", "password1", "common1",
					"format1", "method1", "cert1", "proxy1", "tenant1")
				require.NoError(t, err)
			},
			configName: "config1",
			tenantID:   "tenant1",
			expected: &entity.CIRAConfig{
				ConfigName:         "config1",
				MPSAddress:         "mps_address1",
				MPSPort:            8080,
				Username:           "username1",
				Password:           "password1",
				CommonName:         "common1",
				MPSRootCertificate: "cert1",
				ProxyDetails:       "proxy1",
				TenantID:           "tenant1",
			},
			err: nil,
		},
		{
			name:       "No CIRAConfig found",
			setup:      func(_ *sql.DB) {},
			configName: "config2",
			tenantID:   "tenant2",
			expected:   nil,
			err:        nil,
		},
		{
			name:       QueryExecutionErrorTestName,
			setup:      func(_ *sql.DB) {},
			configName: "config1",
			tenantID:   "tenant1",
			expected:   nil,
			err:        &sqldb.DatabaseError{},
		},
		{
			name: "Rows scan error",
			setup: func(dbConn *sql.DB) {
				_, _ = dbConn.ExecContext(context.Background(), `INSERT INTO ciraconfigs (
					cira_config_name, mps_server_address, mps_port, user_name, password, common_name,
					server_address_format, auth_method, mps_root_certificate, proxydetails, tenant_id
				) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					"config1", "mps_address1", 8080, "username1", "password1", "common1",
					"format1", "method1", "not-an-int", "proxy1", "tenant1")
			},
			configName: "config1",
			tenantID:   "tenant1",
			expected:   nil,
			err:        &sqldb.DatabaseError{},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dbConn := setupDatabase(t)
			defer dbConn.Close()

			tc.setup(dbConn)

			sqlConfig := CreateSQLConfig(dbConn, tc.name == QueryExecutionErrorTestName)

			mockLog := mocks.NewMockLogger(nil)
			repo := sqldb.NewCIRARepo(sqlConfig, mockLog)

			config, err := repo.GetByName(context.Background(), tc.configName, tc.tenantID)

			assertTestResult(t, tc.expected, config, tc.err, err)
		})
	}
}

func TestCIRARepo_Update(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		setup    func(dbConn *sql.DB)
		config   *entity.CIRAConfig
		expected bool
		err      error
	}{
		{
			name: "Successful update",
			setup: func(dbConn *sql.DB) {
				_, err := dbConn.ExecContext(context.Background(), `INSERT INTO ciraconfigs (cira_config_name, tenant_id, mps_server_address, mps_port, user_name, password, common_name, server_address_format, auth_method, mps_root_certificate, proxydetails) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					"config1", "tenant1", "old_address", 443, "old_user", "old_pass", "old_name", "ipv4", "digest", "old_cert", "old_proxy")
				require.NoError(t, err)
			},
			config: &entity.CIRAConfig{
				ConfigName:         "config1",
				TenantID:           "tenant1",
				MPSAddress:         "new_address",
				MPSPort:            8080,
				Username:           "new_user",
				Password:           "new_pass",
				CommonName:         "new_name",
				MPSRootCertificate: "new_cert",
				ProxyDetails:       "new_proxy",
			},
			expected: true,
			err:      nil,
		},
		{
			name:  "Update non-existent config",
			setup: func(_ *sql.DB) {},
			config: &entity.CIRAConfig{
				ConfigName:         "nonexistent_config",
				TenantID:           "tenant1",
				MPSAddress:         "address",
				MPSPort:            443,
				Username:           "user",
				Password:           "pass",
				CommonName:         "name",
				MPSRootCertificate: "cert",
				ProxyDetails:       "proxy",
			},
			expected: false,
			err:      nil,
		},
		{
			name:  QueryExecutionErrorTestName,
			setup: func(_ *sql.DB) {},
			config: &entity.CIRAConfig{
				ConfigName:         "config1",
				TenantID:           "tenant1",
				MPSAddress:         "address",
				MPSPort:            443,
				Username:           "user",
				Password:           "pass",
				CommonName:         "name",
				MPSRootCertificate: "cert",
				ProxyDetails:       "proxy",
			},
			expected: false,
			err:      sqldb.DatabaseError{},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dbConn := setupDatabase(t)
			defer dbConn.Close()

			tc.setup(dbConn)

			sqlConfig := CreateSQLConfig(dbConn, tc.name == QueryExecutionErrorTestName)

			mockLog := mocks.NewMockLogger(nil)
			repo := sqldb.NewCIRARepo(sqlConfig, mockLog)

			updated, err := repo.Update(context.Background(), tc.config)

			assertTestResult(t, nil, nil, tc.err, err)

			if updated != tc.expected {
				t.Errorf("Expected update status %v, got %v", tc.expected, updated)
			}
		})
	}
}

func TestCIRARepo_Insert(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		setup    func(dbConn *sql.DB)
		config   *entity.CIRAConfig
		expected string
		err      error
	}{
		{
			name:  "Successful insert",
			setup: func(_ *sql.DB) {},
			config: &entity.CIRAConfig{
				ConfigName:         "config1",
				MPSAddress:         "mps_address1",
				MPSPort:            443,
				Username:           "username1",
				Password:           "password1",
				CommonName:         "common_name1",
				MPSRootCertificate: "root_cert1",
				ProxyDetails:       "proxy_details1",
				TenantID:           "tenant1",
			},
			expected: "",
			err:      nil,
		},
		{
			name: "Insert with not unique error",
			setup: func(dbConn *sql.DB) {
				_, err := dbConn.ExecContext(context.Background(), `INSERT INTO ciraconfigs (cira_config_name, mps_server_address, mps_port, user_name, password, common_name, server_address_format, auth_method, mps_root_certificate, proxydetails, tenant_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					"config1", "mps_address1", 443, "username1", "password1", "common_name1", "format1", "auth_method1", "root_cert1", "proxy_details1", "tenant1")
				require.NoError(t, err)
			},
			config: &entity.CIRAConfig{
				ConfigName:         "config1",
				MPSAddress:         "mps_address1",
				MPSPort:            443,
				Username:           "username1",
				Password:           "password1",
				CommonName:         "common_name1",
				MPSRootCertificate: "root_cert1",
				ProxyDetails:       "proxy_details1",
				TenantID:           "tenant1",
			},
			expected: "",
			err:      sqldb.NotUniqueError{},
		},
		{
			name:  QueryExecutionErrorTestName,
			setup: func(_ *sql.DB) {},
			config: &entity.CIRAConfig{
				ConfigName:         "config1",
				MPSAddress:         "mps_address1",
				MPSPort:            443,
				Username:           "username1",
				Password:           "password1",
				CommonName:         "common_name1",
				MPSRootCertificate: "root_cert1",
				ProxyDetails:       "proxy_details1",
				TenantID:           "tenant1",
			},
			expected: "",
			err:      &sqldb.DatabaseError{},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dbConn := setupDatabase(t)
			defer dbConn.Close()

			tc.setup(dbConn)

			sqlConfig := CreateSQLConfig(dbConn, tc.name == QueryExecutionErrorTestName)

			mockLog := mocks.NewMockLogger(nil)
			repo := sqldb.NewCIRARepo(sqlConfig, mockLog)

			version, err := repo.Insert(context.Background(), tc.config)

			assertTestResult(t, nil, nil, tc.err, err)

			if version != tc.expected {
				t.Errorf("Expected version %v, got %v", tc.expected, version)
			}
		})
	}
}

func TestCIRARepo_Delete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setup      func(dbConn *sql.DB)
		configName string
		tenantID   string
		expected   bool
		err        error
	}{
		{
			name: "Successful delete",
			setup: func(dbConn *sql.DB) {
				_, err := dbConn.ExecContext(context.Background(), `INSERT INTO ciraconfigs (cira_config_name, mps_server_address, mps_port, user_name, password, common_name, server_address_format, auth_method, mps_root_certificate, proxydetails, tenant_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					"config1", "mps.address", 443, "user1", "pass1", "common1", "dns", "digest", "cert1", "proxy1", "tenant1")
				require.NoError(t, err)
			},
			configName: "config1",
			tenantID:   "tenant1",
			expected:   true,
			err:        nil,
		},
		{
			name:       "No matching config",
			setup:      func(_ *sql.DB) {},
			configName: "config2",
			tenantID:   "tenant2",
			expected:   false,
			err:        nil,
		},
		{
			name:       QueryExecutionErrorTestName,
			setup:      func(_ *sql.DB) {},
			configName: "config1",
			tenantID:   "tenant1",
			expected:   false,
			err:        &sqldb.DatabaseError{},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dbConn := setupDatabase(t)
			defer dbConn.Close()

			tc.setup(dbConn)

			sqlConfig := CreateSQLConfig(dbConn, tc.name == QueryExecutionErrorTestName)

			mockLog := mocks.NewMockLogger(nil)
			repo := sqldb.NewCIRARepo(sqlConfig, mockLog)

			deleted, err := repo.Delete(context.Background(), tc.configName, tc.tenantID)

			assertTestResult(t, nil, nil, tc.err, err)

			if deleted != tc.expected {
				t.Errorf("Expected deleted status %v, got %v", tc.expected, deleted)
			}
		})
	}
}
