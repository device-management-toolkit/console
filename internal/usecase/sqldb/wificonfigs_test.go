package sqldb_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/device-management-toolkit/console/internal/entity"
	"github.com/device-management-toolkit/console/internal/mocks"
	"github.com/device-management-toolkit/console/internal/usecase/sqldb"
	"github.com/device-management-toolkit/console/pkg/db"
	"github.com/device-management-toolkit/console/pkg/logger"
)

type WirelessRepo struct {
	*db.SQL
	logger.Interface
}

var ErrWiFiIEEEForeignKeyViolation = errors.New("IEEE foreign key violation in WirelessRepo")

func TestWirelessRepo_GetCount(t *testing.T) {
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
				_, err := dbConn.ExecContext(context.Background(), `INSERT INTO wirelessconfigs (wireless_profile_name, tenant_id) VALUES (?, ?)`, "profile1", "tenant1")
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

			dbConn, err := sql.Open("sqlite", ":memory:")
			require.NoError(t, err)

			defer dbConn.Close()

			_, err = dbConn.ExecContext(context.Background(), schema)
			require.NoError(t, err)

			tc.setup(dbConn)

			sqlConfig := &db.SQL{
				Builder:    squirrel.StatementBuilder.PlaceholderFormat(squirrel.Question),
				Pool:       dbConn,
				IsEmbedded: true,
			}

			if tc.name == "Query execution error" {
				sqlConfig.Builder = squirrel.StatementBuilder.PlaceholderFormat(squirrel.AtP)
			}

			mockLog := mocks.NewMockLogger(nil)
			repo := sqldb.NewWirelessRepo(sqlConfig, mockLog)

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

func GetWirelessConfigHelper(t *testing.T, tc GetWirelessConfigTestCase, configs []entity.WirelessConfig, err error) {
	t.Helper()

	if (err != nil) != (tc.err != nil) {
		t.Errorf("Expected error: %v, got: %v", tc.err, err)
	}

	if len(configs) == 0 && len(tc.expected) > 0 {
		t.Errorf("Expected %d configs, got %d", len(tc.expected), len(configs))

		return
	}

	if len(configs) != len(tc.expected) {
		t.Errorf("Expected %d configs, got %d", len(tc.expected), len(configs))
	}

	for i := range tc.expected {
		expectedConfig := &tc.expected[i]

		if i >= len(configs) {
			t.Errorf("Expected config %d, but got none", i)

			break
		}

		actualConfig := &configs[i]
		assert.IsType(t, expectedConfig, actualConfig)
	}
}

type GetWirelessConfigTestCase struct {
	name     string
	setup    func(dbConn *sql.DB)
	top      int
	skip     int
	tenantID string
	expected []entity.WirelessConfig
	err      error
}

func TestWirelessRepo_Get(t *testing.T) {
	t.Parallel()

	tests := []GetWirelessConfigTestCase{
		{
			name: "Error in Pool.Query",
			top:  10,
			setup: func(dbConn *sql.DB) {
				_, err := dbConn.ExecContext(context.Background(), `
					INSERT INTO wirelessconfigs
					(wireless_profile_name, authentication_method, encryption_method, ssid, tenant_id)
					VALUES
					('Profile1', 1, 2, 'SSID1', 'tenant2');
				`)
				require.NoError(t, err)

				dbConn.Close()
			},
			skip:     4,
			tenantID: "tenant2",
			expected: nil,
			err:      ErrGeneral,
		},
		{
			name: "Error in rows.Scan",
			top:  10,
			setup: func(dbConn *sql.DB) {
				_, err := dbConn.ExecContext(context.Background(), `
					INSERT INTO wirelessconfigs
					(wireless_profile_name, authentication_method, encryption_method, ssid, psk_value, tenant_id)
					VALUES
					('Profile1', 1, 2, 'SSID1', 'not_an_integer', 'tenant2');
				`)
				require.NoError(t, err)
			},
			skip:     4,
			tenantID: "tenant2",
			expected: nil,
			err:      nil,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dbConn, err := sql.Open("sqlite", ":memory:")
			require.NoError(t, err)

			defer dbConn.Close()

			_, err = dbConn.ExecContext(context.Background(), schema)
			require.NoError(t, err)

			tc.setup(dbConn)

			sqlConfig := &db.SQL{
				Builder:    squirrel.StatementBuilder.PlaceholderFormat(squirrel.Question),
				Pool:       dbConn,
				IsEmbedded: true,
			}

			mockLog := mocks.NewMockLogger(nil)
			repo := sqldb.NewWirelessRepo(sqlConfig, mockLog)

			wireless, err := repo.Get(context.Background(), tc.top, tc.skip, tc.tenantID)

			GetWirelessConfigHelper(t, tc, wireless, err)
		})
	}
}

