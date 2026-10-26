#!/usr/bin/env bash
set -euo pipefail
[[ $# == 4 ]] || { echo 'usage: prepare-candidate-download.sh BINARY SHA256 SOURCE_SHA VERSION' >&2; exit 1; }
binary="$1" expected_sha="$2" expected_source="$3" version="$4"
[[ "$expected_sha" =~ ^[0-9a-f]{64}$ && "$expected_source" =~ ^[0-9a-f]{40}$ ]]
[[ "$version" =~ ^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]]
script_directory="$(cd "${BASH_SOURCE[0]%/*}" && pwd -P)"
action_root="$(cd "$script_directory/../.." && pwd -P)"
[[ "$(git -C "$action_root" rev-parse HEAD)" == "$expected_source" ]] || {
  echo 'candidate source mismatch' >&2; exit 1
}
[[ -f "$binary" ]] || { echo 'candidate binary is missing' >&2; exit 1; }
[[ "$("$binary" --version)" == "gh-settings $version" ]] || { echo 'candidate version mismatch' >&2; exit 1; }
[[ "$(sha256sum "$binary" | cut -d ' ' -f 1)" == "$expected_sha" ]] || {
  echo 'candidate binary checksum mismatch' >&2; exit 1
}
[[ "${RUNNER_TEMP:-}" == /* && -d "$RUNNER_TEMP" && "${GITHUB_PATH:-}" == /* && "${GITHUB_ENV:-}" == /* ]]
fixture="$(mktemp -d "$RUNNER_TEMP/gh-settings-candidate.XXXXXX")"
cp "$binary" "$fixture/gh-settings-linux-amd64"
printf '%s  gh-settings-linux-amd64\n' "$expected_sha" > "$fixture/SHA256SUMS"
printf '%s\n' "$version" > "$fixture/cli-version"
cp "$script_directory/candidate-curl.sh" "$fixture/curl"
chmod 700 "$fixture/curl"
printf '%s\n' "$fixture" >> "$GITHUB_PATH"
printf 'SETTINGS_CANDIDATE_VERSION=%s\n' "$version" >> "$GITHUB_ENV"
printf 'Candidate download fixture: source=%s cli-version=%s cli-sha256=%s\n' \
  "$expected_source" "$version" "$expected_sha"
