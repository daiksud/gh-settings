#!/usr/bin/env bash
set -euo pipefail
asset=gh-settings-linux-amd64
[[ "${RUNNER_OS:-}" == Linux && "${RUNNER_ARCH:-}" == X64 ]] || {
  echo 'requires Linux/X64; use a fresh GitHub-hosted Ubuntu 24.04 x64 runner' >&2; exit 1
}
[[ "${RUNNER_TEMP:-}" == /* && -d "$RUNNER_TEMP" ]] || {
  echo 'RUNNER_TEMP must name an existing absolute directory' >&2; exit 1
}
for tool in curl sha256sum git; do
  command -v "$tool" > /dev/null || { printf 'required tool not found on PATH: %s\n' "$tool" >&2; exit 1; }
done
# A published Action reference identifies its own immutable CLI release.
# Candidate actions invoked locally have no ref and use an explicit test
# version, falling back to the committed development candidate version.
ref="${SETTINGS_ACTION_REF:-}"
if [[ "$ref" =~ ^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]]; then
  version="$ref"
elif [[ "$ref" =~ ^[0-9a-f]{40}$ ]]; then
  tags="$(git ls-remote --tags https://github.com/daiksud/gh-settings.git)" || {
    echo 'cannot resolve the SHA-pinned Action release tag' >&2; exit 1
  }
  version=''
  while read -r commit tag; do
    if [[ "$commit" == "$ref" && "$tag" =~ ^refs/tags/(v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*))(\^\{\})?$ ]]; then
      candidate="${BASH_REMATCH[1]}"
      if [[ -n "$version" && "$version" != "$candidate" ]]; then
        echo 'multiple stable tags point to the SHA-pinned Action' >&2; exit 1
      fi
      version="$candidate"
    fi
  done <<< "$tags"
  [[ -n "$version" ]] || { echo 'SHA-pinned Action has no matching stable release tag' >&2; exit 1; }
elif [[ -z "$ref" ]]; then
  version="${SETTINGS_CANDIDATE_VERSION:-$(cat "${BASH_SOURCE[0]%/*}/cli-version")}"
else
  echo 'unsupported Action reference (expected version tag or full commit SHA)' >&2
  exit 1
fi
[[ "$version" =~ ^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]] || {
  echo 'resolved CLI version must be vMAJOR.MINOR.PATCH' >&2; exit 1
}
directory="$(mktemp -d "$RUNNER_TEMP/gh-settings-cli.XXXXXX")"
binary="$directory/gh-settings"
trap 'rm -rf "$directory"' EXIT
# Both files come from one explicitly selected upstream release over HTTPS.
curl --fail --silent --show-error --location --proto '=https' --tlsv1.2 \
  "https://github.com/daiksud/gh-settings/releases/download/$version/SHA256SUMS" --output "$directory/SHA256SUMS"
checksum=''
while read -r hash name extra; do
  [[ "$name" == "$asset" ]] || continue
  [[ "$hash" =~ ^[0-9a-f]{64}$ && -z "$extra" && -z "$checksum" ]] || {
    echo 'invalid or duplicate CLI checksum' >&2; exit 1
  }
  checksum="$hash"
done < "$directory/SHA256SUMS"
[[ -n "$checksum" ]] || { echo 'missing CLI checksum' >&2; exit 1; }
curl --fail --silent --show-error --location --proto '=https' --tlsv1.2 \
  "https://github.com/daiksud/gh-settings/releases/download/$version/$asset" --output "$binary"
printf '%s  %s\n' "$checksum" "$binary" | sha256sum --check --status
chmod 700 "$binary"
reported_version="$("$binary" --version)" || { echo 'CLI version check failed' >&2; exit 1; }
[[ "$reported_version" == "gh-settings $version" ]] || { echo 'unexpected CLI version' >&2; exit 1; }
trap - EXIT
printf '%s\n' "$binary"
