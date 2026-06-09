package certificates

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockObjectStorager implements ObjectStorager for testing.
type MockObjectStorager struct {
	mock.Mock
}

func (m *MockObjectStorager) GetKeyValue(key string) (string, error) {
	args := m.Called(key)

	return args.String(0), args.Error(1)
}

func (m *MockObjectStorager) SetKeyValue(key, value string) error {
	args := m.Called(key, value)

	return args.Error(0)
}

func (m *MockObjectStorager) DeleteKeyValue(key string) error {
	args := m.Called(key)

	return args.Error(0)
}

func (m *MockObjectStorager) GetObject(key string) (map[string]string, error) {
	args := m.Called(key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	result, ok := args.Get(0).(map[string]string)
	if !ok {
		return nil, args.Error(1)
	}

	return result, args.Error(1)
}

func (m *MockObjectStorager) SetObject(key string, data map[string]string) error {
	args := m.Called(key, data)

	return args.Error(0)
}

// Helper to generate a test certificate and key.
func generateTestCertAndKey(t *testing.T) (*x509.Certificate, *rsa.PrivateKey) {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.NoError(t, err)

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	assert.NoError(t, err)

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   "test-cert",
			Organization: []string{"test-org"},
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour * 24),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	assert.NoError(t, err)

	cert, err := x509.ParseCertificate(certBytes)
	assert.NoError(t, err)

	return cert, privateKey
}

// Helper function to convert cert and key to PEM strings.
func certAndKeyToPEM(cert *x509.Certificate, key *rsa.PrivateKey) (certPEM, keyPEM string) {
	certPEM = ""
	keyPEM = ""

	if cert != nil {
		certPEM = string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw}))
	}

	if key != nil {
		keyPEM = string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}))
	}

	return certPEM, keyPEM
}

func TestParseCertificateFromPEM(t *testing.T) {
	t.Parallel()

	// Generate a test certificate
	cert, key := generateTestCertAndKey(t)

	// Convert to PEM
	certPEM, keyPEM := certAndKeyToPEM(cert, key)

	// Parse back
	parsedCert, parsedKey, err := ParseCertificateFromPEM(certPEM, keyPEM)
	assert.NoError(t, err)
	assert.NotNil(t, parsedCert)
	assert.NotNil(t, parsedKey)
	assert.Equal(t, cert.Subject.CommonName, parsedCert.Subject.CommonName)
}

func TestParseCertificateFromPEM_InvalidCert(t *testing.T) {
	t.Parallel()

	_, _, err := ParseCertificateFromPEM("invalid-pem", "-----BEGIN RSA PRIVATE KEY-----\ntest\n-----END RSA PRIVATE KEY-----")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode certificate PEM")
}

func TestParseCertificateFromPEM_InvalidKey(t *testing.T) {
	t.Parallel()

	cert, _ := generateTestCertAndKey(t)
	certPEM, _ := certAndKeyToPEM(cert, nil)

	_, _, err := ParseCertificateFromPEM(certPEM, "invalid-key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode private key PEM")
}

func TestLoadCertificateFromStore_Success(t *testing.T) {
	t.Parallel()

	cert, key := generateTestCertAndKey(t)
	certPEM, keyPEM := certAndKeyToPEM(cert, key)

	mockStore := new(MockObjectStorager)
	mockStore.On("GetObject", "certs/test-cert").Return(map[string]string{
		"cert": certPEM,
		"key":  keyPEM,
	}, nil)

	loadedCert, loadedKey, err := LoadCertificateFromStore(mockStore, "test-cert")
	assert.NoError(t, err)
	assert.NotNil(t, loadedCert)
	assert.NotNil(t, loadedKey)
	assert.Equal(t, cert.Subject.CommonName, loadedCert.Subject.CommonName)

	mockStore.AssertExpectations(t)
}

func TestLoadCertificateFromStore_NotFound(t *testing.T) {
	t.Parallel()

	mockStore := new(MockObjectStorager)
	mockStore.On("GetObject", "certs/non-existent").Return(nil, assert.AnError)

	_, _, err := LoadCertificateFromStore(mockStore, "non-existent")
	assert.Error(t, err)

	mockStore.AssertExpectations(t)
}

