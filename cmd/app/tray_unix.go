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
	"strconv"
	"strings"
	"syscall"
)

// Kept open for process lifetime so the kernel retains the flock.
var instanceLockFD = -1

const (
	lockDirPerm     os.FileMode = 0o755
	lockFilePerm    os.FileMode = 0o644
	lockReadBufSize             = 1024
	lockPayloadKeys             = 2 // "<pid>\n<url>\n"
)

// ensureSingleInstance acquires a per-user flock; the FD is inherited across
// the background re-exec via DMT_LOCK_FD so the lock survives parent exit.
func ensureSingleInstance(url string) {
	if fdStr := os.Getenv("DMT_LOCK_FD"); fdStr != "" {
		if fd, err := strconv.Atoi(fdStr); err == nil {
			instanceLockFD = fd
			writeLockPayload(fd, url)
		}

		return
	}

	path := lockFilePath()
	if err := os.MkdirAll(filepath.Dir(path), lockDirPerm); err != nil {
		log.Printf("Failed to create lock directory: %v", err)

		return
	}

	fd, err := syscall.Open(path, syscall.O_CREAT|syscall.O_RDWR, uint32(lockFilePerm))
	if err != nil {
		log.Printf("Failed to open lock file %s: %v", path, err)

		return
	}

	if err := syscall.Flock(fd, syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		runningURL := readLockURL(fd)
		if runningURL == "" {
			runningURL = url
		}

		_ = syscall.Close(fd)

		log.Printf("DMT Console is already running; signaling user at %s", runningURL)
		surfaceRunningInstance(runningURL)
		os.Exit(0)
	}

	instanceLockFD = fd

	writeLockPayload(fd, url)
}

// writeLockPayload records "<pid>\n<url>\n" so a duplicate invocation can
// open the running instance's URL instead of the new invocation's config.
func writeLockPayload(fd int, url string) {
	_ = syscall.Ftruncate(fd, 0)
	_, _ = syscall.Pwrite(fd, []byte(strconv.Itoa(os.Getpid())+"\n"+url+"\n"), 0)
}

// readLockURL reads the URL line from a lock file. Returns "" when missing/malformed.
func readLockURL(fd int) string {
	buf := make([]byte, lockReadBufSize)

	n, err := syscall.Pread(fd, buf, 0)
	if err != nil || n <= 0 {
		return ""
	}

	parts := strings.SplitN(strings.TrimRight(string(buf[:n]), "\n"), "\n", lockPayloadKeys)
	if len(parts) < lockPayloadKeys {
		return ""
	}

	return parts[1]
}

func surfaceRunningInstance(url string) {
	opener := "xdg-open"
	if runtime.GOOS == "darwin" {
		opener = "open"
	}

	if err := exec.CommandContext(context.Background(), opener, url).Start(); err != nil {
		log.Printf("Failed to open browser: %v", err)
	}
}

func lockFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "dmt-console.lock")
	}

	if runtime.GOOS == "darwin" {
		return filepath.Join(home, "Library", "Caches", "device-management-toolkit", "dmt-console.lock")
	}

	if dir := os.Getenv("XDG_RUNTIME_DIR"); dir != "" {
		return filepath.Join(dir, "device-management-toolkit", "dmt-console.lock")
	}

	return filepath.Join(home, ".cache", "device-management-toolkit", "dmt-console.lock")
}

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

	// Hand the flock FD to the child (becomes FD 3) so the lock survives parent exit.
	if instanceLockFD >= 0 {
		lockFile := os.NewFile(uintptr(instanceLockFD), "dmt-console.lock")
		cmd.ExtraFiles = []*os.File{lockFile}
		cmd.Env = append(cmd.Env, "DMT_LOCK_FD=3")
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
