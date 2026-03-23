#!/bin/bash
# Device Management Toolkit Console - Configuration Script
# Copyright (c) Intel Corporation
# SPDX-License-Identifier: Apache-2.0

set -e

APP_DIR="/usr/local/device-management-toolkit"
CONFIG_FILE="$APP_DIR/config/config.yml"
VERSION="VERSION_PLACEHOLDER"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo ""
echo -e "${BLUE}=================================================="
echo "Device Management Toolkit Console Configuration"
echo -e "==================================================${NC}"
echo ""

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}Please run as root (sudo dmt-configure)${NC}"
    exit 1
fi

# Function to prompt with default value
prompt_with_default() {
    local prompt="$1"
    local default="$2"
    local result

    read -p "$prompt [$default]: " result
    echo "${result:-$default}"
}

# Function to prompt yes/no
prompt_yes_no() {
    local prompt="$1"
    local default="$2"
    local result

    while true; do
        read -p "$prompt (y/n) [$default]: " result
        result="${result:-$default}"
        case "$result" in
            [Yy]* ) echo "true"; return;;
            [Nn]* ) echo "false"; return;;
            * ) echo "Please answer y or n.";;
        esac
    done
}

# Function to prompt for password (hidden)
# IMPORTANT: echo newline to stderr, NOT stdout (stdout is the return value)
prompt_password() {
    local prompt="$1"
    local default="$2"
    local result

    read -s -p "$prompt [$default]: " result
    echo "" >&2
    echo "${result:-$default}"
}

echo -e "${YELLOW}Step 1: Network Configuration${NC}"
echo ""

HTTP_PORT=$(prompt_with_default "HTTP Port" "8181")

TLS_ENABLED=$(prompt_yes_no "Enable TLS/HTTPS (recommended)" "y")
if [ "$TLS_ENABLED" = "true" ]; then
    echo "  A self-signed certificate will be generated if none is provided."
fi

echo ""
echo -e "${YELLOW}Step 2: Administrator Credentials${NC}"
echo "  (Used for standalone authentication)"
echo ""

ADMIN_USERNAME=$(prompt_with_default "Admin Username" "standalone")
echo -e "${YELLOW}Note: Change the default password for security!${NC}"
ADMIN_PASSWORD=$(prompt_password "Admin Password" "G@ppm0ym")

echo ""
echo -e "${YELLOW}Configuration Summary${NC}"
echo "  Port:       $HTTP_PORT"
echo "  TLS:        $TLS_ENABLED"
echo "  Username:   $ADMIN_USERNAME"
echo "  Password:   ********"
echo ""

read -p "Apply this configuration? (y/n) [y]: " confirm
confirm="${confirm:-y}"

if [[ ! "$confirm" =~ ^[Yy]$ ]]; then
    echo "Configuration cancelled."
    exit 0
fi

echo ""
echo -e "${BLUE}Applying configuration...${NC}"

# Ensure config directory exists
mkdir -p "$(dirname "$CONFIG_FILE")"

# Generate config file
# IMPORTANT: Quote $ADMIN_USERNAME and $ADMIN_PASSWORD with double quotes
# to prevent YAML parsing issues with special characters
cat > "$CONFIG_FILE" << EOF
app:
  name: console
  repo: device-management-toolkit/console
  version: $VERSION
  encryption_key: ""
  allow_insecure_ciphers: false
http:
  host: localhost
  port: "$HTTP_PORT"
  ws_compression: false
  tls:
    enabled: $TLS_ENABLED
    certFile: ""
    keyFile: ""
  allowed_origins:
    - "*"
  allowed_headers:
    - "*"
logger:
  log_level: info
secrets:
  address: http://localhost:8200
  token: ""
postgres:
  pool_max: 2
  url: ""
ea:
  url: http://localhost:8000
  username: ""
  password: ""
auth:
  disabled: false
  adminUsername: "$ADMIN_USERNAME"
  adminPassword: "$ADMIN_PASSWORD"
  jwtKey: your_secret_jwt_key
  jwtExpiration: 24h0m0s
  redirectionJWTExpiration: 5m0s
  clientId: ""
  issuer: ""
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
EOF

# Make config readable by root and dmt group only
chmod 640 "$CONFIG_FILE"
chown root:dmt "$CONFIG_FILE" 2>/dev/null || chmod 644 "$CONFIG_FILE"
echo "  Configuration saved to $CONFIG_FILE"

# Restart the service if it's running
# Use WAS_RUNNING variable to track state; don't re-check later
WAS_RUNNING=false
if systemctl is-active --quiet dmt-console 2>/dev/null; then
    WAS_RUNNING=true
    echo "  Restarting DMT Console service..."
    systemctl restart dmt-console
    echo "  Service restarted."
fi

echo ""
echo -e "${GREEN}=================================================="
echo "Configuration complete!"
echo -e "==================================================${NC}"
echo ""

SCHEME="http"
if [ "$TLS_ENABLED" = "true" ]; then
    SCHEME="https"
fi

if [ "$WAS_RUNNING" = true ]; then
    echo "DMT Console has been restarted with the new configuration."
else
    echo "To start the service:"
    echo "  sudo systemctl start dmt-console"
fi
echo ""
echo "Access the web interface at:"
echo "  $SCHEME://localhost:$HTTP_PORT"
echo ""
