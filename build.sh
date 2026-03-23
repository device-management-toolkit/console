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
chmod +x dist/linux/console_linux_arm64
chmod +x dist/linux/console_linux_arm64_headless
chmod +x dist/darwin/console_mac_arm64
chmod +x dist/darwin/console_mac_arm64_headless

# Prepare Linux installer scripts with version replacement
LINUX_INSTALLER_DIR="installer/linux"
STAGING_DIR=$(mktemp -d)

for script in configure.sh install.sh uninstall.sh dmt-console.service; do
    cp "$LINUX_INSTALLER_DIR/$script" "$STAGING_DIR/$script"
    sed -i "s/VERSION_PLACEHOLDER/$version/g" "$STAGING_DIR/$script"
done

# Package Linux variants (binary + installer scripts)
package_linux() {
    local binary=$1
    local output=$2
    local pkg_dir
    pkg_dir=$(mktemp -d)

    cp "$binary" "$pkg_dir/console"
    chmod +x "$pkg_dir/console"
    cp "$STAGING_DIR/configure.sh" "$pkg_dir/"
    cp "$STAGING_DIR/install.sh" "$pkg_dir/"
    cp "$STAGING_DIR/uninstall.sh" "$pkg_dir/"
    cp "$STAGING_DIR/dmt-console.service" "$pkg_dir/"
    chmod +x "$pkg_dir/configure.sh" "$pkg_dir/install.sh" "$pkg_dir/uninstall.sh"

    tar cvfpz "$output" -C "$pkg_dir" .
    rm -rf "$pkg_dir"
}

package_linux dist/linux/console_linux_x64 console_linux_x64.tar.gz
package_linux dist/linux/console_linux_x64_headless console_linux_x64_headless.tar.gz
package_linux dist/linux/console_linux_arm64 console_linux_arm64.tar.gz
package_linux dist/linux/console_linux_arm64_headless console_linux_arm64_headless.tar.gz

rm -rf "$STAGING_DIR"

# Package macOS variants
tar cvfpz console_mac_arm64.tar.gz dist/darwin/console_mac_arm64
tar cvfpz console_mac_arm64_headless.tar.gz dist/darwin/console_mac_arm64_headless
