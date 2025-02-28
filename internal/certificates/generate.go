package certificates

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net/url"
	"os"
	"time"
)

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
		hash.Write(privateKey.PublicKey.N.Bytes()) // Simplified approach to get a thumbprint-like result
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
	certOut, err := os.Create("root_cert.pem")
	if err != nil {
		return nil, nil, err
	}

	_, err = certOut.Write(certPEM)
	if err != nil {
		return nil, nil, err
	}

	certOut.Close()

	keyOut, err := os.Create("root_key.pem")
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
		hash.Write(keys.PublicKey.N.Bytes()) // Simplified approach to get a thumbprint-like result
		subject.CommonName += "-" + string(hash.Sum(nil)[:3])
	}

	hash := sha256.Sum256(keys.PublicKey.N.Bytes())

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

	// Encoding certificate to PEM format
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes})

	// Save to files (optional)
	certOut, err := os.Create(commonName + "_cert.pem")
	if err != nil {
		return nil, nil, err
	}

	_, err = certOut.Write(certPEM)
	if err != nil {
		return nil, nil, err
	}

	certOut.Close()

	keyOut, err := os.Create(commonName + "_key.pem")
	if err != nil {
		return nil, nil, err
	}

	err = pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(keys)})
	if err != nil {
		return nil, nil, err
	}

	keyOut.Close()

	return &template, keys, nil
}
