#!/usr/bin/env bash
set -euo pipefail
[[ $# == 2 ]] || { echo 'usage: plan.sh VERIFIED_CLI OUTPUT_FILE' >&2; exit 1; }
"$1" validate
"$1" plan > "$2"
cat "$2"
