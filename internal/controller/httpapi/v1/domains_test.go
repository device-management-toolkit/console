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

	"github.com/device-management-toolkit/console/internal/controller/httpapi/middleware"
	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/mocks"
	"github.com/device-management-toolkit/console/internal/usecase/domains"
	"github.com/device-management-toolkit/console/internal/usecase/sqldb"
	"github.com/device-management-toolkit/console/pkg/consoleerrors"
	"github.com/device-management-toolkit/console/pkg/logger"
)

//nolint:gochecknoinits // required to avoid issues when running tests in parallel
func init() {
	gin.SetMode(gin.TestMode)
	gin.DisableBindValidation()
}

func domainsTest(t *testing.T) (*mocks.MockDomainsFeature, *gin.Engine) {
	t.Helper()

	mockCtl := gomock.NewController(t)
	defer mockCtl.Finish()

	log := logger.New("error")
	domain := mocks.NewMockDomainsFeature(mockCtl)

	engine := gin.New()
	engine.Use(middleware.ErrorHandling(log))
	handler := engine.Group("/api/v1/admin")

	NewDomainRoutes(handler, domain, log)

	return domain, engine
}

type test struct {
	name          string
	method        string
	url           string
	mock          func(repo *mocks.MockDomainsFeature)
	response      interface{}
	requestBody   dto.Domain
	expectedCode  int
	expectedError *expectedErrorResponse
}

