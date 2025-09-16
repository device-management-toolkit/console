# this script builds the docker image and packages the binaries into tar files for CI/CD on Github

# Get version from the first argument
version=$1

docker build -t vprodemo.azurecr.io/console:v$version .

# Mark the Unix system outputs as executable
chmod +x dist/linux/console_linux_x64
chmod +x dist/darwin/console_mac_arm64

# Add them to tar files respectively
tar cvfpz console_linux_x64.tar.gz dist/linux/console_linux_x64
tar cvfpz console_mac_arm64.tar.gz dist/darwin/console_mac_arm64
