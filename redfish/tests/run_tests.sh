#!/bin/bash
# run_tests.sh - Run Redfish API tests with mock server

set -eo pipefail

echo "=== Redfish API Test Runner ==="
echo ""

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# Get the repository root (two levels up from redfish/tests)
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

# Change to repo root to ensure consistent paths
cd "${REPO_ROOT}"

# Use port from environment or default to 8181
PORT=${HTTP_PORT:-8181}
TEST_CONFIG_DIR=$(mktemp -d)
TEST_CONFIG="${TEST_CONFIG_DIR}/config.yml"
if [ -n "${NEWMAN_REPORT:-}" ]; then
    REPORT_FILE=$NEWMAN_REPORT
else
    REPORT_DIR=$(mktemp -d "${TMPDIR:-/tmp}/redfish-results.XXXXXX")
    REPORT_FILE="${REPORT_DIR}/newman-report.json"
fi
SERVER_PID=""

cleanup() {
    if [ -n "$SERVER_PID" ]; then
        kill "$SERVER_PID" 2>/dev/null || true
        wait "$SERVER_PID" 2>/dev/null || true
    fi
    rm -f "$TEST_CONFIG"
    rmdir "$TEST_CONFIG_DIR" 2>/dev/null || true
}
trap cleanup EXIT

# Always use the isolated test config instead of a developer's local config.
echo "Creating test configuration..."
cat > "$TEST_CONFIG" << 'EOF'
app:
    name: "console"
    repo: "device-management-toolkit/console"
    version: "test"
    common_name: "localhost"
    encryption_key: "test-encryption-key-for-ci-testing-only"
    allow_insecure_ciphers: false
    disable_cira: true

http:
    host: "localhost"
    port: "8181"
    allowed_origins:
        - "http://localhost:8181"
        - "http://localhost:4200"
        - "http://127.0.0.1:8181"
        - "http://127.0.0.1:4200"
        - "https://localhost:8181"
        - "https://localhost:4200"
        - "https://127.0.0.1:8181"
        - "https://127.0.0.1:4200"
    allowed_headers:
        - "Origin"
        - "Accept"
        - "Content-Type"
        - "Content-Length"
        - "Authorization"
        - "If-Match"
    allow_credentials: true
    ws_compression: false
    tls:
        enabled: false
        certFile: ""
        keyFile: ""

logger:
    log_level: "info"

secrets:
    address: "http://localhost:8200"
    token: ""
    path: "secret/data/console"

postgres:
    provider: "sqlite"
    pool_max: 2
    url: ""

ea:
    url: "http://localhost:8000"
    username: ""
    password: ""

auth:
    disabled: false
    adminUsername: "standalone"
    adminPassword: "G@ppm0ym"
    jwtKey: "test-jwt-key-for-testing"
    jwtExpiration: 1h
    redirectionJWTExpiration: 5m
    clientId: ""
    issuer: ""
    tlsSkipVerify: false
    cookieEnabled: true
    cookieName: "console_session"
    cookieSecure: true
    cookieSameSite: "strict"
    ui:
        clientId: ""
        issuer: ""
        scope: ""
        redirectUri: ""
        responseType: "code"
        requireHttps: false
        strictDiscoveryDocumentValidation: true

ui:
    externalUrl: ""

package:
    rpc_repo: "device-management-toolkit/rpc-go"
    local_dir: ""
    disable_fetch: false
    max_token_ttl: 24h
EOF
echo "✓ Test configuration created"

# Start server with mock repository
echo "Starting server with mock WSMAN repository on port ${PORT}..."
echo "Environment: REDFISH_USE_MOCK=true HTTP_TLS_ENABLED=false HTTP_PORT=${PORT}"

# First, try to build to catch any compilation errors
echo "Building application..."
if ! go build -o /tmp/redfish_test_app ./cmd/app 2>&1 | tee /tmp/redfish_build.log; then
    echo "✗ Build failed. See build log:"
    cat /tmp/redfish_build.log
    exit 1
fi
echo "✓ Build successful"

# Start the built binary with config flag
REDFISH_USE_MOCK=true HTTP_TLS_ENABLED=false HTTP_PORT=${PORT} GIN_MODE=debug AUTH_ADMIN_USERNAME="standalone" AUTH_ADMIN_PASSWORD="G@ppm0ym" APP_ENCRYPTION_KEY="aB3dE5gH7jK9mN1pQ2rS4tU6wX8zC0vB" /tmp/redfish_test_app -config "$TEST_CONFIG" > /tmp/redfish_test_server.log 2>&1 &
SERVER_PID=$!
echo "Server PID: ${SERVER_PID}"

# Wait for server to start
echo "Waiting for server to start..."
for i in {1..10}; do
    sleep 1
    echo "Attempt $i/10: Checking if server is ready..."

    # Check if process is still running
    if ! kill -0 $SERVER_PID 2>/dev/null; then
        echo "✗ Server process died. Check logs:"
        echo ""
        echo "=== Server Log ==="
        cat /tmp/redfish_test_server.log
        echo ""
        exit 1
    fi

    if curl -fsS http://localhost:"${PORT}"/redfish/v1/ > /dev/null 2>&1; then
        echo "✓ Server started successfully on port ${PORT}"
        break
    fi
    if [ $i -eq 10 ]; then
        echo "✗ Server failed to start after 10 attempts"
        echo ""
        echo "=== Server Log ==="
        cat /tmp/redfish_test_server.log
        echo ""
        kill $SERVER_PID 2>/dev/null || true
        exit 1
    fi
done

# Run tests
echo "Running Newman tests..."
echo ""
# Bypass proxy for localhost
export no_proxy=localhost,127.0.0.1,::1
export NO_PROXY=localhost,127.0.0.1,::1
mkdir -p "$(dirname "$REPORT_FILE")"
if newman run "${SCRIPT_DIR}/postman/redfish-collection.json" \
    --environment "${SCRIPT_DIR}/postman/test-environment.json" \
    --env-var "base_url=http://localhost:${PORT}" \
    --reporters cli,json \
    --reporter-json-export "$REPORT_FILE"; then
    TEST_RESULT=0
else
    TEST_RESULT=$?
fi
echo "Newman report: ${REPORT_FILE}"

# Cleanup
echo ""
echo "Stopping server..."
cleanup
SERVER_PID=""

# Show server logs only on failure
if [ $TEST_RESULT -ne 0 ]; then
    echo ""
    echo "=== Server Logs (Test Failed) ==="
    cat /tmp/redfish_test_server.log || echo "No log file found"
    echo ""
    echo "✗ Some tests failed. Check results above."
else
    echo "✓ All tests passed!"
    echo ""
    echo "Note: Server logs available at /tmp/redfish_test_server.log"
fi

exit $TEST_RESULT

