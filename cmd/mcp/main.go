// Command mcp runs the Console MCP server as a standalone process.
//
// Configuration comes from environment variables (see internal/mcpserver.Config
// / the mcp README). To instead embed the MCP server inside the Console binary,
// build Console with the "mcp" build tag.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/device-management-toolkit/console/internal/mcpserver"
	"github.com/device-management-toolkit/console/pkg/logger"
)

func main() {
	cfg := mcpserver.LoadConfig()
	l := logger.New("info")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := mcpserver.Run(ctx, cfg, l); err != nil {
		log.Fatalf("MCP server error: %v", err)
	}
}
