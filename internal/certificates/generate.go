package certificates

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net/url"
	"os"
	"time"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/security"

	"github.com/device-management-toolkit/console/pkg/logger"
)

// Package-level logger for certificate operations.
var log = logger.New("info")

// Certificate file path constants.
const (
	RootCertPath = "config/root_cert.pem"
	RootKeyPath  = "config/root_key.pem"
)

// Sentinel errors for certificate operations.
var (
	ErrCertFieldNotFound     = errors.New("cert field not found in secret")
	ErrKeyFieldNotFound      = errors.New("key field not found in secret")
	ErrStoreNotObjectStorage = errors.New("store does not support object storage")
	ErrDecodeCertificatePEM  = errors.New("failed to decode certificate PEM")
	ErrDecodePrivateKeyPEM   = errors.New("failed to decode private key PEM")
	ErrCertFilesNotFound     = errors.New("certificate files not found")
)

// ObjectStorager extends security.Storager with object storage capabilities.
type ObjectStorager interface {
	security.Storager
	GetObject(key string) (map[string]string, error)
	SetObject(key string, data map[string]string) error
}

// LoadCertificateFromStore loads a certificate and private key from a security.Storager.
// If the store implements ObjectStorager, certificates are stored as {cert, key} fields.
// Path: certs/{name}.
func LoadCertificateFromStore(store security.Storager, name string) (*x509.Certificate, *rsa.PrivateKey, error) {
	// Try object storage first (stores cert/key as proper fields)
	if objStore, ok := store.(ObjectStorager); ok {
		data, err := objStore.GetObject("certs/" + name)
		if err != nil {
			return nil, nil, err
		}

		certPEM, ok := data["cert"]
		if !ok {
			return nil, nil, ErrCertFieldNotFound
		}

		keyPEM, ok := data["key"]
		if !ok {
			return nil, nil, ErrKeyFieldNotFound
		}

		return ParseCertificateFromPEM(certPEM, keyPEM)
	}

	// Fallback: not supported for non-object stores
	return nil, nil, ErrStoreNotObjectStorage
}

// SaveCertificateToStore saves a certificate and private key to a security.Storager.
// If the store implements ObjectStorager, certificates are stored as {cert, key} fields.
// Path: certs/{name}.
func SaveCertificateToStore(store security.Storager, name string, cert *x509.Certificate, key *rsa.PrivateKey) error {
	certPEM := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw}))
	keyPEM := string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}))

	// Try object storage first (stores cert/key as proper fields)
	if objStore, ok := store.(ObjectStorager); ok {
		return objStore.SetObject("certs/"+name, map[string]string{
			"cert": certPEM,
			"key":  keyPEM,
		})
	}

	// Fallback: not supported for non-object stores
	return ErrStoreNotObjectStorage
}

// ParseCertificateFromPEM parses PEM-encoded certificate and key strings.
func ParseCertificateFromPEM(certPEM, keyPEM string) (*x509.Certificate, *rsa.PrivateKey, error) {
	// Decode certificate PEM
	certBlock, _ := pem.Decode([]byte(certPEM))
	if certBlock == nil {
		return nil, nil, ErrDecodeCertificatePEM
	}

	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Decode private key PEM
	keyBlock, _ := pem.Decode([]byte(keyPEM))
	if keyBlock == nil {
		return nil, nil, ErrDecodePrivateKeyPEM
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	return cert, privateKey, nil
}

// LoadCertificateFromFile loads a certificate and private key from PEM files.
func LoadCertificateFromFile(certPath, keyPath string) (*x509.Certificate, *rsa.PrivateKey, error) {
	// Read certificate file
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read certificate file: %w", err)
	}

	// Decode PEM block
	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil {
		return nil, nil, ErrDecodeCertificatePEM
	}

	// Parse certificate
	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Read private key file
	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read private key file: %w", err)
	}

	// Decode PEM block
	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return nil, nil, ErrDecodePrivateKeyPEM
	}

	// Parse private key
	privateKey, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	return cert, privateKey, nil
}

