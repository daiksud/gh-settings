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

The required `settings-plan` job validates the PR merge checkout's operational
`.github/settings.yml` with the same immutable public Action pin as the trusted
PR plan and main apply callers. It uses `command: validate` without a token,
Environment or repository administration credentials, on the Action's supported
fresh Ubuntu 24.04 x64 VM. Invalid configuration fails the check.

The separate required `ci-pr` validates candidate code and current-format
fixtures through its build, process and public Action acceptance stages.
Operational YAML remains tied to its pinned consumer until the declaration and
compatible package are adopted together under
[#382](https://github.com/daiksud/gh-settings/issues/382). There is no automatic
format conversion or compatibility path in the candidate product.

The path-scoped [settings-pr.yml](../../.github/workflows/settings-pr.yml)
separately runs a trusted, authenticated plan when relevant settings files
change. Local validation does not establish live settings differences.
`settings-plan` uses a per-PR concurrency group and does not cancel active runs;
another PR cannot replace its pending run.

The trusted `settings-env` check audits the Environment boundary
using the default-branch policy. Neither the guard nor the source-controlled
workflow alone proves live credential configuration.

## Ruleset transition

Set required contexts to `ci-pr`, `prlint`, `settings-plan` and
`settings-env`, each produced by GitHub Actions. Remove the
old `pass`, `pr-message` and `gh-settings` required contexts. The live
Ruleset is an external GitHub setting; changing job names does not update it.
