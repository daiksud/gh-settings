#!/usr/bin/env bash
# Source from focused offline tests. No real token, network or GitHub repository writes.
set -euo pipefail
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
fixture="$(mktemp -d)"
trap 'rm -rf "$fixture"' EXIT
mkdir -p "$fixture/repo/.github"
cp -R "$repo_root/examples" "$fixture/repo/"
cd "$fixture/repo"
printf 'schemaVersion: 1\nsettings:\n  repository:\n    has_issues: false\n' > .github/settings.yml
git init -q
git remote add origin https://github.com/owner/repo
git add .
git -c commit.gpgsign=false -c user.name=Fixture -c user.email=fixture@example.invalid commit -qm 'test: initial fixture'
GITHUB_SHA="$(git rev-parse HEAD)"
export GITHUB_SHA GITHUB_REPOSITORY=owner/repo
export GITHUB_EVENT_NAME=push GITHUB_REF=refs/heads/main
export GITHUB_ACTOR=owner GITHUB_TRIGGERING_ACTOR=maintainer
export GITHUB_RUN_ID=123 GITHUB_RUN_ATTEMPT=2
export GITHUB_WORKFLOW_REF=owner/repo/.github/workflows/settings-main.yml@refs/heads/main
export GITHUB_WORKFLOW_SHA=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
export RUNNER_TEMP="$fixture/runner" GITHUB_OUTPUT="$fixture/output" GITHUB_STEP_SUMMARY="$fixture/summary"
mkdir -p "$RUNNER_TEMP"
