// Package mcpserver exposes the Device Management Toolkit Console REST API as
// MCP (Model Context Protocol) tools over an SSE (Server-Sent Events) transport.
//
// It can run as a standalone process (see cmd/mcp) or be embedded into the
// Console binary when built with the "mcp" build tag.
package mcpserver

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds runtime configuration for the MCP server and its connection to
// the Console backend.
type Config struct {
	// Console backend.
	ConsoleBaseURL     string
	ConsoleUsername    string
	ConsolePassword    string
	InsecureSkipVerify bool
	RequestTimeout     time.Duration

	// MCP SSE server.
	Addr    string // host:port the SSE server binds to.
	BaseURL string // public base URL advertised to SSE clients.
}

// LoadConfig builds a Config from environment variables, applying defaults that
// match a local Console instance (https://localhost:8181).
func LoadConfig() Config {
	cfg := Config{
		ConsoleBaseURL:     getenv("CONSOLE_BASE_URL", "https://localhost:8181"),
		ConsoleUsername:    os.Getenv("CONSOLE_USERNAME"),
		ConsolePassword:    os.Getenv("CONSOLE_PASSWORD"),
		InsecureSkipVerify: getbool("CONSOLE_INSECURE_SKIP_VERIFY", true),
		RequestTimeout:     time.Duration(getint("CONSOLE_REQUEST_TIMEOUT_SECONDS", 30)) * time.Second,
		Addr:               getenv("MCP_ADDR", ":8080"),
		BaseURL:            getenv("MCP_BASE_URL", "http://localhost:8080"),
	}

	cfg.ConsoleBaseURL = strings.TrimRight(cfg.ConsoleBaseURL, "/")
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")

	return cfg
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}

	return fallback
}

func getbool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}

	return parsed
}

func getint(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}

	return parsed
}
