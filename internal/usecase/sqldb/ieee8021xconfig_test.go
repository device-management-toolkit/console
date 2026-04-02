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
	"github.com/device-management-toolkit/console/internal/usecase/sqldb"
	"github.com/device-management-toolkit/console/pkg/db"
)

func setupIEEE8021xTable(t *testing.T) *sql.DB {
	t.Helper()

	dbConn, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	_, err = dbConn.ExecContext(context.Background(), schema)
	require.NoError(t, err)

	return dbConn
}

func TestIEEE8021xRepo_CheckProfileExists(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		setup       func(dbConn *sql.DB)
		profileName string
		tenantID    string
		expected    bool
		expectedErr error
	}{
		{
			name:        "Profile does not exist",
			setup:       func(_ *sql.DB) {},
			profileName: "profile2",
			tenantID:    "tenant2",
			expected:    false,
			expectedErr: nil,
		},
		{
			name: "Query execution error",
			setup: func(dbConn *sql.DB) {
				_, err := dbConn.ExecContext(context.Background(), `DROP TABLE ieee8021xconfigs`)
				require.NoError(t, err)
			},
			profileName: "profile1",
			tenantID:    "tenant1",
			expected:    false,
			expectedErr: sqldb.ErrIEEE8021xDatabase,
		},
		{
			name: "Profile exists",
			setup: func(dbConn *sql.DB) {
				_, err := dbConn.ExecContext(context.Background(), `INSERT INTO ieee8021xconfigs (profile_name,wired_interface, tenant_id) VALUES (?, ?, ?)`,
					"profile1", 1, "tenant1")
				require.NoError(t, err)

				var count int

				err = dbConn.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM ieee8021xconfigs WHERE profile_name = ? AND tenant_id = ?`, "profile1", "tenant1").Scan(&count)
				require.NoError(t, err)
				require.Equal(t, 1, count)
			},
			profileName: "profile1",
			tenantID:    "tenant1",
			expected:    true,
			expectedErr: nil,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dbConn := setupIEEE8021xTable(t)
			defer dbConn.Close()

			tc.setup(dbConn)

			sqlConfig := &db.SQL{
				Builder:    squirrel.StatementBuilder.PlaceholderFormat(squirrel.Question),
				Pool:       dbConn,
				IsEmbedded: true,
			}

			mockLog := mocks.NewMockLogger(nil)
			repo := sqldb.NewIEEE8021xRepo(sqlConfig, mockLog)

			exists, err := repo.CheckProfileExists(context.Background(), tc.profileName, tc.tenantID)

			if err == nil && tc.expectedErr != nil {
				t.Errorf("Expected error %v, got nil", tc.expectedErr)
			}

			if exists != tc.expected {
				t.Errorf("Expected existence %v, got %v", tc.expected, exists)
			}
		})
	}
}

func TestIEEE8021xRepo_GetCount(t *testing.T) {
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
				_, err := dbConn.ExecContext(context.Background(), `INSERT INTO ieee8021xconfigs (profile_name, auth_protocol, pxe_timeout, wired_interface, tenant_id) VALUES (?, ?, ?, ?, ?)`,
					"profile1", "auth1", 30, "wired1", "tenant1")
				require.NoError(t, err)
			},
			tenantID: "tenant1",
			expected: 1,
			err:      nil,
		},
		{
			name:     "No profiles found",
			setup:    func(_ *sql.DB) {},
			tenantID: "tenant2",
			expected: 0,
			err:      nil,
		},
		{
			name:     "Query execution error",
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

			dbConn := setupIEEE8021xTable(t)
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
			repo := sqldb.NewIEEE8021xRepo(sqlConfig, mockLog)

			count, err := repo.GetCount(context.Background(), tc.tenantID)

			if err == nil && tc.err != nil {
				t.Errorf("Expected error of type %T, got nil", tc.err)
			} else if err != nil {
				var dbError sqldb.DatabaseError

				if !errors.As(err, &dbError) {
					t.Errorf("Expected error of type %T, got %T", tc.err, err)
				}
			}

			if count != tc.expected {
				t.Errorf("Expected count %d, got %d", tc.expected, count)
			}
		})
	}
}

func intPtr(i int) *int {
	return &i
}

func GetIEEEHelper(t *testing.T, tc GetIEEETestCase, configs []entity.IEEE8021xConfig, err error) {
	t.Helper()

	checkErrorType(t, err, tc.err)
	checkConfigs(t, tc.expected, configs)
}

func checkErrorType(t *testing.T, err, expectedErr error) {
	t.Helper()

	if err == nil && expectedErr != nil {
		t.Errorf("Expected error of type %T, got nil", expectedErr)

		return
	}

	if err != nil {
		var dbError sqldb.DatabaseError
		if !errors.As(err, &dbError) {
			t.Errorf("Expected error of type %T, got %T", expectedErr, err)
		}
	}
}

func checkConfigs(t *testing.T, expected, configs []entity.IEEE8021xConfig) {
	t.Helper()

	if expected == nil && configs != nil {
		t.Errorf("Expected nil, got non-nil")

		return
	}

	if expected != nil && configs == nil {
		t.Errorf("Expected non-nil, got nil")

		return
	}

	if expected != nil && configs != nil {
		expectedType := reflect.TypeOf(expected)
		actualType := reflect.TypeOf(configs)

		if expectedType != actualType {
			t.Errorf("Expected type %v, got %v", expectedType, actualType)
		}
	}
}

type GetIEEETestCase struct {
	name     string
	setup    func(dbConn *sql.DB)
	top      int
	skip     int
	tenantID string
	expected []entity.IEEE8021xConfig
	err      error
}

func TestIEEE8021xRepo_Get(t *testing.T) {
	t.Parallel()

	tests := []GetIEEETestCase{
		{
			name: "Successful query",
			setup: func(dbConn *sql.DB) {
				pxeTimeout := 30
				_, err := dbConn.ExecContext(context.Background(), `INSERT INTO ieee8021xconfigs (profile_name, auth_protocol, pxe_timeout, wired_interface, tenant_id) VALUES (?, ?, ?, ?, ?)`,
					"profile1", 1, pxeTimeout, true, "tenant1")
				require.NoError(t, err)
			},
			top:      10,
			skip:     0,
			tenantID: "tenant1",
			expected: []entity.IEEE8021xConfig{
				{
					ProfileName:            "profile1",
					AuthenticationProtocol: 1,
					PXETimeout:             intPtr(30),
					WiredInterface:         true,
					TenantID:               "tenant1",
					Version:                "v1.0",
				},
			},
			err: nil,
		},
		{
			name:     "No profiles found",
			setup:    func(_ *sql.DB) {},
			top:      10,
			skip:     0,
			tenantID: "tenant2",
			expected: []entity.IEEE8021xConfig{},
			err:      nil,
		},
		{
			name:     "Query execution error",
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
				_, _ = dbConn.ExecContext(context.Background(), `INSERT INTO ieee8021xconfigs (profile_name, auth_protocol, pxe_timeout, wired_interface, tenant_id) VALUES (?, ?, ?, ?, ?)`,
					"profile1", "not-an-int", "not-an-int", "not-a-bool", "tenant1")
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

			dbConn := setupIEEE8021xTable(t)
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
			repo := sqldb.NewIEEE8021xRepo(sqlConfig, mockLog)

			configs, err := repo.Get(context.Background(), tc.top, tc.skip, tc.tenantID)

			GetIEEEHelper(t, tc, configs, err)
		})
	}
}

func TestIEEE8021xRepo_GetByName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		setup       func(dbConn *sql.DB)
		profileName string
		tenantID    string
		expected    *entity.IEEE8021xConfig
		expectError bool
	}{
		{
			name: "Successful retrieval",
			setup: func(dbConn *sql.DB) {
				pxeTimeout := 30
				_, err := dbConn.ExecContext(context.Background(), `INSERT INTO ieee8021xconfigs (profile_name, auth_protocol, pxe_timeout, wired_interface, tenant_id) VALUES (?, ?, ?, ?, ?)`,
					"profile1", 1, pxeTimeout, true, "tenant1")
				require.NoError(t, err)
			},
			profileName: "profile1",
			tenantID:    "tenant1",
			expected: &entity.IEEE8021xConfig{
				ProfileName:            "profile1",
				AuthenticationProtocol: 1,
				PXETimeout:             intPtr(30),
				WiredInterface:         true,
				TenantID:               "tenant1",
				Version:                "v1.0",
			},
			expectError: false,
		},
		{
			name:        "No profile found",
			setup:       func(_ *sql.DB) {},
			profileName: "nonexistent-profile",
			tenantID:    "tenant1",
			expected:    nil,
			expectError: false,
		},
		{
			name: "Rows scan error",
			setup: func(dbConn *sql.DB) {
				_, _ = dbConn.ExecContext(context.Background(), `INSERT INTO ieee8021xconfigs (profile_name, auth_protocol, pxe_timeout, wired_interface, tenant_id) VALUES (?, ?, ?, ?, ?)`,
					"profile1", "not-an-int", "not-an-int", "not-a-bool", "tenant1")
			},
			profileName: "profile1",
			tenantID:    "tenant1",
			expected:    nil,
			expectError: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dbConn := setupIEEE8021xTable(t)
			defer dbConn.Close()

			tc.setup(dbConn)

			sqlConfig := &db.SQL{
				Builder:    squirrel.StatementBuilder.PlaceholderFormat(squirrel.Question),
				Pool:       dbConn,
				IsEmbedded: true,
			}

			repo := sqldb.NewIEEE8021xRepo(sqlConfig, mocks.NewMockLogger(nil))

			result, err := repo.GetByName(context.Background(), tc.profileName, tc.tenantID)

			if (err != nil) != tc.expectError {
				t.Errorf("Expected error status %v, got %v", tc.expectError, err != nil)
			}

			if result == nil && tc.expected == nil {
				return
			}

			assert.IsType(t, tc.expected, result)
			assert.IsType(t, tc.expected, result)
		})
	}
}

func TestIEEE8021xRepo_Delete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		setup       func(dbConn *sql.DB)
		profileName string
		tenantID    string
		expected    bool
		expectError bool
	}{
		{
			name: "Successful delete",
			setup: func(dbConn *sql.DB) {
				_, err := dbConn.ExecContext(context.Background(), `INSERT INTO ieee8021xconfigs (profile_name, auth_protocol, pxe_timeout, wired_interface, tenant_id) VALUES (?, ?, ?, ?, ?)`,
					"profile1", 1, 30, true, "tenant1")
				require.NoError(t, err)
			},
			profileName: "profile1",
			tenantID:    "tenant1",
			expected:    true,
			expectError: false,
		},
		{
			name:        "No matching profile",
			setup:       func(_ *sql.DB) {},
			profileName: "nonexistent-profile",
			tenantID:    "tenant2",
			expected:    false,
			expectError: false,
		},
		{
			name: "Query execution error",
			setup: func(dbConn *sql.DB) {
				_, _ = dbConn.ExecContext(context.Background(), `CREATE TABLE ieee8021xconfigs (profile_name TEXT NOT NULL)`)
			},
			profileName: "profile1",
			tenantID:    "tenant1",
			expected:    false,
			expectError: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dbConn := setupIEEE8021xTable(t)
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
			repo := sqldb.NewIEEE8021xRepo(sqlConfig, mockLog)

			deleted, err := repo.Delete(context.Background(), tc.profileName, tc.tenantID)

			if (err != nil) != tc.expectError {
				t.Errorf("Expected error status %v, got %v", tc.expectError, err != nil)
			}

			if deleted != tc.expected {
				t.Errorf("Expected deleted status %v, got %v", tc.expected, deleted)
			}
		})
	}
}

func TestIEEE8021xRepo_Update(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		setup    func(dbConn *sql.DB)
		config   *entity.IEEE8021xConfig
		expected bool
		err      error
	}{
		{
			name: "Successful update",
			setup: func(dbConn *sql.DB) {
				_, err := dbConn.ExecContext(context.Background(), `INSERT INTO ieee8021xconfigs (profile_name, auth_protocol, pxe_timeout, wired_interface, tenant_id) VALUES (?, ?, ?, ?, ?)`,
					"profile1", 1, 30, true, "tenant1")
				require.NoError(t, err)
			},
			config: &entity.IEEE8021xConfig{
				ProfileName:            "profile1",
				AuthenticationProtocol: 2,
				WiredInterface:         false,
				TenantID:               "tenant1",
			},
			expected: true,
			err:      nil,
		},
		{
			name: "Update non-existent profile",
			setup: func(_ *sql.DB) {
			},
			config: &entity.IEEE8021xConfig{
				ProfileName:            "nonexistent-profile",
				AuthenticationProtocol: 2,
				WiredInterface:         false,
				TenantID:               "tenant2",
			},
			expected: false,
			err:      nil,
		},
		{
			name: "Query execution error",
			setup: func(_ *sql.DB) {
			},
			config: &entity.IEEE8021xConfig{
				ProfileName:            "test-domain",
				AuthenticationProtocol: 2,
				WiredInterface:         false,
				TenantID:               "tenant1",
			},
			expected: false,
			err:      &sqldb.DatabaseError{},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dbConn := setupIEEE8021xTable(t)
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
			repo := sqldb.NewIEEE8021xRepo(sqlConfig, mockLog)

			updated, err := repo.Update(context.Background(), tc.config)

			if err == nil && tc.err != nil {
				t.Errorf("Expected error of type %T, got nil", tc.err)
			} else if err != nil {
				var dbError sqldb.DatabaseError

				if !errors.As(err, &dbError) {
					t.Errorf("Expected error of type %T, got %T", tc.err, err)
				}
			}

			if updated != tc.expected {
				t.Errorf("Expected update status %v, got %v", tc.expected, updated)
			}
		})
	}
}

func InsertIEEEHelper(t *testing.T, tc InsertIEEETestCase, version string, err error) {
	t.Helper()

	if err == nil && tc.err != nil {
		t.Errorf("Expected error of type %T, got nil", tc.err)
	} else if err != nil {
		var notUniqueError sqldb.NotUniqueError

		var dbError sqldb.DatabaseError

		if !errors.As(err, &notUniqueError) && !errors.As(err, &dbError) {
			t.Errorf("Expected error of type %T or %T, got %T", tc.err, notUniqueError, err)
		}
	}

	if version != tc.expected {
		t.Errorf("Expected version %v, got %v", tc.expected, version)
	}
}

type InsertIEEETestCase struct {
	name     string
	setup    func(dbConn *sql.DB)
	config   *entity.IEEE8021xConfig
	expected string
	err      error
}

func TestIEEE8021xRepo_Insert(t *testing.T) {
	t.Parallel()

	tests := []InsertIEEETestCase{
		{
			name:  "Successful insert",
			setup: func(_ *sql.DB) {},
			config: &entity.IEEE8021xConfig{
				ProfileName:            "profile1",
				AuthenticationProtocol: 1,
				WiredInterface:         true,
				TenantID:               "tenant1",
			},
			expected: "",
			err:      nil,
		},
		{
			name: "Insert with not unique error",
			setup: func(dbConn *sql.DB) {
				_, err := dbConn.ExecContext(context.Background(), `INSERT INTO ieee8021xconfigs (profile_name, auth_protocol, pxe_timeout, wired_interface, tenant_id) VALUES (?, ?, ?, ?, ?)`,
					"profile1", 1, 30, true, "tenant1")
				require.NoError(t, err)
			},
			config: &entity.IEEE8021xConfig{
				ProfileName:            "profile1",
				AuthenticationProtocol: 1,
				WiredInterface:         true,
				TenantID:               "tenant1",
			},
			expected: "",
			err:      sqldb.NotUniqueError{},
		},
		{
			name:  "Query execution error",
			setup: func(_ *sql.DB) {},
			config: &entity.IEEE8021xConfig{
				ProfileName:            "profile1",
				AuthenticationProtocol: 1,
				WiredInterface:         true,
				TenantID:               "tenant1",
			},
			expected: "",
			err:      &sqldb.DatabaseError{},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dbConn := setupIEEE8021xTable(t)
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
			repo := sqldb.NewIEEE8021xRepo(sqlConfig, mockLog)

			version, err := repo.Insert(context.Background(), tc.config)

			InsertIEEEHelper(t, tc, version, err)
		})
	}
}
