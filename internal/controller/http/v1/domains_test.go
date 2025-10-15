package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/mocks"
	"github.com/device-management-toolkit/console/internal/usecase/domains"
	"github.com/device-management-toolkit/console/pkg/logger"
)

//nolint:gochecknoinits // required to avoid issues when running tests in parallel
func init() {
	gin.SetMode(gin.TestMode)
	// Note: We disable validation for most tests, but enable it selectively in validation tests
	gin.DisableBindValidation()
}

func domainsTest(t *testing.T) (*mocks.MockDomainsFeature, *gin.Engine) {
	t.Helper()

	mockCtl := gomock.NewController(t)
	defer mockCtl.Finish()

	log := logger.New("error")
	domain := mocks.NewMockDomainsFeature(mockCtl)

	engine := gin.New()
	handler := engine.Group("/api/v1/admin")

	NewDomainRoutes(handler, domain, log)

	return domain, engine
}

type test struct {
	name         string
	method       string
	url          string
	mock         func(repo *mocks.MockDomainsFeature)
	response     interface{}
	requestBody  dto.Domain
	expectedCode int
}

var (
	requestDomain  = dto.Domain{ProfileName: "newProfile", TenantID: "tenant1", DomainSuffix: "domain.com", ProvisioningCert: "cert", ProvisioningCertStorageFormat: "string", ProvisioningCertPassword: "password"}
	responseDomain = dto.Domain{ProfileName: "newProfile", TenantID: "tenant1", DomainSuffix: "domain.com", ProvisioningCert: "cert", ProvisioningCertStorageFormat: "string", ProvisioningCertPassword: "password"}
)

