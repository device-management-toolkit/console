#!/bin/bash
# Device Management Toolkit Console - Linux Installation Script
# Copyright (c) Intel Corporation
# SPDX-License-Identifier: Apache-2.0

set -e

VERSION="VERSION_PLACEHOLDER"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

APP_DIR="/usr/local/device-management-toolkit"
CONFIG_DIR="$APP_DIR/config"
DATA_DIR="/var/lib/device-management-toolkit"
SYMLINK_DIR="/usr/local/bin"
SERVICE_FILE="/etc/systemd/system/dmt-console.service"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo ""
echo -e "${BLUE}=================================================="
echo "Device Management Toolkit Console Installer"
echo "Version: $VERSION"
echo -e "==================================================${NC}"
echo ""

# Check for root privileges
if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}This installer must be run as root.${NC}"
    echo "  Usage: sudo ./install.sh"
    exit 1
fi

# Check that required files exist in the script directory
if [ ! -f "$SCRIPT_DIR/console" ]; then
    echo -e "${RED}Error: console binary not found in $SCRIPT_DIR${NC}"
    echo "Please run this script from the extracted archive directory."
    exit 1
fi

echo -e "${BLUE}Installing DMT Console...${NC}"
echo ""

# Create dmt system user if it doesn't exist
if ! id -u dmt > /dev/null 2>&1; then
    echo "  Creating system user 'dmt'..."
    useradd --system --no-create-home --shell /usr/sbin/nologin dmt
    echo -e "  ${GREEN}Created system user 'dmt'${NC}"
else
    echo "  System user 'dmt' already exists."
fi

# Create application directory
echo "  Creating application directory..."
mkdir -p "$APP_DIR"

# Create config directory
if [ ! -d "$CONFIG_DIR" ]; then
    mkdir -p "$CONFIG_DIR"
    chmod 755 "$CONFIG_DIR"
fi

# Create data directory (Linux convention: /var/lib)
if [ ! -d "$DATA_DIR" ]; then
    mkdir -p "$DATA_DIR"
fi
chown dmt:dmt "$DATA_DIR"
chmod 755 "$DATA_DIR"

# Install binary
echo "  Installing binary..."
cp "$SCRIPT_DIR/console" "$APP_DIR/console"
chmod 755 "$APP_DIR/console"

# Install configure script
echo "  Installing configuration script..."
cp "$SCRIPT_DIR/configure.sh" "$APP_DIR/configure.sh"
chmod 755 "$APP_DIR/configure.sh"

# Install uninstall script
echo "  Installing uninstall script..."
cp "$SCRIPT_DIR/uninstall.sh" "$APP_DIR/uninstall.sh"
chmod 755 "$APP_DIR/uninstall.sh"

# Create symlinks in /usr/local/bin for easy CLI access
echo "  Creating symlinks..."
ln -sf "$APP_DIR/console" "$SYMLINK_DIR/dmt-console"
echo "    $SYMLINK_DIR/dmt-console -> $APP_DIR/console"
ln -sf "$APP_DIR/configure.sh" "$SYMLINK_DIR/dmt-configure"
echo "    $SYMLINK_DIR/dmt-configure -> $APP_DIR/configure.sh"
ln -sf "$APP_DIR/uninstall.sh" "$SYMLINK_DIR/dmt-uninstall"
echo "    $SYMLINK_DIR/dmt-uninstall -> $APP_DIR/uninstall.sh"

# Install systemd service file
echo "  Installing systemd service..."
cp "$SCRIPT_DIR/dmt-console.service" "$SERVICE_FILE"
chmod 644 "$SERVICE_FILE"
systemctl daemon-reload
echo -e "  ${GREEN}Systemd service installed${NC}"

# Generate default config if it doesn't exist (preserve existing config on upgrade)
if [ ! -f "$CONFIG_DIR/config.yml" ]; then
    echo "  Generating default configuration..."
    cat > "$CONFIG_DIR/config.yml" << EOF
app:
  name: console
  repo: device-management-toolkit/console
  version: $VERSION
  encryption_key: ""
  allow_insecure_ciphers: false
http:
  host: localhost
  port: "8181"
  ws_compression: false
  tls:
    enabled: true
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
  adminUsername: "standalone"
  adminPassword: "G@ppm0ym"
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
    chmod 640 "$CONFIG_DIR/config.yml"
    chown root:dmt "$CONFIG_DIR/config.yml"
    echo -e "  ${GREEN}Default configuration saved to $CONFIG_DIR/config.yml${NC}"
else
    echo "  Existing configuration preserved at $CONFIG_DIR/config.yml"
    chmod 640 "$CONFIG_DIR/config.yml" 2>/dev/null || true
    chown root:dmt "$CONFIG_DIR/config.yml" 2>/dev/null || true
fi

# Set ownership on application directory
chown -R root:dmt "$APP_DIR"
chown dmt:dmt "$DATA_DIR"

echo ""
echo -e "${GREEN}=================================================="
echo "Device Management Toolkit Console installed!"
echo -e "==================================================${NC}"
echo ""
echo "Next steps:"
echo ""
echo -e "  1. Configure the service (${YELLOW}recommended${NC}):"
echo "       sudo dmt-configure"
echo ""
echo "  2. Start the service:"
echo "       sudo systemctl start dmt-console"
echo ""
echo "  3. Enable auto-start on boot:"
echo "       sudo systemctl enable dmt-console"
echo ""
echo "Other commands:"
echo "  Check status:   systemctl status dmt-console"
echo "  View logs:      journalctl -u dmt-console -f"
echo "  Reconfigure:    sudo dmt-configure"
echo "  Uninstall:      sudo dmt-uninstall"
echo ""

exit 0