// CheckAndLoadOrGenerateRootCertificate checks if root certificate files exist,
// loads them if they do, or generates new ones if they don't.
func CheckAndLoadOrGenerateRootCertificate(addThumbPrintToName bool, commonName, country, organization string, strong bool) (*x509.Certificate, *rsa.PrivateKey, error) {
	// Check if both files exist
	_, certErr := os.Stat(RootCertPath)
	_, keyErr := os.Stat(RootKeyPath)

	if certErr == nil && keyErr == nil {
		// Files exist, try to load them
		cert, key, err := LoadCertificateFromFile(RootCertPath, RootKeyPath)
		if err == nil {
			return cert, key, nil
		}
		// If loading fails, fall through to generation
		log.Warn("Warning: Failed to load existing certificates: %v. Generating new ones...", err)
	}

	// Files don't exist or loading failed, generate new certificates
	return GenerateRootCertificate(addThumbPrintToName, commonName, country, organization, strong)
}

// LoadOrGenerateRootCertificateWithVault attempts to load the root certificate from Vault first,
// falls back to local files, and generates new certificates if neither exists.
// When a new certificate is generated, it is stored in Vault (if available) and locally.
// Certificate is stored at: {basePath}/certs/root.
func LoadOrGenerateRootCertificateWithVault(store security.Storager, addThumbPrintToName bool, commonName, country, organization string, strong bool) (*x509.Certificate, *rsa.PrivateKey, error) {
	const certName = "root"

	log.Info("[Certificate Flow] Loading root certificate (certName=%s)", certName)

	// Try Vault first (primary store for high-value certs)
	if store != nil {
		log.Info("[Certificate Flow] [Vault Storage] Attempting to load root certificate from Vault")
		cert, key, err := LoadCertificateFromStore(store, certName)
		if err == nil {
			log.Info("[Certificate Flow] ✓ SUCCESS: Root certificate loaded from Vault")

			return cert, key, nil
		}

		log.Warn("[Certificate Flow] ✗ Vault load failed: %v. Proceeding to Local Files...", err)
	} else {
		log.Info("[Certificate Flow] Vault client not available (store=nil), skipping Vault Storage")
	}

	// Try local files as fallback
	cert, key, err := tryLoadRootCertFromFiles(store, certName)
	if err == nil {
		return cert, key, nil
	}

	// Generate new certificates
	return generateAndStoreRootCert(store, certName, addThumbPrintToName, commonName, country, organization, strong)
}

// tryLoadRootCertFromFiles attempts to load root certificate from local files and sync to vault.
func tryLoadRootCertFromFiles(store security.Storager, certName string) (*x509.Certificate, *rsa.PrivateKey, error) {
	log.Info("[Certificate Flow] [Local Files] Checking for certificate files: %s, %s", RootCertPath, RootKeyPath)

	_, certErr := os.Stat(RootCertPath)
	_, keyErr := os.Stat(RootKeyPath)

	if certErr != nil || keyErr != nil {
		log.Warn("[Certificate Flow] ✗ Local certificate files not found at %s and %s, proceeding to Generate New", RootCertPath, RootKeyPath)
		return nil, nil, ErrCertFilesNotFound
	}

	log.Info("[Certificate Flow] Certificate files found at %s and %s, attempting to load...", RootCertPath, RootKeyPath)
	cert, key, err := LoadCertificateFromFile(RootCertPath, RootKeyPath)
	if err != nil {
		log.Warn("[Certificate Flow] ✗ Failed to parse local certificates: %v. Proceeding to Generate New", err)

		return nil, nil, err
	}

	log.Info("[Certificate Flow] ✓ SUCCESS: Root certificate loaded from local files (%s)", RootCertPath)

	// Sync to Vault for future use
	if store != nil {
		log.Info("[Certificate Flow] Syncing local certificate to Vault for future use...")
		if syncErr := SaveCertificateToStore(store, certName, cert, key); syncErr != nil {
			log.Warn("[Certificate Flow] ✗ Warning: Failed to sync root certificate to Vault: %v", syncErr)
		} else {
			log.Info("[Certificate Flow] ✓ Root certificate synced to Vault successfully")
		}
	} else {
		log.Info("[Certificate Flow] Vault not available, skipping sync")
	}

	return cert, key, nil
}

