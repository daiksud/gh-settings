# PR validation and required checks

The tracked declaration requires `screening`, `prlint`, and `settings-pr` from
GitHub Actions (App ID `15368`). Apply the reviewed declaration through
`settings-apply` after merge and read back the live Ruleset before claiming
these checks are enforced. Their owning workflows and boundaries are listed in the
[repository automation guide](repository-automation.md#protected-main-and-required-checks).
GitHub reports each job's native check result.

## PR screening

The [pr workflow](../../.github/workflows/pr.yml) uses the default `pull_request`
activities: opened, synchronize, and reopened. Its single `screening` job runs
source lint, unit tests, and a native Linux build on `ubuntu-slim`.
The job calls the focused local Actions directly; [testing](testing.md#ci-validation)
describes the step sequence and cache behavior. A failed lint, unit-test, or
build step fails the job. The separate `ci-push` aggregate checks Screening
and process/public-Action Inspection on main.

All PR-source execution uses read-only tokens and checkouts without persisted
credentials. Screening leaves the PR's Draft or Ready state unchanged.
Title/body edits and draft/readiness transitions do not trigger this workflow;
metadata-only edits remain the responsibility of `prlint`.

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

The `settings-pr` job validates the proposed merge declaration
using the pinned public Action from the PR workflow definition. It uses no
Environment or App token and does not observe live settings. There is no PR plan job.

The repository automation guide owns the
[checkout and validation contract](repository-automation.md#pr-settings-validation)
and the [package pins and credential boundaries](repository-automation.md#active-settings-workflows).
Candidate code and current-format fixtures remain covered by `screening`; they do
not establish that a pinned operational package supports a new declaration.

The Environment scope tests still run in ordinary CI with candidate source.
There is no independent trusted-main guard. These tests and workflow source do
not establish live credential configuration.

The `settings-pr` workflow uses the default `pull_request` activities
(opened, synchronize and reopened) and an unconfigured checkout step on
`ubuntu-slim`. Only `prlint` reruns automatically on metadata-only edits.

## Ruleset enforcement

Review the actual GitHub settings and the current check producer, workflow
revision, event, and candidate SHA before relying on a required check.
The [required-check verification procedure](repository-automation.md#required-check-verification)
distinguishes source configuration from live enforcement. `settings-apply`
remains a separate manual operation after merge; PR validation does not apply
the declaration.
