# GitHub Actions consumer examples

These workflows plan changes from a pull request and apply the merged settings on `main`. They use the [v0.44.0 Action and CLI](../../docs/guides/github-actions.md#published-package-and-consumer-pins). Set up the [App, Environments and review controls](../../docs/guides/github-actions.md#ordered-owner-setup) in the target repository before enabling the credential-bearing jobs.

## Copy the workflows

| Copy | Destination | Runs when |
| --- | --- | --- |
| [settings-pr.yml](settings-pr.yml) | `.github/workflows/settings-pr.yml` | A PR targeting `main` opens, reopens or receives commits affecting a watched path. |
| [settings-main.yml](settings-main.yml) | `.github/workflows/settings-main.yml` | A push to `main` changes `.github/settings.yml` or the main workflow. |

Keep [the declaration](../../docs/references/settings/configuration.md) at the checkout root in `.github/settings.yml`. Review the PR path filter when copying the files. The supplied filter includes this examples directory. A path-filtered plan will not report on every PR.

## Environment and credentials

| Workflow job | Protected Environment | Permissions requested in the sample |
| --- | --- | --- |
| PR plan | `settings-plan` | Contents read for candidate data and Administration read for settings. |
| Main apply | `settings-apply` | Administration write for the target repository. |

Both Environments use the same installed GitHub App. Store its client ID in the Environment variable `SETTINGS_APP_CLIENT_ID` and its private key in the Environment secret `SETTINGS_APP_PRIVATE_KEY`. Each job mints its own short-lived installation token. Review the [permission matrix](../../docs/references/settings/permissions.md) before adding declarations. For example, updating labels also requires Issues or Pull requests write.

## PR review and main apply

The PR workflow executes trusted base code on `pull_request_target`. It reads the proposed commit metadata and the single regular settings blob. The plan identifies the candidate head as the configuration SHA and the containing repository as the target. Keep builds, tests and other PR code out of this job.

The main job waits for Environment approval before checking out the push SHA, minting the token and running apply. Review the actual main configuration and workflow before approving. These examples omit prune, so unlisted labels and Rulesets survive ordinary apply. A listed Ruleset can still change required reviews, checks and bypasses. [Review a pending run](../../docs/guides/protect-apply.md#review-a-pending-run)

For missing reads, failed applies or token cleanup errors, follow the [recovery guide](../../docs/guides/github-actions.md#failure-and-recovery). To change the examples in gh-settings itself, see the [contributor testing guide](../../docs/development/testing.md).
