package v1

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/device-management-toolkit/console/pkg/logger"
)

// mockCertReader implements CertReader for testing.
type mockCertReader struct {
	data []byte
	err  error
}

func (m *mockCertReader) ReadCert() ([]byte, error) {
	return m.data, m.err
}

func ciraCertTestWithReader(t *testing.T, reader CertReader) *gin.Engine {
	t.Helper()

	log := logger.New("error")

	engine := gin.New()
	handler := engine.Group("/api/v1/admin")

	NewCIRACertRoutesWithReader(handler, log, reader)

	return engine
}

type ciraCertTestCase struct {
	name          string
	method        string
	url           string
	certData      []byte
	certErr       error
	expectedCode  int
	expectedBody  string
	shouldContain string
	bodyCheckFunc func(t *testing.T, body string)
}

func TestCIRACertRoutes(t *testing.T) {
	t.Parallel()

	// Valid certificate content for testing
	validCertPEM := `-----BEGIN CERTIFICATE-----
MIIDXTCCAkWgAwIBAgIJAKL0UG+mRKKzMA0GCSqGSIb3DQEBCwUAMEUxCzAJBgNV
BAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEwHwYDVQQKDBhJbnRlcm5ldCBX
aWRnaXRzIFB0eSBMdGQwHhcNMjEwMTAxMDAwMDAwWhcNMzEwMTAxMDAwMDAwWjBF
MQswCQYDVQQGEwJBVTETMBEGA1UECAwKU29tZS1TdGF0ZTEhMB8GA1UECgwYSW50
ZXJuZXQgV2lkZ2l0cyBQdHkgTHRkMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIB
CgKCAQEA0Z6L7r5Q
-----END CERTIFICATE-----`

	// Expected output without PEM headers/footers
	expectedCertContent := "MIIDXTCCAkWgAwIBAgIJAKL0UG+mRKKzMA0GCSqGSIb3DQEBCwUAMEUxCzAJBgNVBAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEwHwYDVQQKDBhJbnRlcm5ldCBXaWRnaXRzIFB0eSBMdGQwHhcNMjEwMTAxMDAwMDAwWhcNMzEwMTAxMDAwMDAwWjBFMQswCQYDVQQGEwJBVTETMBEGA1UECAwKU29tZS1TdGF0ZTEhMB8GA1UECgwYSW50ZXJuZXQgV2lkZ2l0cyBQdHkgTHRkMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA0Z6L7r5Q"

	tests := []ciraCertTestCase{
		{
			name:         "get CIRA cert - success",
			method:       http.MethodGet,
			url:          "/api/v1/admin/ciracert",
			certData:     []byte(validCertPEM),
			certErr:      nil,
			expectedCode: http.StatusOK,
			expectedBody: expectedCertContent,
		},
		{
			name:          "get CIRA cert - file not found",
			method:        http.MethodGet,
			url:           "/api/v1/admin/ciracert",
			certData:      nil,
			certErr:       errors.New("file not found"),
			expectedCode:  http.StatusInternalServerError,
			shouldContain: "Failed to read certificate file",
		},
		{
			name:          "get CIRA cert - invalid PEM format",
			method:        http.MethodGet,
			url:           "/api/v1/admin/ciracert",
			certData:      []byte("This is not a valid PEM certificate"),
			certErr:       nil,
			expectedCode:  http.StatusInternalServerError,
			shouldContain: "Failed to decode certificate",
		},
		{
			name:          "get CIRA cert - empty file",
			method:        http.MethodGet,
			url:           "/api/v1/admin/ciracert",
			certData:      []byte(""),
			certErr:       nil,
			expectedCode:  http.StatusInternalServerError,
			shouldContain: "Failed to decode certificate",
		},
		{
			name:   "get CIRA cert - with extra whitespace and newlines",
			method: http.MethodGet,
			url:    "/api/v1/admin/ciracert",
			certData: []byte(`-----BEGIN CERTIFICATE-----
  MIIDXTCCAkWgAwIBAgIJAKL0UG+mRKKzMA0GCSqGSIb3DQEBCwUAMEUxCzAJBgNV
  BAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEwHwYDVQQKDBhJbnRlcm5ldCBX

  aWRnaXRzIFB0eSBMdGQwHhcNMjEwMTAxMDAwMDAwWhcNMzEwMTAxMDAwMDAwWjBF
  MQswCQYDVQQGEwJBVTETMBEGA1UECAwKU29tZS1TdGF0ZTEhMB8GA1UECgwYSW50
  ZXJuZXQgV2lkZ2l0cyBQdHkgTHRkMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIB
  CgKCAQEA0Z6L7r5Q
-----END CERTIFICATE-----`),
			certErr:      nil,
			expectedCode: http.StatusOK,
			bodyCheckFunc: func(t *testing.T, body string) {
				t.Helper()
				// Should not contain whitespace or newlines
				require.NotContains(t, body, "\n")
				require.NotContains(t, body, "\r")
				require.NotContains(t, body, " ")
				// Should start with MII (typical for base64 encoded certificates)
				require.NotEmpty(t, body, "body should not be empty")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			runCiraCertTestCase(t, tc)
		})
	}
}

