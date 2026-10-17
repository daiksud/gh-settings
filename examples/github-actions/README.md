# GitHub Actions consumer examples

These workflows are examples to copy into a configuration repository. The main example has one job and three steps: checkout, create an App token, and `daiksud/gh-settings`. Consumers need no external scripts.

Both examples pin the published [v0.7.0 release](https://github.com/daiksud/gh-settings/releases/tag/v0.7.0) at `e9edb03f65ac943cd0ebf6a9556c18c485719e66`. The Action installs CLI v0.7.0, verifies its release checksum and exact version, then runs the selected command. The Linux amd64 binary's SHA-256 is `c0ab75712dc7648a6ba9416c468ec9e5f43fd864ab85376fac84aeba3c86a9b0`.

## Copy the workflows

| Source | Destination |
| --- | --- |
| `settings-pr.yml` | `.github/workflows/settings-pr.yml` |
| `settings-main.yml` | `.github/workflows/settings-main.yml` |

Commit `.github/settings.yml` with [schemaVersion 1](../../docs/settings.md#configuration-semantics) at the repository root. The two workflow YAML files and this configuration are the complete consumer setup; installation, validation and PR Summary rendering belong to the Action.

Use a fresh GitHub-hosted `ubuntu-24.04` x64 job. Checkout stays at `GITHUB_WORKSPACE`, uses the event's exact `github.sha`, and persists no credentials. The checkout's GitHub.com `origin` selects the target. See the [public Action guide](../../docs/github-actions.md) for inputs, outputs and runner requirements.

## PR review and main apply

`settings-pr.yml` checks out the containing repository's synthetic PR merge revision and calls `command: plan` with `github.token`. The Action validates before reading settings and writes the review Summary. Differences, no differences and no managed fields all succeed; invalid configuration or a failed read fails the call. Fork PRs use this same read-only path without App credentials.

`settings-main.yml` runs only for a push to Branch `main` that changes `.github/settings.yml` or the adopted main workflow. Its single `apply` job waits for the `settings-apply` Environment approval **before any step runs**. After approval it:

1. Checks out the triggering configuration commit.
2. Creates an App installation token for the containing repository with Administration write.
3. Calls `command: apply` with that token.

Review the PR plan and the exact main commit's configuration before approving the deployment. Main does not produce another plan before approval, depend on the PR run, or consume its plan. The Action step's outcome and logs report apply success or failure. The official token Action retains default post-job revocation.

The caller owns the Environment, App credentials, token issuance/revocation, event/ref guards and concurrency. The Action owns CLI installation and the settings command. Its token input becomes `GH_TOKEN` only for command execution; composite steps still share the approved job.

## Environment and credentials

Use the example Environment `settings-apply`, variable `SETTINGS_APP_CLIENT_ID` and secret `SETTINGS_APP_PRIVATE_KEY`. The App installation must include the target repository and Administration read/write. The token step explicitly narrows the token to this repository and Administration write.

Protect the Environment with the owner reviewer, self-approval allowed, administrator bypass disabled and only a Branch `main` deployment rule before storing credentials. See [owner setup](../../docs/github-actions.md#ordered-owner-setup) for platform requirements and adopting the workflows.

PR concurrency cancels superseded PR runs; main uses a repository-scoped group with `cancel-in-progress: false`. Direct apply sends declared values without rereading remote state, so it can overwrite intervening changes to those fields. A failed output can follow a successful update; inspect current settings before retrying an uncertain result.

## Existing active workflows and verification

This repository's active `.github/workflows/settings-*.yml` now also use the published v0.7.0 Action and the same one-job main apply pattern. They keep Environment `gh-settings`, App `answer26`, `ANSWER26_CLIENT_ID` and `ANSWER26_PRIVATE_KEY`, rather than the example resource names. The legacy handoff verifier is no longer invoked.

Changes in this example directory can trigger the active read-only PR plan, but not main apply by themselves. Main apply watches only its adopted workflow and `.github/settings.yml`. The active `gh-settings` Environment instead uses a [required PR scope guard](../../docs/github-actions.md#active-repository-environment-guard) once its Required reviewers rule is disabled; the `settings-apply` example still requires approval.

CI lints the examples and checks their actual YAML composition, explicit token scope, failure propagation and the absence of caller scripts. Local fixtures simulate approval, downloads and token lifecycle; they do not establish real Environment enforcement, Summary publication or token revocation. The [development guide](../../docs/development.md#settings-example-verification) describes the broader verification boundary.
