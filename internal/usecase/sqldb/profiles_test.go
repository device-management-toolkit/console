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

const schema = `
CREATE TABLE IF NOT EXISTS devices
(
    guid TEXT NOT NULL,
    tags TEXT,
    hostname TEXT,
    mpsinstance TEXT,
    connectionstatus BOOLEAN NOT NULL,
    mpsusername TEXT,
    tenantid TEXT NOT NULL,
    friendlyname TEXT,
    dnssuffix TEXT,
    lastconnected TEXT,
    lastseen TEXT,
    lastdisconnected TEXT,
    deviceinfo TEXT,
    username TEXT,
    password TEXT,
    usetls BOOLEAN NOT NULL,
    allowselfsigned BOOLEAN NOT NULL,
    certhash TEXT,
    PRIMARY KEY (guid, tenantid),
    UNIQUE (guid)
);
CREATE TABLE IF NOT EXISTS ciraconfigs(
  cira_config_name TEXT NOT NULL,
  mps_server_address TEXT,
  mps_port INTEGER,
  user_name TEXT,
  password TEXT,
  common_name TEXT,
  server_address_format INTEGER,
  auth_method INTEGER,
  mps_root_certificate TEXT,
  proxydetails TEXT,
  tenant_id TEXT NOT NULL,
  PRIMARY KEY (cira_config_name, tenant_id)
);

CREATE TABLE IF NOT EXISTS ieee8021xconfigs(
  profile_name TEXT,
  auth_protocol INTEGER,
  pxe_timeout INTEGER,
  wired_interface BOOLEAN NOT NULL,
  tenant_id TEXT NOT NULL,
  PRIMARY KEY (profile_name, tenant_id)
);

CREATE TABLE IF NOT EXISTS wirelessconfigs(
  wireless_profile_name TEXT NOT NULL,
  authentication_method INTEGER,
  encryption_method INTEGER,
  ssid TEXT,
  psk_value INTEGER,
  psk_passphrase TEXT,
  link_policy TEXT,
  creation_date TEXT, -- TIMESTAMP is usually represented as TEXT in SQLite
  created_by TEXT,
  tenant_id TEXT NOT NULL,
  ieee8021x_profile_name TEXT,
  FOREIGN KEY (ieee8021x_profile_name, tenant_id) REFERENCES ieee8021xconfigs(profile_name, tenant_id),
  PRIMARY KEY (wireless_profile_name, tenant_id)
);

CREATE TABLE IF NOT EXISTS profiles(
  profile_name TEXT NOT NULL,
  activation TEXT NOT NULL,
  amt_password TEXT,
  generate_random_password BOOLEAN NOT NULL,
  cira_config_name TEXT,
  creation_date TEXT, -- TIMESTAMP as TEXT
  created_by TEXT,
  mebx_password TEXT,
  generate_random_mebx_password BOOLEAN NOT NULL, 
  tags TEXT,
  dhcp_enabled BOOLEAN NOT NULL, 
  tenant_id TEXT NOT NULL,
  tls_mode INTEGER,
  user_consent TEXT,
  ider_enabled BOOLEAN NOT NULL, 
  kvm_enabled BOOLEAN NOT NULL, 
  sol_enabled BOOLEAN NOT NULL, 
  tls_signing_authority TEXT,
  ip_sync_enabled BOOLEAN NOT NULL, 
  local_wifi_sync_enabled BOOLEAN NOT NULL, 
  ieee8021x_profile_name TEXT,
  FOREIGN KEY (ieee8021x_profile_name, tenant_id) REFERENCES ieee8021xconfigs(profile_name, tenant_id),
  FOREIGN KEY (cira_config_name, tenant_id) REFERENCES ciraconfigs(cira_config_name, tenant_id),
  PRIMARY KEY (profile_name, tenant_id)
);

CREATE TABLE IF NOT EXISTS profiles_wirelessconfigs(
  wireless_profile_name TEXT,
  profile_name TEXT,
  priority INTEGER,
  creation_date TEXT, -- TIMESTAMP as TEXT
  created_by TEXT,
  tenant_id TEXT NOT NULL,
  FOREIGN KEY (wireless_profile_name, tenant_id) REFERENCES wirelessconfigs(wireless_profile_name, tenant_id),
  FOREIGN KEY (profile_name, tenant_id) REFERENCES profiles(profile_name, tenant_id),
  PRIMARY KEY (wireless_profile_name, profile_name, priority, tenant_id)
);

CREATE TABLE IF NOT EXISTS domains(
  name TEXT NOT NULL,
  domain_suffix TEXT NOT NULL,
  provisioning_cert TEXT,
  provisioning_cert_storage_format TEXT,
  provisioning_cert_key TEXT,
  expiration_date TEXT,
  creation_date TEXT, -- TIMESTAMP as TEXT
  created_by TEXT,
  tenant_id TEXT NOT NULL,
  CONSTRAINT domainsuffix UNIQUE (domain_suffix, tenant_id),
  PRIMARY KEY (name, tenant_id)
);

CREATE UNIQUE INDEX lower_name_suffix_idx ON domains (LOWER(name), LOWER(domain_suffix));

PRAGMA foreign_keys = ON;
`

