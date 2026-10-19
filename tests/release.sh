#!/usr/bin/env bash
set -euo pipefail
repo_root="$(cd "$(dirname "$0")/.." && pwd)"
if ! command -v sha256sum > /dev/null; then
  sha256sum() { shasum -a 256 "$@"; }
  export -f sha256sum
fi
fixture="$(mktemp -d)"
trap 'rm -rf "$fixture"' EXIT
mkdir -p "$fixture/bin" "$fixture/assets" "$fixture/release"
cat > "$fixture/bin/gh" <<'MOCK'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >> "$MOCK_ROOT/calls"
release_json() {
  jq -n --arg tag "$VERSION" --rawfile body "$MOCK_ROOT/body" --rawfile draft "$MOCK_ROOT/draft" --argjson assets "$(find "$MOCK_ROOT/assets" -type f -exec basename {} \; | jq -R . | jq -s .)" '{id:123,tag_name:$tag,name:$tag,prerelease:false,draft:($draft == "true\n"),body:$body,assets:[$assets[] | {name:.}]}'
}
if [[ "$1" == api ]]; then
  shift
  if [[ "$1" == -X ]]; then
    case "$2:$3" in
      POST:*/git/refs)
        printf '{"object":{"type":"commit","sha":"%s"}}\n' "$SOURCE_SHA" > "$MOCK_ROOT/tag.json" ;;
      POST:*/releases)
        while [[ $# -gt 0 ]]; do
          if [[ "$1" == --input ]]; then
            jq -r .body "$2" > "$MOCK_ROOT/body"
            break
          fi
          shift
        done
        printf 'true\n' > "$MOCK_ROOT/draft"
        release_json ;;
      PATCH:*/releases/123)
        printf 'false\n' > "$MOCK_ROOT/draft"
        release_json ;;
      *) echo "unexpected mutation: $*" >&2; exit 1 ;;
    esac
    exit 0
  fi
  case "$1" in
    */git/ref/tags/*)
      [[ -f "$MOCK_ROOT/tag.json" ]] || { echo 'gh: Not Found (HTTP 404)' >&2; exit 1; }
      if [[ "${MOCK_TAG_MOVE_ON_SECOND_READ:-}" == 1 ]]; then
        reads=0
        [[ ! -f "$MOCK_ROOT/tag-reads" ]] || reads="$(cat "$MOCK_ROOT/tag-reads")"
        printf '%s\n' "$((reads + 1))" > "$MOCK_ROOT/tag-reads"
        if [[ "$reads" == 1 ]]; then
          printf '{"object":{"type":"commit","sha":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}}\n'
          exit 0
        fi
      fi
      cat "$MOCK_ROOT/tag.json" ;;
    */git/tags/*)
      [[ -f "$MOCK_ROOT/annotated.json" ]] || { echo 'gh: Not Found (HTTP 404)' >&2; exit 1; }
      cat "$MOCK_ROOT/annotated.json" ;;
    */releases/tags/*)
      [[ -f "$MOCK_ROOT/body" ]] || { echo 'gh: Not Found (HTTP 404)' >&2; exit 1; }
      [[ "$(cat "$MOCK_ROOT/draft")" == false ]] || { echo 'gh: Not Found (HTTP 404)' >&2; exit 1; }
      release_json ;;
    */releases\?per_page=100)
      if [[ -f "$MOCK_ROOT/body" ]]; then
        release_json > "$MOCK_ROOT/current.json"
        jq --slurpfile current "$MOCK_ROOT/current.json" '.[0] += $current' "$MOCK_ROOT/catalog.json"
      else
        cat "$MOCK_ROOT/catalog.json"
      fi ;;
    */releases/123)
      [[ -f "$MOCK_ROOT/body" ]] || { echo 'gh: Not Found (HTTP 404)' >&2; exit 1; }
      release_json ;;
    *) echo "unexpected api: $1" >&2; exit 1 ;;
  esac
  exit 0
