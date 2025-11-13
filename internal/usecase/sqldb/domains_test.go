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
)

func setupDomainTable(t *testing.T) *sql.DB {
	t.Helper()

	dbConn, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	_, err = dbConn.ExecContext(context.Background(), schema)
	require.NoError(t, err)

	return dbConn
}

func TestDomainRepo_GetCount(t *testing.T) {
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
				_, err := dbConn.ExecContext(context.Background(), `INSERT INTO domains (name,domain_suffix, tenant_id) VALUES (?,?,?)`,
					"domain1", "suffix.com", "tenant1")
				require.NoError(t, err)
			},
			tenantID: "tenant1",
			expected: 1,
			err:      nil,
		},
		{
			name:     "No domains found",
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

			dbConn := setupDomainTable(t)
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
			repo := sqldb.NewDomainRepo(sqlConfig, mockLog)

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

func TestDomainRepo_Get(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		setup    func(dbConn *sql.DB)
		top      int
		skip     int
		tenantID string
		expected []entity.Domain
		err      error
	}{
		{
			name: "Successful query",
			setup: func(dbConn *sql.DB) {
				_, err := dbConn.ExecContext(context.Background(), `INSERT INTO domains (name, domain_suffix, provisioning_cert_storage_format, provisioning_cert, provisioning_cert_key, expiration_date, tenant_id) VALUES (?, ?, ?, ?, ?, ?, ?)`,
					"domain1", "suffix1", "cert_format1", "cert", "cert-key", "2024-12-31", "tenant1")
				require.NoError(t, err)
			},
			top:      10,
			skip:     0,
			tenantID: "tenant1",
			expected: []entity.Domain{
				{
					ProfileName:                   "domain1",
					DomainSuffix:                  "suffix1",
					ProvisioningCert:              "cert",
					ProvisioningCertStorageFormat: "cert_format1",
					ProvisioningCertPassword:      "cert-key",
					ExpirationDate:                "2024-12-31",
					TenantID:                      "tenant1",
				},
			},
			err: nil,
		},
		{
			name:     "No domains found",
			setup:    func(_ *sql.DB) {},
			top:      10,
			skip:     0,
			tenantID: "tenant2",
			expected: []entity.Domain{},
			err:      nil,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dbConn := setupDomainTable(t)
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
			repo := sqldb.NewDomainRepo(sqlConfig, mockLog)

			domains, err := repo.Get(context.Background(), tc.top, tc.skip, tc.tenantID)

			if tc.err != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Len(t, domains, len(tc.expected))
		})
	}
}

type GetDomainByDomainSuffixTestCase struct {
	name         string
	setup        func(dbConn *sql.DB)
	domainSuffix string
	tenantID     string
	expected     *entity.Domain
	err          error
}

func GetDomainByDomainSuffixHelper(t *testing.T, tc GetDomainByDomainSuffixTestCase, domain *entity.Domain, err error) {
	t.Helper()

	if domain == nil && tc.expected == nil {
		return
	}

	if err == nil && tc.err != nil {
		t.Errorf("Expected error of type %T, got nil", tc.err)
	} else if err != nil {
		var dbError sqldb.DatabaseError

		if !errors.As(err, &dbError) {
			t.Errorf("Expected error of type %T, got %T", tc.err, err)
		}
	}

	if domain == nil && tc.expected == nil {
		return
	}

	assert.IsType(t, tc.expected, domain)
}

func TestDomainRepo_GetDomainByDomainSuffix(t *testing.T) {
	t.Parallel()

	tests := []GetDomainByDomainSuffixTestCase{
		{
			name: "Successful query",
			setup: func(dbConn *sql.DB) {
				_, err := dbConn.ExecContext(context.Background(), `INSERT INTO domains (name, domain_suffix, provisioning_cert, provisioning_cert_storage_format, provisioning_cert_key, expiration_date, tenant_id) VALUES (?, ?, ?, ?, ?, ?, ?)`,
					"domain1", "suffix1", "cert1", "format1", "key1", "2024-12-31", "tenant1")
				require.NoError(t, err)
			},
			domainSuffix: "suffix1",
			tenantID:     "tenant1",
			expected: &entity.Domain{
				ProfileName:                   "domain1",
				DomainSuffix:                  "suffix1",
				ProvisioningCert:              "cert1",
				ProvisioningCertStorageFormat: "format1",
				ProvisioningCertPassword:      "key1",
				ExpirationDate:                "2024-12-31",
				TenantID:                      "tenant1",
			},
			err: nil,
		},
		{
			name:         "No domain found",
			setup:        func(_ *sql.DB) {},
			domainSuffix: "suffix2",
			tenantID:     "tenant2",
			expected:     nil,
			err:          nil,
		},
		{
			name:         "Query execution error",
			setup:        func(_ *sql.DB) {},
			domainSuffix: "suffix1",
			tenantID:     "tenant1",
			expected:     nil,
			err:          &sqldb.DatabaseError{},
		},
		{
			name: "Rows scan error",
			setup: func(dbConn *sql.DB) {
				_, _ = dbConn.ExecContext(context.Background(), `INSERT INTO domains (name, domain_suffix, provisioning_cert, provisioning_cert_storage_format, provisioning_cert_key, expiration_date, tenant_id) VALUES (?, ?, ?, ?, ?, ?, ?)`,
					"domain1", "suffix1", "cert1", "format1", "key1", "invalid-date", "tenant1")
			},
			domainSuffix: "suffix1",
			tenantID:     "tenant1",
			expected:     nil,
			err:          &sqldb.DatabaseError{},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dbConn := setupDomainTable(t)
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
			repo := sqldb.NewDomainRepo(sqlConfig, mockLog)

			domain, err := repo.GetDomainByDomainSuffix(context.Background(), tc.domainSuffix, tc.tenantID)

			GetDomainByDomainSuffixHelper(t, tc, domain, err)
		})
	}
}

