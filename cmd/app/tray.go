//go:build tray

package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/pkg/logger"
	"github.com/device-management-toolkit/console/pkg/tray"
)

var trayBuildEnabled = true

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