// generateAndStoreRootCert generates a new root certificate and stores it.
func generateAndStoreRootCert(store security.Storager, certName string, addThumbPrintToName bool, commonName, country, organization string, strong bool) (*x509.Certificate, *rsa.PrivateKey, error) {
	log.Info("[Certificate Flow] [Generate New] Generating new root certificate...")

	cert, key, err := GenerateRootCertificate(addThumbPrintToName, commonName, country, organization, strong)
	if err != nil {
		log.Error("[Certificate Flow] ✗ FAILED: Certificate generation error: %v", err)
		return nil, nil, err
	}

	log.Info("[Certificate Flow] ✓ New root certificate generated successfully (will be saved to %s)", RootCertPath)

	// Store in Vault (primary)
	if store != nil {
		log.Info("[Certificate Flow] Storing generated certificate in Vault...")
		if storeErr := SaveCertificateToStore(store, certName, cert, key); storeErr != nil {
			log.Warn("[Certificate Flow] ✗ Warning: Failed to store root certificate in Vault: %v", storeErr)
		} else {
			log.Info("[Certificate Flow] ✓ Root certificate stored in Vault")
		}
	} else {
		log.Info("[Certificate Flow] Vault not available, certificate will only be stored locally at %s and %s", RootCertPath, RootKeyPath)
	}

	return cert, key, nil
}

// CheckAndLoadOrGenerateWebServerCertificate checks if web server certificate files exist,
// loads them if they do, or generates new ones if they don't.
func CheckAndLoadOrGenerateWebServerCertificate(rootCert CertAndKeyType, addThumbPrintToName bool, commonName, country, organization string, strong bool) (*x509.Certificate, *rsa.PrivateKey, error) {
	certPath := "config/" + commonName + "_cert.pem"
	keyPath := "config/" + commonName + "_key.pem"

	// Check if both files exist
	_, certErr := os.Stat(certPath)
	_, keyErr := os.Stat(keyPath)

	if certErr == nil && keyErr == nil {
		// Files exist, try to load them
		cert, key, err := LoadCertificateFromFile(certPath, keyPath)
		if err == nil {
			return cert, key, nil
		}
		// If loading fails, fall through to generation
		log.Warn("Warning: Failed to load existing certificates: %v. Generating new ones...", err)
	}

	// Files don't exist or loading failed, generate new certificates
	return IssueWebServerCertificate(rootCert, addThumbPrintToName, commonName, country, organization, strong)
}

// LoadOrGenerateWebServerCertificateWithVault attempts to load the web server certificate from Vault first,
// falls back to local files, and generates new certificates if neither exists.
// When a new certificate is generated, it is stored in Vault (if available) and locally.
// Certificate is stored at: {basePath}/certs/webserver-{commonName}.
func LoadOrGenerateWebServerCertificateWithVault(store security.Storager, rootCert CertAndKeyType, addThumbPrintToName bool, commonName, country, organization string, strong bool) (*x509.Certificate, *rsa.PrivateKey, error) {
	certName := "webserver-" + commonName
	certPath := "config/" + commonName + "_cert.pem"
	keyPath := "config/" + commonName + "_key.pem"

	log.Info("[Certificate Flow] Loading web server certificate (certName=%s, commonName=%s)", certName, commonName)

	// Try Vault first (primary store for high-value certs)
	if store != nil {
		log.Info("[Certificate Flow] [Vault Storage] Attempting to load web server certificate from Vault")
		cert, key, err := LoadCertificateFromStore(store, certName)
		if err == nil {
			log.Info("[Certificate Flow] ✓ SUCCESS: Web server certificate loaded from Vault")

			return cert, key, nil
		}

		log.Warn("[Certificate Flow] ✗ Vault load failed: %v. Proceeding to Local Files...", err)
	} else {
		log.Info("[Certificate Flow] Vault client not available (store=nil), skipping Vault Storage")
	}

	// Try local files as fallback
	cert, key, err := tryLoadWebServerCertFromFiles(store, certName, certPath, keyPath)
	if err == nil {
		return cert, key, nil
	}

	// Generate new certificates
	return generateAndStoreWebServerCert(store, rootCert, certName, addThumbPrintToName, commonName, country, organization, strong)
}

