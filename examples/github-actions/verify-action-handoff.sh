#!/usr/bin/env bash
set -euo pipefail
: "${CONFIG_SHA:?missing prepare configuration SHA}" "${EXPECTED_TARGET:?missing prepare target}"
: "${EXPECTED_ACTION_SHA:?missing prepare Action SHA}" "${REVIEWED_ACTION_SHA:?missing reviewed Action SHA}"
: "${EXPECTED_CLI_VERSION:?missing prepare CLI version}" "${EXPECTED_CLI_SHA256:?missing prepare CLI checksum}"
[[ "$GITHUB_EVENT_NAME" == push && "$GITHUB_REF" == refs/heads/main ]] || { echo 'apply requires a main branch push' >&2; exit 1; }
[[ "$(git rev-parse HEAD)" == "$CONFIG_SHA" && "$CONFIG_SHA" == "$GITHUB_SHA" ]] || { echo 'configuration commit mismatch' >&2; exit 1; }
[[ "$EXPECTED_TARGET" == "$GITHUB_REPOSITORY" ]] || { echo 'prepare target mismatch' >&2; exit 1; }
[[ "$(git remote get-url origin)" == "https://github.com/$EXPECTED_TARGET" || "$(git remote get-url origin)" == "https://github.com/$EXPECTED_TARGET.git" ]] || { echo 'checkout target mismatch' >&2; exit 1; }
[[ "$EXPECTED_ACTION_SHA" == "$REVIEWED_ACTION_SHA" ]] || { echo 'prepare Action revision mismatch' >&2; exit 1; }
# The reviewed immutable Action package fixes this CLI identity; it owns installation.
[[ "$EXPECTED_CLI_VERSION" == v0.5.0 ]] || { echo 'prepare CLI version mismatch' >&2; exit 1; }
[[ "$EXPECTED_CLI_SHA256" == b4a63eb8912657deabc5ea8909a7e6198d375c94cb0a9cd83f7d2179f7bbf5b4 ]] || { echo 'prepare CLI checksum mismatch' >&2; exit 1; }
