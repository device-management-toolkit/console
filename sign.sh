#!/bin/bash

# Sign release artifacts with Cosign keyless (Fulcio/OIDC) signing.
# The calling CI workflow must grant `id-token: write` and install cosign.
# For each artifact this emits a Sigstore bundle (.sigstore.json) in the
# repository root.

set -euo pipefail

repo="${GITHUB_REPOSITORY:-device-management-toolkit/console}"
cert_identity_regexp="^https://github.com/${repo}/.github/workflows/release\\.yml@refs/(heads|tags)/.*$"
oidc_issuer="https://token.actions.githubusercontent.com"

artifacts=(
  console_linux_x64.tar.gz
  console_linux_x64_headless.tar.gz
  console_linux_arm64.tar.gz
  console_linux_arm64_headless.tar.gz
  dist/windows/console_windows_x64.exe
  dist/windows/console_windows_x64_headless.exe
  console_mac_arm64.tar.gz
  console_mac_arm64_headless.tar.gz
)

for artifact in "${artifacts[@]}"; do
  if [ ! -f "$artifact" ]; then
    echo "Artifact not found, cannot sign: $artifact"
    exit 1
  fi

  base_name="$(basename "$artifact")"
  bundle_file="${base_name}.sigstore.json"

  echo "Signing artifact: $artifact"

  cosign sign-blob \
    --yes \
    --bundle "$bundle_file" \
    "$artifact"

  if [ "${COSIGN_SKIP_VERIFY:-0}" = "1" ]; then
    echo "Skipping verification for $artifact (COSIGN_SKIP_VERIFY=1)"
    continue
  fi

  echo "Verifying artifact: $artifact"
  cosign verify-blob \
    --bundle "$bundle_file" \
    --certificate-identity-regexp "$cert_identity_regexp" \
    --certificate-oidc-issuer "$oidc_issuer" \
    "$artifact"
done

echo "Cosign outputs:"
ls -lh ./*.sigstore.json || true