// tryLoadWebServerCertFromFiles attempts to load web server certificate from local files and sync to vault.
func tryLoadWebServerCertFromFiles(store security.Storager, certName, certPath, keyPath string) (*x509.Certificate, *rsa.PrivateKey, error) {
	log.Info("[Certificate Flow] [Local Files] Checking for certificate files: %s, %s", certPath, keyPath)

	_, certErr := os.Stat(certPath)
	_, keyErr := os.Stat(keyPath)

	if certErr != nil || keyErr != nil {
		log.Warn("[Certificate Flow] ✗ Local certificate files not found at %s and %s, proceeding to Generate New", certPath, keyPath)
		return nil, nil, ErrCertFilesNotFound
	}

	log.Info("[Certificate Flow] Certificate files found at %s and %s, attempting to load...", certPath, keyPath)
	cert, key, err := LoadCertificateFromFile(certPath, keyPath)
	if err != nil {
		log.Warn("[Certificate Flow] ✗ Failed to parse local certificates: %v. Proceeding to Generate New", err)

		return nil, nil, err
	}

	log.Info("[Certificate Flow] ✓ SUCCESS: Web server certificate loaded from local files (%s)", certPath)

	// Sync to Vault for future use
	if store != nil {
		log.Info("[Certificate Flow] Syncing local certificate to Vault for future use...")
		if syncErr := SaveCertificateToStore(store, certName, cert, key); syncErr != nil {
			log.Warn("[Certificate Flow] ✗ Warning: Failed to sync web server certificate to Vault: %v", syncErr)
		} else {
			log.Info("[Certificate Flow] ✓ Web server certificate synced to Vault successfully")
		}
	} else {
		log.Info("[Certificate Flow] Vault not available, skipping sync")
	}

	return cert, key, nil
}

// generateAndStoreWebServerCert generates a new web server certificate and stores it.
func generateAndStoreWebServerCert(store security.Storager, rootCert CertAndKeyType, certName string, addThumbPrintToName bool, commonName, country, organization string, strong bool) (*x509.Certificate, *rsa.PrivateKey, error) {
	certPath := "config/" + commonName + "_cert.pem"
	log.Info("[Certificate Flow] [Generate New] Generating new web server certificate (will be saved to %s)...", certPath)

	cert, key, err := IssueWebServerCertificate(rootCert, addThumbPrintToName, commonName, country, organization, strong)
	if err != nil {
		log.Error("[Certificate Flow] ✗ FAILED: Certificate generation error: %v", err)
		return nil, nil, err
	}

	log.Info("[Certificate Flow] ✓ New web server certificate generated successfully")

	// Store in Vault (primary)
	if store != nil {
		log.Info("[Certificate Flow] Storing generated certificate in Vault...")
		if storeErr := SaveCertificateToStore(store, certName, cert, key); storeErr != nil {
			log.Warn("[Certificate Flow] ✗ Warning: Failed to store web server certificate in Vault: %v", storeErr)
		} else {
			log.Info("[Certificate Flow] ✓ Web server certificate stored in Vault")
		}
	} else {
		keyPath := "config/" + commonName + "_key.pem"
		log.Info("[Certificate Flow] Vault not available, certificate will only be stored locally at %s and %s", certPath, keyPath)
	}

	return cert, key, nil
}

