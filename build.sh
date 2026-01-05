# this script builds the docker image and packages the binaries into tar files for CI/CD on Github

# Get version from the first argument
version=$1

# Build Docker images for each variant
# Full build (SQLite + UI)
docker build -t vprodemo.azurecr.io/console:v$version \
             -t vprodemo.azurecr.io/console:latest .

# Headless build (No UI + SQLite)
docker build --build-arg BUILD_TAGS="noui" \
             -t vprodemo.azurecr.io/console:v$version-headless \
             -t vprodemo.azurecr.io/console:latest-headless .

# Minimal build (No UI + PostgreSQL-only)
docker build --build-arg BUILD_TAGS="noui-nosqlite" \
             -t vprodemo.azurecr.io/console:v$version-minimal \
             -t vprodemo.azurecr.io/console:latest-minimal .

# Mark the Unix system outputs as executable
chmod +x dist/linux/console_linux_x64
chmod +x dist/linux/console_linux_x64_headless
chmod +x dist/linux/console_linux_x64_minimal
chmod +x dist/darwin/console_mac_arm64
chmod +x dist/darwin/console_mac_arm64_headless
chmod +x dist/darwin/console_mac_arm64_minimal

# Package Linux variants
tar cvfpz console_linux_x64.tar.gz dist/linux/console_linux_x64
tar cvfpz console_linux_x64_headless.tar.gz dist/linux/console_linux_x64_headless
tar cvfpz console_linux_x64_minimal.tar.gz dist/linux/console_linux_x64_minimal

# Package macOS variants
tar cvfpz console_mac_arm64.tar.gz dist/darwin/console_mac_arm64
tar cvfpz console_mac_arm64_headless.tar.gz dist/darwin/console_mac_arm64_headless
tar cvfpz console_mac_arm64_minimal.tar.gz dist/darwin/console_mac_arm64_minimal