func TestSaveCertificateToStore_Success(t *testing.T) {
	t.Parallel()

	cert, key := generateTestCertAndKey(t)

	mockStore := new(MockObjectStorager)
	mockStore.On("SetObject", "certs/test-cert", mock.AnythingOfType("map[string]string")).Return(nil)

	err := SaveCertificateToStore(mockStore, "test-cert", cert, key)
	assert.NoError(t, err)

	mockStore.AssertExpectations(t)
}

func TestSaveCertificateToStore_Error(t *testing.T) {
	t.Parallel()

	cert, key := generateTestCertAndKey(t)

	mockStore := new(MockObjectStorager)
	mockStore.On("SetObject", "certs/test-cert", mock.AnythingOfType("map[string]string")).Return(assert.AnError)

	err := SaveCertificateToStore(mockStore, "test-cert", cert, key)
	assert.Error(t, err)

	mockStore.AssertExpectations(t)
}

// setupConfigDir creates a temporary config directory and changes to its parent
// so that functions writing to "config/" work correctly.
// Returns a cleanup function that restores the original working directory.
func setupConfigDir(t *testing.T) func() {
	t.Helper()

	origDir, err := os.Getwd()
	assert.NoError(t, err)

	tmpDir := t.TempDir()
	err = os.MkdirAll(tmpDir+"/config", 0o755)
	assert.NoError(t, err)

	err = os.Chdir(tmpDir)
	assert.NoError(t, err)

	return func() {
		_ = os.Chdir(origDir)
	}
}

func TestGenerateRootCertificate_ReturnsParsedCert(t *testing.T) { //nolint:paralleltest // uses os.Chdir
	cleanup := setupConfigDir(t)
	defer cleanup()

	cert, key, err := GenerateRootCertificate(false, "test-root", "US", "TestOrg", false)
	assert.NoError(t, err)
	assert.NotNil(t, cert)
	assert.NotNil(t, key)
	assert.NotEmpty(t, cert.Raw, "cert.Raw must not be empty")

	// Verify .Raw can be re-parsed (the fix we made)
	reparsed, err := x509.ParseCertificate(cert.Raw)
	assert.NoError(t, err)
	assert.Equal(t, "test-root", reparsed.Subject.CommonName)
	assert.True(t, reparsed.IsCA)
}

func TestIssueWebServerCertificate_ReturnsParsedCert(t *testing.T) { //nolint:paralleltest // uses os.Chdir
	cleanup := setupConfigDir(t)
	defer cleanup()

	// Generate root cert first
	rootCert, rootKey, err := GenerateRootCertificate(false, "test-root-ca", "US", "TestOrg", false)
	assert.NoError(t, err)

	root := CertAndKeyType{Cert: rootCert, Key: rootKey}

	cert, key, err := IssueWebServerCertificate(root, false, "test-server", "US", "TestOrg", false)
	assert.NoError(t, err)
	assert.NotNil(t, cert)
	assert.NotNil(t, key)
	assert.NotEmpty(t, cert.Raw, "cert.Raw must not be empty")

	// Verify .Raw can be re-parsed (the fix we made)
	reparsed, err := x509.ParseCertificate(cert.Raw)
	assert.NoError(t, err)
	assert.Equal(t, "test-server", reparsed.Subject.CommonName)
	assert.False(t, reparsed.IsCA)
}

func TestLoadOrGenerateWebServerCertificateWithVault_LoadsFromVault(t *testing.T) { //nolint:paralleltest // uses os.Chdir
	cleanup := setupConfigDir(t)
	defer cleanup()

	cert, key := generateTestCertAndKey(t)
	certPEM, keyPEM := certAndKeyToPEM(cert, key)

	mockStore := new(MockObjectStorager)
	mockStore.On("GetObject", "certs/webserver-test-server").Return(map[string]string{
		"cert": certPEM,
		"key":  keyPEM,
	}, nil)

	root := CertAndKeyType{Cert: cert, Key: key}

	loadedCert, loadedKey, err := LoadOrGenerateWebServerCertificateWithVault(mockStore, root, false, "test-server", "US", "TestOrg", false)
	assert.NoError(t, err)
	assert.NotNil(t, loadedCert)
	assert.NotNil(t, loadedKey)

	// Verify cert files were persisted to disk
	_, certErr := os.Stat("config/test-server_cert.pem")
	assert.NoError(t, certErr, "cert file should exist on disk after Vault load")

	_, keyErr := os.Stat("config/test-server_key.pem")
	assert.NoError(t, keyErr, "key file should exist on disk after Vault load")

	mockStore.AssertExpectations(t)
}