func TestProfileRepo_GetCount(t *testing.T) {
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
				_, err := dbConn.Exec(`INSERT INTO profiles (
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

			dbConn, err := sql.Open("sqlite", ":memory:")
			require.NoError(t, err)
			defer dbConn.Close()

			_, err = dbConn.Exec(schema)
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
			repo := sqldb.NewProfileRepo(sqlConfig, mockLog)

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

func StringPtr(s string) *string {
	return &s
}

func IntPtr(i int) *int {
	return &i
}

func BoolPtr(b bool) *bool {
	return &b
}

func GetProfileHelper(t *testing.T, tc GetProfileTestCase, profiles []entity.Profile, err error) {
	t.Helper()

	if (err != nil) != (tc.err != nil) {
		t.Errorf("Expected error: %v, got: %v", tc.err, err)
	}

	if len(profiles) == 0 && len(tc.expected) > 0 {
		t.Errorf("Expected %d profiles, got %d", len(tc.expected), len(profiles))

		return
	}

	if len(profiles) != len(tc.expected) {
		t.Errorf("Expected %d profiles, got %d", len(tc.expected), len(profiles))
	}

	for i := range tc.expected {
		expectedProfile := &tc.expected[i]

		if i >= len(profiles) {
			t.Errorf("Expected profile %d, but got none", i)

			break
		}

		actualProfile := &profiles[i]
		assert.IsType(t, expectedProfile, actualProfile)
	}
}

type GetProfileTestCase struct {
	name     string
	setup    func(dbConn *sql.DB)
	top      int
	skip     int
	tenantID string
	expected []entity.Profile
	err      error
}

func TestProfileRepo_Get(t *testing.T) {
	t.Parallel()

	tests := []GetProfileTestCase{
		{
			name: "Successful query",
			setup: func(dbConn *sql.DB) {
				_, err := dbConn.Exec(`INSERT INTO wirelessconfigs (
		      wireless_profile_name, authentication_method, encryption_method, ssid, psk_value, psk_passphrase, link_policy, creation_date, created_by, tenant_id
		      ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					"wireless1", 1, 1, "ssid1", 1, "passphrase1", "policy1", "2024-08-01", "user1", "tenant1")
				require.NoError(t, err)

				_, err = dbConn.Exec(`INSERT INTO ciraconfigs (cira_config_name, mps_server_address, mps_port, user_name, password, common_name, server_address_format, auth_method, mps_root_certificate, proxydetails, tenant_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					"cira1", "mpsaddress1", 1234, "user1", "pass1", "common1", 1, 1, "rootcert1", "proxydetail1", "tenant1")
				require.NoError(t, err)

				_, err = dbConn.Exec(`
		            INSERT INTO ieee8021xconfigs (
		                profile_name, pxe_timeout, wired_interface, tenant_id
		            ) VALUES (?, ?, ?, ?);`,
					"ieee1", 30, true, "tenant1")
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
			},
			top:      10,
			skip:     0,
			tenantID: "tenant1",
			expected: []entity.Profile{
				{
					ProfileName:                "profile1",
					AMTPassword:                "password1",
					CreationDate:               "2024-08-01",
					CreatedBy:                  "user1",
					GenerateRandomPassword:     true,
					CIRAConfigName:             StringPtr("cira1"),
					Activation:                 "activation1",
					MEBXPassword:               "mebx1",
					GenerateRandomMEBxPassword: true,
					Tags:                       "tags1",
					DHCPEnabled:                true,
					IPSyncEnabled:              true,
					LocalWiFiSyncEnabled:       true,
					TenantID:                   "tenant1",
					TLSMode:                    1,
					TLSSigningAuthority:        "authority1",
					UserConsent:                "consent1",
					IDEREnabled:                true,
					KVMEnabled:                 true,
					SOLEnabled:                 true,
					IEEE8021xProfileName:       StringPtr("ieee1"),
				},
			},
			err: nil,
		},
		{
			name: "No profiles found",
			setup: func(_ *sql.DB) {
			},
			top:      10,
			skip:     0,
			tenantID: "tenant2",
			expected: []entity.Profile{},
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

			_, err = dbConn.Exec(schema)
			require.NoError(t, err)

			tc.setup(dbConn)

			sqlConfig := &db.SQL{
				Builder:    squirrel.StatementBuilder.PlaceholderFormat(squirrel.Question),
				Pool:       dbConn,
				IsEmbedded: true,
			}

			mockLog := mocks.NewMockLogger(nil)
			repo := sqldb.NewProfileRepo(sqlConfig, mockLog)

			profiles, err := repo.Get(context.Background(), tc.top, tc.skip, tc.tenantID)

			GetProfileHelper(t, tc, profiles, err)
		})
	}
}

func TestProfileRepo_GetByName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		setup       func(dbConn *sql.DB)
		profileName string
		tenantID    string
		expected    *entity.Profile
		expectError bool
	}{
		{
			name: "Successful retrieval",
			setup: func(dbConn *sql.DB) {
				_, err := dbConn.Exec(`
        INSERT INTO ieee8021xconfigs (
            profile_name, pxe_timeout, wired_interface, tenant_id
        ) VALUES (?, ?, ?, ?);`,
					"ieee1", 30, true, "tenant1")
				require.NoError(t, err)

				_, err = dbConn.Exec(`INSERT INTO profiles (
					profile_name, amt_password, creation_date, created_by, generate_random_password,
					activation, mebx_password, generate_random_mebx_password, tags,
					dhcp_enabled, ip_sync_enabled, local_wifi_sync_enabled, tenant_id, tls_mode,
					tls_signing_authority, user_consent, ider_enabled, kvm_enabled, sol_enabled,
					ieee8021x_profile_name
				) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					"profile1", "password1", "2024-08-01", "user1", true,
					"activation1", "mebx1", true, "tags1",
					true, true, true, "tenant1", 1,
					"authority1", "consent1", true, true, true,
					"ieee1")
				require.NoError(t, err)
			},
			profileName: "profile1",
			tenantID:    "tenant1",
			expected:    &entity.Profile{},
			expectError: false,
		},
		{
			name: "No Profile Found",
			setup: func(dbConn *sql.DB) {
				_, err := dbConn.Exec(`
		        INSERT INTO ieee8021xconfigs (
		            profile_name, pxe_timeout, wired_interface, tenant_id
		        ) VALUES (?, ?, ?, ?);`,
					"profile1", 30, true, "tenant1")
				require.NoError(t, err)
			},
			profileName: "dontexist",
			tenantID:    "tenant1",
			expected: &entity.Profile{
				ProfileName:                "profile1",
				AMTPassword:                "password1",
				CreationDate:               "2024-08-01",
				CreatedBy:                  "user1",
				GenerateRandomPassword:     true,
				CIRAConfigName:             StringPtr("cira1"),
				Activation:                 "activation1",
				MEBXPassword:               "mebx1",
				GenerateRandomMEBxPassword: true,
				Tags:                       "tags1",
				DHCPEnabled:                true,
				IPSyncEnabled:              true,
				LocalWiFiSyncEnabled:       true,
				TenantID:                   "tenant1",
				TLSMode:                    1,
				TLSSigningAuthority:        "authority1",
				UserConsent:                "consent1",
				IDEREnabled:                true,
				KVMEnabled:                 true,
				SOLEnabled:                 true,
				IEEE8021xProfileName:       StringPtr("ieee1"),
				Version:                    "v1",
				AuthenticationProtocol:     IntPtr(1),
				ServerName:                 "server1",
				Domain:                     "domain1",
				Username:                   "user1",
				Password:                   "pass1",
				RoamingIdentity:            "identity1",
				ActiveInS0:                 true,
				PXETimeout:                 IntPtr(10),
				WiredInterface:             BoolPtr(true),
			},
			expectError: false,
		},
		// {
		// 	name:        "Query execution error",
		// 	setup:       func(_ *sql.DB) {},
		// 	profileName: "b",
		// 	tenantID:    "tenant1",
		// 	expected:    nil,
		// 	expectError: true,
		// },
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dbConn, err := sql.Open("sqlite", ":memory:")
			require.NoError(t, err)
			defer dbConn.Close()

			_, err = dbConn.Exec(schema)
			require.NoError(t, err)

			tc.setup(dbConn)

			sqlConfig := &db.SQL{
				Builder:    squirrel.StatementBuilder.PlaceholderFormat(squirrel.Question),
				Pool:       dbConn,
				IsEmbedded: true,
			}

			repo := sqldb.NewProfileRepo(sqlConfig, mocks.NewMockLogger(nil))

			profile, err := repo.GetByName(context.Background(), tc.profileName, tc.tenantID)

			if (err != nil) != tc.expectError {
				t.Errorf("Expected error status %v, got %v", tc.expectError, err != nil)
			}

			if profile == nil && tc.expected == nil {
				return
			}

			assert.IsType(t, tc.expected, profile)
			assert.IsType(t, tc.expected, profile)
		})
	}
}

