package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/device-management-toolkit/console/config"
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
