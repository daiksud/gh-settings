# PR validation and required checks

The tracked required PR checks are `ci-pr`, `prlint`, `settings-plan` and
`settings-env`. The guard is removed and validation now produces
`settings-validate`; the required names need the explicitly authorized transition
described below.
Their owning workflows and boundaries are listed in the
[repository automation guide](repository-automation.md#protected-main-and-required-checks).
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

`prlint` checks the current PR title and body together via commitlint. For PRs
that run the job, a green result confirms syntax, not semantic correctness.

PRs authored by `dependabot[bot]` skip the `prlint` job, matching the
[gh-qw workflow](https://github.com/daiksud/gh-qw/blob/main/.github/workflows/prlint.yml).
The condition uses the PR author, so a human rerun still skips a Dependabot PR;
other PRs retain the full validation even when the triggering actor is a bot.
Keep the workflow's PR events and required job name: this is a job-level skip,
not a workflow filter or removal of the required check. GitHub reports a skipped
job as successful for required-check purposes, so the existing `prlint` ruleset
requirement remains satisfied without changing protections. See
[GitHub's job-condition contract](https://docs.github.com/en/actions/how-tos/write-workflows/choose-when-workflows-run/control-jobs-with-conditions).
Normal review, squash-message classification and other required checks still
apply to Dependabot PRs.

The `settings-validate` job always validates the proposed merge declaration
with the operational public Action selected by trusted main code. It uses no
Environment or App token and does not observe live settings. There is no PR plan job.

The repository automation guide owns the
[merge identity and validation contract](repository-automation.md#pr-settings-validation)
and the [package pins and credential boundaries](repository-automation.md#active-settings-workflows).
Candidate code and current-format fixtures remain covered by `ci-pr`; they do
not establish that a pinned operational package supports a new declaration.

The Environment scope tests still run in ordinary CI with candidate source.
There is no independent trusted-main guard. These tests and workflow source do
not establish live credential configuration.

## Ruleset transition

The tracked declaration still requires `settings-plan` and `settings-env`.
The workflow now produces `settings-validate`; it does not provide an old-name
compatibility check. Follow the
[transition procedure](repository-automation.md#required-check-transition):
retire the live guard requirement, merge the job rename with the existing
trusted-main validation still required, verify a fresh post-merge validation
run, then replace the live required name. Update the declaration afterward
before applying settings. Preserve all other required checks and protections.
