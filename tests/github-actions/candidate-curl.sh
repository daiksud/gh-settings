#!/usr/bin/env bash
# CI-only stand-in: serve the same-run candidate through the unchanged installer.
set -euo pipefail
url='' output=''
while [[ $# -gt 0 ]]; do
  case "$1" in
    --fail|--silent|--show-error|--location|--tlsv1.2) shift ;;
    --proto)
      [[ $# -ge 2 && "$2" == '=https' ]] || exit 1
      shift 2 ;;
    --output)
      [[ $# -ge 2 && -z "$output" ]] || exit 1
      output="$2"; shift 2 ;;
    https://*)
      [[ -z "$url" ]] || exit 1
      url="$1"; shift ;;
    *) echo 'unsupported candidate installer request' >&2; exit 1 ;;
  esac
done
[[ -n "$output" ]]
fixture="$(cd "${BASH_SOURCE[0]%/*}" && pwd -P)"
version="$(cat "$fixture/cli-version")"
base="https://github.com/daiksud/gh-settings/releases/download/$version"
case "$url" in
  "$base/SHA256SUMS") asset=SHA256SUMS ;;
  "$base/gh-settings-linux-amd64") asset=gh-settings-linux-amd64 ;;
  *) echo 'unexpected candidate installer URL' >&2; exit 1 ;;
esac
printf '%s\n' "$url" >> "$fixture/requests"
cp "$fixture/$asset" "$output"
