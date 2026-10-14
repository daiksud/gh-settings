#!/usr/bin/env bash
set -euo pipefail
# shellcheck source=tests/github-actions/fixture.sh
source "$(dirname "$0")/fixture.sh"
runner=examples/github-actions/prepare.sh
GH_TOKEN=read-fixture bash "$runner"
grep -q "^config-sha=$GITHUB_SHA$" "$GITHUB_OUTPUT"
grep -q '^target=owner/repo$' "$GITHUB_OUTPUT"
grep -q 'Initiating / rerun actor</th><td>maintainer' "$GITHUB_STEP_SUMMARY"
[[ ! -e "$fixture/api" ]] || exit 1
# A later main ref does not move this run's checked-out configuration.
original="$GITHUB_SHA"
printf '\n' >> .github/settings.yml
git add .github/settings.yml
git -c commit.gpgsign=false -c user.name=Fixture -c user.email=fixture@example.invalid commit -qm 'test: newer main'
git checkout -q "$original"
rm "$GITHUB_OUTPUT" "$GITHUB_STEP_SUMMARY"
GH_TOKEN=read-fixture bash "$runner"
grep -q "^config-sha=$original$" "$GITHUB_OUTPUT"
expect_failure() {
  rm -f "$GITHUB_OUTPUT" "$GITHUB_STEP_SUMMARY" "$fixture/calls"
  if GH_TOKEN=read-fixture bash "$runner" > "$fixture/log" 2>&1; then echo 'unexpected prepare success' >&2; exit 1; fi
  [[ ! -e "$GITHUB_OUTPUT" && ! -e "$fixture/api" ]] || exit 1
}
INVALID=1 expect_failure
[[ "$(cat "$fixture/calls")" != *'plan:'* ]] || exit 1
API_ERROR=1 expect_failure
grep -q 'prepare failed' "$GITHUB_STEP_SUMMARY"
GITHUB_STEP_SUMMARY="$fixture/missing/summary" expect_failure
for context in 'pull_request refs/heads/main' 'push refs/heads/other' 'push refs/tags/main'; do
  read -r event ref <<< "$context"
  GITHUB_EVENT_NAME="$event" GITHUB_REF="$ref" expect_failure
  [[ ! -e "$fixture/calls" ]] || exit 1
done
GITHUB_SHA=bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb expect_failure
printf 'Main prepare fixtures passed.\n'
