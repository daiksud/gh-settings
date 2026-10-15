#!/usr/bin/env bash
set -euo pipefail
repo_root="$(cd "$(dirname "$0")/../.." && pwd)"
bash_path="$(command -v bash)"
fixture="$(mktemp -d)"
trap 'rm -rf "$fixture"' EXIT
mkdir -p "$fixture/bin" "$fixture/action" "$fixture/caller" "$fixture/runner temp"
if [[ "$(uname)" == Darwin ]]; then
  MOCK_SHASUM="$(command -v shasum)"
  export MOCK_SHASUM
  cat > "$fixture/bin/sha256sum" <<'MOCK'
#!/bin/bash
if [[ "$1" == --check ]]; then "$MOCK_SHASUM" -a 256 -c - > /dev/null; else "$MOCK_SHASUM" -a 256 "$@"; fi
MOCK
  chmod +x "$fixture/bin/sha256sum"
fi
export MOCK_ASSET="$fixture/asset" MOCK_CALLS="$fixture/calls" RUNNER_TEMP="$fixture/runner temp"
export RUNNER_OS=Linux RUNNER_ARCH=X64
cat > "$fixture/bin/curl" <<'MOCK'
#!/usr/bin/env bash
set -euo pipefail
while [[ $# -gt 0 ]]; do
  if [[ "$1" == --output ]]; then
    cp "$MOCK_ASSET" "$2"
    [[ "${FAIL_DOWNLOAD:-}" != 1 ]] || { echo 'fixture transfer failed' >&2; exit 22; }
    exit
  fi
  shift
done
exit 1
MOCK
chmod +x "$fixture/bin/curl"
export PATH="$fixture/bin:$PATH"
cat > "$MOCK_ASSET" <<'MOCK'
#!/usr/bin/env bash
printf '%s\n' "$1" >> "$MOCK_CALLS"
[[ "$1" == --version ]] || exit 1
printf '%s\n' "${MOCK_VERSION:-gh-settings v0.5.0}"
exit "${VERSION_STATUS:-0}"
MOCK
# Reuse the bootstrap fixture pattern: only substitute the dummy asset's digest.
installer="$fixture/action/install-cli.sh"
sed "s/^checksum=.*/checksum=$(sha256sum "$MOCK_ASSET" | cut -d ' ' -f 1)/" \
  "$repo_root/action/install-cli.sh" > "$installer"
cd "$fixture/caller"
binary="$(bash "$installer")"
second_binary="$(bash "$installer")"
[[ "$binary" == "$RUNNER_TEMP/"* && -x "$binary" ]] || exit 1
[[ "$("$binary" --version)" == 'gh-settings v0.5.0' ]] || exit 1
[[ "$second_binary" == "$RUNNER_TEMP/"* && -x "$second_binary" && "$binary" != "$second_binary" ]] || exit 1
expect_failure() {
  rm -f "$MOCK_CALLS"
  if PATH="${INSTALLER_PATH:-$PATH}" "$bash_path" "$installer" > "$fixture/output" 2> "$fixture/error"; then
    echo 'installer unexpectedly succeeded' >&2; exit 1
  fi
  [[ ! -s "$fixture/output" ]] || exit 1
  if [[ "${1:-}" == --version ]]; then
    [[ -s "$MOCK_CALLS" ]] || exit 1
    if grep -vx -- --version "$MOCK_CALLS"; then exit 1; fi
  else
    [[ ! -e "$MOCK_CALLS" ]] || exit 1
  fi
  [[ -z "$(find "$fixture/failed runner" "$fixture/caller" -type f)" ]] || exit 1
  [[ -x "$binary" && -x "$second_binary" ]] || exit 1
}
export RUNNER_TEMP="$fixture/failed runner"
mkdir -p "$RUNNER_TEMP"
FAIL_DOWNLOAD=1 expect_failure
cp "$MOCK_ASSET" "$fixture/good-asset"
printf '\n# tampered\n' >> "$MOCK_ASSET"
expect_failure
cp "$fixture/good-asset" "$MOCK_ASSET"
MOCK_VERSION='gh-settings dev' expect_failure --version
VERSION_STATUS=126 expect_failure --version
RUNNER_OS=Windows expect_failure
grep -q 'Linux/X64' "$fixture/error"
RUNNER_ARCH=ARM64 expect_failure
grep -q 'Linux/X64' "$fixture/error"
mkdir 'relative runner'
RUNNER_TEMP='relative runner' expect_failure
grep -q 'RUNNER_TEMP' "$fixture/error"
RUNNER_TEMP='' expect_failure
grep -q 'RUNNER_TEMP' "$fixture/error"
RUNNER_TEMP="$fixture/missing runner" expect_failure
grep -q 'RUNNER_TEMP' "$fixture/error"
mkdir "$fixture/limited-bin"
INSTALLER_PATH="$fixture/limited-bin" expect_failure
grep -q 'curl' "$fixture/error" || { cat "$fixture/error" >&2; exit 1; }
cp "$fixture/bin/curl" "$fixture/limited-bin/curl"
INSTALLER_PATH="$fixture/limited-bin" expect_failure
grep -q 'sha256sum' "$fixture/error" || { cat "$fixture/error" >&2; exit 1; }
printf 'Action CLI install fixtures passed.\n'
