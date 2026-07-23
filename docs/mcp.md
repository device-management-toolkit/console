# MCP Server

Console exposes a subset of its device-management API as a
[Model Context Protocol (MCP)](https://modelcontextprotocol.io) server, so AI
agents and MCP-aware tooling can drive AMT devices through the same use-case
layer the REST API uses.

## Architecture

The MCP server is a **delivery adapter** in `internal/controller/mcp/`, peer to
the REST controller (`internal/controller/httpapi`) and the WebSocket controller
(`internal/controller/ws`). Every tool handler calls straight into the
`devices.Feature` use case:

```
mcp tool handler → devices.Feature → (repo and/or WSMAN)
```

This means MCP tools reuse all existing business logic, validation, WSMAN client
lifecycle, and credential encryption. The adapter never touches repositories or
builds WSMAN clients directly.

It is built on the official [`modelcontextprotocol/go-sdk`](https://github.com/modelcontextprotocol/go-sdk)
and served over the **Streamable HTTP** transport, mounted on the same Gin
engine as the REST API.

## Endpoint & authentication

```
/api/mcp
```

The endpoint is mounted on the protected `/api` router group, so it is covered
by the same JWT auth middleware as the rest of the API (unless `auth.disabled`
is set for local single-user deployments). Obtain a token from
`POST /api/v1/authorize` and send it as a bearer token:

```
Authorization: Bearer <jwt>
```

The Streamable HTTP transport uses `POST` for JSON-RPC requests, `GET` for the
server-to-client event stream, and `DELETE` to end a session.

## Live KVM/SOL/IDER streaming is out of scope

The MCP server covers KVM **configuration and management**, not the live pixel
stream. Real-time KVM/SOL/IDER video is a raw byte stream over
`/relay/webrelay.ashx` (WebSocket → APF channel), which is not a request/response
tool call and stays on its existing WebSocket path.

## Tools

### Power

| Tool | Kind | Description |
|---|---|---|
| `power_get_state` | read-only | Current power state and OS power-saving state. |
| `power_get_capabilities` | read-only | The `power_action` names this device supports. Call before `power_action`. |
| `power_get_boot_sources` | read-only | Configured boot sources / boot-order entries. |
| `power_action` | **destructive** | Perform a power action on physical hardware. |

`power_action` takes a **named** action rather than a raw integer code. Supported
names map to CIM `RequestPowerStateChange` values and OS power-saving transitions:

| Name | Meaning |
|---|---|
| `power_on` | Power on |
| `power_off` | Hard power off |
| `power_off_soft` | Soft power off |
| `power_cycle` | Power cycle (off then on) |
| `reset` | Master bus reset (reboot) |
| `soft_off` | Soft off (graceful) |
| `soft_reset` | Master bus reset (graceful) |
| `sleep` | Sleep |
| `hibernate` | Hibernate |
| `os_to_full_power` | Transition OS to full power |
| `os_to_power_saving` | Transition OS to power saving |

Boot-target actions (PXE, BIOS, diagnostics, IDE-R, HTTPS boot) require the
boot-configuration flow and are not part of `power_action`; they remain on the
REST `power/bootOptions` endpoint.

> `power_action` reboots or powers off real hardware and can interrupt a running
> OS. It is annotated with the MCP destructive hint. An agent should call
> `power_get_capabilities` first to confirm the action is supported.

### KVM

| Tool | Kind | Description |
|---|---|---|
| `kvm_get_screen_settings` | read-only | Per-display resolution, geometry, role, and default screen. |
| `kvm_set_default_display` | idempotent | Set which display is the default KVM screen. |
| `kvm_get_status` | read-only | Whether KVM is enabled and whether the device supports it. |
| `kvm_set_enabled` | idempotent | Enable/disable KVM redirection; other features (SOL, IDE-R, consent) are preserved. |

## Adding new tools

1. Add a handler in `internal/controller/mcp/` that calls `devices.Feature`
   (or the appropriate feature). Validate the GUID with `validateGUID`.
2. Register it in `registerPowerTools` / `registerKVMTools` (or a new
   `register…Tools` grouped by feature) with appropriate `ToolAnnotations`.
3. Return errors via `toolError`/`toolErrorf` so the client sees a tool-call
   error rather than a protocol error.
4. Add tests in `server_test.go` using the in-memory transport helper.