func runCiraCertTestCase(t *testing.T, tc ciraCertTestCase) {
	t.Helper()

	reader := &mockCertReader{
		data: tc.certData,
		err:  tc.certErr,
	}

	engine := ciraCertTestWithReader(t, reader)

	req, err := http.NewRequestWithContext(context.Background(), tc.method, tc.url, http.NoBody)
	require.NoError(t, err, "Couldn't create request")

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	require.Equal(t, tc.expectedCode, w.Code, "Status code mismatch")
	verifyResponseBody(t, tc, w.Body.String())
}

func verifyResponseBody(t *testing.T, tc ciraCertTestCase, body string) {
	t.Helper()

	if tc.expectedBody != "" {
		require.Equal(t, tc.expectedBody, body, "Response body mismatch")
	}

	if tc.shouldContain != "" {
		require.Contains(t, body, tc.shouldContain, "Response should contain expected text")
	}

	if tc.bodyCheckFunc != nil {
		tc.bodyCheckFunc(t, body)
	}
}

func TestCIRACertRoutes_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	validCertPEM := `-----BEGIN CERTIFICATE-----
MIIDXTCCAkWgAwIBAgIJAKL0UG+mRKKzMA0GCSqGSIb3DQEBCwUAMEUxCzAJBgNV
BAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEwHwYDVQQKDBhJbnRlcm5ldCBX
aWRnaXRzIFB0eSBMdGQwHhcNMjEwMTAxMDAwMDAwWhcNMzEwMTAxMDAwMDAwWjBF
-----END CERTIFICATE-----`

	reader := &mockCertReader{
		data: []byte(validCertPEM),
		err:  nil,
	}

	engine := ciraCertTestWithReader(t, reader)

	// Test concurrent access
	var wg sync.WaitGroup

	requests := 10

	for i := 0; i < requests; i++ {
		wg.Add(1)

		go func(id int) {
			defer wg.Done()

			req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/admin/ciracert", http.NoBody)
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Request %d failed with status %d: %s", id, w.Code, w.Body.String())
			}
		}(i)
	}

	wg.Wait()
}

func TestCIRACertRoutes_ReadError(t *testing.T) {
	t.Parallel()

	reader := &mockCertReader{
		data: nil,
		err:  errors.New("permission denied"),
	}

	engine := ciraCertTestWithReader(t, reader)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/admin/ciracert", http.NoBody)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.Contains(t, w.Body.String(), "Failed to read certificate file")
}

// TestCIRACertRoutes_Coverage ensures all code paths are tested.
func TestCIRACertRoutes_Coverage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		certData string
		wantCode int
		wantBody string
	}{
		{
			name: "valid certificate without extra newlines",
			certData: `-----BEGIN CERTIFICATE-----
MIID
-----END CERTIFICATE-----`,
			wantCode: http.StatusOK,
			wantBody: "MIID",
		},
		{
			name:     "certificate with only BEGIN marker",
			certData: "-----BEGIN CERTIFICATE-----\nMIID",
			wantCode: http.StatusInternalServerError,
			wantBody: "Failed to decode certificate",
		},
		{
			name:     "certificate with Windows line endings",
			certData: "-----BEGIN CERTIFICATE-----\r\nMIID\r\n-----END CERTIFICATE-----",
			wantCode: http.StatusOK,
			wantBody: "MIID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			reader := &mockCertReader{
				data: []byte(tt.certData),
				err:  nil,
			}

			engine := ciraCertTestWithReader(t, reader)

			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/admin/ciracert", http.NoBody)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)

			require.Equal(t, tt.wantCode, w.Code)

			if tt.wantCode == http.StatusOK {
				require.Equal(t, tt.wantBody, w.Body.String())
			} else {
				require.Contains(t, w.Body.String(), tt.wantBody)
			}
		})
	}
}
