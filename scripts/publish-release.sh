#!/usr/bin/env bash
set -euo pipefail

fail() { printf 'release: %s\n' "$*" >&2; exit 1; }
digest() { sha256sum "$1" | cut -d ' ' -f 1; }
check_hash() {
  local path="$1" expected="$2"
  [[ -f "$path" && "$(digest "$path")" == "$expected" ]] || fail "artifact hash mismatch: $(basename "$path")"
}

work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT
go build -o "$work/release-publisher" "${BASH_SOURCE[0]%/*}/release-publisher/main.go"
"$work/release-publisher" preflight \
  "${BASH_SOURCE[0]%/*}/../action/cli-version" > "$work/identity.json"
linux=gh-settings-linux-amd64
macos=gh-settings-darwin-arm64
checksums=SHA256SUMS
checksum_sha="$(jq -r '.checksums_sha256' "$work/identity.json")"
marker="$(jq -r '.marker' "$work/identity.json")"
repo="$GITHUB_REPOSITORY"
# Inspect existing release identity before creating any missing tag.
"$work/release-publisher" lookup "$work/identity.json" "$work" > "$work/release-observation.json"
release_exists="$(jq -r '.exists' "$work/release-observation.json")"
release_id="$(jq -r '.id' "$work/release-observation.json")"

# Generate from the frozen local history before creating any remote state.
# Existing drafts retain their reviewed notes when publication is resumed.
if [[ "$release_exists" == false ]]; then
  (
    cd "${BASH_SOURCE[0]%/*}/.."
    go run ./scripts/release-notes/main.go "$repo" "$VERSION" "$SOURCE_SHA" "$work/releases.json"
  ) > "$work/changes.md"
  cat > "$work/notes.md" <<EOF
$marker

$(cat "$work/changes.md")

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
fi

# Resolve both lightweight and annotated tags without moving either.
"$work/release-publisher" tag "$work/identity.json" "$work" existing > "$work/tag-observation.json"
if [[ "$(jq -r '.exists' "$work/tag-observation.json")" == false ]]; then
  [[ "$release_exists" == false ]] || fail 'existing release has no tag; refusing to recreate it'
  gh api -X POST "repos/$repo/git/refs" -f "ref=refs/tags/$VERSION" -f "sha=$SOURCE_SHA" > /dev/null
fi

if [[ "$release_exists" == false ]]; then
  jq -n --arg tag "$VERSION" --arg commit "$SOURCE_SHA" --rawfile body "$work/notes.md" '{tag_name:$tag,target_commitish:$commit,name:$tag,body:$body,draft:true,prerelease:false}' > "$work/create.json"
  gh api -X POST "repos/$repo/releases" --input "$work/create.json" > "$work/release.json"
  "$work/release-publisher" created-id "$work/release.json" > "$work/created-state.json"
  release_id="$(jq -r '.id' "$work/created-state.json")"
fi
"$work/release-publisher" state "$work/identity.json" "$work/release.json" initial > "$work/release-state.json"

verify_assets() {
  local name expected download_dir
  gh api "repos/$repo/releases/$release_id" > "$work/release.json"
  [[ "$(jq -r '[.assets[].name] | unique | length' "$work/release.json")" == "$(jq -r '.assets | length' "$work/release.json")" ]] || fail 'duplicate asset names'
  jq -e --arg m "$macos" --arg l "$linux" --arg s "$checksums" 'all(.assets[].name; . == $m or . == $l or . == $s)' "$work/release.json" > /dev/null || fail 'unexpected release asset'
  download_dir="$(mktemp -d "$work/downloads.XXXXXX")"
  for name in "$macos" "$linux" "$checksums"; do
    case "$name" in
      "$macos") expected="$MACOS_SHA256" ;;
      "$linux") expected="$LINUX_SHA256" ;;
      *) expected="$checksum_sha" ;;
    esac
    if jq -e --arg name "$name" '.assets[] | select(.name == $name)' "$work/release.json" > /dev/null; then
      gh release download "$VERSION" --repo "$repo" --pattern "$name" --dir "$download_dir" > /dev/null
      check_hash "$download_dir/$name" "$expected"
    fi
  done
}

verify_assets
"$work/release-publisher" state "$work/identity.json" "$work/release.json" observed > "$work/release-state.json"
if [[ "$(jq -r '.state' "$work/release-state.json")" == published ]]; then
  [[ "$(jq -r '.assets | length' "$work/release.json")" == 3 ]] || fail 'published release is missing assets'
  printf 'release %s already published with matching identity\n' "$VERSION"
  exit 0
fi

for name in "$macos" "$linux" "$checksums"; do
  if ! jq -e --arg name "$name" '.assets[] | select(.name == $name)' "$work/release.json" > /dev/null; then
    gh release upload "$VERSION" "$RELEASE_DIR/$name" --repo "$repo" > /dev/null
  fi
done

verify_assets
[[ "$(jq -r '.assets | length' "$work/release.json")" == 3 ]] || fail 'draft is missing assets'
"$work/release-publisher" state "$work/identity.json" "$work/release.json" upload > "$work/release-state.json"
"$work/release-publisher" tag "$work/identity.json" "$work" publication > "$work/tag-observation.json"
jq -n '{draft:false}' > "$work/publish.json"
gh api -X PATCH "repos/$repo/releases/$release_id" --input "$work/publish.json" > /dev/null
gh api "repos/$repo/releases/$release_id" > "$work/release.json"
"$work/release-publisher" state "$work/identity.json" "$work/release.json" published > "$work/release-state.json"
printf 'published %s at %s with verified assets\n' "$VERSION" "$SOURCE_SHA"
