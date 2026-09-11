#!/bin/bash
set -e

# Unset empty proxy variables
for var in HTTP_PROXY HTTPS_PROXY NO_PROXY http_proxy https_proxy no_proxy; do
  eval v="\${$var}"
  if [ -z "$v" ]; then unset $var; fi
done

# Strip all trailing '/' or '\\' from proxy URLs for apt config
strip_trailing_slash() {
  local url="$1"
  # Remove all trailing / or \
  url="${url%%*(/|\\)}"
  # Fallback for Bash < 4.0 (no extglob): use sed
  echo "$url" | sed 's%[\\/]*$%%'
}

if [ -n "$HTTP_PROXY" ] || [ -n "$http_proxy" ] || [ -n "$HTTPS_PROXY" ] || [ -n "$https_proxy" ]; then
  echo "Configuring apt to use proxy..."
  sudo mkdir -p /etc/apt/apt.conf.d
  # Remove all trailing / or \\ from proxy URLs
  apt_http_proxy="$(strip_trailing_slash "${HTTP_PROXY:-${http_proxy:-}}")"
  apt_https_proxy="$(strip_trailing_slash "${HTTPS_PROXY:-${https_proxy:-}}")"
  sudo tee /etc/apt/apt.conf.d/99proxy > /dev/null <<EOF
Acquire::http::Proxy "$apt_http_proxy";
Acquire::https::Proxy "$apt_https_proxy";
EOF
fi

if [ -n "$HTTP_PROXY" ] || [ -n "$http_proxy" ] || [ -n "$HTTPS_PROXY" ] || [ -n "$https_proxy" ]; then
  echo "Configuring Docker daemon proxy..."
  docker_http_proxy="${HTTP_PROXY:-${http_proxy:-}}"
  docker_https_proxy="${HTTPS_PROXY:-${https_proxy:-}}"
  docker_no_proxy="${NO_PROXY:-${no_proxy:-}}"
  sudo mkdir -p /etc/docker
  sudo tee /etc/docker/daemon.json > /dev/null <<EOF
{
  "proxies": {
    "httpProxy": "${docker_http_proxy}",
    "httpsProxy": "${docker_https_proxy}",
    "noProxy": "${docker_no_proxy}"
  }
}
EOF

  if command -v systemctl >/dev/null 2>&1 && systemctl is-active --quiet docker; then
    sudo systemctl restart docker
  elif command -v service >/dev/null 2>&1 && service docker status >/dev/null 2>&1; then
    sudo service docker restart
  elif [ -x /usr/local/share/docker-init.sh ]; then
    sudo pkill dockerd || true
    sudo /usr/local/share/docker-init.sh >/tmp/docker-init.log 2>&1 || sudo /usr/local/share/docker-init.sh
  fi
fi

echo "Installing runtime dependencies..."
sudo apt-get update
sudo apt-get install -y dbus dbus-x11 xdg-utils libgtk2.0-0 libgtk-3-0 libgbm-dev libnotify-dev libnss3 libxss1 libasound2 libxtst6 xauth xvfb

go install github.com/air-verse/air@latest
go install github.com/go-delve/delve/cmd/dlv@latest