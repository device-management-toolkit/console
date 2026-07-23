// Package mcp implements a Model Context Protocol (MCP) delivery adapter for
// the Console device-management API. It is a peer to the REST controller in
// internal/controller/httpapi and the WebSocket controller in
// internal/controller/ws: it exposes device operations as MCP tools that call
// straight into the devices.Feature use case, reusing all existing business
// logic, validation, WSMAN client lifecycle, and credential encryption.
//
// The server is mounted over the official MCP Streamable HTTP transport on the
// same Gin engine as the REST API, so it inherits the JWT auth middleware and
// the multi-instance CIRA socket-affinity model for free.
package mcp

import (
	"net/http"

	"github.com/gin-gonic/gin"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/device-management-toolkit/console/internal/usecase/devices"
	"github.com/device-management-toolkit/console/pkg/logger"
)

const (
	serverName    = "console-mcp"
	serverVersion = "v0.1.0"
)

// Controller holds the dependencies shared by all MCP tool handlers.
type Controller struct {
	devices devices.Feature
	log     logger.Interface
}

// ptr is a small helper for the *bool annotation fields in the SDK.
func ptr[T any](v T) *T { return &v }

// NewServer builds an MCP server with every Console tool registered against the
// supplied device use case.
func NewServer(d devices.Feature, l logger.Interface) *sdk.Server {
	c := &Controller{devices: d, log: l}

	server := sdk.NewServer(&sdk.Implementation{
		Name:    serverName,
		Title:   "Intel Device Management Toolkit Console",
		Version: serverVersion,
	}, nil)

	c.registerPowerTools(server)
	c.registerKVMTools(server)

	return server
}

// Register mounts the MCP Streamable HTTP handler on the given Gin router group
// at the "/mcp" path. The group is expected to already carry the auth
// middleware the rest of the protected API uses.
func Register(group *gin.RouterGroup, d devices.Feature, l logger.Interface) {
	server := NewServer(d, l)

	handler := sdk.NewStreamableHTTPHandler(func(*http.Request) *sdk.Server {
		return server
	}, nil)

	// The Streamable HTTP transport uses POST for JSON-RPC requests, GET for the
	// server-to-client event stream, and DELETE to terminate a session.
	group.Any("/mcp", gin.WrapH(handler))
	group.Any("/mcp/*rest", gin.WrapH(handler))
}
