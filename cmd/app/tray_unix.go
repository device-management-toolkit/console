//go:build tray && !windows

package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

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

	// Close our copy now that the child has inherited its own FD.
	_ = f.Close()

	fmt.Printf("DMT Console started in background (PID %d)\n", cmd.Process.Pid)
	fmt.Printf("Logs: %s\n", logPath)

	os.Exit(0)
}
