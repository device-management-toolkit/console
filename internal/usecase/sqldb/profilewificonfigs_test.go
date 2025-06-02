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

var ErrProfileWiFiConfigsForeignKeyViolation = errors.New("foreign key violation in ProfileWiFiConfigs")

func setupProfileWifiConfigsTable(t *testing.T) *sql.DB {
	t.Helper()

	dbConn, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	_, err = dbConn.Exec(schema)
	require.NoError(t, err)

	return dbConn
}

func TestProfileWiFiConfigsRepo_GetByProfileName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		setup       func(dbConn *sql.DB)
		profileName string
		tenantID    string
		expected    []entity.ProfileWiFiConfigs
		expectError bool
	}{
		{
			name: "Successful retrieval",
			setup: func(dbConn *sql.DB) {
				_, err := dbConn.Exec(`INSERT INTO wirelessconfigs (wireless_profile_name, tenant_id) VALUES (?, ?)`, "wireless1", "tenant1")
				require.NoError(t, err)

				_, err = dbConn.Exec(`INSERT INTO profiles (
					profile_name, amt_password, creation_date, created_by, generate_random_password,
					 activation, mebx_password, generate_random_mebx_password, tags,
					dhcp_enabled, ip_sync_enabled, local_wifi_sync_enabled, tenant_id, tls_mode, 
					tls_signing_authority, user_consent, ider_enabled, kvm_enabled, sol_enabled
					
				) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					"profile1", "password1", "2024-08-01", "user1", true,
					"activation1", "mebx1", true, "tags1",
					true, true, true, "tenant1", 1,
					"authority1", "consent1", true, true, true,
				)
				require.NoError(t, err)

				_, err = dbConn.Exec(`INSERT INTO profiles_wirelessconfigs (
					wireless_profile_name, profile_name, priority, tenant_id
				) VALUES (?, ?, ?, ?);`,
					"wireless1", "profile1", 1, "tenant1")
				require.NoError(t, err)
			},
			profileName: "profile1",
			tenantID:    "tenant1",
			expected: []entity.ProfileWiFiConfigs{
				{
					WirelessProfileName: "wireless1",
					ProfileName:         "profile1",
					Priority:            1,
					TenantID:            "tenant1",
				},
			},
			expectError: false,
		},
		{
			name: "No Profile Found",
			setup: func(_ *sql.DB) {
			},
			profileName: "nonexistent",
			tenantID:    "tenant1",
			expected:    []entity.ProfileWiFiConfigs(nil),
			expectError: false,
		},
		// {
		// 	name:        QueryExecutionErrorTestName,
		// 	setup:       func(_ *sql.DB) {},
		// 	profileName: "profile1",
		// 	tenantID:    "tenant1",
		// 	expected:    nil,
		// 	expectError: true,
		// },
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dbConn := setupProfileWifiConfigsTable(t)
			defer dbConn.Close()

			tc.setup(dbConn)

			sqlConfig := &db.SQL{
				Builder:    squirrel.StatementBuilder.PlaceholderFormat(squirrel.Question),
				Pool:       dbConn,
				IsEmbedded: true,
			}

			repo := sqldb.NewProfileWiFiConfigsRepo(sqlConfig, mocks.NewMockLogger(nil))

			configs, err := repo.GetByProfileName(context.Background(), tc.profileName, tc.tenantID)

			if (err != nil) != tc.expectError {
				t.Errorf("Expected error status %v, got %v", tc.expectError, err != nil)
			}

			if configs == nil && tc.expected == nil {
				return
			}

			assert.Equal(t, tc.expected, configs)
		})
	}
}

func TestProfileWiFiConfigsRepo_DeleteByProfileName(t *testing.T) {
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
				_, err := dbConn.Exec(`INSERT INTO wirelessconfigs (
          wireless_profile_name, authentication_method, encryption_method, ssid, psk_value, psk_passphrase, link_policy, creation_date, created_by, tenant_id
          ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					"wirelessProfile1", 1, 1, "ssid1", 1, "passphrase1", "policy1", "2024-08-01", "user1", "tenant1")
				require.NoError(t, err)
				_, err = dbConn.Exec(`INSERT INTO profiles (
					profile_name, amt_password, creation_date, created_by, generate_random_password,
					activation, mebx_password, generate_random_mebx_password, tags,
					dhcp_enabled, ip_sync_enabled, local_wifi_sync_enabled, tenant_id, tls_mode,
					tls_signing_authority, user_consent, ider_enabled, kvm_enabled, sol_enabled
				) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					"profile1", "password1", "2024-08-01", "user1", true,
					"activation1", "mebx1", true, "tags1",
					true, true, true, "tenant1", 1,
					"authority1", "consent1", true, true, true)
				require.NoError(t, err)

				_, err = dbConn.Exec(`INSERT INTO profiles_wirelessconfigs (profile_name, wireless_profile_name, priority, tenant_id) VALUES (?, ?, ?, ?)`,
					"profile1", "wirelessProfile1", 1, "tenant1")
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
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dbConn := setupProfileWifiConfigsTable(t)
			defer dbConn.Close()

			tc.setup(dbConn)

			sqlConfig := &db.SQL{
				Builder:    squirrel.StatementBuilder.PlaceholderFormat(squirrel.Question),
				Pool:       dbConn,
				IsEmbedded: true,
			}

			mockLog := mocks.NewMockLogger(nil)

			repo := sqldb.NewProfileWiFiConfigsRepo(sqlConfig, mockLog)

			deleted, err := repo.DeleteByProfileName(context.Background(), tc.profileName, tc.tenantID)

			if (err != nil) != tc.expectError {
				t.Errorf("Expected error status %v, got %v", tc.expectError, err != nil)
			}

			if deleted != tc.expected {
				t.Errorf("Expected deleted status %v, got %v", tc.expected, deleted)
			}
		})
	}
}

func TestProfileWiFiConfigsRepo_Insert(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		setup       func(dbConn *sql.DB)
		profile     *entity.ProfileWiFiConfigs
		expectedErr bool
	}{
		{
			name: "Successful insertion",
			setup: func(dbConn *sql.DB) {
				_, err := dbConn.Exec(`INSERT INTO wirelessconfigs (
		      wireless_profile_name, authentication_method, encryption_method, ssid, psk_value, psk_passphrase, link_policy, creation_date, created_by, tenant_id
		      ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					"wifiProfile1", 1, 1, "ssid1", 1, "passphrase1", "policy1", "2024-08-01", "user1", "tenant1")
				require.NoError(t, err)
				_, err = dbConn.Exec(`INSERT INTO profiles (
          			profile_name, amt_password, creation_date, created_by, generate_random_password,
          			activation, mebx_password, generate_random_mebx_password, tags,
          			dhcp_enabled, ip_sync_enabled, local_wifi_sync_enabled, tenant_id, tls_mode,
          			tls_signing_authority, user_consent, ider_enabled, kvm_enabled, sol_enabled
          		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					"profile1", "password1", "2024-08-01", "user1", true,
					"activation1", "mebx1", true, "tags1",
					true, true, true, "tenant1", 1,
					"authority1", "consent1", true, true, true)
				require.NoError(t, err)
			},
			profile: &entity.ProfileWiFiConfigs{
				WirelessProfileName: "wifiProfile1",
				ProfileName:         "profile1",
				Priority:            1,
				TenantID:            "tenant1",
			},
			expectedErr: false,
		},
		{
			name: "Insertion with non-unique profile",
			setup: func(dbConn *sql.DB) {
				_, err := dbConn.Exec(`INSERT INTO wirelessconfigs (
		      wireless_profile_name, authentication_method, encryption_method, ssid, psk_value, psk_passphrase, link_policy, creation_date, created_by, tenant_id
		      ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					"wifiProfile1", 1, 1, "ssid1", 1, "passphrase1", "policy1", "2024-08-01", "user1", "tenant1")
				require.NoError(t, err)
				_, err = dbConn.Exec(`INSERT INTO profiles (
					profile_name, amt_password, creation_date, created_by, generate_random_password,
					activation, mebx_password, generate_random_mebx_password, tags,
					dhcp_enabled, ip_sync_enabled, local_wifi_sync_enabled, tenant_id, tls_mode,
					tls_signing_authority, user_consent, ider_enabled, kvm_enabled, sol_enabled
				) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					"profile1", "password1", "2024-08-01", "user1", true,
					"activation1", "mebx1", true, "tags1",
					true, true, true, "tenant1", 1,
					"authority1", "consent1", true, true, true)
				require.NoError(t, err)

				_, err = dbConn.Exec(`
				INSERT INTO profiles_wirelessconfigs (wireless_profile_name, profile_name, priority, tenant_id)
				VALUES (?, ?, ?, ?);`,
					"wifiProfile1", "profile1", 2, "tenant1")
				require.NoError(t, err)
			},
			profile: &entity.ProfileWiFiConfigs{
				WirelessProfileName: "wifiProfile1",
				ProfileName:         "profile1",
				Priority:            2,
				TenantID:            "tenant1",
			},
			expectedErr: true,
		},
		{
			name:  QueryExecutionErrorTestName,
			setup: func(_ *sql.DB) {},
			profile: &entity.ProfileWiFiConfigs{
				WirelessProfileName: "wifiProfile2",
				ProfileName:         "profile2",
				Priority:            1,
				TenantID:            "tenant1",
			},
			expectedErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dbConn := setupProfileWifiConfigsTable(t)
			defer dbConn.Close()

			tc.setup(dbConn)

			sqlConfig := &db.SQL{
				Builder:    squirrel.StatementBuilder.PlaceholderFormat(squirrel.Question),
				Pool:       dbConn,
				IsEmbedded: true,
			}

			repo := sqldb.NewProfileWiFiConfigsRepo(sqlConfig, mocks.NewMockLogger(nil))

			_, err := repo.Insert(context.Background(), tc.profile)

			if (err != nil) != tc.expectedErr {
				t.Errorf("Expected error status %v, got %v", tc.expectedErr, err != nil)
			}

			if !tc.expectedErr {
				var count int
				err := dbConn.QueryRow(`SELECT COUNT(*) FROM profiles_wirelessconfigs WHERE wireless_profile_name = ? AND profile_name = ?`,
					tc.profile.WirelessProfileName, tc.profile.ProfileName).Scan(&count)
				require.NoError(t, err)

				if count == 0 {
					t.Errorf("Expected profile to be inserted, but it was not found in the database")
				}
			}
		})
	}
}