func TestProfileRepo_Delete(t *testing.T) {
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
				_, err := dbConn.Exec(`INSERT INTO profiles (profile_name, activation, generate_random_password, generate_random_mebx_password, tags, dhcp_enabled, tenant_id, tls_mode, user_consent, ider_enabled, kvm_enabled, sol_enabled, tls_signing_authority, ip_sync_enabled, local_wifi_sync_enabled) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					"profile1", true, true, true, "tag1", true, "tenant1", "tls", true, true, true, true, "authority", true, true)
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

			dbConn, err := sql.Open("sqlite", ":memory:")
			require.NoError(t, err)
			defer dbConn.Close()

			_, err = dbConn.Exec(schema)
			require.NoError(t, err)

			tc.setup(dbConn)

			sqlConfig := &db.SQL{
				Builder:    squirrel.StatementBuilder.PlaceholderFormat(squirrel.Question),
				Pool:       dbConn,
				IsEmbedded: true,
			}

			mockLog := mocks.NewMockLogger(nil)
			repo := sqldb.NewProfileRepo(sqlConfig, mockLog)

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

func TestProfileRepo_Update(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		setup       func(dbConn *sql.DB)
		profile     *entity.Profile
		expected    bool
		expectError bool
	}{
		{
			name: "Successful update",
			setup: func(dbConn *sql.DB) {
				_, err := dbConn.Exec(`
        INSERT INTO ieee8021xconfigs (
            profile_name, pxe_timeout, wired_interface, tenant_id
        ) VALUES (?, ?, ?, ?);`,
					"ieee1", 30, true, "tenant1")
				require.NoError(t, err)

				_, err = dbConn.Exec(`
        INSERT INTO ieee8021xconfigs (
            profile_name, pxe_timeout, wired_interface, tenant_id
        ) VALUES (?, ?, ?, ?);`,
					"new-ieee", 30, true, "tenant1")
				require.NoError(t, err)

				_, err = dbConn.Exec(`INSERT INTO ciraconfigs (cira_config_name, tenant_id) VALUES (?,?)`, "cira1", "tenant1")
				require.NoError(t, err)

				_, err = dbConn.Exec(`INSERT INTO ciraconfigs (cira_config_name, tenant_id) VALUES (?,?)`, "new-cira", "tenant1")
				require.NoError(t, err)

				_, err = dbConn.Exec(`INSERT INTO profiles (
					profile_name, amt_password, creation_date, created_by, generate_random_password,
					cira_config_name, activation, mebx_password, generate_random_mebx_password, tags,
					dhcp_enabled, ip_sync_enabled, local_wifi_sync_enabled, tenant_id, tls_mode,
					tls_signing_authority, user_consent, ider_enabled, kvm_enabled, sol_enabled,
					ieee8021x_profile_name
				) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					"profile1", "password1", "2024-08-01", "user1", true,
					"cira1", "activation1", "mebx1", true, "tags1",
					true, true, true, "tenant1", 1,
					"authority1", "consent1", true, true, true,
					"ieee1")
				require.NoError(t, err)
			},
			profile: &entity.Profile{
				ProfileName:                "profile1",
				AMTPassword:                "new-password",
				GenerateRandomPassword:     false,
				CIRAConfigName:             StringPtr("new-cira"),
				Activation:                 "new-activation",
				MEBXPassword:               "new-mebx",
				GenerateRandomMEBxPassword: false,
				Tags:                       "new-tags",
				DHCPEnabled:                false,
				IPSyncEnabled:              false,
				LocalWiFiSyncEnabled:       false,
				TenantID:                   "tenant1",
				TLSMode:                    2,
				TLSSigningAuthority:        "new-authority",
				UserConsent:                "new-consent",
				IDEREnabled:                false,
				KVMEnabled:                 false,
				SOLEnabled:                 false,
				IEEE8021xProfileName:       StringPtr("new-ieee"),
				Version:                    "v2",
				AuthenticationProtocol:     IntPtr(2),
				ServerName:                 "new-server",
				Domain:                     "new-domain",
				Username:                   "new-user",
				Password:                   "new-pass",
				RoamingIdentity:            "new-identity",
				ActiveInS0:                 false,
				PXETimeout:                 IntPtr(20),
				WiredInterface:             BoolPtr(false),
			},
			expected:    true,
			expectError: false,
		},
		{
			name:  "Update non-existent profile",
			setup: func(_ *sql.DB) {},
			profile: &entity.Profile{
				ProfileName:                "nonexistent-profile",
				AMTPassword:                "password",
				GenerateRandomPassword:     true,
				CIRAConfigName:             StringPtr("cira"),
				Activation:                 "activation",
				MEBXPassword:               "mebx",
				GenerateRandomMEBxPassword: true,
				Tags:                       "tags",
				DHCPEnabled:                true,
				IPSyncEnabled:              true,
				LocalWiFiSyncEnabled:       true,
				TenantID:                   "tenant1",
				TLSMode:                    1,
				TLSSigningAuthority:        "authority",
				UserConsent:                "consent",
				IDEREnabled:                true,
				KVMEnabled:                 true,
				SOLEnabled:                 true,
				IEEE8021xProfileName:       StringPtr("ieee"),
				Version:                    "v1",
				AuthenticationProtocol:     IntPtr(1),
				ServerName:                 "server",
				Domain:                     "domain",
				Username:                   "user",
				Password:                   "pass",
				RoamingIdentity:            "identity",
				ActiveInS0:                 true,
				PXETimeout:                 IntPtr(10),
				WiredInterface:             BoolPtr(true),
			},
			expected:    false,
			expectError: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dbConn, err := sql.Open("sqlite", ":memory:")
			require.NoError(t, err)
			defer dbConn.Close()

			_, err = dbConn.Exec(schema)
			require.NoError(t, err)

			tc.setup(dbConn)

			sqlConfig := &db.SQL{
				Builder:    squirrel.StatementBuilder.PlaceholderFormat(squirrel.Question),
				Pool:       dbConn,
				IsEmbedded: true,
			}

			repo := sqldb.NewProfileRepo(sqlConfig, mocks.NewMockLogger(nil))

			updated, err := repo.Update(context.Background(), tc.profile)

			if (err != nil) != tc.expectError {
				t.Errorf("Expected error status %v, got %v", tc.expectError, err != nil)
			}

			if updated != tc.expected {
				t.Errorf("Expected update status %v, got %v", tc.expected, updated)
			}
		})
	}
}

