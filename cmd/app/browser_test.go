//go:build !noui

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockCommandExecutor struct {
	mock.Mock
}

func (m *MockCommandExecutor) Execute(name string, arg ...string) error {
	args := m.Called(name, arg)

	return args.Error(0)
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
