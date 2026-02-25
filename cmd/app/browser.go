//go:build !noui

package main

import (
	"context"
	"os/exec"
	"runtime"

	"github.com/device-management-toolkit/console/config"
)

func launchBrowser(cfg *config.Config) {
	scheme := "http"
	if cfg.TLS.Enabled {
		scheme = "https"
	}

	if err := openBrowser(scheme+"://localhost:"+cfg.Port, runtime.GOOS); err != nil {
		panic(err)
	}
}

// CommandExecutor is an interface to allow for mocking exec.Command in tests.
type CommandExecutor interface {
	Execute(name string, arg ...string) error
}

// RealCommandExecutor is a real implementation of CommandExecutor.
type RealCommandExecutor struct{}

func (e *RealCommandExecutor) Execute(name string, arg ...string) error {
	return exec.CommandContext(context.Background(), name, arg...).Start()
}

// Global command executor, can be replaced in tests.
var cmdExecutor CommandExecutor = &RealCommandExecutor{}

func openBrowser(url, currentOS string) error {
	var cmd string

	var args []string

	switch currentOS {
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", url}
	default:
		cmd = "xdg-open"
		args = []string{url}
	}

	return cmdExecutor.Execute(cmd, args...)
}
