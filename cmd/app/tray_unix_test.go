//go:build tray && !windows

package main

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestLogDirReturnsPlatformPath(t *testing.T) { //nolint:paralleltest // mutates env
	dir := logDir()
	if dir == "" {
		t.Fatal("logDir returned empty string")
	}

	if !filepath.IsAbs(dir) {
		t.Errorf("logDir = %q, want absolute path", dir)
	}

	switch runtime.GOOS {
	case "darwin":
		if !strings.Contains(dir, filepath.Join("Library", "Logs", "device-management-toolkit")) {
			t.Errorf("darwin logDir = %q, want path under Library/Logs/device-management-toolkit", dir)
		}
	default:
		// Linux and other unix should land under a state directory.
		if !strings.Contains(dir, "device-management-toolkit") {
			t.Errorf("unix logDir = %q, want path containing device-management-toolkit", dir)
		}
	}
}

func TestLogDirHonorsXDGStateHome(t *testing.T) { //nolint:paralleltest // mutates env
	if runtime.GOOS == "darwin" {
		t.Skip("XDG_STATE_HOME does not apply on darwin")
	}

	t.Setenv("XDG_STATE_HOME", "/tmp/test-state")

	dir := logDir()

	want := filepath.Join("/tmp/test-state", "device-management-toolkit", "logs")
	if dir != want {
		t.Errorf("logDir = %q, want %q", dir, want)
	}
}