func TestProfileRepo_Insert(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		setup       func(dbConn *sql.DB)
		profile     *entity.Profile
		expectedErr bool
	}{
		{
			name: "Successful insertion",
			setup: func(dbConn *sql.DB) {
				_, err := dbConn.Exec(schema)
				require.NoError(t, err)

				_, err = dbConn.Exec(`INSERT INTO ciraconfigs (cira_config_name, tenant_id) VALUES (?,?)`, "cira1", "tenant1")
				require.NoError(t, err)

				_, err = dbConn.Exec(`INSERT INTO ieee8021xconfigs (profile_name,wired_interface, tenant_id) VALUES (?, ?, ?)`,
					"ieee1", 1, "tenant1")
				require.NoError(t, err)
			},
			profile: &entity.Profile{
				ProfileName:                "profile1",
				Activation:                 "activation1",
				AMTPassword:                "password1",
				GenerateRandomPassword:     true,
				CIRAConfigName:             StringPtr("cira1"),
				MEBXPassword:               "mebx1",
				GenerateRandomMEBxPassword: true,
				Tags:                       "tags1",
				DHCPEnabled:                true,
				TLSMode:                    1,
				UserConsent:                "consent1",
				IDEREnabled:                true,
				KVMEnabled:                 true,
				SOLEnabled:                 true,
				TLSSigningAuthority:        "authority1",
				IEEE8021xProfileName:       StringPtr("ieee1"),
				IPSyncEnabled:              true,
				LocalWiFiSyncEnabled:       true,
				TenantID:                   "tenant1",
			},
			expectedErr: false,
		},
		{
			name: "Insertion with non-unique profile name",
			setup: func(dbConn *sql.DB) {
				_, err := dbConn.Exec(schema)
				require.NoError(t, err)

				_, err = dbConn.Exec(`INSERT INTO ieee8021xconfigs (profile_name, 
        auth_protocol, 
        pxe_timeout, 
        wired_interface, 
        tenant_id) VALUES (?, ?, ?, ?, ?)`,
					"ieee1", 1, 30, true, "tenant1")
				require.NoError(t, err)

				_, err = dbConn.Exec(`INSERT INTO ciraconfigs (cira_config_name, tenant_id) VALUES (?,?)`, "cira1", "tenant1")
				require.NoError(t, err)

				_, err = dbConn.Exec(`
				INSERT INTO profiles (profile_name, activation, amt_password, generate_random_password, cira_config_name, mebx_password, generate_random_mebx_password, tags, dhcp_enabled, tls_mode, user_consent, ider_enabled, kvm_enabled, sol_enabled, tls_signing_authority, ieee8021x_profile_name, ip_sync_enabled, local_wifi_sync_enabled, tenant_id)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`,
					"profile1", "activation1", "password1", true, "cira1", "mebx1", true, "tags1", true, 1, "consent1", true, true, true, "authority1", "ieee1", true, true, "tenant1")
				require.NoError(t, err)
			},
			profile: &entity.Profile{
				ProfileName:                "profile1",
				Activation:                 "activation2",
				AMTPassword:                "password2",
				GenerateRandomPassword:     true,
				CIRAConfigName:             StringPtr("cira1"),
				MEBXPassword:               "mebx2",
				GenerateRandomMEBxPassword: true,
				Tags:                       "tags2",
				DHCPEnabled:                true,
				TLSMode:                    1,
				UserConsent:                "consent2",
				IDEREnabled:                true,
				KVMEnabled:                 true,
				SOLEnabled:                 true,
				TLSSigningAuthority:        "authority2",
				IEEE8021xProfileName:       StringPtr("ieee1"),
				IPSyncEnabled:              true,
				LocalWiFiSyncEnabled:       true,
				TenantID:                   "tenant1",
			},
			expectedErr: true,
		},
		{
			name:  "Query execution error",
			setup: func(_ *sql.DB) {},
			profile: &entity.Profile{
				ProfileName: "profile2",
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

			repo := sqldb.NewProfileRepo(sqlConfig, mocks.NewMockLogger(nil))

			_, err = repo.Insert(context.Background(), tc.profile)

			if (err != nil) != tc.expectedErr {
				t.Errorf("Expected error status %v, got %v", tc.expectedErr, err != nil)
			}

			if !tc.expectedErr {
				var count int
				err := dbConn.QueryRow(`SELECT COUNT(*) FROM profiles WHERE profile_name = ?`, tc.profile.ProfileName).Scan(&count)
				require.NoError(t, err)

				if count == 0 {
					t.Errorf("Expected profile to be inserted, but it was not found in the database")
				}
			}
		})
	}
}
