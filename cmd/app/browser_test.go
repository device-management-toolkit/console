//go:build !noui

package main

import (
	"runtime"
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

// expectedOpenBrowserArgs returns the cmd and args openBrowser will use for the
// current OS, mirroring the switch statement in openBrowser.
func expectedOpenBrowserArgs(url string) (cmd string, args []string) {
	switch runtime.GOOS {
	case "darwin":
		return "open", []string{url}
	case "windows":
		return "cmd", []string{windowsCmdFlag, windowsCmdStart, url}
	default:
		return "xdg-open", []string{url}
	}
}

func TestOpenBrowserWindows(t *testing.T) { //nolint:paralleltest // cannot have simultaneous tests modifying executor.
	mockCmdExecutor := new(MockCommandExecutor)
	original := cmdExecutor

	t.Cleanup(func() { cmdExecutor = original })

	cmdExecutor = mockCmdExecutor

	mockCmdExecutor.On("Execute", "cmd", []string{windowsCmdFlag, windowsCmdStart, "http://localhost:8080"}).Return(nil)

	err := openBrowser("http://localhost:8080", "windows")
	assert.NoError(t, err)
	mockCmdExecutor.AssertExpectations(t)
}

func TestOpenBrowserDarwin(t *testing.T) { //nolint:paralleltest // cannot have simultaneous tests modifying executor.
	mockCmdExecutor := new(MockCommandExecutor)
	original := cmdExecutor

	t.Cleanup(func() { cmdExecutor = original })

	cmdExecutor = mockCmdExecutor

	mockCmdExecutor.On("Execute", "open", []string{"http://localhost:8080"}).Return(nil)

	err := openBrowser("http://localhost:8080", "darwin")
	assert.NoError(t, err)
	mockCmdExecutor.AssertExpectations(t)
}

func TestOpenBrowserLinux(t *testing.T) { //nolint:paralleltest // cannot have simultaneous tests modifying executor.
	mockCmdExecutor := new(MockCommandExecutor)
	original := cmdExecutor

	t.Cleanup(func() { cmdExecutor = original })

	cmdExecutor = mockCmdExecutor

	mockCmdExecutor.On("Execute", "xdg-open", []string{"http://localhost:8080"}).Return(nil)

	err := openBrowser("http://localhost:8080", "ubuntu")
	assert.NoError(t, err)
	mockCmdExecutor.AssertExpectations(t)
}

func TestLaunchBrowserEmptyHostDefaultsToLocalhost(t *testing.T) { //nolint:paralleltest // cannot have simultaneous tests modifying executor.
	mockCmdExecutor := new(MockCommandExecutor)
	original := cmdExecutor

	t.Cleanup(func() { cmdExecutor = original })

	cmdExecutor = mockCmdExecutor

	cfg := &config.Config{
		HTTP: config.HTTP{
			Host: "",
			Port: "8181",
		},
	}

	cmd, args := expectedOpenBrowserArgs("http://localhost:8181")
	mockCmdExecutor.On("Execute", cmd, args).Return(nil)

	launchBrowser(cfg)

	mockCmdExecutor.AssertExpectations(t)
}

func TestLaunchBrowserExplicitHostUsed(t *testing.T) { //nolint:paralleltest // cannot have simultaneous tests modifying executor.
	mockCmdExecutor := new(MockCommandExecutor)
	original := cmdExecutor

	t.Cleanup(func() { cmdExecutor = original })

	cmdExecutor = mockCmdExecutor

	cfg := &config.Config{
		HTTP: config.HTTP{
			Host: "192.168.1.100",
			Port: "8181",
		},
	}

	cmd, args := expectedOpenBrowserArgs("http://192.168.1.100:8181")
	mockCmdExecutor.On("Execute", cmd, args).Return(nil)

	launchBrowser(cfg)

	mockCmdExecutor.AssertExpectations(t)
}

func TestLaunchBrowserTLSUsesHTTPS(t *testing.T) { //nolint:paralleltest // cannot have simultaneous tests modifying executor.
	mockCmdExecutor := new(MockCommandExecutor)
	original := cmdExecutor

	t.Cleanup(func() { cmdExecutor = original })

	cmdExecutor = mockCmdExecutor

	cfg := &config.Config{
		HTTP: config.HTTP{
			Host: "myserver",
			Port: "8443",
			TLS:  config.TLS{Enabled: true},
		},
	}

	cmd, args := expectedOpenBrowserArgs("https://myserver:8443")
	mockCmdExecutor.On("Execute", cmd, args).Return(nil)

	launchBrowser(cfg)

	mockCmdExecutor.AssertExpectations(t)
}

func TestLaunchBrowserTLSAndEmptyHostDefaultsToLocalhost(t *testing.T) { //nolint:paralleltest // cannot have simultaneous tests modifying executor.
	mockCmdExecutor := new(MockCommandExecutor)
	original := cmdExecutor

	t.Cleanup(func() { cmdExecutor = original })

	cmdExecutor = mockCmdExecutor

	cfg := &config.Config{
		HTTP: config.HTTP{
			Host: "",
			Port: "8443",
			TLS:  config.TLS{Enabled: true},
		},
	}

	cmd, args := expectedOpenBrowserArgs("https://localhost:8443")
	mockCmdExecutor.On("Execute", cmd, args).Return(nil)

	launchBrowser(cfg)

	mockCmdExecutor.AssertExpectations(t)
}

func TestLaunchBrowserWildcard0000DefaultsToLocalhost(t *testing.T) { //nolint:paralleltest // cannot have simultaneous tests modifying executor.
	mockCmdExecutor := new(MockCommandExecutor)
	original := cmdExecutor

	t.Cleanup(func() { cmdExecutor = original })

	cmdExecutor = mockCmdExecutor

	cfg := &config.Config{
		HTTP: config.HTTP{Host: "0.0.0.0", Port: "8181"},
	}

	cmd, args := expectedOpenBrowserArgs("http://localhost:8181")
	mockCmdExecutor.On("Execute", cmd, args).Return(nil)

	launchBrowser(cfg)

	mockCmdExecutor.AssertExpectations(t)
}

func TestLaunchBrowserWildcardIPv6DefaultsToLocalhost(t *testing.T) { //nolint:paralleltest // cannot have simultaneous tests modifying executor.
	mockCmdExecutor := new(MockCommandExecutor)
	original := cmdExecutor

	t.Cleanup(func() { cmdExecutor = original })

	cmdExecutor = mockCmdExecutor

	cfg := &config.Config{
		HTTP: config.HTTP{Host: "::", Port: "8181"},
	}

	cmd, args := expectedOpenBrowserArgs("http://localhost:8181")
	mockCmdExecutor.On("Execute", cmd, args).Return(nil)

	launchBrowser(cfg)

	mockCmdExecutor.AssertExpectations(t)
}

func TestLaunchBrowserWildcardBracketedIPv6DefaultsToLocalhost(t *testing.T) { //nolint:paralleltest // cannot have simultaneous tests modifying executor.
	mockCmdExecutor := new(MockCommandExecutor)
	original := cmdExecutor

	t.Cleanup(func() { cmdExecutor = original })

	cmdExecutor = mockCmdExecutor

	cfg := &config.Config{
		HTTP: config.HTTP{Host: "[::]", Port: "8181"},
	}

	cmd, args := expectedOpenBrowserArgs("http://localhost:8181")
	mockCmdExecutor.On("Execute", cmd, args).Return(nil)

	launchBrowser(cfg)

	mockCmdExecutor.AssertExpectations(t)
}

func TestLaunchBrowserIPv6LoopbackWrappedInBrackets(t *testing.T) { //nolint:paralleltest // cannot have simultaneous tests modifying executor.
	mockCmdExecutor := new(MockCommandExecutor)
	original := cmdExecutor

	t.Cleanup(func() { cmdExecutor = original })

	cmdExecutor = mockCmdExecutor

	cfg := &config.Config{
		HTTP: config.HTTP{Host: "::1", Port: "8181"},
	}

	cmd, args := expectedOpenBrowserArgs("http://[::1]:8181")
	mockCmdExecutor.On("Execute", cmd, args).Return(nil)

	launchBrowser(cfg)

	mockCmdExecutor.AssertExpectations(t)
}

func TestLaunchBrowserBracketedIPv6LoopbackStrippedThenRewrapped(t *testing.T) { //nolint:paralleltest // cannot have simultaneous tests modifying executor.
	mockCmdExecutor := new(MockCommandExecutor)
	original := cmdExecutor

	t.Cleanup(func() { cmdExecutor = original })

	cmdExecutor = mockCmdExecutor

	cfg := &config.Config{
		HTTP: config.HTTP{Host: "[::1]", Port: "8181"},
	}

	cmd, args := expectedOpenBrowserArgs("http://[::1]:8181")
	mockCmdExecutor.On("Execute", cmd, args).Return(nil)

	launchBrowser(cfg)

	mockCmdExecutor.AssertExpectations(t)
}
