#!/usr/bin/env bash
set -euo pipefail
# shellcheck source=tests/github-actions/fixture.sh
source "$(dirname "$0")/fixture.sh"
export CONFIG_SHA="$GITHUB_SHA" EXPECTED_TARGET=owner/repo
export EXPECTED_ACTION_SHA=9d01d3258d9f42239b2daeb1ac26bc532cd56105
: "${VERIFY_COMMAND:?missing YAML verifier command}" "${REVIEWED_ACTION_SHA:?missing reviewed YAML Action SHA}"
export EXPECTED_CLI_VERSION=v0.6.1
export EXPECTED_CLI_SHA256=7b93802a64609ed5daf1b3bd02e33ad9384fda26b7776233cd094c8dd85d5874
bash -e -o pipefail -c "$VERIFY_COMMAND"
[[ ! -e "$fixture/calls" && ! -e "$fixture/api" && ! -e "$GITHUB_OUTPUT" ]] || exit 1
for field in CONFIG_SHA EXPECTED_TARGET EXPECTED_ACTION_SHA REVIEWED_ACTION_SHA EXPECTED_CLI_VERSION EXPECTED_CLI_SHA256; do
  if env -u "$field" bash -e -o pipefail -c "$VERIFY_COMMAND" > "$fixture/log" 2>&1; then
    printf 'Missing %s accepted.\n' "$field" >&2; exit 1
  fi
  if env "$field=" bash -e -o pipefail -c "$VERIFY_COMMAND" > "$fixture/log" 2>&1; then
    printf 'Empty %s accepted.\n' "$field" >&2; exit 1
  fi
  [[ ! -e "$fixture/calls" && ! -e "$fixture/api" && ! -e "$GITHUB_OUTPUT" ]] || exit 1
done
for binding in \
  CONFIG_SHA=bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb \
  GITHUB_SHA=bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb \
  EXPECTED_TARGET=owner/other \
  EXPECTED_ACTION_SHA=bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb \
  REVIEWED_ACTION_SHA=bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb \
  EXPECTED_CLI_VERSION=v0.0.0 EXPECTED_CLI_SHA256=bad; do
  if env "$binding" bash -e -o pipefail -c "$VERIFY_COMMAND" > "$fixture/log" 2>&1; then
    printf 'Mismatched %s accepted.\n' "$binding" >&2; exit 1
  fi
  [[ ! -e "$fixture/calls" && ! -e "$fixture/api" && ! -e "$GITHUB_OUTPUT" ]] || exit 1
done
git remote set-url origin https://github.com/owner/other
if bash -e -o pipefail -c "$VERIFY_COMMAND" > "$fixture/log" 2>&1; then echo 'Mismatched checkout target accepted.' >&2; exit 1; fi
[[ ! -e "$fixture/calls" && ! -e "$fixture/api" && ! -e "$GITHUB_OUTPUT" ]] || exit 1
git remote set-url origin https://github.com/owner/repo
for context in 'pull_request refs/heads/main' 'push refs/heads/other' 'push refs/tags/main'; do
  read -r event ref <<< "$context"
  if GITHUB_EVENT_NAME="$event" GITHUB_REF="$ref" bash -e -o pipefail -c "$VERIFY_COMMAND" > "$fixture/log" 2>&1; then
    printf 'Ineligible context %s accepted.\n' "$context" >&2; exit 1
  fi
  [[ ! -e "$fixture/calls" && ! -e "$fixture/api" && ! -e "$GITHUB_OUTPUT" ]] || exit 1
done
printf 'Public Action caller handoff fixtures passed.\n'
