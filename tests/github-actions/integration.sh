#!/usr/bin/env bash
set -euo pipefail
# shellcheck source=tests/github-actions/fixture.sh
source "$(dirname "$0")/fixture.sh"
# The Go test supplies the real YAML run commands. Approval/token provider are local stubs.
if [[ "$SCENARIO" == failed-prepare ]]; then export INVALID=1; fi
if ! GH_TOKEN=read-fixture bash -c "$PREPARE_COMMAND"; then
  [[ "$SCENARIO" == failed-prepare && ! -e "$fixture/mints" && ! -e "$fixture/api" ]] || exit 1
  exit 0
fi
if [[ "$SCENARIO" == denied || "$SCENARIO" == waiting ]]; then
  [[ ! -e "$fixture/mints" && ! -e "$fixture/api" ]] || exit 1
  exit 0
fi
CONFIG_SHA="$(sed -n 's/^config-sha=//p' "$GITHUB_OUTPUT")"
EXPECTED_TARGET="$(sed -n 's/^target=//p' "$GITHUB_OUTPUT")"
EXPECTED_CLI_VERSION="$(sed -n 's/^cli-version=//p' "$GITHUB_OUTPUT")"
EXPECTED_CLI_SHA256="$(sed -n 's/^cli-sha256=//p' "$GITHUB_OUTPUT")"
export CONFIG_SHA EXPECTED_TARGET EXPECTED_CLI_VERSION EXPECTED_CLI_SHA256
export GITHUB_OUTPUT="$fixture/apply-output" RUNNER_TEMP="$fixture/apply-runner"
mkdir -p "$RUNNER_TEMP"
rm "$fixture/calls"
case "$SCENARIO" in
  target-mismatch) export EXPECTED_TARGET=owner/other ;;
  missing-handoff) export CONFIG_SHA='' ;;
  unverified-binary) export CORRUPT_DOWNLOAD=1 ;;
esac
if ! bash -c "$VERIFY_COMMAND"; then
  [[ "$SCENARIO" == target-mismatch || "$SCENARIO" == missing-handoff || "$SCENARIO" == unverified-binary ]] || exit 1
  [[ ! -e "$fixture/mints" && ! -e "$fixture/api" ]] || exit 1
  exit 0
fi
[[ "$(cat "$fixture/calls")" != *token-present* ]] || exit 1
SETTINGS_CLI="$(sed -n 's/^binary-path=//p' "$GITHUB_OUTPUT")"
export SETTINGS_CLI
# Local provider validates a dummy key and requested scope before minting a dummy token.
MOCK_PRIVATE_KEY='fixture-key'
[[ "$SCENARIO" != missing-key ]] || MOCK_PRIVATE_KEY=''
mint_token() {
  [[ -n "$MOCK_PRIVATE_KEY" && "$EXPECTED_TARGET" == owner/repo && "$TOKEN_PERMISSION" == write ]] || return 1
  printf 'mint owner/repo administration:write\n' > "$fixture/mints"
  printf 'app-fixture'
}
if ! app_token="$(mint_token)"; then
  [[ "$SCENARIO" == missing-key && ! -e "$fixture/mints" && ! -e "$fixture/api" ]] || exit 1
  exit 0
fi
cleanup() { printf 'revoke fixture token\n' > "$fixture/cleanup"; }
trap 'cleanup; rm -rf "$fixture"' EXIT
if [[ "$SCENARIO" == apply-failure ]]; then export APPLY_ERROR=1; fi
status=0
GH_TOKEN="$app_token" bash -c "$APPLY_COMMAND" > "$fixture/apply-log" 2>&1 || status=$?
[[ "$(cat "$fixture/api")" == PATCH ]] || exit 1
if [[ "$SCENARIO" == apply-failure ]]; then
  [[ "$status" != 0 ]] || exit 1
  export APPLY_OUTCOME=failure
else
  [[ "$SCENARIO" == approved && "$status" == 0 ]] || exit 1
  export APPLY_OUTCOME=success
fi
bash -c "$SUMMARY_COMMAND"
grep -q "CLI step outcome: $APPLY_OUTCOME" "$GITHUB_STEP_SUMMARY"
cleanup
[[ "$(cat "$fixture/cleanup")" == 'revoke fixture token' ]] || exit 1
[[ "$(cat "$fixture/calls")" == *'apply:token-present'* ]] || exit 1
if grep -q 'app-fixture\|read-fixture' "$GITHUB_STEP_SUMMARY" "$GITHUB_OUTPUT"; then exit 1; fi
