#!/usr/bin/env bash
set -euo pipefail
asset=gh-settings-linux-amd64
[[ "${RUNNER_OS:-}" == Linux && "${RUNNER_ARCH:-}" == X64 ]] || {
  echo 'requires Linux/X64; use a fresh GitHub-hosted Ubuntu 24.04 x64 runner' >&2; exit 1
}
[[ "${RUNNER_TEMP:-}" == /* && -d "$RUNNER_TEMP" ]] || {
  echo 'RUNNER_TEMP must name an existing absolute directory' >&2; exit 1
}
for tool in curl sha256sum; do
  command -v "$tool" > /dev/null || { printf 'required tool not found on PATH: %s\n' "$tool" >&2; exit 1; }
done
version="$(cat "${BASH_SOURCE[0]%/*}/cli-version")"
[[ "$version" =~ ^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]] || {
  echo 'committed CLI version must be vMAJOR.MINOR.PATCH' >&2; exit 1
}
if [[ "${SETTINGS_ACTION_REF:-}" =~ ^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]]; then
  [[ "$SETTINGS_ACTION_REF" == "$version" ]] || { echo 'Action tag differs from committed CLI version' >&2; exit 1; }
fi
directory="$(mktemp -d "$RUNNER_TEMP/gh-settings-cli.XXXXXX")"
binary="$directory/gh-settings"
trap 'rm -rf "$directory"' EXIT
# Both files come from one explicitly selected upstream release over HTTPS.
curl --fail --silent --show-error --location --proto '=https' --tlsv1.2 \
  "https://github.com/daiksud/gh-settings/releases/download/$version/SHA256SUMS" --output "$directory/SHA256SUMS"
checksum=''
while read -r hash name extra; do
  [[ "$name" == "$asset" ]] || continue
  [[ "$hash" =~ ^[0-9a-f]{64}$ && -z "$extra" && -z "$checksum" ]] || {
    echo 'invalid or duplicate CLI checksum' >&2; exit 1
  }
  checksum="$hash"
done < "$directory/SHA256SUMS"
[[ -n "$checksum" ]] || { echo 'missing CLI checksum' >&2; exit 1; }
curl --fail --silent --show-error --location --proto '=https' --tlsv1.2 \
  "https://github.com/daiksud/gh-settings/releases/download/$version/$asset" --output "$binary"
printf '%s  %s\n' "$checksum" "$binary" | sha256sum --check --status
chmod 700 "$binary"
reported_version="$("$binary" --version)" || { echo 'CLI version check failed' >&2; exit 1; }
[[ "$reported_version" == "gh-settings $version" ]] || { echo 'unexpected CLI version' >&2; exit 1; }
trap - EXIT
printf '%s\n' "$binary"
