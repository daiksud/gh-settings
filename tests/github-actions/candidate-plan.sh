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
    [[ $# -le 2 && -S "$PLAN_FIXTURE/api.sock" ]]
    [[ -z ${GH_TOKEN:-} && -z ${GITHUB_TOKEN:-} && -z ${GH_HOST:-} && -z ${GH_REPO:-} ]]
    rm -f "$PLAN_FIXTURE/read403" "$PLAN_FIXTURE/write403"
    scenario="${2:-normal}"
    private=false
    if [[ "$scenario" == private* ]]; then
      private=true
      scenario="${scenario#private}"
      scenario="${scenario#-}"
      scenario="${scenario:-normal}"
    fi
    case "$scenario" in
      normal|invalid-config|apply|apply-invalid|apply-empty) ;;
      read-failure) touch "$PLAN_FIXTURE/read403" ;;
      apply-write-failure) touch "$PLAN_FIXTURE/write403" ;;
      *) exit 1 ;;
    esac
    cat > .github/settings.yml <<'YAML'
schemaVersion: 1
settings:
  repository:
    description: |-
      after <script>alert("fixture")</script>
      line
    has_issues: false
YAML
    if [[ "$scenario" == apply* ]]; then
      printf 'schemaVersion: 1\nsettings:\n  repository:\n    description: ""\n    has_issues: false\n' > .github/settings.yml
    fi
    if [[ "$scenario" == invalid-config || "$scenario" == apply-invalid ]]; then
      printf 'schemaVersion: 1\nsettings:\n  repository:\n    has_issues: "invalid"\n' > .github/settings.yml
    fi
    if [[ "$scenario" == apply-empty ]]; then
      printf 'schemaVersion: 1\nsettings: {}\n' > .github/settings.yml
    fi
    git add .github/settings.yml
    if ! git diff --cached --quiet; then
      git -c commit.gpgsign=false -c user.name=Fixture -c user.email=fixture@example.invalid commit --quiet -m 'test: prepare candidate configuration'
    fi
    git rev-parse HEAD > "$PLAN_FIXTURE/config-sha"
    sha256sum .github/settings.yml > "$PLAN_FIXTURE/config-checksum"
    : > "$PLAN_FIXTURE/requests"
    rm -f "$PLAN_FIXTURE/patch.json"
    : > "$PLAN_FIXTURE/summaries-before"
    shopt -s nullglob
    for summary in "${GITHUB_STEP_SUMMARY%/*}"/step_summary_*; do
      printf '%s\n' "$summary" >> "$PLAN_FIXTURE/summaries-before"
    done
    ;;
  assert)
    [[ $# -le 2 && -S "$PLAN_FIXTURE/api.sock" ]]
    [[ -z ${GH_TOKEN:-} && -z ${GITHUB_TOKEN:-} ]]
    [[ "$(git ls-files)" == .github/settings.yml ]]
    [[ "$(git -C .action rev-parse HEAD)" == "$ACTION_SHA" ]]
    [[ "$(git rev-parse HEAD)" == "$(cat "$PLAN_FIXTURE/config-sha")" ]]
    [[ -z "$(git status --porcelain --untracked-files=no)" ]]
    sha256sum --check "$PLAN_FIXTURE/config-checksum"
    consumer_sha=$(cat "$PLAN_FIXTURE/config-sha")
    [[ "$consumer_sha" != "$ACTION_SHA" ]]
    scenario="${2:-normal}"
    private=false
    if [[ "$scenario" == private* ]]; then
      private=true
      scenario="${scenario#private}"
      scenario="${scenario#-}"
      scenario="${scenario:-normal}"
    fi
    case "$scenario" in
      normal|apply|apply-empty)
        [[ "$PLAN_OUTCOME" == success && "$CONFIG_SHA" == "$consumer_sha" ]]
        [[ "$TARGET" == example-owner/settings-fixture ]]
        [[ "$CLI_VERSION" == "$EXPECTED_CLI_VERSION" && "$CLI_SHA256" == "$EXPECTED_CLI_SHA256" ]]
        ;;
      read-failure|invalid-config|apply-write-failure|apply-invalid)
        [[ "$PLAN_OUTCOME" == failure ]]
        [[ -z "$CONFIG_SHA$TARGET$CLI_VERSION$CLI_SHA256" ]]
        ;;
      *) exit 1 ;;
    esac
    if [[ "$scenario" == invalid-config || "$scenario" == apply-invalid || "$scenario" == apply-empty ]]; then
      [[ ! -s "$PLAN_FIXTURE/requests" && ! -e "$PLAN_FIXTURE/patch.json" ]]
    elif [[ "$scenario" == apply* ]]; then
      [[ "$(cat "$PLAN_FIXTURE/requests")" == 'PATCH /repos/example-owner/settings-fixture' ]]
      [[ "$(jq -cS . "$PLAN_FIXTURE/patch.json")" == '{"description":"","has_issues":false}' ]]
    else
      [[ "$(cat "$PLAN_FIXTURE/requests")" == 'GET /repos/example-owner/settings-fixture' ]]
    fi
    matches=0
    shopt -s nullglob
    for summary in "${GITHUB_STEP_SUMMARY%/*}"/step_summary_*; do
      [[ "$summary" != *-scrubbed && -s "$summary" ]] || continue
      if grep -Fxq "$summary" "$PLAN_FIXTURE/summaries-before"; then continue; fi
      if grep -Eq '^## Settings (review|plan)$' "$summary"; then
        matches=$((matches + 1))
        cp "$summary" "$PLAN_FIXTURE/summary"
      fi
    done
    if [[ "$scenario" == apply* ]]; then
      [[ "$matches" == 0 ]]
      apply_case=${2#apply}
      apply_case=${apply_case#-}
      printf 'Candidate apply %s: source=%s config=%s version=%s sha256=%s outcome=%s\n' \
        "${apply_case:-normal}" "$ACTION_SHA" "$consumer_sha" "$EXPECTED_CLI_VERSION" "$EXPECTED_CLI_SHA256" "$PLAN_OUTCOME"
      if [[ "$2" == apply-invalid || "$2" == apply-empty ]]; then
        printf 'Zero API requests or captured PATCH; unchanged consumer; no new Settings review.\n'
      else
        cat "$PLAN_FIXTURE/requests" "$PLAN_FIXTURE/patch.json"
        printf '\nExact received PATCH and unchanged consumer verified; no GET or new Settings review.\n'
      fi
      if [[ "$2" == apply-write-failure ]]; then
        printf 'A failed PATCH attempt does not establish the remote outcome.\n'
      fi
      exit 0
    fi
    [[ "$matches" == 1 ]]
    summary="$PLAN_FIXTURE/summary"
    if [[ "$private" == true ]]; then
      grep -Fq '## Settings plan' "$summary"
      if [[ "$scenario" == normal ]]; then
        [[ "$PRIVATE_PLAN_OUTCOME" == changed ]]
        grep -Fq 'Outcome: changed' "$summary"
      else
        [[ -z "$PRIVATE_PLAN_OUTCOME" ]]
        grep -Fq "Outcome: $scenario" "$summary"
      fi
      if grep -Eq 'settings\.|description|before|desired|script|bold' "$summary"; then exit 1; fi
      shopt -s nullglob
      reports=("$RUNNER_TEMP"/gh-settings-cli.*/plan.json)
      [[ ${#reports[@]} == 0 ]]
      printf 'Private plan result, read-only API requests and report cleanup verified.\n'
      exit 0
    fi
    grep -Fq "$consumer_sha" "$summary"
    grep -Fq "$EXPECTED_CLI_SHA256" "$summary"
    grep -Fq "$EXPECTED_CLI_VERSION" "$summary"
    grep -Fq 'example-owner/settings-fixture' "$summary"
    if [[ "$scenario" == normal ]]; then
      grep -Fq 'Observed before → desired after.' "$summary"
      grep -Fq 'settings.repository.has_issues: true → false' "$summary"
      grep -Fq '&lt;b&gt;bold&lt;/b&gt;\nline' "$summary"
      grep -Fq '&lt;script&gt;alert(\&#34;fixture\&#34;)&lt;/script&gt;\nline' "$summary"
    else
      grep -Fq 'Error — plan failed.' "$summary"
      if [[ "$2" == read-failure ]]; then
        grep -Fq '<pre>repository read failed</pre>' "$summary"
      else
        grep -Fq '<pre>configuration validation failed</pre>' "$summary"
      fi
    fi
    if grep -Fq '<script>' "$summary"; then exit 1; fi
    printf 'Candidate plan %s: source=%s config=%s version=%s sha256=%s outcome=%s\n' \
      "${2:-normal}" "$ACTION_SHA" "$consumer_sha" "$EXPECTED_CLI_VERSION" "$EXPECTED_CLI_SHA256" "$PLAN_OUTCOME"
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
  *) echo 'usage: candidate-plan.sh start FIXTURE_BINARY | prepare [read-failure|invalid-config|apply|apply-write-failure|apply-invalid|apply-empty] | assert [read-failure|invalid-config|apply|apply-write-failure|apply-invalid|apply-empty] | stop' >&2; exit 1 ;;
esac
