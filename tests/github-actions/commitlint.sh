#!/usr/bin/env bash
set -euo pipefail

# Exercise the selected style rules and prove that unselected limits stay off.
commitlint="$(mise which commitlint)"
lint() {
  printf '%s\n' "$1" | mise exec -- bun run --bun --no-install "$commitlint" --strict >/dev/null 2>&1
}
accept() {
  if ! lint "$1"; then
    printf 'Expected commitlint to accept: %s\n' "$1" >&2
    exit 1
  fi
}
reject() {
  if lint "$1"; then
    printf 'Expected commitlint to reject: %s\n' "$1" >&2
    exit 1
  fi
}

accept 'feat: add topic management'
accept 'feat(API): add topic management'
accept 'feat!: change settings schema'
accept $'feat: change settings schema\n\nBREAKING CHANGE: previous schema is no longer accepted'
accept "feat: $(printf 'x%.0s' {1..120})"
accept "fix: correct a bug"$'\n\n'"$(printf 'x%.0s' {1..120})"
accept 'feat: some Message'
reject 'other: perform maintenance'
reject 'FEAT: add a feature'
reject 'feat: Add a feature'
reject 'feat: ADD A FEATURE'
reject 'feat: FixFeature'
reject 'feat: add a feature.'
reject ' feat: add a feature'
reject 'feat: add a feature '
reject 'feat:'
reject 'fixup! feat: add a feature'
reject $'feat: add a feature\nbody without a blank line'