func TestWirelessRepo_GetByName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		setup       func(dbConn *sql.DB)
		profileName string
		tenantID    string
		expected    *entity.WirelessConfig
		expectError bool
	}{
		{
			name: "Successful retrieval",
			setup: func(dbConn *sql.DB) {
				_, err := dbConn.ExecContext(context.Background(), `INSERT INTO ieee8021xconfigs (profile_name, 
        auth_protocol, 
        pxe_timeout, 
        wired_interface, 
        tenant_id) VALUES (?, ?, ?, ?, ?)`,
					"ieee1", 1, 30, true, "tenant1")
				require.NoError(t, err)

				_, err = dbConn.ExecContext(context.Background(), `
					INSERT INTO wirelessconfigs (
						wireless_profile_name, 
            authentication_method, 
            encryption_method, 
            ssid, 
            psk_value, 
            psk_passphrase,
            link_policy, 
            tenant_id,
						ieee8021x_profile_name
					) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);`,
					"profile1", 1, 2, "SSID1", "5", "encrypted", "policy1", "tenant1", "ieee1")
				require.NoError(t, err)
			},
			profileName: "profile1",
			tenantID:    "tenant1",
			expected: &entity.WirelessConfig{
				ProfileName:          "profile1",
				AuthenticationMethod: 1,
				EncryptionMethod:     2,
				SSID:                 "SSID1",
			},
			expectError: false,
		},
		{
			name: "No Profile Found",
			setup: func(_ *sql.DB) {
			},
			profileName: "dontexist",
			tenantID:    "tenant1",
			expected:    nil,
			expectError: false,
		},
		{
			name: "Error in Pool.Query",
			setup: func(dbConn *sql.DB) {
				_, err := dbConn.ExecContext(context.Background(), `INSERT INTO ieee8021xconfigs (profile_name, 
        auth_protocol, 
        pxe_timeout, 
        wired_interface, 
        tenant_id) VALUES (?, ?, ?, ?, ?)`,
					"ieee1", 1, 30, true, "tenant1")
				require.NoError(t, err)

				_, err = dbConn.ExecContext(context.Background(), `
					INSERT INTO wirelessconfigs (
						wireless_profile_name, authentication_method, encryption_method, ssid, psk_value, link_policy, tenant_id,
						ieee8021x_profile_name
					) VALUES (?, ?, ?, ?, ?, ?, ?, ?);`,
					"profile1", 1, 2, "SSID1", "psk123", "policy1", "tenant1", "ieee1")
				require.NoError(t, err)

				dbConn.Close()
			},
			profileName: "profile1",
			tenantID:    "tenant1",
			expected:    nil,
			expectError: true,
		},
		{
			name: "Error in rows.Scan",
			setup: func(dbConn *sql.DB) {
				_, err := dbConn.ExecContext(context.Background(), `INSERT INTO ieee8021xconfigs (profile_name, 
        auth_protocol, 
        pxe_timeout, 
        wired_interface, 
        tenant_id) VALUES (?, ?, ?, ?, ?)`,
					"ieee1", 1, 30, true, "tenant1")
				require.NoError(t, err)

				_, err = dbConn.ExecContext(context.Background(), `
					INSERT INTO wirelessconfigs (
						wireless_profile_name, authentication_method, encryption_method, ssid, psk_value, link_policy, tenant_id,
						ieee8021x_profile_name
					) VALUES (?, ?, ?, ?, ?, ?, ?, ?);`,
					"profile1", 1, 2, "SSID1", "psk123", "policy1", "tenant1", "ieee1")
				require.NoError(t, err)
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

			dbConn, err := sql.Open("sqlite", ":memory:")
			require.NoError(t, err)

			defer dbConn.Close()

			_, err = dbConn.ExecContext(context.Background(), schema)
			require.NoError(t, err)

			tc.setup(dbConn)

			sqlConfig := &db.SQL{
				Builder:    squirrel.StatementBuilder.PlaceholderFormat(squirrel.Question),
				Pool:       dbConn,
				IsEmbedded: true,
			}

			repo := sqldb.NewWirelessRepo(sqlConfig, mocks.NewMockLogger(nil))

			wirelessConfig, err := repo.GetByName(context.Background(), tc.profileName, tc.tenantID)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if wirelessConfig == nil && tc.expected == nil {
				return
			}

			assert.IsType(t, tc.expected, wirelessConfig)
		})
	}
}

