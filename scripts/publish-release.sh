#!/usr/bin/env bash
set -euo pipefail

fail() { printf 'release: %s\n' "$*" >&2; exit 1; }
digest() { sha256sum "$1" | cut -d ' ' -f 1; }
optional_api() {
  local endpoint="$1" destination="$2" error_file="$work/api-error"
  if gh api "$endpoint" > "$destination" 2> "$error_file"; then
    return 0
  fi
  if grep -q 'HTTP 404' "$error_file"; then
    return 1
  fi
  cat "$error_file" >&2
  fail "GitHub API lookup failed: $endpoint"
}
check_hash() {
  local path="$1" expected="$2"
  [[ -f "$path" && "$(digest "$path")" == "$expected" ]] || fail "artifact hash mismatch: $(basename "$path")"
}

[[ "${VERSION:-}" =~ ^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]] || fail 'version must be vMAJOR.MINOR.PATCH'
[[ "${SOURCE_SHA:-}" =~ ^[0-9a-f]{40}$ ]] || fail 'invalid source commit'
[[ "${LINUX_SHA256:-}" =~ ^[0-9a-f]{64}$ && "${MACOS_SHA256:-}" =~ ^[0-9a-f]{64}$ ]] || fail 'invalid expected binary hash'
[[ "${GITHUB_REPOSITORY:-}" == */* ]] || fail 'missing repository'
[[ -d "${RELEASE_DIR:-}" ]] || fail 'missing release directory'

work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT
linux=gh-settings-linux-amd64
macos=gh-settings-darwin-arm64
checksums=SHA256SUMS
check_hash "$RELEASE_DIR/$linux" "$LINUX_SHA256"
check_hash "$RELEASE_DIR/$macos" "$MACOS_SHA256"
(
  cd "$RELEASE_DIR"
  sha256sum "$macos" "$linux" > "$checksums"
)
checksum_sha="$(digest "$RELEASE_DIR/$checksums")"
manifest="$(jq -cn --arg v "$VERSION" --arg c "$SOURCE_SHA" --arg m "$MACOS_SHA256" --arg l "$LINUX_SHA256" --arg s "$checksum_sha" '{version:$v,commit:$c,macos_sha256:$m,linux_sha256:$l,checksums_sha256:$s}')"
marker="<!-- gh-settings-release: $manifest -->"
repo="$GITHUB_REPOSITORY"

# Resolve both lightweight and annotated tags without moving either.
if optional_api "repos/$repo/git/ref/tags/$VERSION" "$work/tag.json"; then
  tag_type="$(jq -r '.object.type' "$work/tag.json")"
  tag_sha="$(jq -r '.object.sha' "$work/tag.json")"
  if [[ "$tag_type" == tag ]]; then
    gh api "repos/$repo/git/tags/$tag_sha" > "$work/annotated.json"
    tag_type="$(jq -r '.object.type' "$work/annotated.json")"
    tag_sha="$(jq -r '.object.sha' "$work/annotated.json")"
  fi
  [[ "$tag_type" == commit && "$tag_sha" == "$SOURCE_SHA" ]] || fail 'existing tag targets another commit'
else
  gh api -X POST "repos/$repo/git/refs" -f "ref=refs/tags/$VERSION" -f "sha=$SOURCE_SHA" > /dev/null
fi

if optional_api "repos/$repo/releases/tags/$VERSION" "$work/release.json"; then
  [[ "$(jq -r '.tag_name' "$work/release.json")" == "$VERSION" ]] || fail 'existing release has another tag'
  [[ "$(jq -r '.name' "$work/release.json")" == "$VERSION" ]] || fail 'existing release has another title'
  [[ "$(jq -r '.prerelease' "$work/release.json")" == false ]] || fail 'existing release is a prerelease'
  jq -r '.body' "$work/release.json" | grep -Fxq -- "$marker" || fail 'existing release lacks matching version, commit, and binary identity evidence'
else
  gh api -X POST "repos/$repo/releases/generate-notes" -f "tag_name=$VERSION" -f "target_commitish=$SOURCE_SHA" --jq .body > "$work/changes.md"
  cat > "$work/notes.md" <<EOF
$marker

Release $VERSION from main commit \`$SOURCE_SHA\`.

Install with \`gh extension install $repo\`, or download the raw executable matching your platform. The assets are \`$macos\` and \`$linux\`; verify them with \`SHA256SUMS\` before use.

## SHA-256

\`\`\`text
$(cat "$RELEASE_DIR/$checksums")
\`\`\`

## Changes

$(cat "$work/changes.md")
EOF
  gh release create "$VERSION" --repo "$repo" --verify-tag --draft --title "$VERSION" --notes-file "$work/notes.md" > /dev/null
  gh api "repos/$repo/releases/tags/$VERSION" > "$work/release.json"
fi
[[ "$(jq -r '.draft' "$work/release.json")" == true || "$(jq -r '.draft' "$work/release.json")" == false ]] || fail 'unknown release state'
jq -r '.body' "$work/release.json" | grep -Fxq -- "$marker" || fail 'release notes do not match tested identity'

verify_assets() {
  local name expected
  gh api "repos/$repo/releases/tags/$VERSION" > "$work/release.json"
  [[ "$(jq -r '[.assets[].name] | unique | length' "$work/release.json")" == "$(jq -r '.assets | length' "$work/release.json")" ]] || fail 'duplicate asset names'
  jq -e --arg m "$macos" --arg l "$linux" --arg s "$checksums" 'all(.assets[].name; . == $m or . == $l or . == $s)' "$work/release.json" > /dev/null || fail 'unexpected release asset'
  mkdir -p "$work/downloads"
  for name in "$macos" "$linux" "$checksums"; do
    case "$name" in
      "$macos") expected="$MACOS_SHA256" ;;
      "$linux") expected="$LINUX_SHA256" ;;
      *) expected="$checksum_sha" ;;
    esac
    if jq -e --arg name "$name" '.assets[] | select(.name == $name)' "$work/release.json" > /dev/null; then
      gh release download "$VERSION" --repo "$repo" --pattern "$name" --dir "$work/downloads" > /dev/null
      check_hash "$work/downloads/$name" "$expected"
    fi
  done
}

verify_assets
if [[ "$(jq -r '.draft' "$work/release.json")" == false ]]; then
  [[ "$(jq -r '.assets | length' "$work/release.json")" == 3 ]] || fail 'published release is missing assets'
  printf 'release %s already published with matching identity\n' "$VERSION"
  exit 0
fi
[[ "$(jq -r '.draft' "$work/release.json")" == true ]] || fail 'unknown release state'

for name in "$macos" "$linux" "$checksums"; do
  if ! jq -e --arg name "$name" '.assets[] | select(.name == $name)' "$work/release.json" > /dev/null; then
    gh release upload "$VERSION" "$RELEASE_DIR/$name" --repo "$repo" > /dev/null
  fi
done

verify_assets
[[ "$(jq -r '.assets | length' "$work/release.json")" == 3 ]] || fail 'draft is missing assets'
[[ "$(jq -r '.draft' "$work/release.json")" == true ]] || fail 'release changed state during upload'
jq -r '.body' "$work/release.json" | grep -Fxq -- "$marker" || fail 'release notes changed during upload'
gh api "repos/$repo/git/ref/tags/$VERSION" > "$work/tag.json"
tag_type="$(jq -r '.object.type' "$work/tag.json")"
tag_sha="$(jq -r '.object.sha' "$work/tag.json")"
if [[ "$tag_type" == tag ]]; then
  gh api "repos/$repo/git/tags/$tag_sha" > "$work/annotated.json"
  tag_type="$(jq -r '.object.type' "$work/annotated.json")"
  tag_sha="$(jq -r '.object.sha' "$work/annotated.json")"
fi
[[ "$tag_type" == commit && "$tag_sha" == "$SOURCE_SHA" ]] || fail 'tag changed before publication'
gh release edit "$VERSION" --repo "$repo" --draft=false > /dev/null
gh api "repos/$repo/releases/tags/$VERSION" > "$work/release.json"
[[ "$(jq -r '.draft' "$work/release.json")" == false ]] || fail 'release publication not confirmed'
printf 'published %s at %s with verified assets\n' "$VERSION" "$SOURCE_SHA"
