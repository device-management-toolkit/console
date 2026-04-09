#!/bin/bash
# Device Management Toolkit Console - Linux Package Build Script
# Copyright (c) Intel Corporation
# SPDX-License-Identifier: Apache-2.0

set -e

VERSION="${1:-0.0.0}"
ARCH="${2:-amd64}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
BUILD_DIR="$PROJECT_ROOT/installer/linux/build"
OUTPUT_DIR="$PROJECT_ROOT/dist/linux"

echo "Building Linux packages..."
echo "Version: $VERSION"
echo "Architecture: $ARCH"
echo ""

mkdir -p "$OUTPUT_DIR"

# Build binaries
echo "=== Building Binaries ==="

# Build UI binary with tray (requires CGO for systray/webview support)
echo "Building UI binary with tray (CGO_ENABLED=1)..."
UI_BINARY="$OUTPUT_DIR/console_linux_${ARCH}_tray"
CGO_ENABLED=1 GOOS=linux GOARCH=$ARCH go build -tags=tray -ldflags "-s -w" -trimpath -o "$UI_BINARY" "$PROJECT_ROOT/cmd/app"
echo "  Built: $UI_BINARY"

# Build headless binary with tray (requires CGO)
echo "Building headless binary with tray (CGO_ENABLED=1)..."
HEADLESS_BINARY="$OUTPUT_DIR/console_linux_${ARCH}_headless_tray"
CGO_ENABLED=1 GOOS=linux GOARCH=$ARCH go build -tags='tray noui' -ldflags "-s -w" -trimpath -o "$HEADLESS_BINARY" "$PROJECT_ROOT/cmd/app"
echo "  Built: $HEADLESS_BINARY"

echo ""

# Function to build a tar.gz package
build_package() {
    local EDITION="$1"      # "ui" or "headless"
    local BINARY="$2"       # path to binary
    local PKG_NAME="$3"     # output archive name (without extension)

    echo "=== Building $EDITION Package ==="

    # Clean and create build directory
    rm -rf "$BUILD_DIR"
    mkdir -p "$BUILD_DIR/$PKG_NAME"

    # Copy binary
    cp "$BINARY" "$BUILD_DIR/$PKG_NAME/console"
    chmod 755 "$BUILD_DIR/$PKG_NAME/console"

    # Copy and process scripts
    cp "$SCRIPT_DIR/configure.sh" "$BUILD_DIR/$PKG_NAME/configure.sh"
    sed -i "s/VERSION_PLACEHOLDER/$VERSION/g" "$BUILD_DIR/$PKG_NAME/configure.sh"
    chmod 755 "$BUILD_DIR/$PKG_NAME/configure.sh"

    cp "$SCRIPT_DIR/install.sh" "$BUILD_DIR/$PKG_NAME/install.sh"
    sed -i "s/VERSION_PLACEHOLDER/$VERSION/g" "$BUILD_DIR/$PKG_NAME/install.sh"
    chmod 755 "$BUILD_DIR/$PKG_NAME/install.sh"

    cp "$SCRIPT_DIR/uninstall.sh" "$BUILD_DIR/$PKG_NAME/uninstall.sh"
    sed -i "s/VERSION_PLACEHOLDER/$VERSION/g" "$BUILD_DIR/$PKG_NAME/uninstall.sh"
    chmod 755 "$BUILD_DIR/$PKG_NAME/uninstall.sh"

    # Copy systemd service file
    cp "$SCRIPT_DIR/dmt-console.service" "$BUILD_DIR/$PKG_NAME/dmt-console.service"
    chmod 644 "$BUILD_DIR/$PKG_NAME/dmt-console.service"

    # Create tar.gz archive
    echo "  Creating archive..."
    tar -czf "$OUTPUT_DIR/${PKG_NAME}.tar.gz" -C "$BUILD_DIR" "$PKG_NAME"

    echo "  Created: $OUTPUT_DIR/${PKG_NAME}.tar.gz"
    echo ""

    # Clean up
    rm -rf "$BUILD_DIR"
}

# Build UI package (with tray)
build_package "ui" "$UI_BINARY" "console_${VERSION}_linux_${ARCH}"

# Build Headless package
build_package "headless" "$HEADLESS_BINARY" "console_${VERSION}_linux_${ARCH}_headless"

echo "=== Build Complete ==="
echo ""
echo "Packages created:"
echo "  UI:       $OUTPUT_DIR/console_${VERSION}_linux_${ARCH}.tar.gz"
echo "  Headless: $OUTPUT_DIR/console_${VERSION}_linux_${ARCH}_headless.tar.gz"
echo ""
echo "Installation:"
echo "  1. Extract: tar -xzf console_${VERSION}_linux_${ARCH}.tar.gz"
echo "  2. Install: sudo ./install.sh"
echo "  3. Configure: sudo dmt-configure"
echo "  4. Start: sudo systemctl start dmt-console"
