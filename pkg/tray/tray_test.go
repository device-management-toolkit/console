//go:build tray

package tray

import (
	"sync/atomic"
	"testing"
)

func TestNew(t *testing.T) {
	t.Parallel()

	cfg := Config{AppName: "Test", URL: "http://localhost:9999"}

	m := New(cfg)
	if m == nil {
		t.Fatal("New returned nil")
	}

	if m.config.AppName != "Test" {
		t.Errorf("AppName = %q, want %q", m.config.AppName, "Test")
	}

	if m.config.URL != "http://localhost:9999" {
		t.Errorf("URL = %q, want %q", m.config.URL, "http://localhost:9999")
	}
}

func TestOnExitInvokesOnQuit(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32

	m := New(Config{
		OnQuit: func() { calls.Add(1) },
	})

	m.onExit()

	if got := calls.Load(); got != 1 {
		t.Errorf("OnQuit called %d times, want 1", got)
	}
}

func TestOnExitOnlyRunsOnQuitOnce(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32

	m := New(Config{
		OnQuit: func() { calls.Add(1) },
	})

	m.onExit()
	m.onExit()
	m.onExit()

	if got := calls.Load(); got != 1 {
		t.Errorf("OnQuit called %d times, want 1 (sync.Once)", got)
	}
}

func TestOnExitNilOnQuitIsSafe(t *testing.T) {
	t.Parallel()

	m := New(Config{}) // OnQuit unset

	// Should not panic.
	m.onExit()
	m.onExit()
}

func TestGetIconReturnsData(t *testing.T) {
	t.Parallel()

	icon := getIcon()
	if len(icon) == 0 {
		t.Fatal("getIcon returned empty bytes")
	}
}
