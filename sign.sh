#!/bin/bash

# Sign release artifacts with Cosign keyless (Fulcio/OIDC) signing.
# The calling CI workflow must grant `id-token: write` and install cosign.
# For each artifact this emits a path-qualified Cosign bundle in the repository
# root, plus a basename alias for legacy verification when aliases are unique.

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

compatibility_aliases_enabled=1
declare -A compatibility_bundle_sources=()
for artifact in "${artifacts[@]}"; do
  base_name="$(basename "$artifact")"
  if [[ -n "${compatibility_bundle_sources[$base_name]:-}" ]]; then
    printf 'Basename aliases disabled: %s and %s share basename %s; verify path-qualified bundles instead\n' \
      "${compatibility_bundle_sources[$base_name]}" "$artifact" "$base_name" >&2
    compatibility_aliases_enabled=0
  else
    compatibility_bundle_sources[$base_name]="$artifact"
  fi
done

for artifact in "${artifacts[@]}"; do
  if [ ! -f "$artifact" ]; then
    echo "Artifact not found, cannot sign: $artifact"
    exit 1
  fi

  bundle_name="${artifact//%/%25}"
  bundle_name="${bundle_name//\//%2F}"
  bundle_file="${bundle_name}.cosign.bundle.json"
  compatibility_bundle_file="$(basename "$artifact").cosign.bundle.json"

  echo "Signing artifact: $artifact"

  cosign sign-blob \
    --yes \
    --bundle "$bundle_file" \
    "$artifact"

  if [[ "$compatibility_aliases_enabled" == 1 && "$bundle_file" != "$compatibility_bundle_file" ]]; then
    cp "$bundle_file" "$compatibility_bundle_file"
  fi

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
ls -lh ./*.cosign.bundle.json || true
