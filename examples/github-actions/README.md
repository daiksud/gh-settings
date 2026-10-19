# GitHub Actions consumer examples

Copy these workflows into the repository whose settings you want to manage. The main example has one job and three steps: checkout, create an App token and call `daiksud/gh-settings`. Consumers need no external scripts.

Both workflows use a reviewed full Action SHA; the [published package and pins](../../docs/github-actions.md#published-package-and-consumer-pins) identify the selected release. The [public Action guide](../../docs/github-actions.md) owns inputs, outputs, runner requirements and setup instructions.

The selected v0.15.0 package supports all sixteen repository fields including Discussions, merge methods, automatic branch deletion, squash/normal merge commit defaults, auto-merge and branch-update options, topics and labels. Label collections are authoritative; review their [deletion consequences and permission requirements](../../docs/github-actions.md#labels-and-permissions) before adoption.

## Copy the workflows

| Source | Destination |
| --- | --- |
| [settings-pr.yml](settings-pr.yml) | `.github/workflows/settings-pr.yml` |
| [settings-main.yml](settings-main.yml) | `.github/workflows/settings-main.yml` |

Commit `.github/settings.yml` with [schemaVersion 1](../../docs/settings.md#configuration-semantics) at the repository root. These three files are the consumer workflow/configuration files; CLI installation, validation and PR Summary rendering belong to the Action. Copying them does not configure GitHub branch or Environment protections and does not install a workflow-scope guard.

Use a fresh GitHub-hosted `ubuntu-24.04` x64 job. Retain checkout at `GITHUB_WORKSPACE`, the event's exact `github.sha` and `persist-credentials: false`. The checkout's GitHub.com `origin` selects the target.

## Environment and credentials

Complete the [ordered owner setup](../../docs/github-actions.md#ordered-owner-setup) before enabling main apply, including [required reviews/checks on main](../../docs/github-actions.md#protect-main-with-required-checks), the Environment gate, and [protection verification](../../docs/github-actions.md#verify-the-protection-before-use). The example uses these names:

| Resource | Name |
| --- | --- |
| Protected Environment | `settings-apply` |
| Environment variable | `SETTINGS_APP_CLIENT_ID` |
| Environment secret | `SETTINGS_APP_PRIVATE_KEY` |

For repository fields/topics, the App installation must include the target repository with Administration read/write. The token step narrows its token to the containing repository and Administration write, retaining default post-job revocation. Label mutations additionally require separately approved Issues or Pull requests write on the App/installation and matching token input; these examples do not request it automatically. Public zero-difference label reads do not establish mutation access. Configure the Environment's reviewer and `main`-only deployment rule before storing credentials; the workflow does not configure those protections itself.

## PR review and main apply

`settings-pr.yml` checks out the containing repository's synthetic PR merge revision and calls `command: plan` with `github.token`. The Action validates before reading settings and writes a review Summary. Differences, no differences and no managed fields all succeed; invalid configuration or a failed read fails the call. Fork PRs use the same read-only path without App credentials.

`settings-main.yml` runs for a push to Branch `main` that changes `.github/settings.yml` or the adopted main workflow. Its single `apply` job waits for the `settings-apply` Environment approval before any step runs. After approval it:

1. Checks out the triggering configuration commit.
2. Creates the target-scoped App installation token.
3. Calls `command: apply` with that token.

Review the workflow path, event, actors, pinned steps and credential scope as well as the PR plan and exact main configuration before approval; use the [approval checklist](../../docs/github-actions.md#review-a-pending-run). Main does not produce another plan, depend on the PR run or consume its plan. The Action step's outcome and logs report apply results. The [run review guide](../../docs/github-actions.md#review-a-pending-run) explains overwrite semantics and concurrency.

A required check name alone does not identify the workflow that produced it. If PR review is to be the only manual approval, first meet the [independent enforcement conditions](../../docs/github-actions.md#when-pr-review-is-the-only-manual-approval).

The caller owns the Environment, credentials, token lifecycle, event/ref guards and concurrency. Composite steps share the caller job, so keep privileged credentials out of PR code. If apply fails or is cancelled, inspect current settings before retrying; the [recovery guide](../../docs/github-actions.md#failure-and-recovery) covers partial success and token cleanup.

Contributors changing these examples should use the [testing guide](../../docs/contributing/testing.md); this project's own workflows and Environment policy are documented under [repository automation](../../docs/contributing/repository-automation.md).
