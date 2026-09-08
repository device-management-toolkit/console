package main

import (
	"log"

	"github.com/mark3labs/mcp-go/server"
)

func main() {
	cfg := LoadConfig()

	client := NewConsoleClient(cfg)

	mcpServer := server.NewMCPServer(
		"console-mcp",
		"0.1.0",
		server.WithToolCapabilities(true),
		server.WithRecovery(),
	)

	RegisterTools(mcpServer, client, cfg)

	sseServer := server.NewSSEServer(mcpServer, server.WithBaseURL(cfg.BaseURL))

	log.Printf("Console MCP server (SSE) listening on %s", cfg.Addr)
	log.Printf("  SSE endpoint:     %s/sse", cfg.BaseURL)
	log.Printf("  Message endpoint: %s/message", cfg.BaseURL)
	log.Printf("  Console backend:  %s", cfg.ConsoleBaseURL)

	if err := sseServer.Start(cfg.Addr); err != nil {
		log.Fatalf("SSE server error: %v", err)
	}
}
