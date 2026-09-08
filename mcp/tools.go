package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterTools wires every Console-backed tool onto the MCP server.
func RegisterTools(s *server.MCPServer, client *ConsoleClient, cfg Config) {
	registerDeviceTools(s, client)
	registerPowerTools(s, client)
	registerInfoTools(s, client)
	registerRedirectionTools(s, client, cfg)
}

// jsonResult marshals a value into an indented JSON tool result.
func jsonResult(v any) *mcp.CallToolResult {
	pretty, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to encode response", err)
	}

	return mcp.NewToolResultText(string(pretty))
}

// rawResult wraps an already-JSON payload (re-indented) as a tool result.
func rawResult(raw json.RawMessage) *mcp.CallToolResult {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return mcp.NewToolResultText(string(raw))
	}

	return jsonResult(v)
}

func registerDeviceTools(s *server.MCPServer, client *ConsoleClient) {
	s.AddTool(
		mcp.NewTool("list_devices",
			mcp.WithDescription("List AMT devices managed by Console. Supports optional filtering and paging."),
			mcp.WithString("hostname", mcp.Description("Filter by exact device hostname.")),
			mcp.WithString("friendlyName", mcp.Description("Filter by friendly name.")),
			mcp.WithString("tags", mcp.Description("Comma-separated tags to filter by.")),
			mcp.WithNumber("top", mcp.Description("Maximum number of records to return.")),
			mcp.WithNumber("skip", mcp.Description("Number of records to skip (for paging).")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			q := url.Values{}
			if v := req.GetString("hostname", ""); v != "" {
				q.Set("hostname", v)
			}
			if v := req.GetString("friendlyName", ""); v != "" {
				q.Set("friendlyName", v)
			}
			if v := req.GetString("tags", ""); v != "" {
				q.Set("tags", v)
			}
			if v := req.GetFloat("top", 0); v > 0 {
				q.Set("$top", fmt.Sprintf("%d", int(v)))
			}
			if v := req.GetFloat("skip", 0); v > 0 {
				q.Set("$skip", fmt.Sprintf("%d", int(v)))
			}

			path := "/api/v1/devices"
			if encoded := q.Encode(); encoded != "" {
				path += "?" + encoded
			}

			raw, err := client.get(ctx, path)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("list_devices failed", err), nil
			}

			return rawResult(raw), nil
		},
	)

	s.AddTool(
		mcp.NewTool("get_device",
			mcp.WithDescription("Get a single device by its AMT GUID."),
			mcp.WithString("guid", mcp.Required(), mcp.Description("Device GUID.")),
		),
		guidHandler(client, func(guid string) string {
			return "/api/v1/devices/" + escapePath(guid)
		}, "get_device"),
	)

	s.AddTool(
		mcp.NewTool("get_device_stats",
			mcp.WithDescription("Get device counts (total, connected, disconnected)."),
		),
		func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			raw, err := client.get(ctx, "/api/v1/devices/stats")
			if err != nil {
				return mcp.NewToolResultErrorFromErr("get_device_stats failed", err), nil
			}

			return rawResult(raw), nil
		},
	)
}

