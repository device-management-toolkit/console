package mocks

import (
	"gopkg.in/yaml.v2"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/config"
)

type MockCrypto struct{}

// MockError simulates crypto errors for testing.
type MockError struct {
	message string
}

func (e *MockError) Error() string {
	return e.message
}

const encryptedData = "encrypted"

// Encrypt encrypts a string.
func (c MockCrypto) Encrypt(_ string) (string, error) {
	return encryptedData, nil
}

// Encrypt encrypts a string.
func (c MockCrypto) EncryptWithKey(_, _ string) (string, error) {
	return encryptedData, nil
}

func (c MockCrypto) GenerateKey() string {
	return "key"
}

func (c MockCrypto) Decrypt(input string) (string, error) {
	// Special case to simulate decryption failure for testing coverage
	if input == "fail-decrypt" {
		return "", &MockError{message: "mock decrypt failure"}
	}

	return "decrypted", nil
}

// Read encrypted data from file and decrypt it.
func (c MockCrypto) ReadAndDecryptFile(_ string) (config.Configuration, error) {
	var configuration config.Configuration

	err := yaml.Unmarshal([]byte(""), &configuration)
	if err != nil {
		return config.Configuration{}, err
	}

	return configuration, nil
}
