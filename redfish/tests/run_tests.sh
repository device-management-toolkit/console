#!/bin/bash
# run_tests.sh - Run Redfish API tests with mock server

set -e

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

# Create test config file if it doesn't exist or if we're in CI
if [ ! -f config/config.yml ] || [ -n "$CI" ]; then
    echo "Creating test configuration..."
    mkdir -p config
    # Backup existing config if it exists
    [ -f config/config.yml ] && cp config/config.yml config/config.yml.bak 2>/dev/null || true

    cat > config/config.yml << 'EOF'
app:
  name: "console"
  repo: "device-management-toolkit/console"
  version: "test"
  encryption_key: "test-encryption-key-for-ci-testing-only"

http:
  host: "localhost"
  port: "8181"
  allowed_origins: ["*"]
  allowed_headers: ["*"]
  ws_compression: false
  tls:
    enabled: false

logger:
  log_level: "info"

postgres:
  pool_max: 10
  url: ""

auth:
  disabled: false
  adminUsername: "standalone"
  adminPassword: "G@ppm0ym"
  jwtKey: "test-jwt-key-for-testing"
  jwtExpiration: 1h
  redirectionJWTExpiration: 5m
EOF
    echo "✓ Test configuration created"
fi

# Kill any existing servers
pkill -9 -f "redfish_test_app" 2>/dev/null || true
pkill -9 -f "go run.*cmd/app" 2>/dev/null || true
sleep 2

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
REDFISH_USE_MOCK=true HTTP_TLS_ENABLED=false HTTP_PORT=${PORT} GIN_MODE=debug APP_ENCRYPTION_KEY="test-encryption-key-for-ci-testing-only" /tmp/redfish_test_app -config ./config/config.yml > /tmp/redfish_test_server.log 2>&1 &
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

    if curl -s http://localhost:${PORT}/redfish/v1/ > /dev/null 2>&1; then
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
newman run "${SCRIPT_DIR}/postman/redfish-collection.json" \
    --environment "${SCRIPT_DIR}/postman/test-environment.json" \
    --reporters cli,json \
    --reporter-json-export "${SCRIPT_DIR}/postman/results/newman-report.json"

TEST_RESULT=$?

# Cleanup
echo ""
echo "Stopping server..."
kill $SERVER_PID 2>/dev/null || true

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

