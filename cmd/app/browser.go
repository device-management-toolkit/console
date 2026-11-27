//go:build !noui

package main

import (
	"context"
	"log"
	"net"
	"os/exec"
	"runtime"

	"github.com/device-management-toolkit/console/config"
)

func launchBrowser(cfg *config.Config) {
	scheme := "http"
	if cfg.TLS.Enabled {
		scheme = "https"
	}

	host := navigableHost(cfg.Host)

	browserURL := scheme + "://" + net.JoinHostPort(host, cfg.Port)

	if setupURL := setupPageURL(cfg); setupURL != "" {
		browserURL = setupURL
	}

	log.Printf("launchBrowser: opening %s", browserURL)

	if err := openBrowser(browserURL, runtime.GOOS); err != nil {
		log.Printf("Skipping browser launch: %v", err)
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

// windowsURLHandler is the rundll32 entrypoint that asks Windows to open the
// URL with the registered default handler, avoiding cmd.exe parsing.
const (
	windowsURLHandler = "url.dll,FileProtocolHandler"
)

// Global command executor, can be replaced in tests.
var cmdExecutor CommandExecutor = &RealCommandExecutor{}

func openBrowser(targetURL, currentOS string) error {
	var cmd string

	var args []string

	switch currentOS {
	case "darwin":
		cmd = "open"
		args = []string{targetURL}
	case "windows":
		cmd = "rundll32"
		args = []string{windowsURLHandler, targetURL}
	default:
		cmd = "xdg-open"
		args = []string{targetURL}
	}

	return cmdExecutor.Execute(cmd, args...)
}
