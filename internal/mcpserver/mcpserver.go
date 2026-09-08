package mcpserver

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/mark3labs/mcp-go/server"

	"github.com/device-management-toolkit/console/pkg/logger"
)

const (
	serverName    = "console-mcp"
	serverVersion = "0.1.0"

	shutdownTimeout = 5 * time.Second
)

// Run starts the MCP SSE server and blocks until ctx is cancelled or the server
// stops with an error. It is used both by the standalone cmd/mcp binary and by
// the embedded server when Console is built with the "mcp" build tag.
func Run(ctx context.Context, cfg Config, log logger.Interface) error {
	client := NewConsoleClient(cfg)

	mcpServer := server.NewMCPServer(
		serverName,
		serverVersion,
		server.WithToolCapabilities(true),
		server.WithRecovery(),
	)

	RegisterTools(mcpServer, client, cfg)

	sseServer := server.NewSSEServer(mcpServer, server.WithBaseURL(cfg.BaseURL))

	log.Info("Console MCP (SSE) listening on " + cfg.Addr)
	log.Info("  SSE endpoint:     " + cfg.BaseURL + "/sse")
	log.Info("  Message endpoint: " + cfg.BaseURL + "/message")
	log.Info("  Console backend:  " + cfg.ConsoleBaseURL)

	errCh := make(chan error, 1)

	go func() { errCh <- sseServer.Start(cfg.Addr) }()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		return sseServer.Shutdown(shutdownCtx)
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}

		return nil
	}
}
