# GitHub Actions consumer examples

These inert workflows use the public root composite Action. Merging changes under this directory does not install these examples as workflows. This repository's separately maintained active workflows under `.github/workflows` use the released v0.6.1 Action, and these examples use the same released identity. This directory matches the active path filters, so even documentation changes may trigger read-only plan/prepare and queue apply for owner approval. That does not approve apply. See the [public contract](../../docs/adr/0023-public-composite-action.md) and [setup/review/recovery guide](../../docs/github-actions.md).

## Reviewed source and Action identity

Both examples pin `daiksud/gh-settings@9d01d3258d9f42239b2daeb1ac26bc532cd56105` (v0.6.1). That full SHA selects the released root `action.yml` and packaged helpers independently of the caller's workflow/configuration commits. Copy both workflows and their verifier from one approved source checkout containing this reviewed identity; that source checkout is distinct from the pinned Action package.

The immutable [v0.6.1 release](https://github.com/daiksud/gh-settings/releases/tag/v0.6.1) fixes CLI `v0.6.1`, asset `gh-settings-linux-amd64`, SHA-256 `7b93802a64609ed5daf1b3bd02e33ad9384fda26b7776233cd094c8dd85d5874`, and checks the exact reported version before execution. The caller verifier checks the same version/checksum before token issuance. Current source records its explicit candidate version in `action/cli-version`. Existing tags/assets stay immutable: CLI v0.5.0 has no root Action, and published v0.6.0 still contains the earlier installer using CLI v0.5.0. Publishing another release or adopting these files in a consumer requires separate authorization.

## Copy only the caller files

From the reviewed source checkout, place these three files in the configuration repository:

| Source | Destination |
| --- | --- |
| `settings-pr.yml` | `.github/workflows/settings-pr.yml` |
| `settings-main.yml` | `.github/workflows/settings-main.yml` |
| `verify-action-handoff.sh` | `examples/github-actions/verify-action-handoff.sh` |

For example, with `source_dir` set to the absolute reviewed source checkout and the current directory at the target repository root:

```sh
mkdir -p .github/workflows examples/github-actions
cp "$source_dir/examples/github-actions/settings-pr.yml" .github/workflows/settings-pr.yml
cp "$source_dir/examples/github-actions/settings-main.yml" .github/workflows/settings-main.yml
cp "$source_dir/examples/github-actions/verify-action-handoff.sh" examples/github-actions/verify-action-handoff.sh
```

Use this as a future PR preparation procedure, not an activation command for an existing installation. Inspect existing destination files before copying; reconcile them in the PR rather than overwriting operational configuration. Commit root `.github/settings.yml` with [schemaVersion 1](../../docs/settings.md#configuration-semantics). No root `action.yml`, `action/`, old installer, plan wrapper or Summary renderer is needed in the consumer. Keep the caller verifier at the path used by the workflow.

Use fresh GitHub-hosted Ubuntu 24.04 x64 jobs, with Bash, curl, GNU sha256sum, Git and Python 3. Configuration checkout belongs directly at `GITHUB_WORKSPACE`, has a checkout-scoped GitHub.com `origin` and persists no credentials. Other runners/nested checkouts are unsupported. The Action downloads its fixed CLI even for credential-free validate.

## PR review and approved main apply

`settings-pr.yml` uses `pull_request` and the containing repository's synthetic merge revision (`github.sha`), including fork PRs. Its actual public call supplies `command: plan` and an explicit ordinary read token. The Action validates first, renders the quoted review Summary and publishes four success outputs: `config-sha`, `target`, `cli-version`, `cli-sha256`. Invalid configuration/read failure fails the call; diff/no-diff/no managed fields succeed. No App credential, protected Environment or apply path is available to PR code.

`settings-main.yml` permits only a push to Branch `main`. The prepare job guards the triggering checkout, calls the same read-only plan and maps the four outputs directly. Its additional `action-sha` is caller metadata, not an Action output. Failed prepare or detectable Summary rendering/write failure blocks apply. Successful writing does not prove publication; the owner must inspect a complete Summary before approval.

Apply needs prepare and waits at `settings-apply`. After owner approval, it checks out the prepared configuration commit and runs the local verifier against configuration, target, reviewed Action and fixed CLI identities. Only then does the pinned official token Action mint a one-repository Administration-write installation token and pass it to public `command: apply`, retaining default post-job revocation. The always-outcome Summary records the actual apply step as success, failure or skipped. The caller owns Environment protection, App scope, secure credential entry, mint/revoke and concurrency; the composite is not a credential sandbox.

For new installations, the example names are Environment `settings-apply`, variable `SETTINGS_APP_CLIENT_ID` and secret `SETTINGS_APP_PRIVATE_KEY`. Configure verified owner review, self-approval allowed, no administrator bypass and only a Branch `main` deployment rule before storing credentials. See the [ordered adoption boundary](../../docs/github-actions.md#ordered-owner-setup) for plan/visibility prerequisites and trust limits. No setup or live execution is performed by this documentation.

PR concurrency cancels superseded PR runs; main uses a repository-scoped group with `cancel-in-progress: false`, subject to GitHub's pending-run replacement. Direct apply sends declared values without rereading remote state or consuming a saved plan. It can overwrite intervening changes, and output failure can follow a successful update; no rollback, drift/freshness guard or custom approval mechanism is promised.

This repository's active callers use Action SHA `9d01d3258d9f42239b2daeb1ac26bc532cd56105` (v0.6.1), Environment `gh-settings`, App `answer26`, `ANSWER26_CLIENT_ID`, `ANSWER26_PRIVATE_KEY` and the unchanged declaration. Their caller verifier is `.github/scripts/verify-settings-handoff.sh`. Do not replace these resources with the inert names. Main prepare remains read-only; apply still requires the owner's existing Environment approval. Updating this documentation does not approve a deployment or publish a release.

## Caller verifier and validation

The active workflows call the public Action and their own handoff verifier. The example `verify-action-handoff.sh` remains a required consumer file, covered by identity and integration fixtures. Retired installer, plan, Summary and orchestration helpers are removed; current Action-owned helpers and their focused tests remain under `action/` and `tests/github-actions/`.

Operational PR/main coverage now executes the actual public-Action caller YAML through the existing composite fixtures; the obsolete legacy workflow integration test and its sole runner were removed. Tests use isolated local Git, dummy downloads/CLI responses, approval and token providers. Offline composition uses the current candidate Action package copied from the working tree, with a mocked v0.6.1 CLI and the one published consumer identity. It does not load the pinned tag bytes. Copied verifiers replace only the published binary checksum with the dummy binary checksum; separate identity tests use the unchanged verifier. Operational failure coverage includes install failure after token minting, with no apply/API call and normal simulated revocation. Current root tests use the current committed version. Hosted Ubuntu 24.04 smoke explicitly sets its copied package version to published v0.6.0 and uses that real release for validate; built-product acceptance separately checks real CLI semantics. None of these fixtures proves actual Environment enforcement, Summary publication or real token revocation; those need separately authorized platform evidence.
