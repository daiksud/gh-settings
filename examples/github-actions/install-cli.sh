#!/usr/bin/env bash
set -euo pipefail
# Fixed upstream release. Review version, asset and checksum together when updating.
version=v0.5.0
asset=gh-settings-linux-amd64
checksum=b4a63eb8912657deabc5ea8909a7e6198d375c94cb0a9cd83f7d2179f7bbf5b4
[[ $# == 1 && -n "$1" ]] || { echo 'usage: install-cli.sh DESTINATION_DIRECTORY' >&2; exit 1; }
mkdir -p "$1"
binary="$1/gh-settings"
# Never leave a previous or partially downloaded executable available on failure.
rm -f "$binary"
trap 'rm -f "$binary"' EXIT
curl --fail --silent --show-error --location --proto '=https' --tlsv1.2 \
  "https://github.com/daiksud/gh-settings/releases/download/$version/$asset" --output "$binary"
printf '%s  %s\n' "$checksum" "$binary" | sha256sum --check --status
chmod 700 "$binary"
[[ "$("$binary" --version)" == "gh-settings $version" ]] || { echo 'unexpected CLI version' >&2; exit 1; }
trap - EXIT
printf '%s\n' "$binary"
