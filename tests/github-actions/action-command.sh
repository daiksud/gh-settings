#!/usr/bin/env bash
set -euo pipefail
repo_root="$(cd "$(dirname "$0")/../.." && pwd)"
fixture="$(mktemp -d)"
trap 'rm -rf "$fixture"' EXIT
export GITHUB_WORKSPACE="$fixture/caller checkout" MOCK_CALLS="$fixture/calls"
mkdir -p "$GITHUB_WORKSPACE" "$fixture/elsewhere"
cat > "$fixture/verified cli" <<'MOCK'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$1" >> "$MOCK_CALLS"
[[ $# == 1 && "$PWD" == "$GITHUB_WORKSPACE" ]] || exit 90
case "$1" in
  validate)
    [[ -z "${GH_TOKEN:-}" && -z "${GITHUB_TOKEN:-}" ]] || exit 91
    if [[ "${MOCK_INVALID_CONFIG:-}" == true ]]; then
      printf 'invalid configuration\n' >&2
      exit 23
    fi
    printf 'validated stdout\n'
    printf 'validated stderr\n' >&2
    ;;
  plan)
    [[ "${GH_TOKEN:-}" == explicit-fixture ]] || exit 92
    printf '%s\n' "$MOCK_PLAN_OUTPUT"
    printf 'plan stderr\n' >&2
    exit "${MOCK_COMMAND_STATUS:-0}"
    ;;
  apply)
    [[ "${GH_TOKEN:-}" == explicit-fixture ]] || exit 92
    printf 'applied stdout\n'
    printf 'applied stderr\n' >&2
    exit "${MOCK_COMMAND_STATUS:-0}"
    ;;
  *) exit 93 ;;
esac
MOCK
chmod +x "$fixture/verified cli"
runner="$repo_root/action/run-command.sh"
cd "$fixture/elsewhere"
printf 'validated stdout\nvalidated stderr\n' > "$fixture/expected"
GH_TOKEN=ambient-fixture GITHUB_TOKEN=ambient-fixture \
  bash "$runner" "$fixture/verified cli" validate "$fixture/output text" > "$fixture/log" 2>&1 || {
    cat "$fixture/log" >&2; exit 1
  }
[[ "$(cat "$MOCK_CALLS")" == validate ]] || exit 1
[[ ! -s "$fixture/log" ]] || exit 1
cmp "$fixture/expected" "$fixture/output text"
for MOCK_PLAN_OUTPUT in 'No differences.' 'No managed settings.' 'Update description: before -> after'; do
  export MOCK_PLAN_OUTPUT
  : > "$MOCK_CALLS"
  printf 'validated stdout\nvalidated stderr\n%s\nplan stderr\n' "$MOCK_PLAN_OUTPUT" > "$fixture/expected"
  GH_TOKEN=explicit-fixture GITHUB_TOKEN=ambient-fixture \
    bash "$runner" "$fixture/verified cli" plan "$fixture/output text" > "$fixture/log" 2>&1 || {
      cat "$fixture/log" >&2; exit 1
    }
  [[ "$(cat "$MOCK_CALLS")" == $'validate\nplan' ]] || exit 1
  [[ ! -s "$fixture/log" ]] || exit 1
  cmp "$fixture/expected" "$fixture/output text"
done
: > "$MOCK_CALLS"
printf 'applied stdout\napplied stderr\n' > "$fixture/expected"
GH_TOKEN=explicit-fixture bash "$runner" "$fixture/verified cli" apply "$fixture/output text" > "$fixture/log" 2>&1 || {
  cat "$fixture/log" >&2; exit 1
}
[[ "$(cat "$MOCK_CALLS")" == apply ]] || exit 1
[[ ! -s "$fixture/log" ]] || exit 1
cmp "$fixture/expected" "$fixture/output text"
reject_command() {
  : > "$MOCK_CALLS"
  if bash "$runner" "$@" > "$fixture/log" 2>&1; then exit 1; fi
  [[ ! -s "$MOCK_CALLS" && ! -e injected && ! -e "$GITHUB_WORKSPACE/injected" ]] || exit 1
}
reject_command
reject_command "$fixture/verified cli"
reject_command "$fixture/verified cli" validate
reject_command "$fixture/verified cli" validate "$fixture/output text" extra
for command in '' unknown 'validate; touch injected'; do
  reject_command "$fixture/verified cli" "$command" "$fixture/output text"
done
cp "$fixture/verified cli" "$GITHUB_WORKSPACE/verified cli"
cp "$fixture/verified cli" "$fixture/elsewhere/verified cli"
cp "$fixture/verified cli" "$fixture/not executable"
chmod -x "$fixture/not executable"
export PATH="$GITHUB_WORKSPACE:$PATH"
reject_command 'verified cli' validate "$fixture/output text"
reject_command "$fixture/missing cli" validate "$fixture/output text"
reject_command "$fixture/not executable" validate "$fixture/output text"
for command in plan apply; do
  for token in absent empty; do
    : > "$MOCK_CALLS"
    : > "$fixture/output text"
    if [[ "$token" == absent ]]; then
      if env -u GH_TOKEN GITHUB_TOKEN=ambient-fixture \
        bash "$runner" "$fixture/verified cli" "$command" "$fixture/output text" > "$fixture/log" 2>&1; then exit 1; fi
    else
      if GH_TOKEN='' GITHUB_TOKEN=ambient-fixture \
        bash "$runner" "$fixture/verified cli" "$command" "$fixture/output text" > "$fixture/log" 2>&1; then exit 1; fi
    fi
    [[ ! -s "$MOCK_CALLS" ]] || exit 1
    if grep -l 'ambient-fixture' "$fixture/log" "$fixture/output text"; then exit 1; fi
  done
done
for command in validate plan; do
  : > "$MOCK_CALLS"
  printf 'invalid configuration\n' > "$fixture/expected"
  status=0
  GH_TOKEN=explicit-fixture MOCK_INVALID_CONFIG=true \
    bash "$runner" "$fixture/verified cli" "$command" "$fixture/output text" > "$fixture/log" 2>&1 || status=$?
  [[ "$status" == 23 && "$(cat "$MOCK_CALLS")" == validate ]] || exit 1
  [[ ! -s "$fixture/log" ]] || exit 1
  cmp "$fixture/expected" "$fixture/output text"
done
for command in plan apply; do
  : > "$MOCK_CALLS"
  if [[ "$command" == plan ]]; then
    expected_calls=$'validate\nplan'
    printf 'validated stdout\nvalidated stderr\nAPI failure\nplan stderr\n' > "$fixture/expected"
  else
    expected_calls=apply
    printf 'applied stdout\napplied stderr\n' > "$fixture/expected"
  fi
  status=0
  GH_TOKEN=explicit-fixture GITHUB_TOKEN=ambient-fixture MOCK_PLAN_OUTPUT='API failure' MOCK_COMMAND_STATUS=37 \
    bash "$runner" "$fixture/verified cli" "$command" "$fixture/output text" > "$fixture/log" 2>&1 || status=$?
  [[ "$status" == 37 && "$(cat "$MOCK_CALLS")" == "$expected_calls" ]] || exit 1
  [[ ! -s "$fixture/log" ]] || exit 1
  cmp "$fixture/expected" "$fixture/output text"
  if grep -lE 'explicit-fixture|ambient-fixture' "$fixture/log" "$fixture/output text" "$MOCK_CALLS"; then exit 1; fi
done
printf 'Action command fixtures passed.\n'
