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
  jq -n --arg tag "$VERSION" --arg invalid_draft "${MOCK_INVALID_DRAFT:-}" --arg duplicate_assets "${MOCK_DUPLICATE_ASSET:-}" --rawfile body "$MOCK_ROOT/body" --rawfile draft "$MOCK_ROOT/draft" --argjson assets "$(find "$MOCK_ROOT/assets" -type f -exec basename {} \; | jq -R . | jq -s .)" '{id:123,tag_name:$tag,name:$tag,prerelease:false,draft:(if $invalid_draft == "1" then "unknown" else ($draft == "true\n") end),body:$body,assets:([$assets[] | {name:.}] | if $duplicate_assets == "1" then . + [.[0]] else . end)}'
}
if [[ "$1" == api ]]; then
  shift
  if [[ "$1" == -X ]]; then
    case "$2:$3" in
      POST:*/releases/generate-notes)
        [[ $# == 11 && "$3" == "repos/$GITHUB_REPOSITORY/releases/generate-notes" && "$4" == -f && "$5" == "tag_name=$VERSION" && "$6" == -f && "$7" == "target_commitish=$SOURCE_SHA" && "$8" == -f && "$9" == configuration_file_path=.github/release.yml && "${10}" == -f && "${11}" == previous_tag_name=v1.2.2 ]] || {
          echo 'unexpected release-note generation request' >&2; exit 1
        }
        [[ "${MOCK_GENERATION_FAILURE:-}" != 1 ]] || { echo 'fixture generation failed' >&2; exit 1; }
        if [[ "${MOCK_INVALID_NOTES:-}" == 1 ]]; then
          printf '{"body":null}\n'
        else
          jq -n --rawfile body "$MOCK_ROOT/generated-notes" '{body:$body}'
        fi ;;
      POST:*/git/refs)
        [[ $# == 7 && "$3" == "repos/$GITHUB_REPOSITORY/git/refs" && "$4" == -f && "$5" == "ref=refs/tags/$VERSION" && "$6" == -f && "$7" == "sha=$SOURCE_SHA" ]] || {
          echo 'unexpected tag creation request' >&2; exit 1
        }
        printf '{"object":{"type":"commit","sha":"%s"}}\n' "${7#sha=}" > "$MOCK_ROOT/tag.json" ;;
      POST:*/releases)
        [[ $# == 5 && "$3" == "repos/$GITHUB_REPOSITORY/releases" && "$4" == --input ]] || {
          echo 'unexpected release creation arguments' >&2; exit 1
        }
        # The independent complete-notes golden below owns the body bytes.
        jq -e --arg tag "$VERSION" --arg commit "$SOURCE_SHA" '
          . == {tag_name:$tag,target_commitish:$commit,name:$tag,body:.body,draft:true,prerelease:false}
          and (.body | type == "string")
        ' "$5" > /dev/null || { echo 'unexpected release creation payload' >&2; exit 1; }
        jq -r .body "$5" > "$MOCK_ROOT/body"
        jq -r .draft "$5" > "$MOCK_ROOT/draft"
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
      if [[ "${MOCK_TAG_AUTH_FAILURE:-}" == 1 ]]; then
        echo 'gh: Bad credentials (HTTP 401)' >&2
        exit 1
      fi
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
    asset_name="$(basename "$4")"
    [[ "${MOCK_DROP_UPLOAD:-}" != "$asset_name" ]] || exit 0
    cp "$4" "$MOCK_ROOT/assets/$asset_name"
    if [[ "${MOCK_CORRUPT_UPLOAD:-}" == "$asset_name" ]]; then
      printf 'changed uploaded bytes' > "$MOCK_ROOT/assets/$asset_name"
    fi ;;

  download)
    pattern='' destination=''
    while [[ $# -gt 0 ]]; do
      case "$1" in
        --pattern) pattern="$2"; shift 2 ;;
        --dir) destination="$2"; shift 2 ;;
        *) shift ;;
      esac
    done
    if [[ "${MOCK_DOWNLOAD_FAILURE:-}" == "$pattern" ]]; then
      echo 'fixture download failed' >&2
      exit 1
    fi
    [[ ! -e "$destination/$pattern" ]] || { echo "asset already exists at $destination/$pattern" >&2; exit 1; }
    cp "$MOCK_ROOT/assets/$pattern" "$destination/$pattern" ;;
  *) echo "unexpected release operation: $2" >&2; exit 1 ;;
