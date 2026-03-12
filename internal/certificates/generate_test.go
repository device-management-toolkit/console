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
	"github.com/stretchr/testify/require"
)

const testServerName = "test-server"

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

// TestLoadOrGenerateRootCert_FromVault_NoLocalFiles verifies that when local
// certificate files are deleted, the root certificate is loaded directly from
// Vault without attempting file I/O or generating new certificates.
func TestLoadOrGenerateRootCert_FromVault_NoLocalFiles(t *testing.T) {
	t.Parallel()

	cert, key := generateTestCertAndKey(t)
	certPEM, keyPEM := certAndKeyToPEM(cert, key)

	mockStore := new(MockObjectStorager)
	mockStore.On("GetObject", "certs/root").Return(map[string]string{
		"cert": certPEM,
		"key":  keyPEM,
	}, nil)

	// Ensure local files do not exist
	os.Remove(RootCertPath)
	os.Remove(RootKeyPath)

	loadedCert, loadedKey, err := LoadOrGenerateRootCertificateWithVault(mockStore, true, "test", "US", "test-org", false)

	assert.NoError(t, err)
	assert.NotNil(t, loadedCert)
	assert.NotNil(t, loadedKey)
	assert.Equal(t, cert.Subject.CommonName, loadedCert.Subject.CommonName)

	// SetObject should NOT be called — cert was already in Vault
	mockStore.AssertExpectations(t)
	mockStore.AssertNotCalled(t, "SetObject", mock.Anything, mock.Anything)
}

// TestLoadOrGenerateRootCert_FromLocalFiles_SyncsToVault verifies that when
// Vault has no certificate but local files exist, the cert is loaded from
// files and then synced to Vault.
func TestLoadOrGenerateRootCert_FromLocalFiles_SyncsToVault(t *testing.T) {
	t.Parallel()

	// Generate test cert and write to local files
	cert, key := generateTestCertAndKey(t)
	certPEM, keyPEM := certAndKeyToPEM(cert, key)

	// Create config directory if needed
	require.NoError(t, os.MkdirAll("config", 0o755))

	err := os.WriteFile(RootCertPath, []byte(certPEM), 0o600)
	assert.NoError(t, err)

	err = os.WriteFile(RootKeyPath, []byte(keyPEM), 0o600)
	assert.NoError(t, err)

	t.Cleanup(func() {
		os.Remove(RootCertPath)
		os.Remove(RootKeyPath)
	})

	mockStore := new(MockObjectStorager)
	// Vault returns not found
	mockStore.On("GetObject", "certs/root").Return(nil, assert.AnError)
	// Expect sync to Vault after loading from files
	mockStore.On("SetObject", "certs/root", mock.AnythingOfType("map[string]string")).Return(nil)

	loadedCert, loadedKey, loadErr := LoadOrGenerateRootCertificateWithVault(mockStore, true, "test", "US", "test-org", false)

	assert.NoError(t, loadErr)
	assert.NotNil(t, loadedCert)
	assert.NotNil(t, loadedKey)

	// Verify cert was synced to Vault
	mockStore.AssertExpectations(t)
	mockStore.AssertCalled(t, "SetObject", "certs/root", mock.AnythingOfType("map[string]string"))
}

// TestLoadOrGenerateWebServerCert_FromVault_NoLocalFiles verifies that web
// server certificates load from Vault when local files are deleted.
func TestLoadOrGenerateWebServerCert_FromVault_NoLocalFiles(t *testing.T) {
	t.Parallel()

	// Generate root cert for signing
	rootCert, rootKey := generateTestCertAndKey(t)
	// Generate web server cert
	webCert, webKey := generateTestCertAndKey(t)
	webCertPEM, webKeyPEM := certAndKeyToPEM(webCert, webKey)

	commonName := testServerName
	certPath := "config/" + commonName + "_cert.pem"
	keyPath := "config/" + commonName + "_key.pem"

	// Ensure local files do not exist
	os.Remove(certPath)
	os.Remove(keyPath)

	mockStore := new(MockObjectStorager)
	mockStore.On("GetObject", "certs/webserver-"+commonName).Return(map[string]string{
		"cert": webCertPEM,
		"key":  webKeyPEM,
	}, nil)

	rootCertAndKey := CertAndKeyType{Cert: rootCert, Key: rootKey}

	loadedCert, loadedKey, err := LoadOrGenerateWebServerCertificateWithVault(mockStore, rootCertAndKey, false, commonName, "US", "test-org", false)

	assert.NoError(t, err)
	assert.NotNil(t, loadedCert)
	assert.NotNil(t, loadedKey)

	// SetObject should NOT be called — cert was already in Vault
	mockStore.AssertExpectations(t)
	mockStore.AssertNotCalled(t, "SetObject", mock.Anything, mock.Anything)
}