func TestDomainRepo_GetByName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		setup       func(dbConn *sql.DB)
		domainName  string
		tenantID    string
		expected    *entity.Domain
		expectError bool
	}{
		{
			name: "Successful retrieval",
			setup: func(dbConn *sql.DB) {
				_, err := dbConn.ExecContext(context.Background(), `INSERT INTO domains (name, domain_suffix, provisioning_cert, provisioning_cert_storage_format, provisioning_cert_key, expiration_date, tenant_id) VALUES (?, ?, ?, ?, ?, ?, ?)`,
					"test-domain", "test-suffix", "cert", "PEM", "password", "2024-12-31", "tenant1")
				require.NoError(t, err)
			},
			domainName: "test-domain",
			tenantID:   "tenant1",
			expected: &entity.Domain{
				ProfileName:                   "test-domain",
				DomainSuffix:                  "test-suffix",
				ProvisioningCert:              "cert",
				ProvisioningCertStorageFormat: "PEM",
				ProvisioningCertPassword:      "password",
				ExpirationDate:                "2024-12-31",
				TenantID:                      "tenant1",
			},
			expectError: false,
		},
		{
			name:        "No domain found",
			setup:       func(_ *sql.DB) {},
			domainName:  "nonexistent-domain",
			tenantID:    "tenant1",
			expected:    nil,
			expectError: false,
		},
		{
			name:       "Query execution error",
			setup:      func(_ *sql.DB) {},
			domainName: "test-domain",
			tenantID:   "tenant1",
			expected:   nil,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dbConn := setupDomainTable(t)
			defer dbConn.Close()

			tc.setup(dbConn)

			sqlConfig := &db.SQL{
				Builder:    squirrel.StatementBuilder.PlaceholderFormat(squirrel.Question),
				Pool:       dbConn,
				IsEmbedded: true,
			}

			repo := sqldb.NewDomainRepo(sqlConfig, mocks.NewMockLogger(nil))

			domain, err := repo.GetByName(context.Background(), tc.domainName, tc.tenantID)

			if (err != nil) != tc.expectError {
				t.Errorf("Expected error status %v, got %v", tc.expectError, err != nil)
			}

			if domain == nil && tc.expected == nil {
				return
			}

			assert.IsType(t, tc.expected, domain)
			assert.Equal(t, tc.expected, domain)
		})
	}
}

func TestDomainRepo_Delete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setup      func(dbConn *sql.DB)
		domainName string
		tenantID   string
		expected   bool
		err        error
	}{
		{
			name: "Successful delete",
			setup: func(dbConn *sql.DB) {
				_, err := dbConn.ExecContext(context.Background(), `INSERT INTO domains (name, domain_suffix, provisioning_cert_storage_format, expiration_date, tenant_id) VALUES (?, ?, ?, ?, ?)`,
					"test-domain", "test-suffix", "PEM", "2024-12-31", "tenant1")
				require.NoError(t, err)
			},
			domainName: "test-domain",
			tenantID:   "tenant1",
			expected:   true,
			err:        nil,
		},
		{
			name:       "No matching domain",
			setup:      func(_ *sql.DB) {},
			domainName: "nonexistent-domain",
			tenantID:   "tenant2",
			expected:   false,
			err:        nil,
		},
		{
			name:       "Query execution error",
			setup:      func(_ *sql.DB) {},
			domainName: "test-domain",
			tenantID:   "tenant1",
			expected:   false,
			err:        &sqldb.DatabaseError{},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dbConn := setupDomainTable(t)
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
			repo := sqldb.NewDomainRepo(sqlConfig, mockLog)

			deleted, err := repo.Delete(context.Background(), tc.domainName, tc.tenantID)

			if err == nil && tc.err != nil {
				t.Errorf("Expected error of type %T, got nil", tc.err)
			} else if err != nil {
				var dbError sqldb.DatabaseError

				if !errors.As(err, &dbError) {
					t.Errorf("Expected error of type %T, got %T", tc.err, err)
				}
			}

			if deleted != tc.expected {
				t.Errorf("Expected deleted status %v, got %v", tc.expected, deleted)
			}
		})
	}
}

