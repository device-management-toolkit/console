//go:build tray && !windows

package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/pkg/logger"
	"github.com/device-management-toolkit/console/pkg/tray"
)

func init() {
	// Enable tray mode by default when built with tray tag
	trayBuildEnabled = true
}

var trayBuildEnabled = false

// logDir returns the macOS-conventional log directory for the app.
func logDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return os.TempDir()
	}

	return filepath.Join(home, "Library", "Logs", "device-management-toolkit")
}

// relaunchInBackground re-execs the current process detached from the terminal,
// redirecting output to a log file. It exits the parent process on success.
func relaunchInBackground() {
	dir := logDir()

	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Fatalf("Failed to create log directory: %v", err)
	}

	logPath := filepath.Join(dir, "console.log")

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}

	exePath, err := os.Executable()
	if err != nil {
		log.Fatalf("Failed to get executable path: %v", err)
	}

	cmd := exec.Command(exePath, os.Args[1:]...)
	cmd.Stdout = f
	cmd.Stderr = f
	cmd.Env = append(os.Environ(), "DMT_BACKGROUND=1")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}

	if err := cmd.Start(); err != nil {
		log.Fatalf("Failed to start in background: %v", err)
	}

	fmt.Printf("DMT Console started in background (PID %d)\n", cmd.Process.Pid)
	fmt.Printf("Logs: %s\n", logPath)

	os.Exit(0)
}

// isTerminal returns true if stdin is connected to a terminal.
func isTerminal() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}

	return fi.Mode()&os.ModeCharDevice != 0
}

func runWithTray(cfg *config.Config, l logger.Interface) {
	// When launched from a terminal, re-exec in the background so the
	// user gets their shell back and logs go to a file.
	if os.Getenv("DMT_BACKGROUND") == "" && isTerminal() {
		relaunchInBackground()
	}

	// Build the URL for the web UI
	scheme := "http"
	if cfg.TLS.Enabled {
		scheme = "https"
	}
	url := scheme + "://localhost:" + cfg.Port

	// Create tray manager
	trayManager := tray.New(tray.Config{
		AppName:  "DMT Console",
		URL:      url,
		Headless: isHeadlessBuild,
		OnReady: func() {
			// Start the server in a goroutine
			go runAppFunc(cfg, l)
			log.Printf("DMT Console running at %s", url)
		},
		OnQuit: func() {
			log.Println("Shutting down DMT Console...")
			// Send interrupt signal to trigger graceful shutdown
			p, _ := os.FindProcess(os.Getpid())
			_ = p.Signal(os.Interrupt)
		},
	})

	// Catch Ctrl+C / SIGTERM so the tray unblocks on terminal interrupt.
	// app.Run also listens for these signals to shut down the HTTP server;
	// Go delivers to all registered channels, so both handlers fire.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		trayManager.Quit()
	}()

	// Run the tray (this blocks until quit)
	trayManager.Run()
}