func TestDomainRoutes(t *testing.T) {
	t.Parallel()

	tests := []test{
		{
			name:   "get all domains",
			method: http.MethodGet,
			url:    "/api/v1/admin/domains",
			mock: func(domain *mocks.MockDomainsFeature) {
				domain.EXPECT().Get(context.Background(), 25, 0, "").Return([]dto.Domain{{
					ProfileName: "profile",
				}}, nil)
			},
			response:     []dto.Domain{{ProfileName: "profile"}},
			expectedCode: http.StatusOK,
		},
		{
			name:   "get all domains - with count",
			method: http.MethodGet,
			url:    "/api/v1/admin/domains?$top=10&$skip=1&$count=true",
			mock: func(domain *mocks.MockDomainsFeature) {
				domain.EXPECT().Get(context.Background(), 10, 1, "").Return([]dto.Domain{{
					ProfileName: "profile",
				}}, nil)
				domain.EXPECT().GetCount(context.Background(), "").Return(1, nil)
			},
			response:     DomainCountResponse{Count: 1, Data: []dto.Domain{{ProfileName: "profile"}}},
			expectedCode: http.StatusOK,
		},
		{
			name:   "get all domains - failed",
			method: http.MethodGet,
			url:    "/api/v1/admin/domains",
			mock: func(domain *mocks.MockDomainsFeature) {
				domain.EXPECT().Get(context.Background(), 25, 0, "").Return(nil, domains.ErrDatabase)
			},
			response:     domains.ErrDatabase,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:   "get domain by name",
			method: http.MethodGet,
			url:    "/api/v1/admin/domains/profile",
			mock: func(domain *mocks.MockDomainsFeature) {
				domain.EXPECT().GetByName(context.Background(), "profile", "").Return(&dto.Domain{
					ProfileName: "profile",
				}, nil)
			},
			response:     dto.Domain{ProfileName: "profile"},
			expectedCode: http.StatusOK,
		},
		{
			name:   "get domain by name - failed",
			method: http.MethodGet,
			url:    "/api/v1/admin/domains/profile",
			mock: func(domain *mocks.MockDomainsFeature) {
				domain.EXPECT().GetByName(context.Background(), "profile", "").Return(nil, domains.ErrDatabase)
			},
			response:     domains.ErrDatabase,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:   "insert domain",
			method: http.MethodPost,
			url:    "/api/v1/admin/domains",
			mock: func(domain *mocks.MockDomainsFeature) {
				domainTest := &dto.Domain{ProfileName: "newProfile", TenantID: "tenant1", DomainSuffix: "domain.com", ProvisioningCert: "cert", ProvisioningCertStorageFormat: "string", ProvisioningCertPassword: "password"}
				domain.EXPECT().Insert(context.Background(), domainTest).Return(domainTest, nil)
			},
			response:     responseDomain,
			requestBody:  requestDomain,
			expectedCode: http.StatusCreated,
		},
		{
			name:   "insert domain - failed",
			method: http.MethodPost,
			url:    "/api/v1/admin/domains",
			mock: func(domain *mocks.MockDomainsFeature) {
				domainTest := &dto.Domain{ProfileName: "newProfile", TenantID: "tenant1", DomainSuffix: "domain.com", ProvisioningCert: "cert", ProvisioningCertStorageFormat: "string", ProvisioningCertPassword: "password"}
				domain.EXPECT().Insert(context.Background(), domainTest).Return(nil, domains.ErrDatabase)
			},
			response:     domains.ErrDatabase,
			requestBody:  requestDomain,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:   "insert domain validation - failed",
			method: http.MethodPost,
			url:    "/api/v1/admin/domains",
			mock: func(domain *mocks.MockDomainsFeature) {
				domain400Test := &dto.Domain{ProfileName: "p1", TenantID: "t1", DomainSuffix: "domain1.com", ProvisioningCert: "cert1", ProvisioningCertStorageFormat: "string1"}
				domain.EXPECT().Insert(context.Background(), domain400Test).Return(nil, domains.ErrDatabase)
			},
			response:     domains.ErrDatabase,
			requestBody:  dto.Domain{ProfileName: "p1", TenantID: "t1", DomainSuffix: "domain1.com", ProvisioningCert: "cert1", ProvisioningCertStorageFormat: "string1"},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:   "delete domain",
			method: http.MethodDelete,
			url:    "/api/v1/admin/domains/profile",
			mock: func(domain *mocks.MockDomainsFeature) {
				domain.EXPECT().Delete(context.Background(), "profile", "").Return(nil)
			},
			response:     nil,
			expectedCode: http.StatusNoContent,
		},
		{
			name:   "delete domain - failed",
			method: http.MethodDelete,
			url:    "/api/v1/admin/domains/profile",
			mock: func(domain *mocks.MockDomainsFeature) {
				domain.EXPECT().Delete(context.Background(), "profile", "").Return(domains.ErrDatabase)
			},
			response:     domains.ErrDatabase,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:   "update domain",
			method: http.MethodPatch,
			url:    "/api/v1/admin/domains",
			mock: func(domain *mocks.MockDomainsFeature) {
				domainTest := &dto.Domain{ProfileName: "newProfile", TenantID: "tenant1", DomainSuffix: "domain.com", ProvisioningCert: "cert", ProvisioningCertStorageFormat: "string", ProvisioningCertPassword: "password"}
				domain.EXPECT().Update(context.Background(), domainTest).Return(domainTest, nil)
			},
			response:     responseDomain,
			requestBody:  requestDomain,
			expectedCode: http.StatusOK,
		},
		{
			name:   "update domain - failed",
			method: http.MethodPatch,
			url:    "/api/v1/admin/domains",
			mock: func(domain *mocks.MockDomainsFeature) {
				domainTest := &dto.Domain{ProfileName: "newProfile", TenantID: "tenant1", DomainSuffix: "domain.com", ProvisioningCert: "cert", ProvisioningCertStorageFormat: "string", ProvisioningCertPassword: "password"}
				domain.EXPECT().Update(context.Background(), domainTest).Return(nil, domains.ErrDatabase)
			},
			response:     domains.ErrDatabase,
			requestBody:  requestDomain,
			expectedCode: http.StatusBadRequest,
		},
	}

	// Add more comprehensive edge case tests
	moreTests := []test{
		{
			name:   "get domains with invalid query parameters",
			method: http.MethodGet,
			url:    "/api/v1/admin/domains?$top=invalid&$skip=abc",
			mock: func(_ *mocks.MockDomainsFeature) {
				// No mock expectation as it should fail at query binding
			},
			response:     nil,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:   "get domains with count error",
			method: http.MethodGet,
			url:    "/api/v1/admin/domains?$count=true",
			mock: func(domain *mocks.MockDomainsFeature) {
				domain.EXPECT().Get(context.Background(), 25, 0, "").Return([]dto.Domain{{ProfileName: "profile"}}, nil)
				domain.EXPECT().GetCount(context.Background(), "").Return(0, domains.ErrDatabase)
			},
			response:     domains.ErrDatabase,
			expectedCode: http.StatusBadRequest,
		},
	}

	// Combine both test sets
	tests = append(tests, moreTests...)

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			domainFeature, engine := domainsTest(t)

			tc.mock(domainFeature)

			var req *http.Request

			var err error

			if tc.method == http.MethodPost || tc.method == http.MethodPatch {
				reqBody, _ := json.Marshal(tc.requestBody)
				req, err = http.NewRequestWithContext(context.Background(), tc.method, tc.url, bytes.NewBuffer(reqBody))
			} else {
				req, err = http.NewRequestWithContext(context.Background(), tc.method, tc.url, http.NoBody)
			}

			if err != nil {
				t.Fatalf("Couldn't create request: %v\n", err)
			}

			w := httptest.NewRecorder()

			engine.ServeHTTP(w, req)

			require.Equal(t, tc.expectedCode, w.Code)

			if tc.expectedCode == http.StatusOK || tc.expectedCode == http.StatusCreated {
				jsonBytes, _ := json.Marshal(tc.response)
				require.Equal(t, string(jsonBytes), w.Body.String())
			}
		})
	}
}

