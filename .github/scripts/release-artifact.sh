#!/usr/bin/env bash
set -euo pipefail

usage() {
  echo 'usage: release-artifact.sh write TARGET VERSION SOURCE_SHA ASSET_PATH DESTINATION | verify TARGET VERSION SOURCE_SHA ARTIFACT_DIR OUTPUT' >&2
  exit 2
}

asset_name() {
  case "$1" in
    linux-amd64) printf '%s\n' gh-settings-linux-amd64 ;;
    darwin-arm64) printf '%s\n' gh-settings-darwin-arm64 ;;
    *) echo "unsupported release target: $1" >&2; exit 1 ;;
  esac
}

hash_file() {
  if command -v sha256sum >/dev/null; then
    sha256sum "$1" | cut -d ' ' -f 1
  else
    shasum -a 256 "$1" | cut -d ' ' -f 1
  fi
}

validate_identity() {
  [[ "$1" =~ ^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]] || {
    echo 'invalid release version' >&2; exit 1
  }
  [[ "$2" =~ ^[0-9a-f]{40}$ ]] || {
    echo 'invalid release source SHA' >&2; exit 1
  }
}

[[ $# -ge 1 ]] || usage
command=$1
shift

case "$command" in
  write)
    [[ $# -eq 5 ]] || usage
    target=$1 version=$2 source_sha=$3 source_path=$4 destination=$5
    validate_identity "$version" "$source_sha"
    asset=$(asset_name "$target")
    [[ -f "$source_path" && ! -L "$source_path" ]] || {
      echo 'release asset must be a regular file' >&2; exit 1
    }
    [[ ! -e "$destination" ]] || {
      echo 'release artifact destination already exists' >&2; exit 1
    }
    mkdir -p "$destination"
    cp "$source_path" "$destination/$asset"
    chmod 0755 "$destination/$asset"
    digest=$(hash_file "$destination/$asset")
    [[ "$digest" =~ ^[0-9a-f]{64}$ ]] || {
      echo 'failed to calculate release artifact digest' >&2; exit 1
    }
    printf 'source-sha=%s\nversion=%s\ntarget=%s\nasset=%s\nsha256=%s\n' \
      "$source_sha" "$version" "$target" "$asset" "$digest" > "$destination/release-manifest"
    ;;
  verify)
    [[ $# -eq 5 ]] || usage
    target=$1 version=$2 source_sha=$3 directory=$4 output=$5
    validate_identity "$version" "$source_sha"
    asset=$(asset_name "$target")
    [[ -d "$directory" && ! -L "$directory" ]] || {
      echo 'release artifact directory is invalid' >&2; exit 1
    }
    inventory=$(find "$directory" -mindepth 1 -maxdepth 1 -print | sed "s|$directory/||" | LC_ALL=C sort)
    [[ "$inventory" == "$asset"$'\n'release-manifest ]] || {
      echo 'release artifact inventory differs from the manifest contract' >&2; exit 1
    }
    manifest=$directory/release-manifest
    product=$directory/$asset
    [[ -f "$manifest" && ! -L "$manifest" && -f "$product" && ! -L "$product" ]] || {
      echo 'release artifact contains a non-regular file' >&2; exit 1
    }
    [[ "$(wc -l < "$manifest" | tr -d ' ')" == 5 ]] || {
      echo 'release manifest has an invalid line count' >&2; exit 1
    }
    [[ "$(sed -n '1p' "$manifest")" == "source-sha=$source_sha" ]] || {
      echo 'release manifest source mismatch' >&2; exit 1
    }
    [[ "$(sed -n '2p' "$manifest")" == "version=$version" ]] || {
      echo 'release manifest version mismatch' >&2; exit 1
    }
    [[ "$(sed -n '3p' "$manifest")" == "target=$target" ]] || {
      echo 'release manifest target mismatch' >&2; exit 1
    }
    [[ "$(sed -n '4p' "$manifest")" == "asset=$asset" ]] || {
      echo 'release manifest asset mismatch' >&2; exit 1
    }
    digest_line=$(sed -n '5p' "$manifest")
    [[ "$digest_line" =~ ^sha256=([0-9a-f]{64})$ ]] || {
      echo 'release manifest digest is invalid' >&2; exit 1
    }
    expected=${BASH_REMATCH[1]}
    actual=$(hash_file "$product")
    [[ "$actual" == "$expected" ]] || {
      echo 'release artifact bytes differ from the manifest' >&2; exit 1
    }
    printf 'asset-path=%s\nsha256=%s\n' "$product" "$expected" >> "$output"
    ;;
  *) usage ;;
esac