func registerPowerTools(s *server.MCPServer, client *ConsoleClient) {
	s.AddTool(
		mcp.NewTool("get_power_state",
			mcp.WithDescription("Get the current AMT power state of a device."),
			mcp.WithString("guid", mcp.Required(), mcp.Description("Device GUID.")),
		),
		guidHandler(client, func(guid string) string {
			return "/api/v1/amt/power/state/" + escapePath(guid)
		}, "get_power_state"),
	)

	s.AddTool(
		mcp.NewTool("get_power_capabilities",
			mcp.WithDescription("Get the supported AMT power actions for a device."),
			mcp.WithString("guid", mcp.Required(), mcp.Description("Device GUID.")),
		),
		guidHandler(client, func(guid string) string {
			return "/api/v1/amt/power/capabilities/" + escapePath(guid)
		}, "get_power_capabilities"),
	)

	s.AddTool(
		mcp.NewTool("send_power_action",
			mcp.WithDescription("Send an AMT power action to a device. Common actions: "+
				"2=Power On, 5=Power Cycle (Off Soft), 6=Power Off (Hard), 8=Power Off (Soft), "+
				"9=Power Cycle (Off Hard), 10=Reset (Master Bus Reset), 11=Diagnostic Interrupt (NMI), "+
				"12=Power Off (Soft Graceful), 13=Power Off (Hard Graceful), 14=Reset Graceful."),
			mcp.WithString("guid", mcp.Required(), mcp.Description("Device GUID.")),
			mcp.WithNumber("action", mcp.Required(), mcp.Description("Numeric AMT power action code.")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			guid, err := req.RequireString("guid")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			action, err := req.RequireFloat("action")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			raw, err := client.post(ctx, "/api/v1/amt/power/action/"+escapePath(guid), map[string]int{
				"action": int(action),
			})
			if err != nil {
				return mcp.NewToolResultErrorFromErr("send_power_action failed", err), nil
			}

			return rawResult(raw), nil
		},
	)
}

func registerInfoTools(s *server.MCPServer, client *ConsoleClient) {
	infoTools := []struct {
		name string
		desc string
		path string
	}{
		{"get_hardware_info", "Get AMT hardware inventory (CPU, memory, media, etc.) for a device.", "/api/v1/amt/hardwareInfo/"},
		{"get_disk_info", "Get AMT disk information for a device.", "/api/v1/amt/diskInfo/"},
		{"get_general_settings", "Get AMT general settings for a device.", "/api/v1/amt/generalSettings/"},
		{"get_amt_version", "Get AMT firmware/version information for a device.", "/api/v1/amt/version/"},
	}

	for _, t := range infoTools {
		base := t.path

		s.AddTool(
			mcp.NewTool(t.name,
				mcp.WithDescription(t.desc),
				mcp.WithString("guid", mcp.Required(), mcp.Description("Device GUID.")),
			),
			guidHandler(client, func(guid string) string {
				return base + escapePath(guid)
			}, t.name),
		)
	}
}

func registerRedirectionTools(s *server.MCPServer, client *ConsoleClient, cfg Config) {
	s.AddTool(
		mcp.NewTool("get_redirect_status",
			mcp.WithDescription("Get the KVM/SOL/IDER redirection connection status of a device."),
			mcp.WithString("guid", mcp.Required(), mcp.Description("Device GUID.")),
		),
		guidHandler(client, func(guid string) string {
			return "/api/v1/devices/redirectstatus/" + escapePath(guid)
		}, "get_redirect_status"),
	)

	s.AddTool(
		mcp.NewTool("get_kvm_screen_settings",
			mcp.WithDescription("Get the KVM display/screen settings for a device."),
			mcp.WithString("guid", mcp.Required(), mcp.Description("Device GUID.")),
		),
		guidHandler(client, func(guid string) string {
			return "/api/v1/amt/kvm/displays/" + escapePath(guid)
		}, "get_kvm_screen_settings"),
	)

	s.AddTool(
		mcp.NewTool("create_redirection_session",
			mcp.WithDescription("Create a short-lived redirection token for a KVM or SOL (Serial-over-LAN) "+
				"session and return the WebSocket relay URL to connect to. The token must be sent as the "+
				"'Sec-Websocket-Protocol' header (subprotocol) when opening the relay WebSocket."),
			mcp.WithString("guid", mcp.Required(), mcp.Description("Device GUID.")),
			mcp.WithString("mode", mcp.Required(), mcp.Description("Redirection mode."), mcp.Enum("kvm", "sol")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			guid, err := req.RequireString("guid")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			mode, err := req.RequireString("mode")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			raw, err := client.get(ctx, "/api/v1/devices/authorize/redirection/"+escapePath(guid))
			if err != nil {
				return mcp.NewToolResultErrorFromErr("create_redirection_session failed", err), nil
			}

			var tokenResp struct {
				Token string `json:"token"`
			}
			if err := json.Unmarshal(raw, &tokenResp); err != nil {
				return mcp.NewToolResultErrorFromErr("decode redirection token", err), nil
			}

			relay := strings.Replace(cfg.ConsoleBaseURL, "https://", "wss://", 1)
			relay = strings.Replace(relay, "http://", "ws://", 1)
			relayURL := fmt.Sprintf("%s/relay/webrelay.ashx?host=%s&mode=%s",
				relay, url.QueryEscape(guid), url.QueryEscape(mode))

			return jsonResult(map[string]string{
				"guid":             guid,
				"mode":             mode,
				"token":            tokenResp.Token,
				"relayWebSocket":   relayURL,
				"connectInstructions": "Open a WebSocket to relayWebSocket and pass the token as the " +
					"Sec-Websocket-Protocol subprotocol header.",
			}), nil
		},
	)

	s.AddTool(
		mcp.NewTool("capture_kvm_frame",
			mcp.WithDescription("Capture a single KVM screen frame from a device as raw pixel data for "+
				"visual comparison/analysis. Returns a full-screen framebuffer in RGB332 (1 byte/pixel), "+
				"row-major top-to-bottom, base64-encoded in 'dataBase64'. Each byte packs colour as bits 5-7 "+
				"red (0-7), bits 2-4 green (0-7), bits 0-1 blue (0-3); expand to 8-bit RGB with "+
				"r=(b>>5)*255/7, g=((b>>2)&7)*255/7, blue=(b&3)*255/3. Works whether Console runs headless "+
				"or with its UI. Decoding/rendering to an image is the caller's responsibility."),
			mcp.WithString("guid", mcp.Required(), mcp.Description("Device GUID.")),
			mcp.WithNumber("width", mcp.Description("Optional target width hint for downstream scaling (px).")),
			mcp.WithNumber("height", mcp.Description("Optional target height hint for downstream scaling (px).")),
			mcp.WithNumber("timeoutSeconds", mcp.Description("Optional capture timeout in seconds.")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			guid, err := req.RequireString("guid")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			q := url.Values{}
			if v := req.GetFloat("width", 0); v > 0 {
				q.Set("width", fmt.Sprintf("%d", int(v)))
			}
			if v := req.GetFloat("height", 0); v > 0 {
				q.Set("height", fmt.Sprintf("%d", int(v)))
			}
			if v := req.GetFloat("timeoutSeconds", 0); v > 0 {
				q.Set("timeoutSeconds", fmt.Sprintf("%d", int(v)))
			}

			path := "/api/v1/amt/kvm/frame/" + escapePath(guid)
			if encoded := q.Encode(); encoded != "" {
				path += "?" + encoded
			}

			raw, err := client.get(ctx, path)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("capture_kvm_frame failed", err), nil
			}

			return rawResult(raw), nil
		},
	)
}

// guidHandler builds a tool handler that reads a required "guid" argument, GETs
// the path produced by pathFn, and returns the JSON response.
func guidHandler(client *ConsoleClient, pathFn func(guid string) string, toolName string) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		guid, err := req.RequireString("guid")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		raw, err := client.get(ctx, pathFn(guid))
		if err != nil {
			return mcp.NewToolResultErrorFromErr(toolName+" failed", err), nil
		}

		return rawResult(raw), nil
	}
}
