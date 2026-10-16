#!/usr/bin/env bash
# Inspect actual candidate execution and runner Summary bytes, not publication.
set -euo pipefail

case "${1:-}" in
  start)
    [[ $# == 2 && -x "$2" && -f "$GITHUB_ENV" ]]
    state=$(mktemp -d /tmp/ghs-plan.XXXXXX)
    mkdir "$state/config"
    printf 'http_unix_socket: %s/api.sock\n' "$state" > "$state/config/config.yml"
    "$2" "$state" > "$state/server.log" 2>&1 &
    printf '%s\n' "$!" > "$state/pid"
    # Export the owned state before readiness checks so always() cleanup can use it.
    {
      printf 'PLAN_FIXTURE=%s\nGH_CONFIG_DIR=%s/config\n' "$state" "$state"
      printf 'GH_PATH_DISABLED=1\nGIT_CONFIG_NOSYSTEM=1\nGIT_CONFIG_GLOBAL=/dev/null\n'
      printf 'HTTP_PROXY=http://127.0.0.1:1\nHTTPS_PROXY=http://127.0.0.1:1\n'
      printf 'http_proxy=http://127.0.0.1:1\nhttps_proxy=http://127.0.0.1:1\nNO_PROXY=\nno_proxy=\n'
    } >> "$GITHUB_ENV"
    for ((i=0; i<100; i++)); do
      [[ -e "$state/ready" ]] && break
      kill -0 "$(cat "$state/pid")"
      sleep 0.05
    done
    [[ -S "$state/api.sock" && -e "$state/ready" ]]
    ;;
  prepare)
    [[ $# == 1 && -S "$PLAN_FIXTURE/api.sock" ]]
    [[ -z ${GH_TOKEN:-} && -z ${GITHUB_TOKEN:-} && -z ${GH_HOST:-} && -z ${GH_REPO:-} ]]
    cat > .github/settings.yml <<'YAML'
schemaVersion: 1
settings:
  repository:
    description: |-
      after <script>alert("fixture")</script>
      line
    has_issues: false
YAML
    git add .github/settings.yml
    git -c commit.gpgsign=false -c user.name=Fixture -c user.email=fixture@example.invalid commit --quiet -m 'test: prepare candidate plan'
    git rev-parse HEAD > "$PLAN_FIXTURE/config-sha"
    sha256sum .github/settings.yml > "$PLAN_FIXTURE/config-checksum"
    : > "$PLAN_FIXTURE/requests"
    : > "$PLAN_FIXTURE/summaries-before"
    shopt -s nullglob
    for summary in "${GITHUB_STEP_SUMMARY%/*}"/step_summary_*; do
      printf '%s\n' "$summary" >> "$PLAN_FIXTURE/summaries-before"
    done
    ;;
  assert)
    [[ $# == 1 && -S "$PLAN_FIXTURE/api.sock" && "$PLAN_OUTCOME" == success ]]
    [[ -z ${GH_TOKEN:-} && -z ${GITHUB_TOKEN:-} ]]
    [[ "$(git ls-files)" == .github/settings.yml ]]
    [[ "$(git -C .action rev-parse HEAD)" == "$ACTION_SHA" ]]
    [[ "$(git rev-parse HEAD)" == "$(cat "$PLAN_FIXTURE/config-sha")" ]]
    [[ -z "$(git status --porcelain --untracked-files=no)" ]]
    sha256sum --check "$PLAN_FIXTURE/config-checksum"
    [[ "$CONFIG_SHA" == "$(cat "$PLAN_FIXTURE/config-sha")" && "$CONFIG_SHA" != "$ACTION_SHA" ]]
    [[ "$TARGET" == example-owner/settings-fixture ]]
    [[ "$CLI_VERSION" == "$EXPECTED_CLI_VERSION" && "$CLI_SHA256" == "$EXPECTED_CLI_SHA256" ]]
    [[ "$(cat "$PLAN_FIXTURE/requests")" == 'GET /repos/example-owner/settings-fixture' ]]
    matches=0
    shopt -s nullglob
    for summary in "${GITHUB_STEP_SUMMARY%/*}"/step_summary_*; do
      [[ "$summary" != *-scrubbed && -s "$summary" ]] || continue
      if grep -Fxq "$summary" "$PLAN_FIXTURE/summaries-before"; then continue; fi
      if grep -Fq '## Settings review' "$summary"; then
        matches=$((matches + 1))
        cp "$summary" "$PLAN_FIXTURE/summary"
      fi
    done
    [[ "$matches" == 1 ]]
    summary="$PLAN_FIXTURE/summary"
    grep -Fq "$CONFIG_SHA" "$summary"
    grep -Fq "$CLI_SHA256" "$summary"
    grep -Fq "$CLI_VERSION" "$summary"
    grep -Fq 'example-owner/settings-fixture' "$summary"
    grep -Fq 'Observed before → desired after.' "$summary"
    grep -Fq 'settings.repository.has_issues: true → false' "$summary"
    grep -Fq '&lt;b&gt;bold&lt;/b&gt;\nline' "$summary"
    grep -Fq '&lt;script&gt;alert(\&#34;fixture\&#34;)&lt;/script&gt;\nline' "$summary"
    if grep -Fq '<script>' "$summary"; then exit 1; fi
    printf 'Candidate plan: source=%s config=%s version=%s sha256=%s\n' "$ACTION_SHA" "$CONFIG_SHA" "$CLI_VERSION" "$CLI_SHA256"
    cat "$PLAN_FIXTURE/requests" "$summary"
    printf 'Actual plan and Summary bytes verified; publication is not asserted.\n'
    ;;
  stop)
    [[ $# == 1 ]]
    if [[ -n ${PLAN_FIXTURE:-} ]]; then
      [[ "$PLAN_FIXTURE" == /tmp/ghs-plan.* && -f "$PLAN_FIXTURE/pid" ]]
      kill "$(cat "$PLAN_FIXTURE/pid")"
      for ((i=0; i<100; i++)); do
        [[ ! -S "$PLAN_FIXTURE/api.sock" ]] && break
        sleep 0.05
      done
      [[ ! -S "$PLAN_FIXTURE/api.sock" && ! -s "$PLAN_FIXTURE/server.log" ]]
      rm -rf "$PLAN_FIXTURE"
    fi
    ;;
  *) echo 'usage: candidate-plan.sh start FIXTURE_BINARY | prepare | assert | stop' >&2; exit 1 ;;
esac
