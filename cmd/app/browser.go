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

	host := cfg.Host

	// Strip brackets from bracketed IPv6 literals (e.g. [::1] → ::1) before
	// the wildcard check so both [::] and [::1] are handled uniformly.
	if len(host) >= 2 && host[0] == '[' && host[len(host)-1] == ']' {
		host = host[1 : len(host)-1]
	}

	// Wildcard bind addresses are not directly navigable; normalise them to
	// localhost. Preserve loopback addresses such as ::1 so net.JoinHostPort
	// can format them correctly for browser navigation.
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "localhost"
	}

	url := scheme + "://" + net.JoinHostPort(host, cfg.Port)
	log.Printf("launchBrowser: opening %s", url)

	if err := openBrowser(url, runtime.GOOS); err != nil {
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

// windowsCmdFlag is the /c flag passed to cmd.exe to run a command and exit.
// windowsCmdStart is the Windows shell verb that opens a URL in the default browser.
const (
	windowsCmdFlag  = "/c"
	windowsCmdStart = "start"
)

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
		args = []string{windowsCmdFlag, windowsCmdStart, url}
	default:
		cmd = "xdg-open"
		args = []string{url}
	}

	return cmdExecutor.Execute(cmd, args...)
}
