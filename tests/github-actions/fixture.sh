#!/usr/bin/env bash
# Source from focused offline tests. No real token, network or GitHub repository writes.
set -euo pipefail
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
if [[ "$(uname)" == Darwin ]]; then
  sha256sum() {
    if [[ "$1" == --check ]]; then shasum -a 256 -c - > /dev/null; else shasum -a 256 "$@"; fi
  }
  export -f sha256sum
fi
fixture="$(mktemp -d)"
trap 'rm -rf "$fixture"' EXIT
mkdir -p "$fixture/bin" "$fixture/repo/.github"
cp -R "$repo_root/examples" "$fixture/repo/"
export MOCK_ROOT="$fixture" MOCK_ASSET="$fixture/asset"
cat > "$MOCK_ASSET" <<'MOCK'
#!/usr/bin/env bash
set -euo pipefail
printf '%s:%s\n' "$1" "${GH_TOKEN:+token-present}" >> "$MOCK_ROOT/calls"
case "$1" in
  --version) printf 'gh-settings v0.5.0\n' ;;
  validate) [[ "${INVALID:-}" != 1 ]] ;;
  plan)
    [[ "${API_ERROR:-}" != 1 ]] || { echo 'read permission denied' >&2; exit 1; }
    printf 'settings.repository.has_issues: true → false\n' ;;
  apply)
    [[ "${GH_TOKEN:-}" == app-fixture ]] || exit 1
    printf 'PATCH\n' >> "$MOCK_ROOT/api"
    [[ "${APPLY_ERROR:-}" != 1 ]] || { echo 'API error' >&2; exit 1; }
    printf 'Applied repository settings.\n' ;;
  *) exit 1 ;;
esac
MOCK
cat > "$fixture/bin/curl" <<'MOCK'
#!/usr/bin/env bash
set -euo pipefail
while [[ $# -gt 0 ]]; do
  if [[ "$1" == --output ]]; then
    cp "$MOCK_ASSET" "$2"
    [[ "${CORRUPT_DOWNLOAD:-}" != 1 ]] || printf 'tampered\n' >> "$2"
    exit
  fi
  shift
done
exit 1
MOCK
chmod +x "$fixture/bin/curl"
export PATH="$fixture/bin:$PATH"
installer="$fixture/repo/examples/github-actions/install-cli.sh"
sed "s/^checksum=.*/checksum=$(sha256sum "$MOCK_ASSET" | cut -d ' ' -f 1)/" "$installer" > "$fixture/installer"
cp "$fixture/installer" "$installer"
cd "$fixture/repo"
printf 'schemaVersion: 1\nsettings:\n  repository:\n    has_issues: false\n' > .github/settings.yml
git init -q
git remote add origin https://github.com/owner/repo
git add .
git -c commit.gpgsign=false -c user.name=Fixture -c user.email=fixture@example.invalid commit -qm 'test: initial fixture'
GITHUB_SHA="$(git rev-parse HEAD)"
export GITHUB_SHA GITHUB_REPOSITORY=owner/repo
export GITHUB_EVENT_NAME=push GITHUB_REF=refs/heads/main
export GITHUB_ACTOR=owner GITHUB_TRIGGERING_ACTOR=maintainer
export GITHUB_RUN_ID=123 GITHUB_RUN_ATTEMPT=2
export GITHUB_WORKFLOW_REF=owner/repo/.github/workflows/settings-main.yml@refs/heads/main
export GITHUB_WORKFLOW_SHA=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
export RUNNER_TEMP="$fixture/runner" GITHUB_OUTPUT="$fixture/output" GITHUB_STEP_SUMMARY="$fixture/summary"
mkdir -p "$RUNNER_TEMP"

# Apply jobs start from committed input on a separate fresh runner/worktree.
fresh_apply_checkout() {
  git clone --quiet --no-local "$fixture/repo" "$fixture/apply-repo"
  cd "$fixture/apply-repo"
  git checkout -q "$CONFIG_SHA"
  git remote set-url origin https://github.com/owner/repo
}