func GenerateRootCertificate(addThumbPrintToName bool, commonName, country, organization string, strong bool) (*x509.Certificate, *rsa.PrivateKey, error) {
	keyLength := 2048
	if strong {
		keyLength = 3072
	}

	// Generate RSA keys
	privateKey, err := rsa.GenerateKey(rand.Reader, keyLength)
	if err != nil {
		return nil, nil, err
	}

	// Preparing the certificate
	var maxValue uint = 128

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), maxValue))
	if err != nil {
		return nil, nil, err
	}

	thirtyYears := 30

	if addThumbPrintToName {
		hash := sha256.New()
		hash.Write(privateKey.N.Bytes()) // Simplified approach to get a thumbprint-like result
		commonName += "-" + fmt.Sprintf("%x", hash.Sum(nil)[:3])
	}

	if country == "" {
		country = "unknown country"
	}

	if organization == "" {
		organization = "unknown organization"
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   commonName,
			Organization: []string{organization},
			Country:      []string{country},
		},
		NotBefore: time.Now().AddDate(-1, 0, 0),
		NotAfter:  time.Now().AddDate(thirtyYears, 0, 0),

		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	// Create a self-signed certificate
	certBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, nil, err
	}

	// Encoding certificate to PEM format
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes})

	// Save to files (optional)
	certOut, err := os.Create("config/root_cert.pem")
	if err != nil {
		return nil, nil, err
	}

	_, err = certOut.Write(certPEM)
	if err != nil {
		return nil, nil, err
	}

	certOut.Close()

	keyOut, err := os.Create("config/root_key.pem")
	if err != nil {
		return nil, nil, err
	}

	err = pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})
	if err != nil {
		return nil, nil, err
	}

	keyOut.Close()

	return &template, privateKey, nil
}

type CertAndKeyType struct {
	Cert *x509.Certificate
	Key  *rsa.PrivateKey
}

func IssueWebServerCertificate(rootCert CertAndKeyType, addThumbPrintToName bool, commonName, country, organization string, strong bool) (*x509.Certificate, *rsa.PrivateKey, error) {
	keyLength := 2048
	if strong {
		keyLength = 3072
	}

	// Generate RSA keys
	keys, err := rsa.GenerateKey(rand.Reader, keyLength)
	if err != nil {
		return nil, nil, err
	}

	var maxValue uint = 128

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), maxValue))
	if err != nil {
		return nil, nil, err
	}

	thirtyYears := 30
	notBefore := time.Now().AddDate(-1, 0, 0)
	notAfter := time.Now().AddDate(thirtyYears, 0, 0)

	subject := pkix.Name{
		CommonName: commonName,
	}

	if country != "" {
		subject.Country = []string{country}
	}

	if organization != "" {
		subject.Organization = []string{organization}
	}

	if addThumbPrintToName {
		hash := sha256.New()
		hash.Write(keys.N.Bytes()) // Simplified approach to get a thumbprint-like result
		subject.CommonName += "-" + string(hash.Sum(nil)[:3])
	}

	hash := sha256.Sum256(keys.N.Bytes())

	template := x509.Certificate{
		SerialNumber:          serialNumber,
		Subject:               subject,
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment | x509.KeyUsageCertSign | x509.KeyUsageDataEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  false,
		SubjectKeyId:          hash[:],
	}

	// Subject Alternative Name
	uri, _ := url.Parse("http://" + commonName + "/")
	template.DNSNames = []string{commonName, "localhost"}
	template.URIs = []*url.URL{uri}

	// Sign the certificate with root certificate private key
	certBytes, err := x509.CreateCertificate(rand.Reader, &template, rootCert.Cert, &keys.PublicKey, rootCert.Key)
	if err != nil {
		return nil, nil, err
	}

	// Save certificate and key to files
	if err := saveCertAndKeyToFiles(commonName, certBytes, keys); err != nil {
		return nil, nil, err
	}

	return &template, keys, nil
}

// saveCertAndKeyToFiles saves a certificate and private key to PEM files.
func saveCertAndKeyToFiles(commonName string, certBytes []byte, keys *rsa.PrivateKey) error {
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes})

	certOut, err := os.Create("config/" + commonName + "_cert.pem")
	if err != nil {
		return err
	}
	defer certOut.Close()

	if _, err = certOut.Write(certPEM); err != nil {
		return err
	}

	keyOut, err := os.Create("config/" + commonName + "_key.pem")
	if err != nil {
		return err
	}
	defer keyOut.Close()

	return pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(keys)})
}
