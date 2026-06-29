//go:build tray && windows

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"
)

// detachedProcess is the Windows CreationFlag that runs the child without
// inheriting the parent console. Defined here to avoid pulling in
// golang.org/x/sys/windows just for the constant.
// inheriting the parent console.
const detachedProcess = 0x00000008

// CreateMutexW returns ERROR_ALREADY_EXISTS when another instance holds the named mutex.
const mutexName = "Local\\DMTConsoleTray"
const errorAlreadyExists uint32 = 183

// ensureSingleInstance prevents concurrent tray processes via a named mutex.
//
// The mutex handle does not survive process exit and cannot be passed to the
// re-execed child on Windows, so:
//   - In the terminal parent (which is about to fork and exit), we probe with
//     OpenMutexW; if a duplicate is found, surface it and exit, otherwise let
//     the re-execed child take the persistent hold.
//   - In the re-execed child or non-terminal launch (GUI/service), we hold
//     the mutex with CreateMutexW for process lifetime.
func ensureSingleInstance(url string) {
	namePtr, err := windows.UTF16PtrFromString(mutexName)
	if err != nil {
		return
	}

	kernel32 := windows.NewLazySystemDLL("kernel32.dll")

	if shouldHoldInstanceMutex() {
		acquireInstanceMutex(kernel32, namePtr, url)
		return
	}

	probeInstanceMutex(kernel32, namePtr, url)
}

// shouldHoldInstanceMutex reports whether this process is the persistent tray
// process (re-execed child or non-terminal launch). The terminal parent that
// is about to call relaunchInBackground returns false.
func shouldHoldInstanceMutex() bool {
	return os.Getenv("DMT_BACKGROUND") != "" || !isTerminal()
}

func acquireInstanceMutex(kernel32 *windows.LazyDLL, namePtr *uint16, url string) {
	createMutex := kernel32.NewProc("CreateMutexW")

	handle, _, err := createMutex.Call(
		0,
		0,
		uintptr(unsafe.Pointer(namePtr)),
	)

	if handle == 0 {
		return
	}

	if errno, ok := err.(windows.Errno); ok && uint32(errno) == errorAlreadyExists {
		log.Printf("DMT Console is already running; signalling user at %s", url)
		surfaceRunningInstance(url)
		os.Exit(0)
	}
}

func probeInstanceMutex(kernel32 *windows.LazyDLL, namePtr *uint16, url string) {
	openMutex := kernel32.NewProc("OpenMutexW")

	const synchronize = 0x00100000 // SYNCHRONIZE

	handle, _, _ := openMutex.Call(
		synchronize,
		0,
		uintptr(unsafe.Pointer(namePtr)),
	)

	if handle == 0 {
		return
	}

	closeHandle := kernel32.NewProc("CloseHandle")
	_, _, _ = closeHandle.Call(handle)

	log.Printf("DMT Console is already running; signalling user at %s", url)
	surfaceRunningInstance(url)
	os.Exit(0)
}

func surfaceRunningInstance(url string) {
	if isHeadlessBuild {
		user32 := windows.NewLazySystemDLL("user32.dll")
		messageBox := user32.NewProc("MessageBoxW")

		text, _ := windows.UTF16PtrFromString(
			"DMT Console is already running in the system tray.\nAPI: " + url,
		)
		caption, _ := windows.UTF16PtrFromString("DMT Console")

		const mbIconInformation = 0x40

		messageBox.Call(
			0,
			uintptr(unsafe.Pointer(text)),
			uintptr(unsafe.Pointer(caption)),
			uintptr(mbIconInformation),
		)

		return
	}
  
// rundll32 avoids cmd.exe's metacharacter parsing on URLs with querystrings.
	if err := exec.CommandContext(
		context.Background(),
		"rundll32",
		"url.dll,FileProtocolHandler",
		url,
	).Start(); err != nil {
		log.Printf("Failed to open browser: %v", err)
	}
}

// logDir returns the Windows-conventional log directory for the app.
func logDir() string {
	if dir := os.Getenv("LOCALAPPDATA"); dir != "" {
		return filepath.Join(dir, "device-management-toolkit", "logs")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return os.TempDir()
	}

	return filepath.Join(home, "AppData", "Local", "device-management-toolkit", "logs")
}

// relaunchInBackground re-execs the current process detached from the console,
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

	cmd.SysProcAttr = &windows.SysProcAttr{
		HideWindow:    true,
		CreationFlags: detachedProcess,
	}

	if err := cmd.Start(); err != nil {
		log.Fatalf("Failed to start in background: %v", err)
	}

	_ = f.Close()

	fmt.Printf("DMT Console started in background (PID %d)\n", cmd.Process.Pid)
	fmt.Printf("Logs: %s\n", logPath)

	os.Exit(0)
}