// TestLoadOrGenerateWebServerCert_FromLocalFiles_SyncsToVault verifies that
// web server certificates load from local files and sync to Vault when Vault
// has no copy.
func TestLoadOrGenerateWebServerCert_FromLocalFiles_SyncsToVault(t *testing.T) {
	t.Parallel()

	rootCert, rootKey := generateTestCertAndKey(t)
	webCert, webKey := generateTestCertAndKey(t)
	webCertPEM, webKeyPEM := certAndKeyToPEM(webCert, webKey)

	commonName := testServerName
	certPath := "config/" + commonName + "_cert.pem"
	keyPath := "config/" + commonName + "_key.pem"

	require.NoError(t, os.MkdirAll("config", 0o755))

	err := os.WriteFile(certPath, []byte(webCertPEM), 0o600)
	assert.NoError(t, err)

	err = os.WriteFile(keyPath, []byte(webKeyPEM), 0o600)
	assert.NoError(t, err)

	t.Cleanup(func() {
		os.Remove(certPath)
		os.Remove(keyPath)
	})

	mockStore := new(MockObjectStorager)
	mockStore.On("GetObject", "certs/webserver-"+commonName).Return(nil, assert.AnError)
	mockStore.On("SetObject", "certs/webserver-"+commonName, mock.AnythingOfType("map[string]string")).Return(nil)

	rootCertAndKey := CertAndKeyType{Cert: rootCert, Key: rootKey}

	loadedCert, loadedKey, loadErr := LoadOrGenerateWebServerCertificateWithVault(mockStore, rootCertAndKey, false, commonName, "US", "test-org", false)

	assert.NoError(t, loadErr)
	assert.NotNil(t, loadedCert)
	assert.NotNil(t, loadedKey)

	mockStore.AssertExpectations(t)
	mockStore.AssertCalled(t, "SetObject", "certs/webserver-"+commonName, mock.AnythingOfType("map[string]string"))
}

// TestLoadOrGenerateRootCert_NilStore_GeneratesLocally verifies that when
// store is nil, the root certificate is generated and saved locally only.
func TestLoadOrGenerateRootCert_NilStore_GeneratesLocally(t *testing.T) {
	t.Parallel()

	require.NoError(t, os.MkdirAll("config", 0o755))

	t.Cleanup(func() {
		os.Remove(RootCertPath)
		os.Remove(RootKeyPath)
	})

	// Remove any existing local files so generation is triggered
	os.Remove(RootCertPath)
	os.Remove(RootKeyPath)

	cert, key, err := LoadOrGenerateRootCertificateWithVault(nil, true, "test-nil-store", "US", "test-org", false)

	assert.NoError(t, err)
	assert.NotNil(t, cert)
	assert.NotNil(t, key)
}

// TestLoadOrGenerateWebServerCert_NilStore_GeneratesLocally verifies that when
// store is nil, the web server certificate is generated and saved locally only.
func TestLoadOrGenerateWebServerCert_NilStore_GeneratesLocally(t *testing.T) {
	t.Parallel()

	require.NoError(t, os.MkdirAll("config", 0o755))

	rootCert, rootKey := generateTestCertAndKey(t)

	commonName := "test-nil-store-ws"
	certPath := "config/" + commonName + "_cert.pem"
	keyPath := "config/" + commonName + "_key.pem"

	t.Cleanup(func() {
		os.Remove(certPath)
		os.Remove(keyPath)
	})

	rootCertAndKey := CertAndKeyType{Cert: rootCert, Key: rootKey}

	cert, key, err := LoadOrGenerateWebServerCertificateWithVault(nil, rootCertAndKey, false, commonName, "US", "test-org", false)

	assert.NoError(t, err)
	assert.NotNil(t, cert)
	assert.NotNil(t, key)
}
