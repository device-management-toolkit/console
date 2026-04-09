#!/bin/bash
# Device Management Toolkit Console - Linux Uninstall Script
# Copyright (c) Intel Corporation
# SPDX-License-Identifier: Apache-2.0

set -e

APP_DIR="/usr/local/device-management-toolkit"
DATA_DIR="/var/lib/device-management-toolkit"
SERVICE_FILE="/etc/systemd/system/dmt-console.service"
SYMLINKS=(
    "/usr/local/bin/dmt-console"
    "/usr/local/bin/dmt-configure"
    "/usr/local/bin/dmt-uninstall"
)

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo ""
echo -e "${BLUE}=================================================="
echo "Device Management Toolkit Console Uninstaller"
echo -e "==================================================${NC}"
echo ""

# Check for root privileges
if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}Please run as root: sudo dmt-uninstall${NC}"
    exit 1
fi

# Check if installed
if [ ! -d "$APP_DIR" ] && [ ! -f "$SERVICE_FILE" ]; then
    echo "DMT Console does not appear to be installed."
    exit 0
fi

# Ask about data preservation
echo "Do you want to remove configuration and data files?"
echo "  - Configuration: $APP_DIR/config/"
echo "  - Data: $DATA_DIR/"
echo ""
read -p "Remove all data? [y/N]: " REMOVE_DATA

# Stop and disable the systemd service
echo ""
echo -e "${BLUE}Stopping service...${NC}"
if systemctl is-active --quiet dmt-console 2>/dev/null; then
    systemctl stop dmt-console
    echo "  Service stopped."
fi
if systemctl is-enabled --quiet dmt-console 2>/dev/null; then
    systemctl disable dmt-console
    echo "  Service disabled."
fi

# Remove systemd service file
if [ -f "$SERVICE_FILE" ]; then
    rm -f "$SERVICE_FILE"
    systemctl daemon-reload
    echo "  Service file removed and systemd reloaded."
fi

# Remove symlinks
echo ""
echo -e "${BLUE}Removing symlinks...${NC}"
for link in "${SYMLINKS[@]}"; do
    if [ -L "$link" ]; then
        rm -f "$link"
        echo "  Removed: $link"
    fi
done

# Remove application files
echo ""
echo -e "${BLUE}Removing application files...${NC}"
rm -f "$APP_DIR/console"
rm -f "$APP_DIR/configure.sh"
rm -f "$APP_DIR/uninstall.sh"
echo "  Removed binaries and scripts."

# Handle data removal
if [[ "$REMOVE_DATA" =~ ^[Yy]$ ]]; then
    echo ""
    echo -e "${BLUE}Removing configuration and data...${NC}"
    rm -rf "$APP_DIR/config"
    echo "  Removed: $APP_DIR/config/"

    rm -rf "$DATA_DIR"
    echo "  Removed: $DATA_DIR/"

    # Remove entire application directory if empty
    rmdir "$APP_DIR" 2>/dev/null && echo "  Removed: $APP_DIR/" || true

    # Remove dmt system user
    if id -u dmt > /dev/null 2>&1; then
        userdel dmt 2>/dev/null || true
        echo "  Removed system user 'dmt'."
    fi
else
    echo ""
    echo -e "${YELLOW}Keeping configuration and data files.${NC}"
fi

echo ""
echo -e "${GREEN}=================================================="
echo "DMT Console has been uninstalled."
echo -e "==================================================${NC}"

if [[ ! "$REMOVE_DATA" =~ ^[Yy]$ ]]; then
    echo ""
    echo "Configuration and data preserved at:"
    echo "  $APP_DIR/config/"
    echo "  $DATA_DIR/"
    echo ""
    echo "To completely remove all data, run:"
    echo "  sudo rm -rf $APP_DIR $DATA_DIR"
    echo "  sudo userdel dmt"
fi
echo ""

exit 0
