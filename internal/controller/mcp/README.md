# Console MCP API Reference

Console exposes a subset of its device-management API as a
[Model Context Protocol (MCP)](https://modelcontextprotocol.io) server so AI
agents and MCP-aware tooling can drive Intel® AMT devices through the same
use-case layer the REST API uses.

This guide serves two purposes:

1. **Call the MCP API directly** — every tool below has a ready-to-run `curl`
   command against the `/api/mcp` endpoint, so you can drive AMT devices from a
   shell or any HTTP client with no test harness or wrapper script.
2. **Integrate Console into an agentic AI** — point any MCP-aware AI client
   (VS Code, Claude, Cursor, custom agents, and so on) at the same endpoint and
   the model can call these tools autonomously. See
   [Integrate with an agentic AI client](#integrate-with-an-agentic-ai-client).

For architecture and design notes see [../../../docs/mcp.md](../../../docs/mcp.md).

- Server name: `console-mcp`
- Server version: `v0.1.0`
- Transport: **Streamable HTTP**
- Endpoint: `POST /api/mcp`

## Endpoint & authentication

The MCP endpoint is mounted on the protected `/api` router group, so it is
covered by the same JWT auth middleware as the rest of the API (unless
`auth.disabled` is set). Obtain a token from `POST /api/v1/authorize` and send
it as a bearer token on every MCP request:

```
Authorization: Bearer <jwt>
```

Two environment notes for local testing:

- Console serves **HTTPS** with a self-signed certificate — use `https://` and
  `curl -k` (not `http://`).
- A corporate proxy may intercept `localhost` — pass `curl --noproxy '*'`.

## Calling a tool (handshake)

The Streamable HTTP transport requires a short handshake before any tool call.
Responses arrive as Server-Sent Events, so the JSON payload is on a `data:` line.

```sh
BASE="https://localhost:8181"
USER="standalone"
PASS='TestPass123!'

# 1. Get a JWT.
TOKEN=$(curl -sk --noproxy '*' -X POST "$BASE/api/v1/authorize" \
  -H 'Content-Type: application/json' \
  -d "{\"username\":\"$USER\",\"password\":\"$PASS\"}" \
  | sed -E 's/.*"token":"([^"]+)".*/\1/')

# 2. initialize — the session id comes back in the Mcp-Session-Id response header.
SID=$(curl -sk --noproxy '*' -D - -o /dev/null -X POST "$BASE/api/mcp" \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -H 'Accept: application/json, text/event-stream' \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{
        "protocolVersion":"2025-06-18","capabilities":{},
        "clientInfo":{"name":"curl","version":"1.0"}}}' \
  | tr -d '\r' | sed -n 's/^Mcp-Session-Id: //Ip')

# 3. Tell the server initialization is complete (required before tools/call).
curl -sk --noproxy '*' -o /dev/null -X POST "$BASE/api/mcp" \
  -H "Authorization: Bearer $TOKEN" -H "Mcp-Session-Id: $SID" \
  -H 'Content-Type: application/json' -H 'Accept: application/json, text/event-stream' \
  -d '{"jsonrpc":"2.0","method":"notifications/initialized"}'
```

Every per-tool example below is a full, standalone `curl` command. They reuse
the `$BASE`, `$TOKEN`, and `$SID` shell variables set during the handshake
above; replace `<device-guid>` with a real device GUID.

List every registered tool at any time:

```sh
curl -sk --noproxy '*' -X POST "$BASE/api/mcp" \
  -H "Authorization: Bearer $TOKEN" -H "Mcp-Session-Id: $SID" \
  -H 'Content-Type: application/json' -H 'Accept: application/json, text/event-stream' \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/list"}' | sed -n 's/^data: //p'
```

## Complete standalone example

The per-tool sections below reuse the `$BASE`, `$TOKEN`, and `$SID` shell
variables set during the handshake above. If you just want one self-contained
command to copy, paste, and run, the script below does everything — authorize,
handshake, and call `power_get_state` — with no prior setup. Replace the two
placeholders and run it in `bash`:

```sh
#!/usr/bin/env bash
set -euo pipefail

BASE="https://localhost:8181"
USER="standalone"
PASS='TestPass123!'
GUID="f20ee10b-5518-45c0-a7d8-81ca28cd68db"   # from GET /api/v1/devices

# 1. Authorize -> JWT.
TOKEN=$(curl -sk --noproxy '*' -X POST "$BASE/api/v1/authorize" \
  -H 'Content-Type: application/json' \
  -d "{\"username\":\"$USER\",\"password\":\"$PASS\"}" \
  | sed -E 's/.*"token":"([^"]+)".*/\1/')

# 2. initialize -> session id (returned in the Mcp-Session-Id response header).
SID=$(curl -sk --noproxy '*' -D - -o /dev/null -X POST "$BASE/api/mcp" \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -H 'Accept: application/json, text/event-stream' \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{
        "protocolVersion":"2025-06-18","capabilities":{},
        "clientInfo":{"name":"curl","version":"1.0"}}}' \
  | tr -d '\r' | sed -n 's/^Mcp-Session-Id: //Ip')

# 3. notifications/initialized (required before any tools/call).
curl -sk --noproxy '*' -o /dev/null -X POST "$BASE/api/mcp" \
  -H "Authorization: Bearer $TOKEN" -H "Mcp-Session-Id: $SID" \
  -H 'Content-Type: application/json' -H 'Accept: application/json, text/event-stream' \
  -d '{"jsonrpc":"2.0","method":"notifications/initialized"}'

# 4. tools/call -> power_get_state for the device.
curl -sk --noproxy '*' -X POST "$BASE/api/mcp" \
  -H "Authorization: Bearer $TOKEN" -H "Mcp-Session-Id: $SID" \
  -H 'Content-Type: application/json' -H 'Accept: application/json, text/event-stream' \
  -d "{\"jsonrpc\":\"2.0\",\"id\":10,\"method\":\"tools/call\",
       \"params\":{\"name\":\"power_get_state\",\"arguments\":{\"guid\":\"$GUID\"}}}" \
  | sed -n 's/^data: //p'
```

Expected output (the tool result on the SSE `data:` line):

```json
{"jsonrpc":"2.0","id":10,"result":{"content":[{"type":"text","text":"{\"powerState\":2,\"osPowerSavingState\":0}"}],"structuredContent":{"powerState":2,"osPowerSavingState":0}}}
```

To call any other tool, change the `name` and `arguments` in step 4 — for
example a destructive reset:

```sh
  -d "{\"jsonrpc\":\"2.0\",\"id\":11,\"method\":\"tools/call\",
       \"params\":{\"name\":\"power_action\",\"arguments\":{\"guid\":\"$GUID\",\"action\":\"reset\"}}}" \
```

## Integrate with an agentic AI client

Any MCP client that speaks the **Streamable HTTP** transport can consume these
tools directly — the model then discovers them via `tools/list` and calls them
autonomously, performing the `initialize` handshake and session handling for
you. You only need to provide the endpoint URL and the bearer token.

**Endpoint:** `https://<console-host>:8181/api/mcp`
**Auth header:** `Authorization: Bearer <jwt>` (from `POST /api/v1/authorize`).
For a local single-user deployment you can set `auth.disabled: true` in the
Console config to drop the token requirement entirely.

### VS Code (GitHub Copilot / agent mode)

Add an HTTP MCP server in `.vscode/mcp.json`. The `inputs` block prompts once
for the token and injects it as the auth header:

```json
{
  "inputs": [
    { "id": "consoleToken", "type": "promptString", "description": "Console JWT", "password": true }
  ],
  "servers": {
    "console": {
      "type": "http",
      "url": "https://localhost:8181/api/mcp",
      "headers": { "Authorization": "Bearer ${input:consoleToken}" }
    }
  }
}
```

### Generic MCP client (Claude Desktop, Cursor, custom agents)

Clients that support remote HTTP servers use the same three fields — transport
type, URL, and an `Authorization` header:

```json
{
  "mcpServers": {
    "console": {
      "type": "streamable-http",
      "url": "https://localhost:8181/api/mcp",
      "headers": { "Authorization": "Bearer <jwt>" }
    }
  }
}
```

For a client that only speaks the stdio transport, bridge to the HTTP endpoint
with [`mcp-remote`](https://github.com/geelen/mcp-remote):

```json
{
  "mcpServers": {
    "console": {
      "command": "npx",
      "args": [
        "-y", "mcp-remote",
        "https://localhost:8181/api/mcp",
        "--header", "Authorization: Bearer <jwt>"
      ]
    }
  }
}
```

### Notes for agent use

- Console's self-signed TLS certificate must be trusted by the client, or the
  client must be configured to skip verification (equivalent to `curl -k`).
- `power_action` is annotated with the MCP **destructive** hint, so compliant
  clients should ask the user to confirm before running it. Instruct the agent
  to call `power_get_capabilities` first to confirm the action is supported.
- Every tool takes a `guid`; have the agent resolve device names to GUIDs via
  the REST `GET /api/v1/devices` endpoint before calling a tool.

## Tool summary

| Tool | Kind | Description |
|---|---|---|
| [`power_get_state`](#power_get_state) | read-only | Current power state and OS power-saving state. |
| [`power_get_capabilities`](#power_get_capabilities) | read-only | The `power_action` names this device supports. |
| [`power_get_boot_sources`](#power_get_boot_sources) | read-only | Configured boot sources / boot-order entries. |
| [`power_action`](#power_action) | **destructive** | Perform a power action on physical hardware. |
| [`kvm_get_screen_settings`](#kvm_get_screen_settings) | read-only | Per-display resolution, geometry, role, default screen. |
| [`kvm_set_default_display`](#kvm_set_default_display) | idempotent | Set which display is the default KVM screen. |
| [`kvm_get_status`](#kvm_get_status) | read-only | Whether KVM is enabled and supported. |
| [`kvm_set_enabled`](#kvm_set_enabled) | idempotent | Enable/disable KVM redirection. |

All tools take a `guid` argument: the device GUID (UUID) as registered in
Console. Get one from `GET /api/v1/devices`. An invalid UUID is rejected before
any device call with a tool-level error.

---

## Power tools

### `power_get_state`

Get the current power state of an AMT device, including its OS power-saving
state. **Read-only.**

**Input**

| Field | Type | Description |
|---|---|---|
| `guid` | string (UUID) | Device GUID as registered in Console. |

**Output**

| Field | Type | Description |
|---|---|---|
| `powerState` | int | CIM power state value. |
| `osPowerSavingState` | int | OS power-saving state (0 = unknown). |

**Call**

```sh
curl -sk --noproxy '*' -X POST "$BASE/api/mcp" \
  -H "Authorization: Bearer $TOKEN" -H "Mcp-Session-Id: $SID" \
  -H 'Content-Type: application/json' -H 'Accept: application/json, text/event-stream' \
  -d '{"jsonrpc":"2.0","id":10,"method":"tools/call","params":{"name":"power_get_state","arguments":{"guid":"<device-guid>"}}}' \
  | sed -n 's/^data: //p'
```

**Example result**

```json
{"jsonrpc":"2.0","id":10,"result":{
  "content":[{"type":"text","text":"{\"powerState\":2,\"osPowerSavingState\":0}"}],
  "structuredContent":{"powerState":2,"osPowerSavingState":0}}}
```

---

### `power_get_capabilities`

List the power actions this AMT device supports. **Call this before
`power_action`** to discover valid action names for the device. **Read-only.**

**Input**

| Field | Type | Description |
|---|---|---|
| `guid` | string (UUID) | Device GUID as registered in Console. |

**Output**

| Field | Type | Description |
|---|---|---|
| `supportedActions` | string[] | `power_action` names this device supports. |

**Call**

```sh
curl -sk --noproxy '*' -X POST "$BASE/api/mcp" \
  -H "Authorization: Bearer $TOKEN" -H "Mcp-Session-Id: $SID" \
  -H 'Content-Type: application/json' -H 'Accept: application/json, text/event-stream' \
  -d '{"jsonrpc":"2.0","id":11,"method":"tools/call","params":{"name":"power_get_capabilities","arguments":{"guid":"<device-guid>"}}}' \
  | sed -n 's/^data: //p'
```

**Example result**

```json
{"jsonrpc":"2.0","id":11,"result":{
  "structuredContent":{"supportedActions":["power_cycle","power_off","power_on","reset"]}}}
```

---

### `power_get_boot_sources`

List the configured boot sources (boot-order entries) for an AMT device.
**Read-only.**

**Input**

| Field | Type | Description |
|---|---|---|
| `guid` | string (UUID) | Device GUID as registered in Console. |

**Output** — `sources` is an array of boot-source objects:

| Field | Type | Description |
|---|---|---|
| `biosBootString` | string | BIOS boot string. |
| `bootString` | string | Boot string. |
| `elementName` | string | Element name, e.g. `Intel® AMT: Boot Source`. |
| `failThroughSupported` | int | Fail-through support code. |
| `instanceID` | string | Instance ID, e.g. `Intel® AMT: Force Hard-drive Boot`. |
| `structuredBiosBootString` | string | Structured boot string, e.g. `CIM:Hard-Disk:1`. |

**Call**

```sh
curl -sk --noproxy '*' -X POST "$BASE/api/mcp" \
  -H "Authorization: Bearer $TOKEN" -H "Mcp-Session-Id: $SID" \
  -H 'Content-Type: application/json' -H 'Accept: application/json, text/event-stream' \
  -d '{"jsonrpc":"2.0","id":12,"method":"tools/call","params":{"name":"power_get_boot_sources","arguments":{"guid":"<device-guid>"}}}' \
  | sed -n 's/^data: //p'
```

---

### `power_action`

Perform a power action on physical AMT hardware. **Destructive** — this reboots
or powers off a real machine and can interrupt a running OS. It carries the MCP
`destructiveHint`, so compliant clients prompt for confirmation. Call
`power_get_capabilities` first to confirm the action is supported.

**Input**

| Field | Type | Description |
|---|---|---|
| `guid` | string (UUID) | Device GUID as registered in Console. |
| `action` | string | One of the action names below. |

**Valid `action` values**

| Name | Meaning |
|---|---|
| `power_on` | Power on. |
| `power_off` | Hard power off. |
| `power_off_soft` | Soft power off. |
| `power_cycle` | Power cycle (off then on). |
| `reset` | Master bus reset (reboot). |
| `soft_off` | Soft off (graceful). |
| `soft_reset` | Master bus reset (graceful). |
| `sleep` | Sleep. |
| `hibernate` | Hibernate. |
| `os_to_full_power` | Transition OS to full power. |
| `os_to_power_saving` | Transition OS to power saving. |

> Boot-target actions (PXE, BIOS, diagnostics, IDE-R, HTTPS boot) are **not**
> part of `power_action`; they remain on the REST `power/bootOptions` endpoint.

**Output**

| Field | Type | Description |
|---|---|---|
| `returnValue` | int | WSMAN return code; `0` indicates success. |

**Call**

```sh
curl -sk --noproxy '*' -X POST "$BASE/api/mcp" \
  -H "Authorization: Bearer $TOKEN" -H "Mcp-Session-Id: $SID" \
  -H 'Content-Type: application/json' -H 'Accept: application/json, text/event-stream' \
  -d '{"jsonrpc":"2.0","id":13,"method":"tools/call","params":{"name":"power_action","arguments":{"guid":"<device-guid>","action":"reset"}}}' \
  | sed -n 's/^data: //p'
```

**Example result**

```json
{"jsonrpc":"2.0","id":13,"result":{"structuredContent":{"returnValue":0}}}
```

An unknown action is rejected with a tool-level error listing the valid names.

---

## KVM tools

### `kvm_get_screen_settings`

Get the KVM screen/display configuration for an AMT device, including each
display's resolution, geometry, role, and which is the default KVM screen.
**Read-only.**

**Input**

| Field | Type | Description |
|---|---|---|
| `guid` | string (UUID) | Device GUID as registered in Console. |

**Output** — `displays` is an array of display objects:

| Field | Type | Description |
|---|---|---|
| `displayIndex` | int | 0-based display index. |
| `isActive` | bool | Whether the display is active. |
| `resolutionX` | int | Horizontal resolution. |
| `resolutionY` | int | Vertical resolution. |
| `upperLeftX` | int | Upper-left X offset. |
| `upperLeftY` | int | Upper-left Y offset. |
| `role` | string | `primary`, `secondary`, `tertiary`, or `quaternary`. |
| `isDefault` | bool | Whether this is the default KVM screen. |

**Call**

```sh
curl -sk --noproxy '*' -X POST "$BASE/api/mcp" \
  -H "Authorization: Bearer $TOKEN" -H "Mcp-Session-Id: $SID" \
  -H 'Content-Type: application/json' -H 'Accept: application/json, text/event-stream' \
  -d '{"jsonrpc":"2.0","id":20,"method":"tools/call","params":{"name":"kvm_get_screen_settings","arguments":{"guid":"<device-guid>"}}}' \
  | sed -n 's/^data: //p'
```

---

### `kvm_set_default_display`

Set which display an AMT device presents as the default KVM screen.
**Idempotent** (not destructive).

**Input**

| Field | Type | Description |
|---|---|---|
| `guid` | string (UUID) | Device GUID as registered in Console. |
| `displayIndex` | int | 0-based display index to set as the default KVM screen. |

**Output** — same `displays` array as `kvm_get_screen_settings`, reflecting the
updated state.

**Call**

```sh
curl -sk --noproxy '*' -X POST "$BASE/api/mcp" \
  -H "Authorization: Bearer $TOKEN" -H "Mcp-Session-Id: $SID" \
  -H 'Content-Type: application/json' -H 'Accept: application/json, text/event-stream' \
  -d '{"jsonrpc":"2.0","id":21,"method":"tools/call","params":{"name":"kvm_set_default_display","arguments":{"guid":"<device-guid>","displayIndex":0}}}' \
  | sed -n 's/^data: //p'
```

---

### `kvm_get_status`

Report whether KVM redirection is enabled and whether the device firmware
supports it. **Read-only.**

**Input**

| Field | Type | Description |
|---|---|---|
| `guid` | string (UUID) | Device GUID as registered in Console. |

**Output**

| Field | Type | Description |
|---|---|---|
| `enabled` | bool | Whether KVM redirection is currently enabled. |
| `available` | bool | Whether the device firmware supports KVM. |

**Call**

```sh
curl -sk --noproxy '*' -X POST "$BASE/api/mcp" \
  -H "Authorization: Bearer $TOKEN" -H "Mcp-Session-Id: $SID" \
  -H 'Content-Type: application/json' -H 'Accept: application/json, text/event-stream' \
  -d '{"jsonrpc":"2.0","id":22,"method":"tools/call","params":{"name":"kvm_get_status","arguments":{"guid":"<device-guid>"}}}' \
  | sed -n 's/^data: //p'
```

---

### `kvm_set_enabled`

Enable or disable KVM redirection on an AMT device. Other redirection features
(SOL, IDE-R) and the user-consent setting are left unchanged. **Idempotent**
(not destructive).

**Input**

| Field | Type | Description |
|---|---|---|
| `guid` | string (UUID) | Device GUID as registered in Console. |
| `enabled` | bool | `true` to enable KVM redirection, `false` to disable it. |

**Output**

| Field | Type | Description |
|---|---|---|
| `enabled` | bool | Resulting KVM enabled state. |
| `available` | bool | Whether the device firmware supports KVM. |

**Call**

```sh
curl -sk --noproxy '*' -X POST "$BASE/api/mcp" \
  -H "Authorization: Bearer $TOKEN" -H "Mcp-Session-Id: $SID" \
  -H 'Content-Type: application/json' -H 'Accept: application/json, text/event-stream' \
  -d '{"jsonrpc":"2.0","id":23,"method":"tools/call","params":{"name":"kvm_set_enabled","arguments":{"guid":"<device-guid>","enabled":true}}}' \
  | sed -n 's/^data: //p'
```

---

## Error handling

Tools return errors as **tool-level** results (not JSON-RPC protocol errors), so
the payload has `"isError":true` and a text message rather than a top-level
`error` object:

```json
{"jsonrpc":"2.0","id":10,"result":{
  "content":[{"type":"text","text":"guid must be a valid UUID: \"not-a-uuid\""}],
  "isError":true}}
```

Common cases:

- **Invalid GUID** — `guid must be a valid UUID: "<value>"`.
- **Unknown power action** — `unknown power action "<value>"; valid actions: ...`.
- **Device/credential/WSMAN failures** — surfaced from `devices.Feature` (for
  example `cipher: message authentication failed` when the stored device
  credentials were encrypted with a different `APP_ENCRYPTION_KEY`).

## See also

- [../../../docs/mcp.md](../../../docs/mcp.md) — architecture and design.
- [../../../integration-test/mcp/README.md](../../../integration-test/mcp/README.md) — manual end-to-end test script.
- [Model Context Protocol](https://modelcontextprotocol.io)
