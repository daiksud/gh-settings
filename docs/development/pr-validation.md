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

The screening job has Contents read permission, and checkout uses its default credential persistence. PR-controlled checks can use that read-only credential during the job. Screening leaves Draft and Ready state unchanged. Metadata-only edits trigger the separate message check.

## Message and settings validation

The [PRLint workflow](../../.github/workflows/prlint.yml) checks the current PR title and body together with commitlint, including on title and body edits. It disables checkout credential persistence. A green result verifies message syntax. Reviewers assess scope, compatibility and the actual effect of the change under the [code-review policy](../../.agents/skills/code-review/SKILL.md).

PRs authored by `dependabot[bot]` skip the `prlint` job. The condition uses the PR author, so a human rerun still skips that job. GitHub treats a skipped job as successful for required-check purposes. The requirement, the other checks and normal review still apply. See [GitHub's job-condition contract](https://docs.github.com/en/actions/how-tos/write-workflows/choose-when-workflows-run/control-jobs-with-conditions).

The [settings-pr workflow](../../.github/workflows/settings-pr.yml) validates the proposed merge declaration with the public Action pin selected by that PR workflow. It runs on the default opened, synchronize and reopened activities for PRs targeting main. Checkout uses the default merge ref and credential persistence in a Contents-read job. The Action receives no settings token, selects no Environment and performs no plan or remote settings observation.

A passing `settings-pr` run establishes that the selected package accepts the declaration. Candidate-source checks and Environment-scope tests provide separate evidence. Use the [checkout and validation contract](repository-automation.md#pr-settings-validation) for implementation details and the [active settings workflows](repository-automation.md#active-settings-workflows) for the operational pins and App bindings.

## Ruleset enforcement

Review the actual GitHub settings and the current check producer, workflow
revision, event, and candidate SHA before relying on a required check.
The [required-check verification procedure](repository-automation.md#required-check-verification)
distinguishes source configuration from live enforcement. `settings-apply`
remains a separate manual operation after merge; PR validation does not apply
the declaration.
