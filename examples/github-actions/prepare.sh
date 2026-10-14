#!/usr/bin/env bash
set -euo pipefail
[[ "$GITHUB_EVENT_NAME" == push && "$GITHUB_REF" == refs/heads/main ]] || { echo 'prepare requires a main branch push' >&2; exit 1; }
config_sha="$(git rev-parse HEAD)"
[[ "$config_sha" == "$GITHUB_SHA" ]] || { echo 'checkout does not match configuration commit' >&2; exit 1; }
[[ "$(git remote get-url origin)" == "https://github.com/$GITHUB_REPOSITORY" || "$(git remote get-url origin)" == "https://github.com/$GITHUB_REPOSITORY.git" ]] || { echo 'checkout target mismatch' >&2; exit 1; }
cli="$(bash examples/github-actions/install-cli.sh "$RUNNER_TEMP/settings-cli")"
export SETTINGS_ACTOR="$GITHUB_ACTOR" SETTINGS_TRIGGERING_ACTOR="$GITHUB_TRIGGERING_ACTOR"
export SETTINGS_EVENT="$GITHUB_EVENT_NAME" SETTINGS_RUN_ID="$GITHUB_RUN_ID" SETTINGS_RUN_ATTEMPT="$GITHUB_RUN_ATTEMPT"
export SETTINGS_WORKFLOW_REF="$GITHUB_WORKFLOW_REF" SETTINGS_WORKFLOW_SHA="$GITHUB_WORKFLOW_SHA"
export SETTINGS_CONFIG_SHA="$config_sha" SETTINGS_TARGET="$GITHUB_REPOSITORY"
SETTINGS_CLI_VERSION="$("$cli" --version)"
SETTINGS_CLI_VERSION="${SETTINGS_CLI_VERSION#gh-settings }"
SETTINGS_CLI_SHA256="$(sha256sum "$cli" | cut -d ' ' -f 1)"
export SETTINGS_CLI_VERSION SETTINGS_CLI_SHA256
status=0
bash examples/github-actions/plan.sh "$cli" "$RUNNER_TEMP/settings-plan.txt" > "$RUNNER_TEMP/settings-prepare.txt" 2>&1 || status=$?
python3 examples/github-actions/render-summary.py "$RUNNER_TEMP/settings-prepare.txt" "$status" >> "$GITHUB_STEP_SUMMARY"
[[ "$status" == 0 ]] || exit "$status"
printf 'config-sha=%s\ncli-version=%s\ncli-sha256=%s\ntarget=%s\n' \
  "$config_sha" "$SETTINGS_CLI_VERSION" "$SETTINGS_CLI_SHA256" "$GITHUB_REPOSITORY" >> "$GITHUB_OUTPUT"
