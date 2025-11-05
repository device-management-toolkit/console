# this script builds the docker image and packages the binaries into tar files for CI/CD on Github

# Get version from the first argument
version=$1

# Set proxy environment variables if needed
export HTTP_PROXY="${HTTP_PROXY:-${http_proxy:-}}"
export HTTPS_PROXY="${HTTPS_PROXY:-${https_proxy:-}}"
export NO_PROXY="${NO_PROXY:-${no_proxy:-localhost,127.0.0.1}}"


docker  build --build-arg HTTP_PROXY=$HTTP_PROXY --build-arg HTTPS_PROXY=$HTTPS_PROXY --build-arg NO_PROXY=$NO_PROXY -t vprodemo.azurecr.io/console:v$version .

# Mark the Unix system outputs as executable
chmod +x dist/linux/console_linux_x64
chmod +x dist/darwin/console_mac_arm64

# Add them to tar files respectively
tar cvfpz console_linux_x64.tar.gz dist/linux/console_linux_x64
tar cvfpz console_mac_arm64.tar.gz dist/darwin/console_mac_arm64
