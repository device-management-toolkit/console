// Package crypto provides a lightweight fake implementation of
// security.Cryptor for use in tests. It lives in its own leaf package (no
// dependency on internal/usecase/devices) so it can be imported by both
// internal (package foo) and external (package foo_test) test files without
// creating an import cycle through internal/mocks.
package crypto

import (
	"gopkg.in/yaml.v2"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/config"
)

type MockCrypto struct{}

const encryptedData = "encrypted"

// Encrypt encrypts a string.
func (c MockCrypto) Encrypt(_ string) (string, error) {
	return encryptedData, nil
}

// EncryptWithKey encrypts a string with the provided key.
func (c MockCrypto) EncryptWithKey(_, _ string) (string, error) {
	return encryptedData, nil
}

func (c MockCrypto) GenerateKey() string {
	return "key"
}

func (c MockCrypto) Decrypt(_ string) (string, error) {
	return "decrypted", nil
}

// ReadAndDecryptFile reads encrypted data from a file and decrypts it.
func (c MockCrypto) ReadAndDecryptFile(_ string) (config.Configuration, error) {
	var configuration config.Configuration

	err := yaml.Unmarshal([]byte(""), &configuration)
	if err != nil {
		return config.Configuration{}, err
	}

	return configuration, nil
}
