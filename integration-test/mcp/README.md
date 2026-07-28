# MCP manual tests

`test-mcp-power.sh` drives the Console MCP server (`/api/mcp`) end-to-end with
curl over the Streamable HTTP transport, against a running Console and a real
AMT device.

This is a **manual** test, not part of `go test ./...`. The unit tests in
`internal/controller/mcp/server_test.go` cover the same tools against a mocked
`devices.Feature` and need no device; this script verifies the parts they can't:
real HTTP transport, JWT middleware, session handling, and live WSMAN calls.

## Start Console

```sh
CGO_ENABLED=0 go build -o ./bin/console ./cmd/app

GIN_MODE=debug APP_DISABLE_CIRA=false HTTP_PORT=8181 \
  AUTH_ADMIN_USERNAME=standalone AUTH_ADMIN_PASSWORD='TestPass123!' \
  AUTH_JWT_KEY=test_jwt_key_for_mcp_testing \
  APP_ENCRYPTION_KEY=0123456789abcdef0123456789abcdef \
  ./bin/console
```

Set `APP_ENCRYPTION_KEY` explicitly. Without it, `handleEncryptionKey` in
`cmd/app/main.go` falls through to the OS keyring and may prompt
interactively, which hangs a non-interactive run.

Keep `APP_DISABLE_CIRA=false` when testing against a real device — the device
reaches Console over its CIRA connection on `:4433`.

## Run the tests

```sh
export CONSOLE_PASS='TestPass123!'

# protocol, auth, and input validation only (no device needed)
./integration-test/mcp/test-mcp-power.sh

# add the live read-only tools
./integration-test/mcp/test-mcp-power.sh <device-guid>

# add one destructive power action (prompts for confirmation)
./integration-test/mcp/test-mcp-power.sh <device-guid> --action reset
```

Get a device GUID from `GET /api/v1/devices`.

| Variable | Default | Purpose |
|---|---|---|
| `CONSOLE_URL` | `https://localhost:8181` | Base URL |
| `CONSOLE_USER` | `standalone` | Admin username |
| `CONSOLE_PASS` | *(required)* | Admin password (`auth.adminPassword`) |
| `DEVICE_GUID` | — | Device GUID, instead of the positional argument |
| `CURL_TIMEOUT` | `30` | Per-request timeout in seconds |

`power_action` is destructive and never runs unless you pass `--action`; even
then it asks you to type the action name to confirm. Everything else is
read-only. Run `power_get_capabilities` first — it reports which actions the
device actually supports.

## Two environment gotchas

**Console serves HTTPS with a self-signed certificate.** Use `https://`, not
`http://`. Over `http://` the server replies `400 Bad Request: Client sent an
HTTP request to an HTTPS server`. The script uses `curl -k` throughout.

**A corporate proxy will intercept localhost.** With `HTTP_PROXY` /
`HTTPS_PROXY` set and `NO_PROXY` not covering localhost, requests return a proxy
`403` HTML page that reads like a Console auth failure but never reaches
Console. The script passes `--noproxy '*'` on every request; do the same for
ad-hoc curl calls.

## Transport notes

Two details matter when hand-writing MCP curl calls:

1. `initialize` returns the session id in the **`Mcp-Session-Id` response
   header** (not the body). Echo it on every later request.
2. Send the `notifications/initialized` notification after `initialize`, before
   any `tools/call`.

Responses arrive as Server-Sent Events, so the JSON payload is on a `data: `
line — hence the `sed -n 's/^data: //p'` in the script.

## Expected output

```
== Preflight ==
  PASS  GET /healthz -> 200
  PASS  POST /api/v1/authorize -> JWT (105 chars)

== MCP handshake ==
  PASS  initialize -> session JSNYHWTGNPJMSP3MTQHZMYENM4
  PASS  initialize -> serverInfo
  PASS  notifications/initialized sent

== Auth enforcement ==
  PASS  no token -> 401
  PASS  bad token -> 401

== Tool registration ==
  PASS  tools/list contains power_get_state
  ...
  PASS  power_action annotated destructiveHint:true

== Input validation ==
  PASS  power_action rejects unknown action
  PASS  error message lists valid actions
  ...
```
