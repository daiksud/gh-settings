#!/usr/bin/env bash
set -euo pipefail
repo_root="$(cd "$(dirname "$0")/../.." && pwd)"
fixture="$(mktemp -d)"
trap 'rm -rf "$fixture"' EXIT
export MOCK_ROOT="$fixture"
cat > "$fixture/cli" <<'MOCK'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$1" >> "$MOCK_ROOT/calls"
case "$1" in
  validate) [[ "${INVALID:-}" != 1 ]] ;;
  plan)
    [[ "${API_ERROR:-}" != 1 ]] || { echo 'read permission denied' >&2; exit 1; }
    printf 'settings.repository.has_issues: true → false\n' ;;
  *) echo 'unexpected operation' >&2; exit 1 ;;
esac
MOCK
chmod +x "$fixture/cli"
runner="$repo_root/examples/github-actions/plan.sh"
bash "$runner" "$fixture/cli" "$fixture/plan" > "$fixture/log"
[[ "$(cat "$fixture/calls")" == $'validate\nplan' ]] || exit 1
cmp "$fixture/plan" "$fixture/log"
rm "$fixture/calls"
if INVALID=1 bash "$runner" "$fixture/cli" "$fixture/plan" > "$fixture/log" 2>&1; then exit 1; fi
[[ "$(cat "$fixture/calls")" == validate ]] || exit 1
rm "$fixture/calls"
if API_ERROR=1 bash "$runner" "$fixture/cli" "$fixture/plan" > "$fixture/log" 2>&1; then exit 1; fi
[[ "$(cat "$fixture/calls")" == $'validate\nplan' ]] || exit 1
printf 'Read-only plan sequence fixtures passed.\n'
