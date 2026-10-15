#!/usr/bin/env bash
set -euo pipefail
[[ $# == 3 ]] || { echo 'usage: run-command.sh VERIFIED_CLI COMMAND OUTPUT_FILE' >&2; exit 1; }
cli="$1" command="$2" output="$3"
[[ "$cli" == /* && -x "$cli" ]] || { echo 'missing verified absolute CLI' >&2; exit 1; }
cd "${GITHUB_WORKSPACE:?missing caller checkout}"
case "$command" in
  validate) env -u GH_TOKEN -u GITHUB_TOKEN "$cli" validate > "$output" 2>&1 ;;
  plan)
    : "${GH_TOKEN:?missing caller token}"
    { env -u GH_TOKEN -u GITHUB_TOKEN "$cli" validate && "$cli" plan; } > "$output" 2>&1
    ;;
  apply)
    : "${GH_TOKEN:?missing caller token}"
    "$cli" apply > "$output" 2>&1
    ;;
  *) echo 'unsupported command' >&2; exit 1 ;;
esac
