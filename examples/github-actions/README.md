# GitHub Actions consumer examples

These inert workflows use the public root composite Action. Merging changes under this directory does not install these examples as workflows. This repository's separately maintained active workflows under `.github/workflows` use the released v0.6.1 Action, while these examples retain their historical pin. This directory matches the active path filters, so even documentation changes may trigger read-only plan/prepare and queue apply for owner approval. That does not approve apply. See the [public contract](../../docs/adr/0023-public-composite-action.md) and [setup/review/recovery guide](../../docs/github-actions.md).

## Reviewed source and Action identity

Both examples pin `daiksud/gh-settings@93d8217d146f5d430aaf90597119114218990e70`. That full SHA contains the implemented root `action.yml` and packaged helpers, independently of the caller's workflow/configuration commits. Example source revision `897eab7fc143d848abf37d7355bd64126527ef4c` contains both updated workflows and the caller verifier below; its Action package matches the examples' pin.

That historical Action fixes CLI `v0.5.0`, asset `gh-settings-linux-amd64`, SHA-256 `b4a63eb8912657deabc5ea8909a7e6198d375c94cb0a9cd83f7d2179f7bbf5b4`, and checks exact reported version before execution. CLI tag `v0.5.0` has no root Action: it cannot replace the Action SHA. Existing tags/assets stay immutable. Current source instead records the explicit version in `action/cli-version`, with matching future release tags and same-release checksum verification. Published v0.6.0 still contains the earlier installer using v0.5.0; keep its tag/assets unchanged. These inert caller pins and verifier remain historical, and adopting a newer reviewed Action/CLI identity requires updating them together through review. Publishing another release/Marketplace listing requires a separate owner decision.

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

## Retained helper inventory

The active workflows call the public Action and their own handoff verifier. Historical helpers remain with their focused compatibility fixtures:

| Retained file | Existing reference / compatibility reason |
| --- | --- |
| `install-cli.sh` | Legacy prepare/verify wrappers; fixed CLI installer retained with bootstrap fixtures. |
| `plan.sh` | Legacy prepare; retained validate-before-plan fixtures. |
| `render-summary.py` | Legacy prepare; retained escaped Summary fixtures. |
| `prepare.sh` | Retained legacy handoff/prepare fixtures. |
| `verify-apply.sh` | Retained legacy approved-apply fixtures. |
| `apply.sh` | Retained legacy step-token/failure fixtures. |
| `verify-action-handoff.sh` | Inert public-Action main caller; required consumer file, production-identity and integration fixtures. |

The ordinary CI keeps the focused legacy helper fixtures. Operational PR/main coverage now executes the actual public-Action caller YAML through the existing composite fixtures; the obsolete legacy workflow integration test and its sole runner were removed. Tests use isolated local Git, dummy downloads/CLI responses, approval and token providers. Both historical v0.5.0 and operational v0.6.1 identities are modeled explicitly; copied verifiers use the corresponding dummy binary checksum. Operational failure coverage includes install failure after token minting, with no apply/API call and normal simulated revocation. Current root tests use the current committed version. Hosted Ubuntu 24.04 smoke explicitly sets its copied package version to published v0.6.0 and uses that real release for validate; built-product acceptance separately checks real CLI semantics. None of these fixtures proves actual Environment enforcement, Summary publication or real token revocation; those need separately authorized platform evidence.
