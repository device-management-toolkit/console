package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/internal/usecase"
	"github.com/device-management-toolkit/console/pkg/logger"
)

type MockCommandExecutor struct {
	mock.Mock
}

func (m *MockCommandExecutor) Execute(name string, arg ...string) error {
	args := m.Called(name, arg)

	return args.Error(0)
}

func TestMainFunction(_ *testing.T) { //nolint:paralleltest // cannot have simultaneous tests modifying env variables.
	os.Setenv("GIN_MODE", "debug")

	// Mock functions
	initializeConfigFunc = func() (*config.Config, error) {
		return &config.Config{HTTP: config.HTTP{Port: "8080"}, App: config.App{EncryptionKey: "test"}}, nil
	}

	initializeAppFunc = func(_ *config.Config) error {
		return nil
	}

	runAppFunc = func(_ *config.Config) {}

	// Call the main function
	main()
}

func TestOpenBrowserWindows(t *testing.T) { //nolint:paralleltest // cannot have simultaneous tests modifying executor.
	mockCmdExecutor := new(MockCommandExecutor)
	cmdExecutor = mockCmdExecutor

	mockCmdExecutor.On("Execute", "cmd", []string{"/c", "start", "http://localhost:8080"}).Return(nil)

	err := openBrowser("http://localhost:8080", "windows")
	assert.NoError(t, err)
	mockCmdExecutor.AssertExpectations(t)
}

func TestOpenBrowserDarwin(t *testing.T) { //nolint:paralleltest // cannot have simultaneous tests modifying executor.
	mockCmdExecutor := new(MockCommandExecutor)
	cmdExecutor = mockCmdExecutor

	mockCmdExecutor.On("Execute", "open", []string{"http://localhost:8080"}).Return(nil)

	err := openBrowser("http://localhost:8080", "darwin")
	assert.NoError(t, err)
	mockCmdExecutor.AssertExpectations(t)
}

func TestOpenBrowserLinux(t *testing.T) { //nolint:paralleltest // cannot have simultaneous tests modifying executor.
	mockCmdExecutor := new(MockCommandExecutor)
	cmdExecutor = mockCmdExecutor

	mockCmdExecutor.On("Execute", "xdg-open", []string{"http://localhost:8080"}).Return(nil)

	err := openBrowser("http://localhost:8080", "ubuntu")
	assert.NoError(t, err)
	mockCmdExecutor.AssertExpectations(t)
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

	err := handleOpenAPIGeneration()
	assert.NoError(t, err)

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

	err := handleOpenAPIGeneration()
	assert.Error(t, err)

	mockGen.AssertExpectations(t)
}