func TestWirelessRepo_Delete(t *testing.T) {
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
				_, err := dbConn.ExecContext(context.Background(), `
					INSERT INTO wirelessconfigs (
						wireless_profile_name, authentication_method, encryption_method, ssid, psk_value, link_policy, tenant_id
						
					) VALUES (?, ?, ?, ?, ?, ?, ?);`,
					"profile1", 1, 2, "SSID1", "psk123", "policy1", "tenant1")
				require.NoError(t, err)
			},
			profileName: "profile1",
			tenantID:    "tenant1",
			expected:    true,
			expectError: false,
		},
		{
			name: "Foreign key violation",
			setup: func(dbConn *sql.DB) {
				_, err := dbConn.ExecContext(context.Background(), `
					INSERT INTO wirelessconfigs (
						wireless_profile_name, authentication_method, encryption_method, ssid, psk_value, link_policy, tenant_id
					) VALUES (?, ?, ?, ?, ?, ?, ?);`,
					"wirelessProfile1", 1, 2, "SSID1", "psk123", "policy1", "tenant1")
				require.NoError(t, err)

				_, err = dbConn.ExecContext(context.Background(), `INSERT INTO profiles (
					profile_name, amt_password, creation_date, created_by, generate_random_password,
					activation, mebx_password, generate_random_mebx_password, tags,
					dhcp_enabled, ip_sync_enabled, local_wifi_sync_enabled, tenant_id, tls_mode, 
					tls_signing_authority, user_consent, ider_enabled, kvm_enabled, sol_enabled, uefi_wifi_sync_enabled
				) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					"profile1", "password1", "2024-08-01", "user1", true,
					"activation1", "mebx1", true, "tags1",
					true, true, true, "tenant1", 1,
					"authority1", "consent1", true, true, true, true,
				)
				require.NoError(t, err)

				_, err = dbConn.ExecContext(context.Background(), `INSERT INTO profiles_wirelessconfigs (
          profile_name, wireless_profile_name, priority, tenant_id
        ) VALUES (?, ?, ?, ?)`,
					"profile1", "wirelessProfile1", 1, "tenant1")
				require.NoError(t, err)
			},
			profileName: "wirelessProfile1",
			tenantID:    "tenant1",
			expected:    false,
			expectError: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dbConn, err := sql.Open("sqlite", ":memory:")
			require.NoError(t, err)

			defer dbConn.Close()

			_, err = dbConn.ExecContext(context.Background(), schema)
			require.NoError(t, err)

			tc.setup(dbConn)

			sqlConfig := &db.SQL{
				Builder:    squirrel.StatementBuilder.PlaceholderFormat(squirrel.Question),
				Pool:       dbConn,
				IsEmbedded: true,
			}

			mockLog := mocks.NewMockLogger(nil)
			repo := sqldb.NewWirelessRepo(sqlConfig, mockLog)

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

func TestWirelessRepo_Update(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		setup       func(dbConn *sql.DB)
		config      *entity.WirelessConfig
		expected    bool
		expectError bool
	}{
		{
			name: "Successful update",
			setup: func(dbConn *sql.DB) {
				_, err := dbConn.ExecContext(context.Background(), `
					CREATE TABLE wirelessconfigs (
						wireless_profile_name TEXT NOT NULL,
						authentication_method INTEGER NOT NULL,
						encryption_method INTEGER NOT NULL,
						ssid TEXT NOT NULL,
						psk_value TEXT,
						psk_passphrase TEXT,
						link_policy TEXT,
						ieee8021x_profile_name TEXT,
						tenant_id TEXT NOT NULL,
						PRIMARY KEY (wireless_profile_name, tenant_id)
					);
				`)
				require.NoError(t, err)

				_, err = dbConn.ExecContext(context.Background(), `
					INSERT INTO wirelessconfigs (
						wireless_profile_name, authentication_method, encryption_method, ssid, psk_value, psk_passphrase, link_policy, ieee8021x_profile_name, tenant_id
					) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);`,
					"profile1", 1, 2, "SSID1", "psk123", "passphrase", "policy1", "ieee1", "tenant1")
				require.NoError(t, err)
			},
			config: &entity.WirelessConfig{
				ProfileName:          "profile1",
				AuthenticationMethod: 2,
				EncryptionMethod:     3,
				SSID:                 "NewSSID",
				PSKPassphrase:        "newpassphrase",
				TenantID:             "tenant1",
			},
			expected:    true,
			expectError: false,
		},
		{
			name: "Update non-existent profile",
			setup: func(dbConn *sql.DB) {
				_, err := dbConn.ExecContext(context.Background(), `
					CREATE TABLE wirelessconfigs (
						wireless_profile_name TEXT NOT NULL,
						authentication_method INTEGER NOT NULL,
						encryption_method INTEGER NOT NULL,
						ssid TEXT NOT NULL,
						psk_value TEXT,
						psk_passphrase TEXT,
						link_policy TEXT,
						ieee8021x_profile_name TEXT,
						tenant_id TEXT NOT NULL,
						PRIMARY KEY (wireless_profile_name, tenant_id)
					);
				`)
				require.NoError(t, err)
			},
			config: &entity.WirelessConfig{
				ProfileName:          "nonexistent-profile",
				AuthenticationMethod: 2,
				EncryptionMethod:     3,
				SSID:                 "NewSSID",
				PSKPassphrase:        "newpassphrase",
				TenantID:             "tenant2",
			},
			expected:    false,
			expectError: false,
		},
		{
			name:  QueryExecutionErrorTestName,
			setup: func(_ *sql.DB) {},
			config: &entity.WirelessConfig{
				ProfileName:          "nonexistent-profile",
				AuthenticationMethod: 2,
				EncryptionMethod:     3,
				SSID:                 "NewSSID",
				PSKPassphrase:        "newpassphrase",
				TenantID:             "tenant2",
			},
			expected:    false,
			expectError: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dbConn, err := sql.Open("sqlite", ":memory:")
			require.NoError(t, err)

			defer dbConn.Close()

			tc.setup(dbConn)

			sqlConfig := &db.SQL{
				Builder:    squirrel.StatementBuilder.PlaceholderFormat(squirrel.Question),
				Pool:       dbConn,
				IsEmbedded: true,
			}

			repo := sqldb.NewWirelessRepo(sqlConfig, mocks.NewMockLogger(nil))

			updated, err := repo.Update(context.Background(), tc.config)

			if (err != nil) != tc.expectError {
				t.Errorf("Expected error status %v, got %v", tc.expectError, err != nil)
			}

			if updated != tc.expected {
				t.Errorf("Expected update status %v, got %v", tc.expected, updated)
			}
		})
	}
}

func TestWirelessRepo_Insert(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		setup       func(dbConn *sql.DB)
		wirelessCfg *entity.WirelessConfig
		expectedErr bool
	}{
		{
			name: "Successful insertion",
			setup: func(dbConn *sql.DB) {
				_, err := dbConn.ExecContext(context.Background(), `
					CREATE TABLE wirelessconfigs (
						wireless_profile_name TEXT NOT NULL PRIMARY KEY,
						authentication_method TEXT,
						encryption_method TEXT,
						ssid TEXT,
						psk_value TEXT,
						psk_passphrase TEXT,
						link_policy TEXT,
						creation_date TEXT,
						tenant_id TEXT NOT NULL,
						ieee8021x_profile_name TEXT
					);
				`)
				require.NoError(t, err)
			},
			wirelessCfg: &entity.WirelessConfig{
				ProfileName:          "profile1",
				SSID:                 "SSID1",
				PSKPassphrase:        "passphrase123",
				TenantID:             "tenant1",
				IEEE8021xProfileName: StringPtr("ieee1"),
			},
			expectedErr: false,
		},
		{
			name: "Insertion with non-unique profile name",
			setup: func(dbConn *sql.DB) {
				_, err := dbConn.ExecContext(context.Background(), `
					CREATE TABLE wirelessconfigs (
						wireless_profile_name TEXT NOT NULL PRIMARY KEY,
						authentication_method TEXT,
						encryption_method TEXT,
						ssid TEXT,
						psk_value TEXT,
						psk_passphrase TEXT,
						link_policy TEXT,
						creation_date TEXT,
						tenant_id TEXT NOT NULL,
						ieee8021x_profile_name TEXT
					);
				`)
				require.NoError(t, err)

				_, err = dbConn.ExecContext(context.Background(), `
					INSERT INTO wirelessconfigs (wireless_profile_name, authentication_method, encryption_method, ssid, psk_value, psk_passphrase, link_policy, creation_date, tenant_id, ieee8021x_profile_name)
					VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`,
					"profile1", "WPA2", "AES", "SSID1", "psk123", "passphrase123", "policy1", time.Now().Format("2006-01-02 15:04:05"), "tenant1", "ieee1")
				require.NoError(t, err)
			},
			wirelessCfg: &entity.WirelessConfig{
				ProfileName:          "profile1",
				SSID:                 "SSID2",
				PSKPassphrase:        "passphrase456",
				TenantID:             "tenant1",
				IEEE8021xProfileName: StringPtr("ieee2"),
			},
			expectedErr: true,
		},
		{
			name:  "Query execution error",
			setup: func(_ *sql.DB) {},
			wirelessCfg: &entity.WirelessConfig{
				ProfileName:          "profile2",
				SSID:                 "SSID3",
				PSKPassphrase:        "passphrase789",
				TenantID:             "tenant1",
				IEEE8021xProfileName: StringPtr("ieee3"),
			},
			expectedErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dbConn, err := sql.Open("sqlite", ":memory:")
			require.NoError(t, err)

			defer dbConn.Close()

			tc.setup(dbConn)

			sqlConfig := &db.SQL{
				Builder:    squirrel.StatementBuilder.PlaceholderFormat(squirrel.Question),
				Pool:       dbConn,
				IsEmbedded: true,
			}

			repo := sqldb.NewWirelessRepo(sqlConfig, mocks.NewMockLogger(nil))

			_, err = repo.Insert(context.Background(), tc.wirelessCfg)

			if (err != nil) != tc.expectedErr {
				t.Errorf("Expected error status %v, got %v", tc.expectedErr, err != nil)
			}

			if !tc.expectedErr {
				var count int

				err := dbConn.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM wirelessconfigs WHERE wireless_profile_name = ?`, tc.wirelessCfg.ProfileName).Scan(&count)
				require.NoError(t, err)

				if count == 0 {
					t.Errorf("Expected wireless config to be inserted, but it was not found in the database")
				}
			}
		})
	}
}
