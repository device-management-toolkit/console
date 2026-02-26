package main

import (
	"crypto/rsa"
	"crypto/x509"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/security"

	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/internal/certificates"
	"github.com/device-management-toolkit/console/internal/usecase"
	"github.com/device-management-toolkit/console/pkg/logger"
)

func TestMainFunction(_ *testing.T) { //nolint:paralleltest // cannot have simultaneous tests modifying env variables.
	os.Setenv("GIN_MODE", "debug")

	// Mock functions
	initializeConfigFunc = func() (*config.Config, error) {
		return &config.Config{HTTP: config.HTTP{Port: "8080"}, App: config.App{EncryptionKey: "test"}, Log: config.Log{Level: "info"}}, nil
	}

	initializeAppFunc = func(_ *config.Config) error {
		return nil
	}

	runAppFunc = func(_ *config.Config, _ logger.Interface) {}

	// Mock certificate functions
	loadOrGenerateRootCertFunc = func(_ security.Storager, _ bool, _, _, _ string, _ bool) (*x509.Certificate, *rsa.PrivateKey, error) {
		return &x509.Certificate{}, &rsa.PrivateKey{}, nil
	}

	loadOrGenerateWebServerCertFunc = func(_ security.Storager, _ certificates.CertAndKeyType, _ bool, _, _, _ string, _ bool) (*x509.Certificate, *rsa.PrivateKey, error) {
		return &x509.Certificate{}, &rsa.PrivateKey{}, nil
	}

	// Call the main function
	main()
}

type MockGenerator struct {
	mock.Mock
}

func (m *MockGenerator) GenerateSpec() ([]byte, error) {
	args := m.Called()

	var b []byte

	if v := args.Get(0); v != nil {
		if bb, ok := v.([]byte); ok {
			b = bb
		}
	}

	return b, args.Error(1)
}

func (m *MockGenerator) SaveSpec(b []byte, path string) error {
	args := m.Called(b, path)

	return args.Error(0)
}

//nolint:paralleltest // modifies package-level NewGeneratorFunc
func TestHandleOpenAPIGeneration_Success(t *testing.T) {
	mockGen := new(MockGenerator)

	NewGeneratorFunc = func(_ usecase.Usecases, _ logger.Interface) interface {
		GenerateSpec() ([]byte, error)
		SaveSpec([]byte, string) error
	} {
		return mockGen
	}

	expectedSpec := []byte("{}")
	mockGen.On("GenerateSpec").Return(expectedSpec, nil)
	mockGen.On("SaveSpec", expectedSpec, "doc/openapi.json").Return(nil)

	handleOpenAPIGeneration(logger.New("info"))

	mockGen.AssertExpectations(t)
}

//nolint:paralleltest // modifies package-level NewGeneratorFunc
func TestHandleOpenAPIGeneration_GenerateFails(t *testing.T) {
	mockGen := new(MockGenerator)

	NewGeneratorFunc = func(_ usecase.Usecases, _ logger.Interface) interface {
		GenerateSpec() ([]byte, error)
		SaveSpec([]byte, string) error
	} {
		return mockGen
	}

	mockGen.On("GenerateSpec").Return([]byte(nil), assert.AnError)

	handleOpenAPIGeneration(logger.New("info"))

	mockGen.AssertExpectations(t)
}
