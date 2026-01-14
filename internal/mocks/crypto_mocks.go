package mocks

import (
	"errors"

	"gopkg.in/yaml.v2"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/config"
)

// MockCrypto is a simple mock for the security.Cryptor interface that always succeeds.
type MockCrypto struct{}

// ErrDecryptFailed is a test error for decrypt failures.
var ErrDecryptFailed = errors.New("decrypt failed")

const encryptedData = "encrypted"

// Encrypt encrypts a string.
func (c MockCrypto) Encrypt(_ string) (string, error) {
	return encryptedData, nil
}

// EncryptWithKey encrypts a string with a key.
func (c MockCrypto) EncryptWithKey(_, _ string) (string, error) {
	return encryptedData, nil
}

// GenerateKey generates a key.
func (c MockCrypto) GenerateKey() string {
	return "key"
}

// Decrypt decrypts a string.
func (c MockCrypto) Decrypt(_ string) (string, error) {
	return "decrypted", nil
}

// ReadAndDecryptFile reads encrypted data from file and decrypts it.
func (c MockCrypto) ReadAndDecryptFile(_ string) (config.Configuration, error) {
	var configuration config.Configuration

	err := yaml.Unmarshal([]byte(""), &configuration)
	if err != nil {
		return config.Configuration{}, err
	}

	return configuration, nil
}