fi
[[ "$1" == release ]] || exit 1
case "$2" in
  create)
    while [[ $# -gt 0 ]]; do
      if [[ "$1" == --notes-file ]]; then cp "$2" "$MOCK_ROOT/body"; break; fi
      shift
    done
    printf 'true\n' > "$MOCK_ROOT/draft" ;;
  upload)
    cp "$4" "$MOCK_ROOT/assets/$(basename "$4")" ;;
  download)
    pattern='' destination=''
    while [[ $# -gt 0 ]]; do
      case "$1" in
        --pattern) pattern="$2"; shift 2 ;;
        --dir) destination="$2"; shift 2 ;;
        *) shift ;;
      esac
    done
    [[ ! -e "$destination/$pattern" ]] || { echo "asset already exists at $destination/$pattern" >&2; exit 1; }
    cp "$MOCK_ROOT/assets/$pattern" "$destination/$pattern" ;;
  *) echo "unexpected release operation: $2" >&2; exit 1 ;;
esac
MOCK
chmod +x "$fixture/bin/gh"
export MOCK_ROOT="$fixture" PATH="$fixture/bin:$PATH" GITHUB_REPOSITORY=owner/repo
export VERSION=v1.2.3 RELEASE_DIR="$fixture/release"
printf 'macos-binary' > "$RELEASE_DIR/gh-settings-darwin-arm64"
printf 'linux-binary' > "$RELEASE_DIR/gh-settings-linux-amd64"
MACOS_SHA256="$(sha256sum "$RELEASE_DIR/gh-settings-darwin-arm64" | cut -d ' ' -f 1)"
LINUX_SHA256="$(sha256sum "$RELEASE_DIR/gh-settings-linux-amd64" | cut -d ' ' -f 1)"
export MACOS_SHA256 LINUX_SHA256
mkdir -p "$fixture/source/scripts/release-notes" "$fixture/source/action"
cp "$repo_root/scripts/publish-release.sh" "$fixture/source/scripts/publish-release.sh"
cp "$repo_root/scripts/release-notes/main.go" "$fixture/source/scripts/release-notes/main.go"
printf 'v1.2.3\n' > "$fixture/source/action/cli-version"
git -C "$fixture/source" init --quiet
git -C "$fixture/source" config user.name Fixture
git -C "$fixture/source" config user.email fixture@example.invalid
git -C "$fixture/source" config commit.gpgsign false
git -C "$fixture/source" commit --quiet --allow-empty -m 'chore: initialize release fixture'
git -C "$fixture/source" -c tag.gpgsign=false tag v1.2.2
printf 'feat!: update the test fixture\n\nBREAKING CHANGE: Migrate the previous fixture.\n' > "$fixture/message"
if [[ "${MOCK_LONG_NOTES:-1}" == 1 ]]; then
  head -c 262144 /dev/zero | tr '\0' x >> "$fixture/message"
  printf '\n' >> "$fixture/message"
fi
git -C "$fixture/source" commit --quiet --allow-empty -F "$fixture/message"
SOURCE_SHA="$(git -C "$fixture/source" rev-parse HEAD)"
export SOURCE_SHA
git -C "$fixture/source" commit --quiet --allow-empty -m 'feat: do not include later HEAD'
printf '[[{"id":120,"tag_name":"v1.2.2","draft":false,"prerelease":false,"published_at":"2026-10-01T00:00:00Z"}]]\n' > "$fixture/catalog.json"
publish() { bash "$fixture/source/scripts/publish-release.sh"; }
reject() {
  if publish > "$fixture/output" 2>&1; then
    cat "$fixture/output" >&2
    echo 'expected release rejection' >&2
    exit 1
  fi
}

# The source version must agree before any GitHub lookup or mutation.
printf 'v1.2.4\n' > "$fixture/source/action/cli-version"
reject
[[ ! -e "$fixture/calls" ]]
grep -q 'committed CLI version' "$fixture/output"
printf 'v1.2.3\n' > "$fixture/source/action/cli-version"

# Incomplete predecessor history must fail before creating a remote tag or draft.
mv "$fixture/catalog.json" "$fixture/catalog.saved"
jq '.[0][0].tag_name = "v1.2.1"' "$fixture/catalog.saved" > "$fixture/catalog.json"
reject
grep -q 'resolve previous release v1.2.1' "$fixture/output"
if grep -q '^api -X ' "$fixture/calls"; then
  echo 'history failure created remote state' >&2
  exit 1
fi
[[ ! -e "$fixture/tag.json" && ! -e "$fixture/body" ]]
mv "$fixture/catalog.saved" "$fixture/catalog.json"

