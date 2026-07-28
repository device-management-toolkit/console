#!/usr/bin/env bash
#
# Manual end-to-end test for the Console MCP server (power + KVM tools).
#
# Drives /api/mcp over the Streamable HTTP transport with curl: obtains a JWT,
# performs the MCP initialize handshake, then exercises the power and KVM tools
# against a real AMT device.
#
# Read-only by default. power_action is destructive (it reboots or powers off
# physical hardware) and only runs when --action is passed explicitly.
#
# Usage:
#   ./test-mcp-power.sh                              # protocol + validation only
#   ./test-mcp-power.sh <device-guid>                # + live read-only tools
#   ./test-mcp-power.sh <device-guid> --action reset  # + one destructive action
#
# Environment:
#   CONSOLE_URL   base URL           (default https://localhost:8181)
#   CONSOLE_USER  admin username     (default standalone)
#   CONSOLE_PASS  admin password     (required)
#   DEVICE_GUID   device GUID        (alternative to the positional argument)
#   CURL_TIMEOUT  per-request seconds (default 30)
#
# Notes:
#   * Console serves HTTPS with a self-signed certificate, so curl runs with -k.
#   * Every request passes --noproxy '*'; a corporate HTTP_PROXY otherwise
#     intercepts localhost and returns a 403 that looks like an auth failure.

set -uo pipefail

BASE="${CONSOLE_URL:-https://localhost:8181}"
USER_NAME="${CONSOLE_USER:-standalone}"
PASSWORD="${CONSOLE_PASS:-}"
TIMEOUT="${CURL_TIMEOUT:-30}"
GUID="${DEVICE_GUID:-}"
ACTION=""

# --- argument parsing --------------------------------------------------------

while [ $# -gt 0 ]; do
  case "$1" in
    --action)
      ACTION="${2:-}"
      if [ -z "$ACTION" ]; then
        echo "error: --action requires a power action name" >&2
        exit 2
      fi
      shift 2
      ;;
    -h|--help)
      sed -n '3,28p' "$0" | sed 's/^# \{0,1\}//'
      exit 0
      ;;
    -*)
      echo "error: unknown flag $1" >&2
      exit 2
      ;;
    *)
      GUID="$1"
      shift
      ;;
  esac
done

# --- output helpers ----------------------------------------------------------

if [ -t 1 ]; then
  RED=$'\033[31m'; GREEN=$'\033[32m'; YELLOW=$'\033[33m'; BOLD=$'\033[1m'; OFF=$'\033[0m'
else
  RED=""; GREEN=""; YELLOW=""; BOLD=""; OFF=""
fi

PASSED=0
FAILED=0
SKIPPED=0

section() { printf '\n%s== %s ==%s\n' "$BOLD" "$1" "$OFF"; }
pass()    { PASSED=$((PASSED + 1)); printf '  %sPASS%s  %s\n' "$GREEN" "$OFF" "$1"; }
fail()    { FAILED=$((FAILED + 1)); printf '  %sFAIL%s  %s\n' "$RED" "$OFF" "$1"; }
skip()    { SKIPPED=$((SKIPPED + 1)); printf '  %sSKIP%s  %s\n' "$YELLOW" "$OFF" "$1"; }
detail()  { printf '        %s\n' "$1"; }
die()     { printf '\n%serror:%s %s\n' "$RED" "$OFF" "$1" >&2; exit 1; }

# Pretty-print JSON when jq is available, otherwise emit it raw.
show() {
  if command -v jq >/dev/null 2>&1; then
    printf '%s\n' "$1" | jq -c . 2>/dev/null || printf '%s\n' "$1"
  else
    printf '%s\n' "$1"
  fi
}

# --- transport ---------------------------------------------------------------

# rpc <json-body> -- send a JSON-RPC request, print the SSE data payload.
rpc() {
  curl -sk --noproxy '*' --max-time "$TIMEOUT" -X POST "$BASE/api/mcp" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Mcp-Session-Id: $SID" \
    -H 'Content-Type: application/json' \
    -H 'Accept: application/json, text/event-stream' \
    -d "$1" | sed -n 's/^data: //p'
}

# call_tool <id> <tool-name> <arguments-json>
call_tool() {
  rpc "{\"jsonrpc\":\"2.0\",\"id\":$1,\"method\":\"tools/call\",\"params\":{\"name\":\"$2\",\"arguments\":$3}}"
}

is_error()  { printf '%s' "$1" | grep -q '"isError":true'; }
has_result() { printf '%s' "$1" | grep -q '"result"'; }

# assert_tool_error <label> <response>  -- expects a tool-level error result.
assert_tool_error() {
  if is_error "$2"; then
    pass "$1"
  else
    fail "$1 (expected isError:true)"
    detail "$(show "$2")"
  fi
}

