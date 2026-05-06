//go:build tray && !windows

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
)

// logDir returns the conventional log directory for the current platform.
// macOS: ~/Library/Logs/device-management-toolkit
// Linux: $XDG_STATE_HOME/device-management-toolkit/logs (fallback ~/.local/state/...)
func logDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return os.TempDir()
	}

	if runtime.GOOS == "darwin" {
		return filepath.Join(home, "Library", "Logs", "device-management-toolkit")
	}

	// Linux / other unix: follow XDG Base Directory spec for state files.
	if state := os.Getenv("XDG_STATE_HOME"); state != "" {
		return filepath.Join(state, "device-management-toolkit", "logs")
	}

	return filepath.Join(home, ".local", "state", "device-management-toolkit", "logs")
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

	cmd := exec.CommandContext(context.Background(), exePath, os.Args[1:]...)
	cmd.Stdout = f
	cmd.Stderr = f
	cmd.Env = append(os.Environ(), "DMT_BACKGROUND=1")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}

	if err := cmd.Start(); err != nil {
		log.Fatalf("Failed to start in background: %v", err)
	}

	// Close our copy now that the child has inherited its own FD.
	_ = f.Close()

	fmt.Printf("DMT Console started in background (PID %d)\n", cmd.Process.Pid)
	fmt.Printf("Logs: %s\n", logPath)

	os.Exit(0)
}
