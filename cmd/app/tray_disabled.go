//go:build !tray

package main

import (
	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/pkg/logger"
)

var trayBuildEnabled = false

func runWithTray(cfg *config.Config, l logger.Interface) {
	// Tray not available in this build, fall back to standard mode
	handleDebugMode(cfg, l)
	runAppFunc(cfg, l)
}
