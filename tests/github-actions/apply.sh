#!/usr/bin/env bash
set -euo pipefail
# shellcheck source=tests/github-actions/fixture.sh
source "$(dirname "$0")/fixture.sh"
GH_TOKEN=read-fixture bash examples/github-actions/prepare.sh
CONFIG_SHA="$(sed -n 's/^config-sha=//p' "$GITHUB_OUTPUT")"
EXPECTED_TARGET="$(sed -n 's/^target=//p' "$GITHUB_OUTPUT")"
EXPECTED_CLI_VERSION="$(sed -n 's/^cli-version=//p' "$GITHUB_OUTPUT")"
EXPECTED_CLI_SHA256="$(sed -n 's/^cli-sha256=//p' "$GITHUB_OUTPUT")"
export CONFIG_SHA EXPECTED_TARGET EXPECTED_CLI_VERSION EXPECTED_CLI_SHA256
fresh_apply_checkout
export GITHUB_OUTPUT="$fixture/apply-output" RUNNER_TEMP="$fixture/apply-runner"
mkdir -p "$RUNNER_TEMP"
rm "$fixture/calls"
runner=examples/github-actions/verify-apply.sh
bash "$runner"
SETTINGS_CLI="$(sed -n 's/^binary-path=//p' "$GITHUB_OUTPUT")"
export SETTINGS_CLI
[[ "$(cat "$fixture/calls")" != *token-present* ]] || exit 1
if bash examples/github-actions/apply.sh > "$fixture/log" 2>&1; then exit 1; fi
[[ ! -e "$fixture/api" ]] || exit 1
GH_TOKEN=app-fixture bash examples/github-actions/apply.sh > "$fixture/log"
[[ "$(cat "$fixture/api")" == PATCH ]] || exit 1
grep -q 'Applied repository settings.' "$fixture/log"
if GH_TOKEN=app-fixture APPLY_ERROR=1 bash examples/github-actions/apply.sh > "$fixture/log" 2>&1; then exit 1; fi
if grep -q 'Applied repository settings.' "$fixture/log"; then exit 1; fi
rm "$fixture/api"
expect_failure() {
  rm -f "$GITHUB_OUTPUT" "$fixture/calls"
  if bash "$runner" > "$fixture/log" 2>&1; then echo 'unexpected apply verification success' >&2; exit 1; fi
  [[ ! -e "$GITHUB_OUTPUT" && ! -e "$fixture/api" ]] || exit 1
}
EXPECTED_TARGET=owner/other expect_failure
CONFIG_SHA='' expect_failure
EXPECTED_CLI_VERSION=v0.0.0 expect_failure
EXPECTED_CLI_SHA256=bad expect_failure
CORRUPT_DOWNLOAD=1 expect_failure
printf 'Approved apply input/token fixtures passed.\n'
