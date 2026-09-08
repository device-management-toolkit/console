//go:build mcp

package app

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/internal/mcpserver"
	"github.com/device-management-toolkit/console/pkg/logger"
)

// startEmbeddedMCP launches the MCP SSE server in-process. Compiled in only when
// Console is built with the "mcp" build tag. It auto-wires to this Console
// instance unless the corresponding CONSOLE_* env vars are set explicitly, and
// can be turned off at runtime with MCP_ENABLED=false.
func startEmbeddedMCP(cfg *config.Config, log logger.Interface) {
	if v, err := strconv.ParseBool(os.Getenv("MCP_ENABLED")); err == nil && !v {
		log.Info("embedded MCP server disabled via MCP_ENABLED=false")

		return
	}

	mcpCfg := mcpserver.LoadConfig()

	if os.Getenv("CONSOLE_BASE_URL") == "" {
		scheme := "http"
		if cfg.TLS.Enabled {
			scheme = "https"
		}

		mcpCfg.ConsoleBaseURL = fmt.Sprintf("%s://127.0.0.1:%s", scheme, cfg.Port)
	}

	// Reuse Console's admin credentials unless overridden; skipped when auth is disabled.
	if !cfg.Disabled {
		if os.Getenv("CONSOLE_USERNAME") == "" {
			mcpCfg.ConsoleUsername = cfg.AdminUsername
		}

		if os.Getenv("CONSOLE_PASSWORD") == "" {
			mcpCfg.ConsolePassword = cfg.AdminPassword
		}
	}

	go func() {
		log.Info("starting embedded MCP SSE server on " + mcpCfg.Addr + " (backend " + mcpCfg.ConsoleBaseURL + ")")

		if err := mcpserver.Run(context.Background(), mcpCfg, log); err != nil {
			log.Error(fmt.Errorf("embedded MCP server: %w", err))
		}
	}()
}
