//go:build !mcp

package app

import (
	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/pkg/logger"
)

// startEmbeddedMCP is a no-op unless Console is built with the "mcp" build tag.
func startEmbeddedMCP(_ *config.Config, _ logger.Interface) {}