# assert_tool_ok <label> <response>  -- expects a successful tool result.
assert_tool_ok() {
  if [ -z "$2" ]; then
    fail "$1 (empty response)"
  elif is_error "$2"; then
    fail "$1"
    detail "$(show "$2")"
  elif has_result "$2"; then
    pass "$1"
    detail "$(show "$2")"
  else
    fail "$1 (unexpected response)"
    detail "$(show "$2")"
  fi
}

# --- preflight ---------------------------------------------------------------

section "Preflight"

[ -n "$PASSWORD" ] || die "CONSOLE_PASS is not set. Export the admin password first:
  export CONSOLE_PASS='<your auth.adminPassword>'"

code=$(curl -sk --noproxy '*' --max-time "$TIMEOUT" -o /dev/null -w '%{http_code}' "$BASE/healthz")
if [ "$code" = "200" ]; then
  pass "GET /healthz -> 200"
else
  fail "GET /healthz -> $code"
  die "Console is not reachable at $BASE.
  * Is the server running?
  * Console serves HTTPS -- use https:// (not http://) in CONSOLE_URL."
fi

TOKEN=$(curl -sk --noproxy '*' --max-time "$TIMEOUT" -X POST "$BASE/api/v1/authorize" \
  -H 'Content-Type: application/json' \
  -d "{\"username\":\"$USER_NAME\",\"password\":\"$PASSWORD\"}" \
  | sed -E 's/.*"token":"([^"]+)".*/\1/')

if [ -n "$TOKEN" ] && [ "${#TOKEN}" -gt 20 ]; then
  pass "POST /api/v1/authorize -> JWT (${#TOKEN} chars)"
else
  fail "POST /api/v1/authorize"
  die "Could not obtain a JWT. Check CONSOLE_USER / CONSOLE_PASS."
fi

# --- MCP handshake -----------------------------------------------------------

section "MCP handshake"

SID=""
init_headers=$(curl -sk --noproxy '*' --max-time "$TIMEOUT" -D - -o /tmp/mcp_init_body.$$ \
  -X POST "$BASE/api/mcp" \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -H 'Accept: application/json, text/event-stream' \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{
        "protocolVersion":"2025-06-18","capabilities":{},
        "clientInfo":{"name":"test-mcp-power.sh","version":"1.0"}}}')

SID=$(printf '%s' "$init_headers" | tr -d '\r' | sed -n 's/^Mcp-Session-Id: //Ip')
init_body=$(sed -n 's/^data: //p' /tmp/mcp_init_body.$$ 2>/dev/null)
rm -f /tmp/mcp_init_body.$$

if [ -n "$SID" ]; then
  pass "initialize -> session $SID"
else
  fail "initialize (no Mcp-Session-Id header)"
  detail "$(printf '%s' "$init_headers" | head -5)"
  die "MCP handshake failed."
fi

if printf '%s' "$init_body" | grep -q '"serverInfo"'; then
  pass "initialize -> serverInfo"
  detail "$(show "$init_body")"
else
  fail "initialize -> serverInfo missing"
fi

# The spec requires this notification before regular requests.
curl -sk --noproxy '*' --max-time "$TIMEOUT" -o /dev/null -X POST "$BASE/api/mcp" \
  -H "Authorization: Bearer $TOKEN" -H "Mcp-Session-Id: $SID" \
  -H 'Content-Type: application/json' -H 'Accept: application/json, text/event-stream' \
  -d '{"jsonrpc":"2.0","method":"notifications/initialized"}'
pass "notifications/initialized sent"

# --- auth enforcement --------------------------------------------------------

section "Auth enforcement"

for label in "no token" "bad token"; do
  if [ "$label" = "no token" ]; then
    hdr=(-H 'Content-Type: application/json')
  else
    hdr=(-H 'Authorization: Bearer garbage.token.here' -H 'Content-Type: application/json')
  fi

  code=$(curl -sk --noproxy '*' --max-time "$TIMEOUT" -o /dev/null -w '%{http_code}' \
    -X POST "$BASE/api/mcp" "${hdr[@]}" \
    -H 'Accept: application/json, text/event-stream' \
    -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{
          "protocolVersion":"2025-06-18","capabilities":{},
          "clientInfo":{"name":"c","version":"1"}}}')

  if [ "$code" = "401" ]; then
    pass "$label -> 401"
  else
    fail "$label -> $code (expected 401)"
  fi
done

# --- tool registration -------------------------------------------------------

section "Tool registration"

tools=$(rpc '{"jsonrpc":"2.0","id":2,"method":"tools/list"}')

for want in power_get_state power_get_capabilities power_get_boot_sources power_action \
            kvm_get_screen_settings kvm_set_default_display kvm_get_status kvm_set_enabled; do
  if printf '%s' "$tools" | grep -q "\"name\":\"$want\""; then
    pass "tools/list contains $want"
  else
    fail "tools/list missing $want"
  fi
done

