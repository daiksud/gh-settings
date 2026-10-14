#!/usr/bin/env bash
set -euo pipefail
: "${CONFIG_SHA:?missing prepare configuration SHA}" "${EXPECTED_TARGET:?missing prepare target}"
: "${EXPECTED_CLI_VERSION:?missing prepare CLI version}" "${EXPECTED_CLI_SHA256:?missing prepare CLI checksum}"
[[ "$GITHUB_EVENT_NAME" == push && "$GITHUB_REF" == refs/heads/main ]] || { echo 'apply requires a main branch push' >&2; exit 1; }
[[ "$EXPECTED_TARGET" == "$GITHUB_REPOSITORY" ]] || { echo 'prepare target mismatch' >&2; exit 1; }
[[ "$(git rev-parse HEAD)" == "$CONFIG_SHA" ]] || { echo 'checkout does not match prepare configuration commit' >&2; exit 1; }
[[ "$(git remote get-url origin)" == "https://github.com/$EXPECTED_TARGET" || "$(git remote get-url origin)" == "https://github.com/$EXPECTED_TARGET.git" ]] || { echo 'checkout target mismatch' >&2; exit 1; }
cli="$(bash examples/github-actions/install-cli.sh "$RUNNER_TEMP/settings-cli")"
[[ "$("$cli" --version)" == "gh-settings $EXPECTED_CLI_VERSION" ]] || { echo 'prepare CLI version mismatch' >&2; exit 1; }
[[ "$(sha256sum "$cli" | cut -d ' ' -f 1)" == "$EXPECTED_CLI_SHA256" ]] || { echo 'prepare CLI checksum mismatch' >&2; exit 1; }
printf 'binary-path=%s\n' "$cli" >> "$GITHUB_OUTPUT"