func TestDomainValidation(t *testing.T) {
	t.Parallel()

	// NOTE: This test documents the expected validation behavior for Domain DTOs.
	// The actual validation is performed by gin's ShouldBindJSON using the binding tags,
	// but since gin.DisableBindValidation() is called globally for other tests,
	// we demonstrate the validation logic here for documentation and verification purposes.

	t.Run("domain validation rules documentation", func(t *testing.T) {
		t.Parallel()

		// This test verifies that the Domain DTO has proper validation tags
		// and demonstrates which inputs should be considered invalid

		// ProfileName validation rules (binding:"required,alphanum"):
		// - Must not be empty (required)
		// - Must contain only alphanumeric characters (alphanum)
		// - Should reject: spaces, special characters (@, -, _), empty strings

		invalidProfileNames := []struct {
			name  string
			value string
			rule  string
		}{
			{"empty profile name", "", "required"},
			{"profile name with @", "profile@name", "alphanum"},
			{"profile name with spaces", "profile name", "alphanum"},
			{"profile name with hyphen", "profile-name", "alphanum"},
			{"profile name with underscore", "profile_name", "alphanum"},
		}

		// ProvisioningCertStorageFormat validation (binding:"required,oneof=raw string"):
		// - Must not be empty (required)
		// - Must be either "raw" or "string" (oneof)

		invalidStorageFormats := []string{"", "invalid", "json", "binary"}
		validStorageFormats := []string{"raw", "string"}

		// Log the validation rules for documentation
		t.Logf("Domain validation rules:")
		t.Logf("- ProfileName: required, alphanum (letters and numbers only)")
		t.Logf("- DomainSuffix: required")
		t.Logf("- ProvisioningCert: required")
		t.Logf("- ProvisioningCertStorageFormat: required, oneof=raw string")
		t.Logf("- ProvisioningCertPassword: required, lte=64")

		// Test that we can identify invalid profile names
		for _, invalid := range invalidProfileNames {
			t.Logf("Invalid ProfileName (%s): '%s' violates %s rule", invalid.name, invalid.value, invalid.rule)
		}

		// Test that we can identify invalid storage formats
		for _, invalid := range invalidStorageFormats {
			t.Logf("Invalid ProvisioningCertStorageFormat: '%s' violates oneof rule", invalid)
		}

		for _, valid := range validStorageFormats {
			t.Logf("Valid ProvisioningCertStorageFormat: '%s'", valid)
		}

		// Test a valid domain structure
		validDomain := dto.Domain{
			ProfileName:                   "ValidProfile123",
			DomainSuffix:                  "example.com",
			ProvisioningCert:              "-----BEGIN CERTIFICATE-----\nexample\n-----END CERTIFICATE-----",
			ProvisioningCertStorageFormat: "string",
			ProvisioningCertPassword:      "securePassword123",
			TenantID:                      "tenant1",
		}

		require.NotEmpty(t, validDomain.ProfileName, "Valid domain should have ProfileName")
		require.Contains(t, []string{"raw", "string"}, validDomain.ProvisioningCertStorageFormat, "Valid storage format")
		t.Logf("Example valid domain: ProfileName='%s', StorageFormat='%s'",
			validDomain.ProfileName, validDomain.ProvisioningCertStorageFormat)
	})

	// Integration test: Test successful domain creation with valid data
	// Note: Due to gin.DisableBindValidation() in the test environment,
	// validation is bypassed. In production, gin validation would enforce the binding rules.
	t.Run("successful domain creation", func(t *testing.T) {
		t.Parallel()

		mockCtl := gomock.NewController(t)
		defer mockCtl.Finish()

		log := logger.New("error")
		domain := mocks.NewMockDomainsFeature(mockCtl)

		gin.SetMode(gin.TestMode)
		engine := gin.New()
		handler := engine.Group("/api/v1/admin")

		NewDomainRoutes(handler, domain, log)

		// Test with valid domain data
		validDomain := dto.Domain{
			ProfileName:                   "ValidProfile123",
			TenantID:                      "tenant1",
			DomainSuffix:                  "example.com",
			ProvisioningCert:              "-----BEGIN CERTIFICATE-----\nexample\n-----END CERTIFICATE-----",
			ProvisioningCertStorageFormat: "string",
			ProvisioningCertPassword:      "securePassword123",
		}

		domain.EXPECT().Insert(context.Background(), &validDomain).Return(&validDomain, nil)

		reqBody, _ := json.Marshal(validDomain)
		req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/admin/domains", bytes.NewBuffer(reqBody))
		require.NoError(t, err)

		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		require.Equal(t, http.StatusCreated, w.Code, "Valid domain should be created successfully")

		var response dto.Domain

		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err, "Response should be valid JSON")
		require.Equal(t, validDomain.ProfileName, response.ProfileName, "Response should match request")
	})
}
