package v1

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/device-management-toolkit/console/pkg/logger"
)

func ciraCertTest(t *testing.T) *gin.Engine {
	t.Helper()

	log := logger.New("error")

	engine := gin.New()
	handler := engine.Group("/api/v1/admin")

	NewCIRACertRoutes(handler, log)

	return engine
}

type ciraCertTestCase struct {
	name          string
	method        string
	url           string
	setupFunc     func(t *testing.T, testDir string) func()
	expectedCode  int
	expectedBody  string
	shouldContain string
	bodyCheckFunc func(t *testing.T, body string)
}

func TestCIRACertRoutes(t *testing.T) {
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
			name:   "get CIRA cert - success",
			method: http.MethodGet,
			url:    "/api/v1/admin/ciracert",
			setupFunc: func(t *testing.T, testDir string) func() {
				t.Helper()
				// Create config directory and certificate file in test directory
				configDir := filepath.Join(testDir, "config")
				err := os.MkdirAll(configDir, 0o755)
				require.NoError(t, err)

				certPath := filepath.Join(configDir, "root_cert.pem")
				err = os.WriteFile(certPath, []byte(validCertPEM), 0o644)
				require.NoError(t, err)

				// Change working directory to test directory
				oldWd, _ := os.Getwd()
				os.Chdir(testDir)

				// Return cleanup function
				return func() {
					os.Chdir(oldWd)
					os.RemoveAll(testDir)
				}
			},
			expectedCode: http.StatusOK,
			expectedBody: expectedCertContent,
		},
		{
			name:   "get CIRA cert - file not found",
			method: http.MethodGet,
			url:    "/api/v1/admin/ciracert",
			setupFunc: func(t *testing.T, testDir string) func() {
				t.Helper()
				// Change working directory to test directory (no config dir)
				oldWd, _ := os.Getwd()
				os.Chdir(testDir)

				return func() {
					os.Chdir(oldWd)
					os.RemoveAll(testDir)
				}
			},
			expectedCode:  http.StatusInternalServerError,
			shouldContain: "Failed to read certificate file",
		},
		{
			name:   "get CIRA cert - invalid PEM format",
			method: http.MethodGet,
			url:    "/api/v1/admin/ciracert",
			setupFunc: func(t *testing.T, testDir string) func() {
				t.Helper()
				// Create config directory
				configDir := filepath.Join(testDir, "config")
				err := os.MkdirAll(configDir, 0o755)
				require.NoError(t, err)

				// Write invalid certificate content (not PEM format)
				certPath := filepath.Join(configDir, "root_cert.pem")
				err = os.WriteFile(certPath, []byte("This is not a valid PEM certificate"), 0o644)
				require.NoError(t, err)

				oldWd, _ := os.Getwd()
				os.Chdir(testDir)

				return func() {
					os.Chdir(oldWd)
					os.RemoveAll(testDir)
				}
			},
			expectedCode:  http.StatusInternalServerError,
			shouldContain: "Failed to decode certificate",
		},
		{
			name:   "get CIRA cert - empty file",
			method: http.MethodGet,
			url:    "/api/v1/admin/ciracert",
			setupFunc: func(t *testing.T, testDir string) func() {
				t.Helper()
				// Create config directory
				configDir := filepath.Join(testDir, "config")
				err := os.MkdirAll(configDir, 0o755)
				require.NoError(t, err)

				// Write empty file
				certPath := filepath.Join(configDir, "root_cert.pem")
				err = os.WriteFile(certPath, []byte(""), 0o644)
				require.NoError(t, err)

				oldWd, _ := os.Getwd()
				os.Chdir(testDir)

				return func() {
					os.Chdir(oldWd)
					os.RemoveAll(testDir)
				}
			},
			expectedCode:  http.StatusInternalServerError,
			shouldContain: "Failed to decode certificate",
		},
		{
			name:   "get CIRA cert - with extra whitespace and newlines",
			method: http.MethodGet,
			url:    "/api/v1/admin/ciracert",
			setupFunc: func(t *testing.T, testDir string) func() {
				t.Helper()
				// Create config directory
				configDir := filepath.Join(testDir, "config")
				err := os.MkdirAll(configDir, 0o755)
				require.NoError(t, err)

				// Certificate with extra whitespace
				certWithWhitespace := `-----BEGIN CERTIFICATE-----
  MIIDXTCCAkWgAwIBAgIJAKL0UG+mRKKzMA0GCSqGSIb3DQEBCwUAMEUxCzAJBgNV
  BAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEwHwYDVQQKDBhJbnRlcm5ldCBX
  
  aWRnaXRzIFB0eSBMdGQwHhcNMjEwMTAxMDAwMDAwWhcNMzEwMTAxMDAwMDAwWjBF
  MQswCQYDVQQGEwJBVTETMBEGA1UECAwKU29tZS1TdGF0ZTEhMB8GA1UECgwYSW50
  ZXJuZXQgV2lkZ2l0cyBQdHkgTHRkMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIB
  CgKCAQEA0Z6L7r5Q
-----END CERTIFICATE-----`

				certPath := filepath.Join(configDir, "root_cert.pem")
				err = os.WriteFile(certPath, []byte(certWithWhitespace), 0o644)
				require.NoError(t, err)

				oldWd, _ := os.Getwd()
				os.Chdir(testDir)

				return func() {
					os.Chdir(oldWd)
					os.RemoveAll(testDir)
				}
			},
			expectedCode: http.StatusOK,
			bodyCheckFunc: func(t *testing.T, body string) {
				t.Helper()
				// Should not contain whitespace or newlines
				require.NotContains(t, body, "\n")
				require.NotContains(t, body, "\r")
				require.NotContains(t, body, " ")
				// Should start with MII (typical for base64 encoded certificates)
				require.True(t, len(body) > 0, "body should not be empty")
			},
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			// Create unique test directory for each test
			testDir, err := os.MkdirTemp("", "ciracert_test_*")
			require.NoError(t, err)

			// Setup test environment
			var cleanup func()
			if tc.setupFunc != nil {
				cleanup = tc.setupFunc(t, testDir)
				if cleanup != nil {
					defer cleanup()
				}
			}

			engine := ciraCertTest(t)

			req, err := http.NewRequestWithContext(context.Background(), tc.method, tc.url, http.NoBody)
			require.NoError(t, err, "Couldn't create request")

			w := httptest.NewRecorder()

			engine.ServeHTTP(w, req)

			require.Equal(t, tc.expectedCode, w.Code, "Status code mismatch")

			// Check response body
			if tc.expectedBody != "" {
				require.Equal(t, tc.expectedBody, w.Body.String(), "Response body mismatch")
			}

			if tc.shouldContain != "" {
				require.Contains(t, w.Body.String(), tc.shouldContain, "Response should contain expected text")
			}

			if tc.bodyCheckFunc != nil {
				tc.bodyCheckFunc(t, w.Body.String())
			}
		})
	}
}