# power_action must carry the destructive hint so MCP clients prompt the user.
if printf '%s' "$tools" | grep -q '"destructiveHint":true'; then
  pass "power_action annotated destructiveHint:true"
else
  fail "power_action missing destructiveHint:true"
fi

if printf '%s' "$tools" | grep -q '"readOnlyHint":true'; then
  pass "read-only tools annotated readOnlyHint:true"
else
  fail "no readOnlyHint:true annotations found"
fi

# --- input validation (no device required) -----------------------------------

section "Input validation"

VALID_GUID="038d0240-0f01-4a4b-9c8c-6c2f5b1234ab"

res=$(call_tool 10 power_action "{\"guid\":\"$VALID_GUID\",\"action\":\"explode\"}")
assert_tool_error "power_action rejects unknown action" "$res"
if printf '%s' "$res" | grep -q 'valid actions'; then
  pass "error message lists valid actions"
  # The message embeds escaped quotes, so cut on the closing "}] instead of ".
  detail "$(printf '%s' "$res" | sed -E 's/.*"text":"(.*)"\}\].*/\1/' | sed 's/\\"/"/g')"
else
  fail "error message does not list valid actions"
fi

res=$(call_tool 11 power_action '{"guid":"not-a-uuid","action":"reset"}')
assert_tool_error "power_action rejects malformed GUID" "$res"

res=$(call_tool 12 power_get_state '{"guid":"not-a-uuid"}')
assert_tool_error "power_get_state rejects malformed GUID" "$res"

res=$(call_tool 13 kvm_get_status '{"guid":"not-a-uuid"}')
assert_tool_error "kvm_get_status rejects malformed GUID" "$res"

# --- live device, read-only --------------------------------------------------

section "Live device (read-only)"

if [ -z "$GUID" ]; then
  skip "no device GUID supplied -- pass one as the first argument"
  detail "example: $0 $VALID_GUID"
else
  detail "device: $GUID"

  res=$(call_tool 20 power_get_state "{\"guid\":\"$GUID\"}")
  assert_tool_ok "power_get_state" "$res"

  res=$(call_tool 21 power_get_capabilities "{\"guid\":\"$GUID\"}")
  assert_tool_ok "power_get_capabilities" "$res"
  # Only summarise when the array is actually present; on error it is null.
  if printf '%s' "$res" | grep -q '"supportedActions":\['; then
    SUPPORTED=$(printf '%s' "$res" \
      | sed -E 's/.*"supportedActions":\[([^]]*)\].*/\1/' | tr -d '"')
    [ -n "$SUPPORTED" ] && detail "supported: $SUPPORTED"
  fi

  res=$(call_tool 22 power_get_boot_sources "{\"guid\":\"$GUID\"}")
  assert_tool_ok "power_get_boot_sources" "$res"

  res=$(call_tool 23 kvm_get_status "{\"guid\":\"$GUID\"}")
  assert_tool_ok "kvm_get_status" "$res"

  res=$(call_tool 24 kvm_get_screen_settings "{\"guid\":\"$GUID\"}")
  assert_tool_ok "kvm_get_screen_settings" "$res"
fi

# --- live device, destructive ------------------------------------------------

section "Live device (destructive)"

if [ -z "$ACTION" ]; then
  skip "power_action -- pass --action <name> to run it"
  detail "example: $0 $GUID --action reset"
elif [ -z "$GUID" ]; then
  skip "power_action -- --action given but no device GUID"
else
  printf '  %sWARNING%s power_action %s on device %s\n' "$YELLOW" "$OFF" "$ACTION" "$GUID"
  printf '        This reboots or powers off physical hardware and can interrupt a running OS.\n'
  printf '        Type the action name to confirm: '
  read -r confirm

  if [ "$confirm" != "$ACTION" ]; then
    skip "power_action $ACTION (not confirmed)"
  else
    res=$(call_tool 30 power_action "{\"guid\":\"$GUID\",\"action\":\"$ACTION\"}")
    assert_tool_ok "power_action $ACTION" "$res"
    if printf '%s' "$res" | grep -q '"returnValue":0'; then
      pass "power_action returnValue:0 (WSMAN accepted)"
    else
      fail "power_action non-zero returnValue"
    fi
  fi
fi

# --- teardown ----------------------------------------------------------------

curl -sk --noproxy '*' --max-time "$TIMEOUT" -o /dev/null -X DELETE "$BASE/api/mcp" \
  -H "Authorization: Bearer $TOKEN" -H "Mcp-Session-Id: $SID"

section "Summary"
printf '  passed: %s  failed: %s  skipped: %s\n' "$PASSED" "$FAILED" "$SKIPPED"

if [ "$FAILED" -gt 0 ]; then
  printf '\n%sFAILED%s\n' "$RED" "$OFF"
  exit 1
fi

printf '\n%sOK%s\n' "$GREEN" "$OFF"
