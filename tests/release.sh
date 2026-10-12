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
if [[ "$1" == api ]]; then
  shift
  if [[ "$1" == -X ]]; then
    [[ "$2" == POST ]]
    if [[ "$3" == */releases/generate-notes ]]; then
      printf '{"body":"- Test change"}\n' | jq -r .body
      exit 0
    fi
    printf '{"object":{"type":"commit","sha":"%s"}}\n' "$SOURCE_SHA" > "$MOCK_ROOT/tag.json"
    exit 0
  fi
  case "$1" in
    */git/ref/tags/*)
      [[ -f "$MOCK_ROOT/tag.json" ]] || { echo 'gh: Not Found (HTTP 404)' >&2; exit 1; }
      cat "$MOCK_ROOT/tag.json" ;;
    */releases/tags/*)
      [[ -f "$MOCK_ROOT/body" ]] || { echo 'gh: Not Found (HTTP 404)' >&2; exit 1; }
      jq -n --arg tag "$VERSION" --rawfile body "$MOCK_ROOT/body" --rawfile draft "$MOCK_ROOT/draft" --argjson assets "$(find "$MOCK_ROOT/assets" -type f -exec basename {} \; | jq -R . | jq -s .)" '{tag_name:$tag,name:$tag,prerelease:false,draft:($draft == "true\n"),body:$body,assets:[$assets[] | {name:.}]}';;
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
  edit) printf 'false\n' > "$MOCK_ROOT/draft" ;;
  *) echo "unexpected release operation: $2" >&2; exit 1 ;;
esac
MOCK
chmod +x "$fixture/bin/gh"
export MOCK_ROOT="$fixture" PATH="$fixture/bin:$PATH" GITHUB_REPOSITORY=owner/repo
export VERSION=v1.2.3 SOURCE_SHA=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa RELEASE_DIR="$fixture/release"
printf 'macos-binary' > "$RELEASE_DIR/gh-settings-darwin-arm64"
printf 'linux-binary' > "$RELEASE_DIR/gh-settings-linux-amd64"
MACOS_SHA256="$(sha256sum "$RELEASE_DIR/gh-settings-darwin-arm64" | cut -d ' ' -f 1)"
LINUX_SHA256="$(sha256sum "$RELEASE_DIR/gh-settings-linux-amd64" | cut -d ' ' -f 1)"
export MACOS_SHA256 LINUX_SHA256
publish() { bash "$repo_root/scripts/publish-release.sh"; }
reject() {
  if publish > "$fixture/output" 2>&1; then
    cat "$fixture/output" >&2
    echo 'expected release rejection' >&2
    exit 1
  fi
}

# First run creates a tag, draft and three assets, then publishes.
publish
[[ "$(cat "$fixture/draft")" == false ]]
[[ "$(find "$fixture/assets" -type f | wc -l | tr -d ' ')" == 3 ]]
[[ "$(grep -c '^release upload ' "$fixture/calls")" == 3 ]]

# A completed run is an identity-checked no-op.
publish
[[ "$(grep -c '^release upload ' "$fixture/calls")" == 3 ]]

# Interrupted draft: retain matching assets and upload only the absent one.
printf 'true\n' > "$fixture/draft"
rm "$fixture/assets/SHA256SUMS"
publish
[[ "$(grep -c '^release upload ' "$fixture/calls")" == 4 ]]
[[ "$(cat "$fixture/draft")" == false ]]

# Version, tag target, draft manifest, and bytes each reject conflicts.
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
echo 'release mock tests passed'
