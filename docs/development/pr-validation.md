# PR validation and required checks

The [default-branch Ruleset and its required checks](repository-automation.md#protected-main-and-required-checks) govern merging to `main`. Each required job reports its result to GitHub.

## PR screening

The [PR workflow](../../.github/workflows/pr.yml) runs on opened, synchronized,
reopened and base-retargeted PRs. Its `screening` job installs the manifest-selected
tools and runs the [shared Linux checks](../../.github/actions/screening/action.yml)
on the merge candidate. Lint, unit tests and a native build must all succeed.
Checkout disables credential persistence, and the job has Contents read permission.
Concurrent updates to one PR cancel superseded screening runs.

Editing only the PR title or body reruns `commitlint`. Screening and settings validation
skip that edit under separate display names so a skipped result cannot satisfy
either required check for a new merge candidate. The [main CI guide](testing/ci.md)
describes the inspections that run after merge.

## Message and settings validation

The [commitlint workflow](../../.github/workflows/commitlint.yml) runs for every opened, reopened, synchronized or edited PR, including drafts and Dependabot updates. It checks out the event's head SHA with full history and a read-only token, with credential persistence disabled. It validates every commit in `base..head` and the title and body together as the proposed squash message. A separate step rejects any HTML comment opener in the PR body, even in code samples.

The repository preset supplies the message rules and the normal commitlint warning behavior. Correct any failing commit messages on the PR branch and edit the final title and body as needed. Dependabot updates can include generated HTML comments and overlong lines. Reviewers assess scope, compatibility and the actual effect of the change under the [code-review policy](../../.agents/skills/code-review/SKILL.md).

The [settings-pr workflow](../../.github/workflows/settings-pr.yml) checks newly opened, synchronized, reopened and base-retargeted PRs targeting `main`. Its only job uses the default merge checkout and a Contents-read token on `ubuntu-slim`. The public Action pinned in the PR workflow runs `validate` on `.github/settings.yml` without an Environment or settings token. A passing check means the selected package accepts that declaration. The [manual apply workflow](repository-automation.md#active-settings-workflows) owns remote settings changes.

## Ruleset enforcement

Review the actual GitHub settings and the current check producer, workflow
revision, event, and candidate SHA before relying on a required check.
The [required-check verification procedure](repository-automation.md#required-check-verification)
distinguishes source configuration from live enforcement. `settings-apply`
remains a separate manual operation after merge; PR validation does not apply
the declaration.
