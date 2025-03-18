//go:build windows

package app

import (
	"golang.org/x/sys/windows"
)

// hideTerminalWindows hides the terminal on Windows.
func hideTerminalWindows() {
	modKernel32 := windows.NewLazySystemDLL("kernel32.dll")
	procGetConsoleWindow := modKernel32.NewProc("GetConsoleWindow")
	procShowWindow := modKernel32.NewProc("ShowWindow")

	hwnd, _, _ := procGetConsoleWindow.Call()
	if hwnd != 0 {
		procShowWindow.Call(hwnd, 0) // SW_HIDE = 0
	}
}
