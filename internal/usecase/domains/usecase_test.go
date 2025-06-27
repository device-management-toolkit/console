package domains_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/device-management-toolkit/console/internal/entity"
	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/mocks"
	"github.com/device-management-toolkit/console/internal/usecase/domains"
	"github.com/device-management-toolkit/console/pkg/logger"
)

type test struct {
	name         string
	top          int
	skip         int
	domainName   string
	domainSuffix string
	tenantID     string
	input        entity.Domain
	mock         func(repo *mocks.MockDomainsRepository)
	res          interface{}
	err          error
}

func domainsTest(t *testing.T) (*domains.UseCase, *mocks.MockDomainsRepository) {
	t.Helper()

	mockCtl := gomock.NewController(t)
	defer mockCtl.Finish()

	repo := mocks.NewMockDomainsRepository(mockCtl)
	log := logger.New("error")
	crypto := mocks.MockCrypto{}
	useCase := domains.New(repo, log, crypto)

	return useCase, repo
}

func TestGetCount(t *testing.T) {
	t.Parallel()

	tests := []test{
		{
			name: "empty result",
			mock: func(repo *mocks.MockDomainsRepository) {
				repo.EXPECT().GetCount(context.Background(), "").Return(0, nil)
			},
			res: 0,
			err: nil,
		},
		{
			name: "result with error",
			mock: func(repo *mocks.MockDomainsRepository) {
				repo.EXPECT().GetCount(context.Background(), "").Return(0, domains.ErrDatabase)
			},
			res: 0,
			err: domains.ErrDatabase,
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, repo := domainsTest(t)

			tc.mock(repo)

			res, err := useCase.GetCount(context.Background(), "")

			require.Equal(t, tc.res, res)
			require.IsType(t, tc.err, err)
		})
	}
}