esac
MOCK
chmod +x "$fixture/bin/gh"
export MOCK_ROOT="$fixture" PATH="$fixture/bin:$PATH" GITHUB_REPOSITORY=owner/repo
export VERSION=v1.2.3 PREVIOUS_TAG=v1.2.2 RELEASE_DIR="$fixture/release"
printf 'macos-binary' > "$RELEASE_DIR/gh-settings-darwin-arm64"
printf 'linux-binary' > "$RELEASE_DIR/gh-settings-linux-amd64"
MACOS_SHA256="$(sha256sum "$RELEASE_DIR/gh-settings-darwin-arm64" | cut -d ' ' -f 1)"
LINUX_SHA256="$(sha256sum "$RELEASE_DIR/gh-settings-linux-amd64" | cut -d ' ' -f 1)"
export MACOS_SHA256 LINUX_SHA256
mkdir -p "$fixture/source/scripts/release-metadata" "$fixture/source/scripts/release-publisher" "$fixture/source/action"
cp "$repo_root/scripts/publish-release.sh" "$fixture/source/scripts/publish-release.sh"
cp "$repo_root/scripts/release-publisher/main.go" "$fixture/source/scripts/release-publisher/main.go"
cp "$repo_root/scripts/release-metadata/main.go" "$fixture/source/scripts/release-metadata/main.go"
printf 'v1.2.3\n' > "$fixture/source/action/cli-version"
mkdir -p "$fixture/source/.github"
cp "$repo_root/.github/release.yml" "$fixture/source/.github/release.yml"
printf '## 🚨 BREAKING CHANGE\n\n* GitHub fixture PR #1\n' > "$fixture/generated-notes"
git -C "$fixture/source" init --quiet
git -C "$fixture/source" config user.name Fixture
git -C "$fixture/source" config user.email fixture@example.invalid
git -C "$fixture/source" config commit.gpgsign false
git -C "$fixture/source" add .github/release.yml action/cli-version
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
PREVIOUS_SHA="$(git -C "$fixture/source" rev-parse 'v1.2.2^{commit}')"
export PREVIOUS_SHA
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

# Every preflight failure must stop before even a GitHub lookup.
VERSION='v1.2.3;unsafe' reject
export VERSION=v1.2.3
SOURCE_SHA=invalid reject
[[ "$(cat "$fixture/output")" == 'release: invalid source commit' ]]
LINUX_SHA256=invalid reject
MACOS_SHA256=invalid reject
GITHUB_REPOSITORY=owner reject
RELEASE_DIR="$fixture/absent" reject
for asset in gh-settings-linux-amd64 gh-settings-darwin-arm64; do
  cp "$RELEASE_DIR/$asset" "$fixture/$asset.saved"
  rm "$RELEASE_DIR/$asset"
  reject
  printf 'changed bytes' > "$RELEASE_DIR/$asset"
  reject
  mv "$fixture/$asset.saved" "$RELEASE_DIR/$asset"
done
[[ ! -e "$fixture/calls" && ! -e "$RELEASE_DIR/SHA256SUMS" ]]

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

# Note generation failure cannot leave a tag, draft, or uploaded asset.
MOCK_GENERATION_FAILURE=1 reject
grep -q 'GitHub release-note generation failed' "$fixture/output"
MOCK_INVALID_NOTES=1 reject
grep -q 'GitHub returned no release-note body' "$fixture/output"
[[ ! -e "$fixture/tag.json" && ! -e "$fixture/body" ]]
[[ "$(find "$fixture/assets" -type f | wc -l | tr -d ' ')" == 0 ]]

# The tag endpoint hides drafts in this mock, as it does in GitHub's REST API.
# First run must use the created draft ID, then publish its three assets.
publish
# Freeze the exact legacy checksum order, separators, newlines, and marker bytes.
printf '69e398f9f4479aba0fcc353dc9acb6819c0a7d260f6e104ca65aae7ac2c21b85  gh-settings-darwin-arm64\n8e05650e3597d536838387f4fb4f08fbd95624760f1ce44bbff4e35de8a353e8  gh-settings-linux-amd64\n' > "$fixture/expected-checksums"
cmp "$fixture/expected-checksums" "$RELEASE_DIR/SHA256SUMS"
expected_manifest="{\"version\":\"v1.2.3\",\"commit\":\"$SOURCE_SHA\",\"macos_sha256\":\"69e398f9f4479aba0fcc353dc9acb6819c0a7d260f6e104ca65aae7ac2c21b85\",\"linux_sha256\":\"8e05650e3597d536838387f4fb4f08fbd95624760f1ce44bbff4e35de8a353e8\",\"checksums_sha256\":\"98f9ece90dea10c02544983c6f8bc3e3ccef162fa20869eecebe1df30b297abe\"}"
IFS= read -r actual_marker < "$fixture/body"
[[ "$actual_marker" == "<!-- gh-settings-release: $expected_manifest -->" ]]
# Independently assemble the native response plus the retained migration supplement.
{
  cat "$fixture/generated-notes"
  printf '\n## ⚠️ Migration guidance\n\n- [update the test fixture](https://github.com/%s/commit/%s)\n\n  > Migrate the previous fixture.\n' "$GITHUB_REPOSITORY" "$SOURCE_SHA"
  if [[ "${MOCK_LONG_NOTES:-1}" == 1 ]]; then
    printf '  > '
    head -c 262144 /dev/zero | tr '\0' x
    printf '\n'
  fi
  previous_sha="$(git -C "$fixture/source" rev-parse 'v1.2.2^{commit}')"
  printf '\n**Frozen comparison**: [v1.2.2 → %s](https://github.com/%s/compare/%s...%s)\n' "$VERSION" "$GITHUB_REPOSITORY" "$previous_sha" "$SOURCE_SHA"
} > "$fixture/expected-changes"
marker="<!-- gh-settings-release: $expected_manifest -->"
repo="$GITHUB_REPOSITORY"
macos=gh-settings-darwin-arm64
linux=gh-settings-linux-amd64
checksums=SHA256SUMS
cat > "$fixture/expected-notes" <<EOF
$marker

