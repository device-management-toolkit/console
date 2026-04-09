#!/bin/bash
# Device Management Toolkit Console - Windows Installer Build Script
# Copyright (c) Intel Corporation
# SPDX-License-Identifier: Apache-2.0
#
# Builds two NSIS installers: one for UI edition, one for headless.
# Requires: NSIS (makensis) installed and on PATH.

set -e

VERSION="${1:-0.0.0}"
ARCH="${2:-x64}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
NSI_FILE="$PROJECT_ROOT/installer/console.nsi"
OUTPUT_DIR="$PROJECT_ROOT/dist/windows"

echo "Building Windows NSIS installers..."
echo "Version: $VERSION"
echo "Architecture: $ARCH"
echo ""

mkdir -p "$OUTPUT_DIR"

# Build binaries
echo "=== Building Binaries ==="

echo "Building UI binary..."
UI_BINARY="$OUTPUT_DIR/console_windows_${ARCH}.exe"
CGO_ENABLED=1 GOOS=windows GOARCH=amd64 go build -tags=tray -ldflags "-s -w" -trimpath -o "$UI_BINARY" "$PROJECT_ROOT/cmd/app"
echo "  Built: $UI_BINARY"

echo "Building headless binary..."
HEADLESS_BINARY="$OUTPUT_DIR/console_windows_${ARCH}_headless.exe"
CGO_ENABLED=1 GOOS=windows GOARCH=amd64 go build -tags='tray noui' -ldflags "-s -w" -trimpath -o "$HEADLESS_BINARY" "$PROJECT_ROOT/cmd/app"
echo "  Built: $HEADLESS_BINARY"

echo ""

# Build NSIS installers
echo "=== Building NSIS Installers ==="

echo "Building UI installer..."
makensis -DVERSION="$VERSION" -DARCH="$ARCH" -DEDITION=ui -DBINARY="$UI_BINARY" "$NSI_FILE"
echo "  Created: console_${VERSION}_windows_${ARCH}_setup.exe"

echo "Building headless installer..."
makensis -DVERSION="$VERSION" -DARCH="$ARCH" -DEDITION=headless -DBINARY="$HEADLESS_BINARY" "$NSI_FILE"
echo "  Created: console_${VERSION}_windows_${ARCH}_headless_setup.exe"

echo ""
echo "=== Build Complete ==="
echo ""
echo "Installers created:"
echo "  UI:       console_${VERSION}_windows_${ARCH}_setup.exe"
echo "  Headless: console_${VERSION}_windows_${ARCH}_headless_setup.exe"