func TestGet(t *testing.T) {
	t.Parallel()

	testDomains := []entity.Domain{
		{
			ProfileName: "test-domain-1",
			TenantID:    "tenant-id-456",
		},
		{
			ProfileName: "test-domain-2",
			TenantID:    "tenant-id-456",
		},
	}

	testDomainDTOs := []dto.Domain{
		{
			ProfileName: "test-domain-1",
			TenantID:    "tenant-id-456",
		},
		{
			ProfileName: "test-domain-2",
			TenantID:    "tenant-id-456",
		},
	}

	tests := []test{
		{
			name:     "successful retrieval",
			top:      10,
			skip:     0,
			tenantID: "tenant-id-456",
			mock: func(repo *mocks.MockDomainsRepository) {
				repo.EXPECT().
					Get(context.Background(), 10, 0, "tenant-id-456").
					Return(testDomains, nil)
			},
			res: testDomainDTOs,
			err: nil,
		},
		{
			name:     "database error",
			top:      5,
			skip:     0,
			tenantID: "tenant-id-456",
			mock: func(repo *mocks.MockDomainsRepository) {
				repo.EXPECT().
					Get(context.Background(), 5, 0, "tenant-id-456").
					Return(nil, domains.ErrDatabase)
			},
			res: []dto.Domain(nil),
			err: domains.ErrDatabase,
		},
		{
			name:     "zero results",
			top:      10,
			skip:     20,
			tenantID: "tenant-id-456",
			mock: func(repo *mocks.MockDomainsRepository) {
				repo.EXPECT().
					Get(context.Background(), 10, 20, "tenant-id-456").
					Return([]entity.Domain{}, nil)
			},
			res: []dto.Domain{},
			err: nil,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, repo := domainsTest(t)

			tc.mock(repo)

			results, err := useCase.Get(context.Background(), tc.top, tc.skip, tc.tenantID)

			require.Equal(t, tc.res, results)

			if tc.err != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGetDomainByDomainSuffix(t *testing.T) {
	t.Parallel()

	domain := entity.Domain{
		ProfileName:                   "test-domain",
		DomainSuffix:                  "test.com",
		ProvisioningCert:              "test-cert",
		ProvisioningCertStorageFormat: "test-format",
		ProvisioningCertPassword:      "test-password",
		TenantID:                      "tenant-id-456",
		Version:                       "1.0.0",
	}

	domainDTO := dto.Domain{
		ProfileName:                   "test-domain",
		DomainSuffix:                  "test.com",
		ProvisioningCert:              "",
		ProvisioningCertStorageFormat: "test-format",
		ProvisioningCertPassword:      "",
		TenantID:                      "tenant-id-456",
		Version:                       "1.0.0",
	}

	tests := []test{
		{
			name:         "successful retrieval",
			domainSuffix: "test.com",
			tenantID:     "tenant-id-456",
			mock: func(repo *mocks.MockDomainsRepository) {
				repo.EXPECT().
					GetDomainByDomainSuffix(context.Background(), "test.com", "tenant-id-456").
					Return(&domain, nil)
			},
			res: &domainDTO,
			err: nil,
		},
		{
			name:         "domain not found",
			domainSuffix: "unknown.com",
			tenantID:     "tenant-id-456",
			mock: func(repo *mocks.MockDomainsRepository) {
				repo.EXPECT().
					GetDomainByDomainSuffix(context.Background(), "unknown.com", "tenant-id-456").
					Return(nil, nil)
			},
			res: nil,
			err: domains.ErrNotFound,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, repo := domainsTest(t)

			tc.mock(repo)

			domain, err := useCase.GetDomainByDomainSuffix(context.Background(), tc.domainSuffix, tc.tenantID)

			if tc.res != nil {
				require.NotNil(t, domain)
				require.Equal(t, tc.res, domain)
			} else {
				require.Nil(t, domain)
			}

			if tc.err != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGetByName(t *testing.T) {
	t.Parallel()

	domain := &entity.Domain{
		ProfileName:                   "test-domain",
		DomainSuffix:                  "test-domain",
		ProvisioningCert:              "test-cert",
		ProvisioningCertStorageFormat: "test-format",
		ProvisioningCertPassword:      "test-password",
		TenantID:                      "tenant-id-456",
		Version:                       "1.0.0",
	}

	domainDTO := &dto.Domain{
		ProfileName:                   "test-domain",
		DomainSuffix:                  "test-domain",
		ProvisioningCert:              "",
		ProvisioningCertStorageFormat: "test-format",
		ProvisioningCertPassword:      "",
		TenantID:                      "tenant-id-456",
		Version:                       "1.0.0",
	}

	tests := []test{
		{
			name: "successful retrieval",
			input: entity.Domain{
				ProfileName: "test-domain",
				TenantID:    "tenant-id-456",
			},
			mock: func(repo *mocks.MockDomainsRepository) {
				repo.EXPECT().
					GetByName(context.Background(), "test-domain", "tenant-id-456").
					Return(domain, nil)
			},
			res: domainDTO,
			err: nil,
		},
		{
			name: "domain not found",
			input: entity.Domain{
				ProfileName: "unknown-domain",
				TenantID:    "tenant-id-456",
			},
			mock: func(repo *mocks.MockDomainsRepository) {
				repo.EXPECT().
					GetByName(context.Background(), "unknown-domain", "tenant-id-456").
					Return(nil, nil)
			},
			res: (*dto.Domain)(nil),
			err: domains.ErrNotFound,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, repo := domainsTest(t)

			tc.mock(repo)

			res, err := useCase.GetByName(context.Background(), tc.input.ProfileName, tc.input.TenantID)

			require.Equal(t, tc.res, res)

			if tc.err != nil {
				require.Contains(t, err.Error(), tc.err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	t.Parallel()

	tests := []test{
		{
			name:       "successful deletion",
			domainName: "example-domain",
			tenantID:   "tenant-id-456",
			mock: func(repo *mocks.MockDomainsRepository) {
				repo.EXPECT().
					Delete(context.Background(), "example-domain", "tenant-id-456").
					Return(true, nil)
			},
			err: nil,
		},
		{
			name:       "deletion fails - domain not found",
			domainName: "nonexistent-domain",
			tenantID:   "tenant-id-456",
			mock: func(repo *mocks.MockDomainsRepository) {
				repo.EXPECT().
					Delete(context.Background(), "nonexistent-domain", "tenant-id-456").
					Return(false, nil)
			},
			err: domains.ErrNotFound,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, repo := domainsTest(t)

			tc.mock(repo)

			err := useCase.Delete(context.Background(), tc.domainName, tc.tenantID)

			if tc.err != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestUpdate(t *testing.T) {
	t.Parallel()

	domain := &entity.Domain{
		ProfileName:              "example-domain",
		TenantID:                 "tenant-id-456",
		ProvisioningCertPassword: "encrypted",
		Version:                  "1.0.0",
	}
	domainDTO := &dto.Domain{
		ProfileName: "example-domain",
		TenantID:    "tenant-id-456",
		Version:     "1.0.0",
	}

	tests := []test{
		{
			name: "successful update",
			mock: func(repo *mocks.MockDomainsRepository) {
				repo.EXPECT().
					Update(context.Background(), domain).
					Return(true, nil)
				repo.EXPECT().
					GetByName(context.Background(), domain.ProfileName, domain.TenantID).
					Return(domain, nil)
			},
			res: domainDTO,
			err: nil,
		},
		{
			name: "update fails - not found",
			mock: func(repo *mocks.MockDomainsRepository) {
				repo.EXPECT().
					Update(context.Background(), domain).
					Return(false, nil)
			},
			res: (*dto.Domain)(nil),
			err: domains.ErrNotFound,
		},
		{
			name: "update fails - database error",
			mock: func(repo *mocks.MockDomainsRepository) {
				repo.EXPECT().
					Update(context.Background(), domain).
					Return(false, domains.ErrDatabase)
			},
			res: (*dto.Domain)(nil),
			err: domains.ErrDatabase,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, repo := domainsTest(t)

			tc.mock(repo)

			result, err := useCase.Update(context.Background(), domainDTO)

			require.Equal(t, tc.res, result)
			require.IsType(t, tc.err, err)
		})
	}
}

func TestInsert(t *testing.T) {
	t.Parallel()

	domain := &entity.Domain{
		ProfileName:                   "new-domain",
		DomainSuffix:                  "newdomain.com",
		ProvisioningCert:              generateTestPFX(),
		ProvisioningCertStorageFormat: "PEM",
		ProvisioningCertPassword:      "encrypted",
		TenantID:                      "tenant-id-789",
		ExpirationDate:                "2033-08-01T07:12:09Z",
		Version:                       "1.0.0",
	}
	domainDTO := &dto.Domain{
		ProfileName:                   "new-domain",
		DomainSuffix:                  "newdomain.com",
		ProvisioningCert:              generateTestPFX(),
		ProvisioningCertStorageFormat: "PEM",
		ProvisioningCertPassword:      "P@ssw0rd",
		ExpirationDate:                time.Date(2033, time.August, 1, 7, 12, 9, 0, time.UTC),
		TenantID:                      "tenant-id-789",
		Version:                       "1.0.0",
	}
	returnDomainDTO := &dto.Domain{
		ProfileName:                   "new-domain",
		DomainSuffix:                  "newdomain.com",
		ProvisioningCertStorageFormat: "PEM",
		ExpirationDate:                time.Date(2033, time.August, 1, 7, 12, 9, 0, time.UTC),
		TenantID:                      "tenant-id-789",
		Version:                       "1.0.0",
	}
	tests := []test{
		{
			name: "successful insertion",
			mock: func(repo *mocks.MockDomainsRepository) {
				repo.EXPECT().
					Insert(context.Background(), domain).
					Return("unique-domain-id", nil)
				repo.EXPECT().
					GetByName(context.Background(), domain.ProfileName, domain.TenantID).
					Return(domain, nil)
			},
			res: returnDomainDTO,
			err: nil,
		},
		{
			name: "insertion fails - database error",
			mock: func(repo *mocks.MockDomainsRepository) {
				repo.EXPECT().
					Insert(context.Background(), domain).
					Return("", domains.ErrDatabase)
			},
			res: (*dto.Domain)(nil),
			err: domains.ErrDatabase,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			useCase, repo := domainsTest(t)

			tc.mock(repo)

			id, err := useCase.Insert(context.Background(), domainDTO)

			require.Equal(t, tc.res, id)

			if tc.err != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func generateTestPFX() string {
	return "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"
}

func TestDecryptAndCheckCertExpiration_Valid(t *testing.T) {
	t.Parallel()

	domain := dto.Domain{
		ProvisioningCert:         generateTestPFX(),
		ProvisioningCertPassword: "P@ssw0rd",
	}

	// Call the function
	x509Cert, err := domains.DecryptAndCheckCertExpiration(domain)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, x509Cert)
}

func TestDecryptAndCheckCertExpiration_Expired(t *testing.T) {
	t.Parallel()

	domain := dto.Domain{
		ProvisioningCert:         "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",
		ProvisioningCertPassword: "",
	}

	// Call the function
	x509Cert, err := domains.DecryptAndCheckCertExpiration(domain)

	// Assertions
	assert.Error(t, err)
	assert.Nil(t, x509Cert)
	assert.Contains(t, err.Error(), "certificate has expired")
}

func TestDecryptAndCheckCertExpiration_IncorrectPassword(t *testing.T) {
	t.Parallel()

	domain := dto.Domain{
		ProvisioningCert:         generateTestPFX(),
		ProvisioningCertPassword: "WrongP@ssw0rd",
	}

	// Call the function
	x509Cert, err := domains.DecryptAndCheckCertExpiration(domain)

	// Assertions
	assert.Error(t, err)
	assert.Nil(t, x509Cert)
	assert.Contains(t, err.Error(), "pkcs12: decryption password incorrect")
}