$(cat "$fixture/expected-changes")

---

## 📦 Installation

\`\`\`sh
gh extension install $repo
\`\`\`

Already installed? Run \`gh extension upgrade ${repo##*/}\`.

Or download \`$macos\` (macOS Apple Silicon) or \`$linux\` (Linux x86_64) below and verify it with \`SHA256SUMS\`.

<details>
<summary>🔐 SHA-256 checksums</summary>

\`\`\`text
$(cat "$RELEASE_DIR/$checksums")
\`\`\`

</details>

Built from main commit [\`${SOURCE_SHA:0:7}\`](https://github.com/$repo/commit/$SOURCE_SHA).
EOF
# The local gh stand-in writes decoded body with jq -r, adding one newline.
printf '\n' >> "$fixture/expected-notes"
cmp "$fixture/expected-notes" "$fixture/body"
[[ "$(cat "$fixture/draft")" == false ]]
[[ "$(find "$fixture/assets" -type f | wc -l | tr -d ' ')" == 3 ]]
[[ "$(grep -c '^release upload ' "$fixture/calls")" == 3 ]]
grep -q '^## 🚨 BREAKING CHANGE$' "$fixture/body"
grep -q 'Migrate the previous fixture' "$fixture/body"
if grep -q 'do not include later HEAD\|initialize release fixture' "$fixture/body" "$fixture/calls"; then
  echo 'release notes used history outside the frozen range' >&2
  exit 1
fi
[[ "$(grep -n '^## 🚨' "$fixture/body" | cut -d: -f1)" -lt "$(grep -n '^## 📦' "$fixture/body" | cut -d: -f1)" ]]
if [[ "${MOCK_LONG_NOTES:-1}" == 1 ]]; then
  [[ "$(wc -c < "$fixture/body")" -gt 262144 ]]
fi
cp "$fixture/body" "$fixture/original-notes"
generations="$(grep -c '^api -X POST repos/owner/repo/releases/generate-notes ' "$fixture/calls")"

# Later publications cannot change notes or make an existing release regenerate.
# The intentionally unavailable new predecessor would fail first-time generation.
printf '[[{"id":999,"tag_name":"v9.0.0","draft":false,"prerelease":false,"published_at":"2026-10-02T00:00:00Z"}]]\n' > "$fixture/catalog.json"

# A completed run is an identity-checked no-op.
publish
[[ "$(grep -c '^release upload ' "$fixture/calls")" == 3 ]]
cmp "$fixture/original-notes" "$fixture/body"

# A published release missing an asset is immutable, not a draft to repair.
previous_uploads="$(grep -c '^release upload ' "$fixture/calls")"
previous_mutations="$(grep -c '^api -X ' "$fixture/calls")"
rm "$fixture/assets/SHA256SUMS"
reject
grep -q 'published release is missing assets' "$fixture/output"
[[ "$(grep -c '^release upload ' "$fixture/calls")" == "$previous_uploads" ]]
[[ "$(grep -c '^api -X ' "$fixture/calls")" == "$previous_mutations" ]]
[[ "$(cat "$fixture/draft")" == false ]]
cp "$RELEASE_DIR/SHA256SUMS" "$fixture/assets/SHA256SUMS"

# Interrupted draft: retain matching assets and upload only the absent one.
printf 'true\n' > "$fixture/draft"
rm "$fixture/assets/SHA256SUMS"
publish
[[ "$(grep -c '^release upload ' "$fixture/calls")" == 4 ]]
[[ "$(cat "$fixture/draft")" == false ]]
cmp "$fixture/original-notes" "$fixture/body"

# Reject all name conflicts before downloading any existing bytes.
previous_downloads="$(grep -c '^release download ' "$fixture/calls")"
previous_uploads="$(grep -c '^release upload ' "$fixture/calls")"
previous_mutations="$(grep -c '^api -X ' "$fixture/calls")"
MOCK_DUPLICATE_ASSET=1 reject
grep -q 'duplicate asset names' "$fixture/output"
printf 'unexpected' > "$fixture/assets/unexpected"
reject
grep -q 'unexpected release asset' "$fixture/output"
rm "$fixture/assets/unexpected"
[[ "$(grep -c '^release download ' "$fixture/calls")" == "$previous_downloads" ]]
MOCK_DOWNLOAD_FAILURE=gh-settings-linux-amd64 reject
grep -q 'fixture download failed' "$fixture/output"
[[ "$(grep -c '^release download ' "$fixture/calls")" == "$((previous_downloads + 2))" ]]
[[ "$(grep -c '^release upload ' "$fixture/calls")" == "$previous_uploads" ]]
[[ "$(grep -c '^api -X ' "$fixture/calls")" == "$previous_mutations" ]]

# Draft state is checked only after tag processing, and before asset reads.
previous_mutations="$(grep -c '^api -X ' "$fixture/calls")"
previous_downloads="$(grep -c '^release download ' "$fixture/calls")"
previous_tag_reads="$(grep -c '^api repos/owner/repo/git/ref/tags/' "$fixture/calls")"
mv "$fixture/tag.json" "$fixture/tag.saved"
MOCK_INVALID_DRAFT=1 reject
grep -q 'existing release has no tag' "$fixture/output"
[[ ! -e "$fixture/tag.json" ]]
mv "$fixture/tag.saved" "$fixture/tag.json"
MOCK_INVALID_DRAFT=1 reject
grep -q 'unknown release state' "$fixture/output"
[[ "$(grep -c '^api repos/owner/repo/git/ref/tags/' "$fixture/calls")" == "$((previous_tag_reads + 2))" ]]
[[ "$(grep -c '^release download ' "$fixture/calls")" == "$previous_downloads" ]]

# Authentication/transport errors must not be mistaken for an absent tag.
MOCK_TAG_AUTH_FAILURE=1 reject
grep -q 'HTTP 401' "$fixture/output"
grep -q 'GitHub API lookup failed: repos/owner/repo/git/ref/tags/v1.2.3' "$fixture/output"
[[ "$(grep -c '^api -X ' "$fixture/calls")" == "$previous_mutations" ]]
cmp "$fixture/original-notes" "$fixture/body"

# A successful upload response is insufficient: verify all bytes again.
printf 'true\n' > "$fixture/draft"
rm "$fixture/assets/SHA256SUMS"
previous_publishes="$(grep -c '^api -X PATCH ' "$fixture/calls")"
MOCK_DROP_UPLOAD=SHA256SUMS reject
grep -q 'draft is missing assets' "$fixture/output"
[[ "$(cat "$fixture/draft")" == true ]]
[[ "$(grep -c '^api -X PATCH ' "$fixture/calls")" == "$previous_publishes" ]]
publish
cmp "$fixture/original-notes" "$fixture/body"

printf 'true\n' > "$fixture/draft"
rm "$fixture/assets/SHA256SUMS"
previous_publishes="$(grep -c '^api -X PATCH ' "$fixture/calls")"
MOCK_CORRUPT_UPLOAD=SHA256SUMS reject
grep -q 'artifact hash mismatch: SHA256SUMS' "$fixture/output"
[[ "$(cat "$fixture/draft")" == true ]]
[[ "$(grep -c '^api -X PATCH ' "$fixture/calls")" == "$previous_publishes" ]]
previous_uploads="$(grep -c '^release upload ' "$fixture/calls")"
reject
grep -q 'artifact hash mismatch: SHA256SUMS' "$fixture/output"
[[ "$(grep -c '^release upload ' "$fixture/calls")" == "$previous_uploads" ]]
[[ "$(grep -c '^api -X PATCH ' "$fixture/calls")" == "$previous_publishes" ]]
# Reset only the owned local fixture; the publisher must not replace bad bytes.
cp "$RELEASE_DIR/SHA256SUMS" "$fixture/assets/SHA256SUMS"
printf 'false\n' > "$fixture/draft"
cmp "$fixture/original-notes" "$fixture/body"

# Version, tag target, draft manifest, and bytes each reject conflicts.
rm "$fixture/tag.json"
SOURCE_SHA=bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb reject
[[ ! -e "$fixture/tag.json" ]] || { echo 'conflicting release recreated a missing tag' >&2; exit 1; }
printf '{"object":{"type":"commit","sha":"%s"}}\n' "$SOURCE_SHA" > "$fixture/tag.json"
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
[[ "$(grep -c '^api -X POST repos/owner/repo/releases/generate-notes ' "$fixture/calls")" == "$generations" ]]
echo 'release mock tests passed'
