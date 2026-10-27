# PR validation and required checks

The required PR checks are named for their workflow file and aggregate job:
`ci-pr.yml` → `ci-pr`, `prlint.yml` → `prlint`, and
`settings-plan.yml` → `settings-plan`. The independent
`settings-env.yml` reports `settings-env`.
No workflow publishes the shared `pass` commit status.

## Staged code validation

Every PR event (including title/body edits) runs Draft, metadata resolution,
Commit stage, Ready and Acceptance stage. The final `ci-pr` job succeeds only
when every prerequisite succeeds. A failed, skipped or cancelled prerequisite
cannot report success. The separate `ci-push` aggregate checks main after merge.
The PR aggregate uses `always()` so cancelled prerequisites are evaluated as
non-success rather than skipping the required job.

All PR-source execution uses read-only tokens and checkouts without persisted
credentials. The Draft/Ready jobs change only PR state; the aggregate job has
no write permission. Repeated metadata edits can rerun expensive CI, but they
cannot create a skipped-success replacement for a prior failed job.

## Message and settings validation

`prlint` checks the current PR title and body together via commitlint. A green
result confirms syntax, not semantic correctness.

The required `settings-plan` job runs `validate` on the PR merge candidate
without repository administration credentials. It does not claim to run an
authenticated GitHub settings difference plan. The path-scoped
[`settings-pr.yml`](../../.github/workflows/settings-pr.yml) separately runs
a trusted, authenticated plan when relevant settings files change.
The dispatch-only, Environment-protected plan within `settings-plan.yml` remains
inactive until explicitly enabled. Its `plan` job is not itself a required PR
check.

PR validation uses per-PR concurrency groups, separate from protected manual
dispatches. Neither mode cancels active runs; newer events within the same PR
may still replace its pending run, but cannot evict another PR or a manual plan.

The trusted `settings-env` check audits the Environment boundary
using the default-branch policy. Neither the guard nor the source-controlled
workflow alone proves live credential configuration.

## Ruleset transition

Set required contexts to `ci-pr`, `prlint`, `settings-plan` and
`settings-env`, each produced by GitHub Actions. Remove the
old `pass`, `pr-message` and `gh-settings` required contexts. The live
Ruleset is an external GitHub setting; changing job names does not update it.
