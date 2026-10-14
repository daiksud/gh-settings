#!/usr/bin/env bash
set -euo pipefail
repo_root="$(cd "$(dirname "$0")/../.." && pwd)"
if [[ "$(uname)" == Darwin ]]; then
  sha256sum() {
    if [[ "$1" == --check ]]; then shasum -a 256 -c - > /dev/null; else shasum -a 256 "$@"; fi
  }
  export -f sha256sum
fi
fixture="$(mktemp -d)"
trap 'rm -rf "$fixture"' EXIT
mkdir -p "$fixture/bin"
cat > "$fixture/bin/curl" <<'MOCK'
#!/usr/bin/env bash
set -euo pipefail
[[ "${FAIL_DOWNLOAD:-}" != 1 ]] || exit 22
while [[ $# -gt 0 ]]; do
  if [[ "$1" == --output ]]; then cp "$MOCK_ASSET" "$2"; exit; fi
  shift
done
exit 1
MOCK
chmod +x "$fixture/bin/curl"
export PATH="$fixture/bin:$PATH"
export MOCK_ASSET="$fixture/asset"
cat > "$MOCK_ASSET" <<'MOCK'
#!/usr/bin/env bash
[[ "$1" == --version ]] || exit 1
printf 'gh-settings v0.5.0\n'
MOCK
# Test the real installer with only the fixed checksum replaced by the fixture digest.
installer="$fixture/install-cli.sh"
sed "s/^checksum=.*/checksum=$(sha256sum "$MOCK_ASSET" | cut -d ' ' -f 1)/" \
  "$repo_root/examples/github-actions/install-cli.sh" > "$installer"
expect_failure() {
  if bash "$installer" "$fixture/installed" > "$fixture/output" 2> "$fixture/error"; then
    echo 'installer unexpectedly succeeded' >&2; exit 1
  fi
  [[ ! -e "$fixture/installed/gh-settings" ]] || exit 1
}
[[ "$(bash "$installer" "$fixture/installed")" == "$fixture/installed/gh-settings" ]] || exit 1
[[ "$("$fixture/installed/gh-settings" --version)" == 'gh-settings v0.5.0' ]] || exit 1
printf 'tampered\n' >> "$MOCK_ASSET"
expect_failure
FAIL_DOWNLOAD=1 expect_failure
rm "$MOCK_ASSET"
expect_failure
for content in "printf 'gh-settings dev\\n'" 'exit 126'; do
  printf '#!/usr/bin/env bash\n%s\n' "$content" > "$MOCK_ASSET"
  sed "s/^checksum=.*/checksum=$(sha256sum "$MOCK_ASSET" | cut -d ' ' -f 1)/" \
    "$repo_root/examples/github-actions/install-cli.sh" > "$installer"
  expect_failure
done
printf 'CLI bootstrap fixtures passed.\n'
