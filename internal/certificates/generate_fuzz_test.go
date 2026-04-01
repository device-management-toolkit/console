package certificates

import (
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	mathrand "math/rand"
	"strings"
	"testing"
	"time"
)

func FuzzParseCertificateFromPEM(f *testing.F) {
	validCertPEM, validKeyPEM := generateFuzzPEMCertificate(f, 1)
	otherCertPEM, otherKeyPEM := generateFuzzPEMCertificate(f, 2)
	truncatedCertPEM := validCertPEM[:len(validCertPEM)/2]
	truncatedKeyPEM := validKeyPEM[:len(validKeyPEM)/2]
	corruptedCertPEM := corruptPEMBody(validCertPEM)
	corruptedKeyPEM := corruptPEMBody(validKeyPEM)

	seedInputs := []struct {
		certPEM string
		keyPEM  string
	}{
		{certPEM: validCertPEM, keyPEM: validKeyPEM},
		{certPEM: otherCertPEM, keyPEM: otherKeyPEM},
		{certPEM: validCertPEM, keyPEM: otherKeyPEM},
		{certPEM: truncatedCertPEM, keyPEM: validKeyPEM},
		{certPEM: validCertPEM, keyPEM: truncatedKeyPEM},
		{certPEM: corruptedCertPEM, keyPEM: validKeyPEM},
		{certPEM: validCertPEM, keyPEM: corruptedKeyPEM},
		{certPEM: "", keyPEM: ""},
		{certPEM: "invalid-pem", keyPEM: validKeyPEM},
		{certPEM: invalidBase64PEM("CERTIFICATE"), keyPEM: validKeyPEM},
		{certPEM: validCertPEM, keyPEM: "invalid-key"},
		{certPEM: validCertPEM, keyPEM: invalidBase64PEM("RSA PRIVATE KEY")},
		{certPEM: validKeyPEM, keyPEM: validCertPEM},
		{certPEM: "前置\n" + validCertPEM + "後置", keyPEM: "🔐\n" + validKeyPEM},
		{certPEM: strings.Repeat("A", 256), keyPEM: strings.Repeat("B", 256)},
	}

	for _, input := range seedInputs {
		f.Add(input.certPEM, input.keyPEM)
	}

	f.Fuzz(func(t *testing.T, certPEM, keyPEM string) {
		firstCert, firstKey, firstErr := ParseCertificateFromPEM(certPEM, keyPEM)
		secondCert, secondKey, secondErr := ParseCertificateFromPEM(certPEM, keyPEM)

		if (firstErr == nil) != (secondErr == nil) {
			t.Fatalf("ParseCertificateFromPEM error mismatch for cert len=%d key len=%d: first=%v second=%v", len(certPEM), len(keyPEM), firstErr, secondErr)
		}

		if firstErr != nil {
			verifyParseCertError(t, certPEM, keyPEM, firstErr, secondErr, firstCert, secondCert, firstKey, secondKey)

			return
		}

		verifyParseCertSuccess(t, certPEM, keyPEM, firstCert, secondCert, firstKey, secondKey)
	})
}

func verifyParseCertError(t *testing.T, certPEM, keyPEM string, firstErr, secondErr error, firstCert, secondCert *x509.Certificate, firstKey, secondKey *rsa.PrivateKey) {
	t.Helper()

	if firstErr.Error() != secondErr.Error() {
		t.Fatalf("ParseCertificateFromPEM error text mismatch for cert len=%d key len=%d: first=%q second=%q", len(certPEM), len(keyPEM), firstErr.Error(), secondErr.Error())
	}

	if firstCert != nil || secondCert != nil || firstKey != nil || secondKey != nil {
		t.Fatalf("ParseCertificateFromPEM returned parsed data alongside an error for cert len=%d key len=%d", len(certPEM), len(keyPEM))
	}
}

func verifyParseCertSuccess(t *testing.T, certPEM, keyPEM string, firstCert, secondCert *x509.Certificate, firstKey, secondKey *rsa.PrivateKey) {
	t.Helper()

	if firstCert == nil || secondCert == nil || firstKey == nil || secondKey == nil {
		t.Fatalf("ParseCertificateFromPEM returned nil data without an error for cert len=%d key len=%d", len(certPEM), len(keyPEM))
	}

	if firstCert.SerialNumber.Cmp(secondCert.SerialNumber) != 0 {
		t.Fatalf("ParseCertificateFromPEM serial number mismatch for cert len=%d key len=%d", len(certPEM), len(keyPEM))
	}

	if firstCert.Subject.CommonName != secondCert.Subject.CommonName {
		t.Fatalf("ParseCertificateFromPEM common name mismatch for cert len=%d key len=%d: first=%q second=%q", len(certPEM), len(keyPEM), firstCert.Subject.CommonName, secondCert.Subject.CommonName)
	}

	if firstKey.E != secondKey.E || firstKey.N.Cmp(secondKey.N) != 0 || firstKey.D.Cmp(secondKey.D) != 0 {
		t.Fatalf("ParseCertificateFromPEM returned different private keys for cert len=%d key len=%d", len(certPEM), len(keyPEM))
	}
}

func invalidBase64PEM(blockType string) string {
	return "-----BEGIN " + blockType + "-----\n!!!!\n-----END " + blockType + "-----\n"
}

func corruptPEMBody(input string) string {
	block, _ := pem.Decode([]byte(input))
	if block == nil || len(block.Bytes) == 0 {
		return input
	}

	corrupted := append([]byte(nil), block.Bytes...)
	index := len(corrupted) / 2
	corrupted[index] ^= 0xff

	return string(pem.EncodeToMemory(&pem.Block{Type: block.Type, Bytes: corrupted}))
}

func generateFuzzPEMCertificate(tb testing.TB, seed int64) (certPEM, keyPEM string) {
	tb.Helper()

	rng := mathrand.New(mathrand.NewSource(seed))

	privateKey, err := rsa.GenerateKey(rng, 1024)
	if err != nil {
		tb.Fatalf("failed to generate RSA key: %v", err)
	}

	serialNumber := new(big.Int).SetInt64(rng.Int63())

	fixedTime := time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   "fuzz-cert",
			Organization: []string{"console"},
		},
		NotBefore:             fixedTime.Add(-time.Hour),
		NotAfter:              fixedTime.Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}

	certBytes, err := x509.CreateCertificate(rng, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		tb.Fatalf("failed to create certificate: %v", err)
	}

	certPEM = string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes}))
	keyPEM = string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)}))

	return certPEM, keyPEM
}