# The tag endpoint hides drafts in this mock, as it does in GitHub's REST API.
# First run must use the created draft ID, then publish its three assets.
publish
[[ "$(cat "$fixture/draft")" == false ]]
[[ "$(find "$fixture/assets" -type f | wc -l | tr -d ' ')" == 3 ]]
[[ "$(grep -c '^release upload ' "$fixture/calls")" == 3 ]]
grep -q '^## 🚨 BREAKING CHANGE$' "$fixture/body"
grep -q 'Migrate the previous fixture' "$fixture/body"
if grep -q 'do not include later HEAD\|initialize release fixture\|generate-notes' "$fixture/body" "$fixture/calls"; then
  echo 'release notes used history outside the frozen range or GitHub generation' >&2
  exit 1
fi
[[ "$(grep -n '^## 🚨' "$fixture/body" | cut -d: -f1)" -lt "$(grep -n '^## 📦' "$fixture/body" | cut -d: -f1)" ]]
if [[ "${MOCK_LONG_NOTES:-1}" == 1 ]]; then
  [[ "$(wc -c < "$fixture/body")" -gt 262144 ]]
fi
cp "$fixture/body" "$fixture/original-notes"

# Later publications cannot change notes or make an existing release regenerate.
# The intentionally unavailable new predecessor would fail first-time generation.
printf '[[{"id":999,"tag_name":"v9.0.0","draft":false,"prerelease":false,"published_at":"2026-10-02T00:00:00Z"}]]\n' > "$fixture/catalog.json"

# A completed run is an identity-checked no-op.
publish
[[ "$(grep -c '^release upload ' "$fixture/calls")" == 3 ]]
cmp "$fixture/original-notes" "$fixture/body"

# Interrupted draft: retain matching assets and upload only the absent one.
printf 'true\n' > "$fixture/draft"
rm "$fixture/assets/SHA256SUMS"
publish
[[ "$(grep -c '^release upload ' "$fixture/calls")" == 4 ]]
[[ "$(cat "$fixture/draft")" == false ]]
cmp "$fixture/original-notes" "$fixture/body"

# Version, tag target, draft manifest, and bytes each reject conflicts.
rm "$fixture/tag.json"
SOURCE_SHA=bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb reject
[[ ! -e "$fixture/tag.json" ]] || { echo 'conflicting release recreated a missing tag' >&2; exit 1; }
printf '{"object":{"type":"commit","sha":"%s"}}\n' "$SOURCE_SHA" > "$fixture/tag.json"
VERSION='v1.2.3;unsafe' reject
export VERSION=v1.2.3
printf '{"object":{"type":"commit","sha":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}}\n' > "$fixture/tag.json"
reject
printf '{"object":{"type":"commit","sha":"%s"}}\n' "$SOURCE_SHA" > "$fixture/tag.json"
cp "$fixture/body" "$fixture/body.saved"
printf 'other release\n' > "$fixture/body"
reject
cp "$fixture/body.saved" "$fixture/body"
printf 'changed bytes' > "$fixture/assets/gh-settings-linux-amd64"
reject

# A tag moved after initial inspection must still block final publication.
cp "$RELEASE_DIR/gh-settings-linux-amd64" "$fixture/assets/gh-settings-linux-amd64"
printf 'true\n' > "$fixture/draft"
previous_publishes="$(grep -c '^api -X PATCH ' "$fixture/calls")"
export MOCK_TAG_MOVE_ON_SECOND_READ=1
reject
unset MOCK_TAG_MOVE_ON_SECOND_READ
[[ "$(cat "$fixture/tag-reads")" == 2 ]]
grep -q 'tag changed before publication' "$fixture/output"
[[ "$(cat "$fixture/draft")" == true ]]
[[ "$(grep -c '^api -X PATCH ' "$fixture/calls")" == "$previous_publishes" ]]

# An annotated tag is resolved afresh at both checkpoints.
printf '{"object":{"type":"tag","sha":"cccccccccccccccccccccccccccccccccccccccc"}}\n' > "$fixture/tag.json"
printf '{"object":{"type":"commit","sha":"%s"}}\n' "$SOURCE_SHA" > "$fixture/annotated.json"
previous_resolutions="$(grep -c '^api repos/owner/repo/git/tags/' "$fixture/calls" || true)"
publish
[[ "$(cat "$fixture/draft")" == false ]]
[[ "$(grep -c '^api repos/owner/repo/git/tags/' "$fixture/calls")" == "$((previous_resolutions + 2))" ]]
echo 'release mock tests passed'
