package domains_test

import (
	"bytes"
	"crypto/x509"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/usecase/domains"
)

func FuzzDecryptAndCheckCertExpiration(f *testing.F) {
	validPFX := generateTestPFX()
	expiredPFX := expiredTestPFX()
	truncatedValidPFX := validPFX[:len(validPFX)/2]
	corruptedValidPFX := corruptBase64String(validPFX)

	seedInputs := []struct {
		cert     string
		password string
	}{
		{cert: validPFX, password: "P@ssw0rd"},
		{cert: validPFX, password: "WrongP@ssw0rd"},
		{cert: validPFX, password: "pässwörd"},
		{cert: validPFX, password: "秘密"},
		{cert: validPFX, password: "🔐password"},
		{cert: expiredPFX, password: ""},
		{cert: "", password: ""},
		{cert: "not-base64", password: ""},
		{cert: base64.StdEncoding.EncodeToString([]byte("not a pkcs12 blob")), password: "P@ssw0rd"},
		{cert: truncatedValidPFX, password: "P@ssw0rd"},
		{cert: corruptedValidPFX, password: "P@ssw0rd"},
		{cert: strings.Repeat("A", 256), password: strings.Repeat("B", 32)},
	}

	for _, input := range seedInputs {
		f.Add(input.cert, input.password)
	}

	f.Fuzz(func(t *testing.T, cert, password string) {
		domain := dto.Domain{
			ProvisioningCert:         cert,
			ProvisioningCertPassword: password,
		}

		firstCert, firstErr := domains.DecryptAndCheckCertExpiration(domain)
		secondCert, secondErr := domains.DecryptAndCheckCertExpiration(domain)

		if (firstErr == nil) != (secondErr == nil) {
			t.Fatalf("DecryptAndCheckCertExpiration error mismatch for cert len=%d password len=%d: first=%v second=%v", len(cert), len(password), firstErr, secondErr)
		}

		if firstErr != nil {
			verifyDecryptError(t, cert, password, firstErr, secondErr, firstCert, secondCert)

			return
		}

		verifyDecryptSuccess(t, cert, password, firstCert, secondCert)
	})
}

func verifyDecryptError(t *testing.T, cert, password string, firstErr, secondErr error, firstCert, secondCert *x509.Certificate) {
	t.Helper()

	if firstErr.Error() != secondErr.Error() {
		t.Fatalf("DecryptAndCheckCertExpiration error text mismatch for cert len=%d password len=%d: first=%q second=%q", len(cert), len(password), firstErr.Error(), secondErr.Error())
	}

	if firstCert != nil || secondCert != nil {
		t.Fatalf("DecryptAndCheckCertExpiration returned a certificate alongside an error for cert len=%d password len=%d", len(cert), len(password))
	}
}

func verifyDecryptSuccess(t *testing.T, cert, password string, firstCert, secondCert *x509.Certificate) {
	t.Helper()

	if firstCert == nil || secondCert == nil {
		t.Fatalf("DecryptAndCheckCertExpiration returned nil certificate without an error for cert len=%d password len=%d", len(cert), len(password))
	}

	if !firstCert.NotAfter.Equal(secondCert.NotAfter) {
		t.Fatalf("DecryptAndCheckCertExpiration NotAfter mismatch for cert len=%d password len=%d: first=%s second=%s", len(cert), len(password), firstCert.NotAfter, secondCert.NotAfter)
	}

	if !bytes.Equal(firstCert.Raw, secondCert.Raw) {
		t.Fatalf("DecryptAndCheckCertExpiration returned different certificate bytes for cert len=%d password len=%d", len(cert), len(password))
	}
}

func corruptBase64String(input string) string {
	decoded, err := base64.StdEncoding.DecodeString(input)
	if err != nil || len(decoded) == 0 {
		return input
	}

	corrupted := append([]byte(nil), decoded...)
	index := len(corrupted) / 2
	corrupted[index] ^= 0xff

	return base64.StdEncoding.EncodeToString(corrupted)
}
