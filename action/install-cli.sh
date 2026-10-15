#!/usr/bin/env bash
set -euo pipefail
# Fixed upstream release. Review version, asset and checksum together when updating.
version=v0.5.0
asset=gh-settings-linux-amd64
checksum=b4a63eb8912657deabc5ea8909a7e6198d375c94cb0a9cd83f7d2179f7bbf5b4
[[ "${RUNNER_OS:-}" == Linux && "${RUNNER_ARCH:-}" == X64 ]] || {
  echo 'requires Linux/X64; use a fresh GitHub-hosted Ubuntu 24.04 x64 runner' >&2; exit 1
}
[[ "${RUNNER_TEMP:-}" == /* && -d "$RUNNER_TEMP" ]] || {
  echo 'RUNNER_TEMP must name an existing absolute directory' >&2; exit 1
}
for tool in curl sha256sum; do
  command -v "$tool" > /dev/null || { printf 'required tool not found on PATH: %s\n' "$tool" >&2; exit 1; }
done
directory="$(mktemp -d "$RUNNER_TEMP/gh-settings-cli.XXXXXX")"
binary="$directory/gh-settings"
trap 'rm -rf "$directory"' EXIT
curl --fail --silent --show-error --location --proto '=https' --tlsv1.2 \
  "https://github.com/daiksud/gh-settings/releases/download/$version/$asset" --output "$binary"
printf '%s  %s\n' "$checksum" "$binary" | sha256sum --check --status
chmod 700 "$binary"
reported_version="$("$binary" --version)" || { echo 'CLI version check failed' >&2; exit 1; }
[[ "$reported_version" == "gh-settings $version" ]] || { echo 'unexpected CLI version' >&2; exit 1; }
trap - EXIT
printf '%s\n' "$binary"
