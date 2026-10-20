#!/usr/bin/env bash
set -euo pipefail

work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT
go build -o "$work/release-publisher" "${BASH_SOURCE[0]%/*}/release-publisher/main.go"
"$work/release-publisher" "${BASH_SOURCE[0]%/*}/.." "$work"