type expectedErrorResponse struct {
	Code    string
	Message string
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
				dbErr := sqldb.DatabaseError{Console: consoleerrors.CreateConsoleError("DomainsUseCase")}
				domain.EXPECT().Get(context.Background(), 25, 0, "").Return(nil, dbErr.Wrap("Get", "uc.repo.Get", nil))
			},
			expectedCode: http.StatusBadRequest,
			expectedError: &expectedErrorResponse{
				Code:    consoleerrors.ErrCodeDatabase,
				Message: "database error",
			},
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
				dbErr := sqldb.DatabaseError{Console: consoleerrors.CreateConsoleError("DomainsUseCase")}
				domain.EXPECT().GetByName(context.Background(), "profile", "").Return(nil, dbErr.Wrap("GetByName", "uc.repo.GetByName", nil))
			},
			expectedCode: http.StatusBadRequest,
			expectedError: &expectedErrorResponse{
				Code:    consoleerrors.ErrCodeDatabase,
				Message: "database error",
			},
		},
		{
			name:   "get domain by name - not found",
			method: http.MethodGet,
			url:    "/api/v1/admin/domains/nonexistent",
			mock: func(domain *mocks.MockDomainsFeature) {
				domain.EXPECT().GetByName(context.Background(), "nonexistent", "").Return(nil, domains.ErrNotFound)
			},
			expectedCode: http.StatusNotFound,
			expectedError: &expectedErrorResponse{
				Code:    consoleerrors.ErrCodeNotFound,
				Message: "resource not found",
			},
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
				dbErr := sqldb.DatabaseError{Console: consoleerrors.CreateConsoleError("DomainsUseCase")}
				domain.EXPECT().Insert(context.Background(), domainTest).Return(nil, dbErr.Wrap("Insert", "uc.repo.Insert", nil))
			},
			requestBody:  requestDomain,
			expectedCode: http.StatusBadRequest,
			expectedError: &expectedErrorResponse{
				Code:    consoleerrors.ErrCodeDatabase,
				Message: "database error",
			},
		},
		{
			name:   "insert domain - cert password error",
			method: http.MethodPost,
			url:    "/api/v1/admin/domains",
			mock: func(domain *mocks.MockDomainsFeature) {
				domainTest := &dto.Domain{ProfileName: "newProfile", TenantID: "tenant1", DomainSuffix: "domain.com", ProvisioningCert: "cert", ProvisioningCertStorageFormat: "string", ProvisioningCertPassword: "password"}
				certPwdErr := domains.CertPasswordError{Console: consoleerrors.CreateConsoleError("DomainsUseCase")}
				domain.EXPECT().Insert(context.Background(), domainTest).Return(nil, certPwdErr.Wrap("Insert", "pkcs12.Decode", nil))
			},
			requestBody:  requestDomain,
			expectedCode: http.StatusBadRequest,
			expectedError: &expectedErrorResponse{
				Code:    consoleerrors.ErrCodeCertPassword,
				Message: "unable to decrypt certificate, incorrect password",
			},
		},
		{
			name:   "insert domain - cert expiration error",
			method: http.MethodPost,
			url:    "/api/v1/admin/domains",
			mock: func(domain *mocks.MockDomainsFeature) {
				domainTest := &dto.Domain{ProfileName: "newProfile", TenantID: "tenant1", DomainSuffix: "domain.com", ProvisioningCert: "cert", ProvisioningCertStorageFormat: "string", ProvisioningCertPassword: "password"}
				certExpErr := domains.CertExpirationError{Console: consoleerrors.CreateConsoleError("DomainsUseCase")}
				domain.EXPECT().Insert(context.Background(), domainTest).Return(nil, certExpErr.Wrap("Insert", "cert.NotAfter.Before", nil))
			},
			requestBody:  requestDomain,
			expectedCode: http.StatusBadRequest,
			expectedError: &expectedErrorResponse{
				Code:    consoleerrors.ErrCodeCertExpired,
				Message: "certificate has expired",
			},
		},
		{
			name:   "insert domain - not unique error",
			method: http.MethodPost,
			url:    "/api/v1/admin/domains",
			mock: func(domain *mocks.MockDomainsFeature) {
				domainTest := &dto.Domain{ProfileName: "newProfile", TenantID: "tenant1", DomainSuffix: "domain.com", ProvisioningCert: "cert", ProvisioningCertStorageFormat: "string", ProvisioningCertPassword: "password"}
				notUniqueErr := sqldb.NotUniqueError{Console: consoleerrors.CreateConsoleError("DomainsUseCase")}
				domain.EXPECT().Insert(context.Background(), domainTest).Return(nil, notUniqueErr.Wrap("profile_name"))
			},
			requestBody:  requestDomain,
			expectedCode: http.StatusBadRequest,
			expectedError: &expectedErrorResponse{
				Code:    consoleerrors.ErrCodeNotUnique,
				Message: "unique constraint violation: profile_name",
			},
		},
		{
			name:   "insert domain validation - failed",
			method: http.MethodPost,
			url:    "/api/v1/admin/domains",
			mock: func(domain *mocks.MockDomainsFeature) {
				domain400Test := &dto.Domain{ProfileName: "p1", TenantID: "t1", DomainSuffix: "domain1.com", ProvisioningCert: "cert1", ProvisioningCertStorageFormat: "string1"}
				dbErr := sqldb.DatabaseError{Console: consoleerrors.CreateConsoleError("DomainsUseCase")}
				domain.EXPECT().Insert(context.Background(), domain400Test).Return(nil, dbErr.Wrap("Insert", "uc.repo.Insert", nil))
			},
			requestBody:  dto.Domain{ProfileName: "p1", TenantID: "t1", DomainSuffix: "domain1.com", ProvisioningCert: "cert1", ProvisioningCertStorageFormat: "string1"},
			expectedCode: http.StatusBadRequest,
			expectedError: &expectedErrorResponse{
				Code:    consoleerrors.ErrCodeDatabase,
				Message: "database error",
			},
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
				dbErr := sqldb.DatabaseError{Console: consoleerrors.CreateConsoleError("DomainsUseCase")}
				domain.EXPECT().Delete(context.Background(), "profile", "").Return(dbErr.Wrap("Delete", "uc.repo.Delete", nil))
			},
			expectedCode: http.StatusBadRequest,
			expectedError: &expectedErrorResponse{
				Code:    consoleerrors.ErrCodeDatabase,
				Message: "database error",
			},
		},
		{
			name:   "delete domain - not found",
			method: http.MethodDelete,
			url:    "/api/v1/admin/domains/nonexistent",
			mock: func(domain *mocks.MockDomainsFeature) {
				domain.EXPECT().Delete(context.Background(), "nonexistent", "").Return(domains.ErrNotFound)
			},
			expectedCode: http.StatusNotFound,
			expectedError: &expectedErrorResponse{
				Code:    consoleerrors.ErrCodeNotFound,
				Message: "resource not found",
			},
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
				dbErr := sqldb.DatabaseError{Console: consoleerrors.CreateConsoleError("DomainsUseCase")}
				domain.EXPECT().Update(context.Background(), domainTest).Return(nil, dbErr.Wrap("Update", "uc.repo.Update", nil))
			},
			requestBody:  requestDomain,
			expectedCode: http.StatusBadRequest,
			expectedError: &expectedErrorResponse{
				Code:    consoleerrors.ErrCodeDatabase,
				Message: "database error",
			},
		},
		{
			name:   "update domain - not found",
			method: http.MethodPatch,
			url:    "/api/v1/admin/domains",
			mock: func(domain *mocks.MockDomainsFeature) {
				domainTest := &dto.Domain{ProfileName: "newProfile", TenantID: "tenant1", DomainSuffix: "domain.com", ProvisioningCert: "cert", ProvisioningCertStorageFormat: "string", ProvisioningCertPassword: "password"}
				domain.EXPECT().Update(context.Background(), domainTest).Return(nil, domains.ErrNotFound)
			},
			requestBody:  requestDomain,
			expectedCode: http.StatusNotFound,
			expectedError: &expectedErrorResponse{
				Code:    consoleerrors.ErrCodeNotFound,
				Message: "resource not found",
			},
		},
	}

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

			// Verify error response format when expectedError is set
			if tc.expectedError != nil {
				var errResp consoleerrors.ErrorResponse
				err = json.Unmarshal(w.Body.Bytes(), &errResp)
				require.NoError(t, err, "Failed to unmarshal error response")
				require.Equal(t, tc.expectedError.Code, errResp.Code, "Error code mismatch")
				require.Equal(t, tc.expectedError.Message, errResp.Message, "Error message mismatch")
				require.NotEmpty(t, errResp.TraceID, "TraceID should not be empty")
			}
		})
	}
}
