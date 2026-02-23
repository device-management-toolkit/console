//go:build noui

package main

import "github.com/device-management-toolkit/console/config"

// launchBrowser is a no-op in noui builds; there is no UI to open.
func launchBrowser(_ *config.Config) {}
