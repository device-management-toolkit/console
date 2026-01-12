#!/bin/bash
# This script builds Docker images and packages binaries into tar files for CI/CD on Github
#
# With CGO_ENABLED=0, Go produces static binaries that are cross-platform compatible.
# All platform binaries are built using cross-compilation from a single environment.

# Get version from the first argument
version=$1

# Build Docker images for each variant
# Full build (with UI)
docker build -t vprodemo.azurecr.io/console:v$version \
             -t vprodemo.azurecr.io/console:latest .

# Headless build (No UI)
docker build --build-arg BUILD_TAGS="noui" \
             -t vprodemo.azurecr.io/console:v$version-headless \
             -t vprodemo.azurecr.io/console:latest-headless .

# Mark the Unix system outputs as executable
chmod +x dist/linux/console_linux_x64
chmod +x dist/linux/console_linux_x64_headless
chmod +x dist/darwin/console_mac_arm64
chmod +x dist/darwin/console_mac_arm64_headless

# Package Linux variants
tar cvfpz console_linux_x64.tar.gz dist/linux/console_linux_x64
tar cvfpz console_linux_x64_headless.tar.gz dist/linux/console_linux_x64_headless

# Package macOS variants
tar cvfpz console_mac_arm64.tar.gz dist/darwin/console_mac_arm64
tar cvfpz console_mac_arm64_headless.tar.gz dist/darwin/console_mac_arm64_headless