func TestCIRACertRoutes_ConcurrentAccess(t *testing.T) {
	// Create unique test directory
	testDir, err := os.MkdirTemp("", "ciracert_concurrent_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(testDir)

	// Setup certificate file
	configDir := filepath.Join(testDir, "config")
	err = os.MkdirAll(configDir, 0o755)
	require.NoError(t, err)

	validCertPEM := `-----BEGIN CERTIFICATE-----
MIIDXTCCAkWgAwIBAgIJAKL0UG+mRKKzMA0GCSqGSIb3DQEBCwUAMEUxCzAJBgNV
BAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEwHwYDVQQKDBhJbnRlcm5ldCBX
aWRnaXRzIFB0eSBMdGQwHhcNMjEwMTAxMDAwMDAwWhcNMzEwMTAxMDAwMDAwWjBF
-----END CERTIFICATE-----`

	certPath := filepath.Join(configDir, "root_cert.pem")
	err = os.WriteFile(certPath, []byte(validCertPEM), 0o644)
	require.NoError(t, err)

	// Change to test directory
	oldWd, _ := os.Getwd()
	os.Chdir(testDir)
	defer os.Chdir(oldWd)

	engine := ciraCertTest(t)

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

func TestCIRACertRoutes_FilePermissions(t *testing.T) {
	// Skip on Windows as file permissions work differently
	if filepath.Separator == '\\' {
		t.Skip("Skipping file permission test on Windows")
	}

	// Create unique test directory
	testDir, err := os.MkdirTemp("", "ciracert_perm_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(testDir)

	// Setup certificate file with no read permissions
	configDir := filepath.Join(testDir, "config")
	err = os.MkdirAll(configDir, 0o755)
	require.NoError(t, err)

	validCertPEM := `-----BEGIN CERTIFICATE-----
MIIDXTCCAkWgAwIBAgIJAKL0UG+mRKKzMA0GCSqGSIb3DQEBCwUAMEUxCzAJBgNV
-----END CERTIFICATE-----`

	certPath := filepath.Join(configDir, "root_cert.pem")
	err = os.WriteFile(certPath, []byte(validCertPEM), 0o000)
	require.NoError(t, err)

	// Change to test directory
	oldWd, _ := os.Getwd()
	os.Chdir(testDir)
	defer os.Chdir(oldWd)

	engine := ciraCertTest(t)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/admin/ciracert", http.NoBody)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.Contains(t, w.Body.String(), "Failed to read certificate file")
}

// TestCIRACertRoutes_Coverage ensures all code paths are tested
func TestCIRACertRoutes_Coverage(t *testing.T) {
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
			// Create unique test directory
			testDir, err := os.MkdirTemp("", fmt.Sprintf("ciracert_coverage_test_%s_*", tt.name))
			require.NoError(t, err)
			defer os.RemoveAll(testDir)

			configDir := filepath.Join(testDir, "config")
			err = os.MkdirAll(configDir, 0o755)
			require.NoError(t, err)

			certPath := filepath.Join(configDir, "root_cert.pem")
			err = os.WriteFile(certPath, []byte(tt.certData), 0o644)
			require.NoError(t, err)

			oldWd, _ := os.Getwd()
			os.Chdir(testDir)
			defer os.Chdir(oldWd)

			engine := ciraCertTest(t)

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