func TestDomainRepo_Update(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		setup    func(dbConn *sql.DB)
		domain   *entity.Domain
		expected bool
		err      error
	}{
		{
			name: "Successful update",
			setup: func(dbConn *sql.DB) {
				_, err := dbConn.ExecContext(context.Background(), `INSERT INTO domains (name, domain_suffix, provisioning_cert, provisioning_cert_storage_format, provisioning_cert_key, expiration_date, tenant_id) VALUES (?, ?, ?, ?, ?, ?, ?)`,
					"test-domain", "test-suffix", "cert-data", "PEM", "cert-key", "2024-12-31", "tenant1")
				require.NoError(t, err)
			},
			domain: &entity.Domain{
				ProfileName:                   "test-domain",
				DomainSuffix:                  "updated-suffix",
				ProvisioningCert:              "updated-cert-data",
				ProvisioningCertStorageFormat: "DER",
				ProvisioningCertPassword:      "updated-cert-key",
				ExpirationDate:                "2025-01-01",
				TenantID:                      "tenant1",
			},
			expected: true,
			err:      nil,
		},
		{
			name:  "Update non-existent domain",
			setup: func(_ *sql.DB) {},
			domain: &entity.Domain{
				ProfileName:                   "nonexistent-domain",
				DomainSuffix:                  "suffix",
				ProvisioningCert:              "cert-data",
				ProvisioningCertStorageFormat: "PEM",
				ProvisioningCertPassword:      "cert-key",
				ExpirationDate:                "2024-12-31",
				TenantID:                      "tenant2",
			},
			expected: false,
			err:      nil,
		},
		{
			name:  "Query execution error",
			setup: func(_ *sql.DB) {},
			domain: &entity.Domain{
				ProfileName:                   "test-domain",
				DomainSuffix:                  "suffix",
				ProvisioningCert:              "cert-data",
				ProvisioningCertStorageFormat: "PEM",
				ProvisioningCertPassword:      "cert-key",
				ExpirationDate:                "2024-12-31",
				TenantID:                      "tenant1",
			},
			expected: false,
			err:      &sqldb.DatabaseError{},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dbConn := setupDomainTable(t)
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
			repo := sqldb.NewDomainRepo(sqlConfig, mockLog)

			updated, err := repo.Update(context.Background(), tc.domain)

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

type DomainInsertTestCase struct {
	name     string
	setup    func(dbConn *sql.DB)
	domain   *entity.Domain
	expected string
	err      error
}

func DomainInsertHelper(t *testing.T, tc DomainInsertTestCase, version string, err error) {
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

func TestDomainRepo_Insert(t *testing.T) {
	t.Parallel()

	tests := []DomainInsertTestCase{
		{
			name:  "Successful insert",
			setup: func(_ *sql.DB) {},
			domain: &entity.Domain{
				ProfileName:                   "profile1",
				DomainSuffix:                  "suffix1",
				ProvisioningCert:              "cert1",
				ProvisioningCertStorageFormat: "format1",
				ProvisioningCertPassword:      "password1",
				TenantID:                      "tenant1",
			},
			expected: "",
			err:      nil,
		},
		{
			name: "Insert with not unique error",
			setup: func(dbConn *sql.DB) {
				_, err := dbConn.ExecContext(context.Background(), `INSERT INTO domains (name, domain_suffix, provisioning_cert, provisioning_cert_storage_format, provisioning_cert_key, expiration_date, tenant_id) VALUES (?, ?, ?, ?, ?, ?, ?)`,
					"profile1", "suffix1", "cert1", "format1", "password1", time.Now().AddDate(1, 0, 0), "tenant1")
				require.NoError(t, err)
			},
			domain: &entity.Domain{
				ProfileName:                   "profile1",
				DomainSuffix:                  "suffix1",
				ProvisioningCert:              "cert1",
				ProvisioningCertStorageFormat: "format1",
				ProvisioningCertPassword:      "password1",
				TenantID:                      "tenant1",
			},
			expected: "",
			err:      sqldb.NotUniqueError{},
		},
		{
			name:  "Query execution error",
			setup: func(_ *sql.DB) {},
			domain: &entity.Domain{
				ProfileName:                   "profile1",
				DomainSuffix:                  "suffix1",
				ProvisioningCert:              "cert1",
				ProvisioningCertStorageFormat: "format1",
				ProvisioningCertPassword:      "password1",
				TenantID:                      "tenant1",
			},
			expected: "",
			err:      &sqldb.DatabaseError{},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dbConn := setupDomainTable(t)
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
			repo := sqldb.NewDomainRepo(sqlConfig, mockLog)

			version, err := repo.Insert(context.Background(), tc.domain)

			DomainInsertHelper(t, tc, version, err)
		})
	}
}
