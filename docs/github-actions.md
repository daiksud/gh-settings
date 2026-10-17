---
type: Guide
title: Use the public settings Action
description: Invoke the composite Action and separately adopt owner-approved consumer workflows.
sources:
  - id: composite-metadata
    resource: https://docs.github.com/en/actions/reference/workflows-and-actions/metadata-syntax
  - id: environment-controls
    resource: https://docs.github.com/en/actions/how-tos/deploy/configure-and-manage-deployments/manage-environments
  - id: app-token-action
    resource: https://github.com/actions/create-github-app-token/tree/64ab729a6294000f5aa4a4bc0b73d1f32bc45510
---

## Public Action and supported checkout

The root [composite Action](../action.yml) implements the [ADR 0023 contract](adr/0023-public-composite-action.md) and [behavior specification](features/actions.feature.md#public-composite-action-contract). Current source selects **published v0.7.0** and invokes one verified Go executable for metadata, settings operations and typed plan/Summary presentation. Shell retains installation bootstrap and Actions wiring; no Python runtime or command wrapper is required by this source. Workflow/configuration, Action and CLI identities remain distinct. Composite inputs/outputs use normal `with` and step-output bindings.[^composite-metadata]

Only a fresh GitHub-hosted `ubuntu-24.04` x64 job is supported, with Bash, curl, GNU sha256sum and Git. macOS, Windows, ARM, containers and self-hosted runners are outside this contract. The caller checks out its configuration repository at `GITHUB_WORKSPACE`, without a checkout `path` or persisted credentials, and commits `.github/settings.yml` at that root. A resolvable checkout commit and checkout-scoped GitHub.com `origin` are required even for validate's output metadata. The Action repository is not the target; `GH_REPO` does not override the CLI's origin resolution. See the [configuration semantics](settings.md#configuration-semantics).

The Action uses its own packaged installation bootstrap and verified absolute CLI in a per-invocation runner-temp directory; consumers do not install an extension or copy Action internals. Validate needs no settings authentication/API access, but the Action still downloads the public CLI asset. Workflow, configuration and Action revisions must remain available for ordinary reruns.

## Inputs, outputs and failures

| Input | Contract |
| --- | --- |
| `command` | Required, no default; exactly `validate`, `plan` or `apply`. |
| `token` | No default; unused for validate, required and nonempty for plan/apply. Supply it explicitly; no ambient or implicit `github.token` fallback. |

Validate checks local configuration without credentials. Plan validates first, then reads settings with an explicit read-only caller token, for example `token: ${{ github.token }}` in the read-only PR/prepare jobs. Apply directly invokes the existing CLI using the caller's approved, target-scoped App installation token with Administration write. The Action does not verify the token's actual grants or create it. No export, arbitrary arguments, configuration-path, target, version or download-URL input is supported.

| Success output | Meaning |
| --- | --- |
| `config-sha` | Full checked-out configuration commit (`HEAD`), not the file hash, workflow SHA or Action SHA. |
| `target` | Configuration checkout's resolved GitHub `origin`, in `OWNER/REPO` form. |
| `cli-version` | Verified installed CLI release selected by the Action package. |
| `cli-sha256` | SHA-256 of the verified installed binary. |

Use `steps.<id>.outputs.<name>` only after successful completion. Outputs are published after the command and, for plan, successful Summary rendering/writing. Missing command/token, unsupported prerequisites, install/metadata/output failures and CLI/configuration/API errors fail the Action. CLI failure status is preserved; differences, no differences and no managed fields all exit successfully. Failed calls have no successful output contract. Plan Summary quotes settings/error text and separates workflow, configuration, Action and CLI identities; validate/apply do not render a plan Summary. The main example uses the Action step's outcome and logs to report apply results.

Current source selects **published v0.7.0** in [`action/cli-version`](../action/cli-version) for the verified executable’s Go metadata and typed plan/Summary entries, sharing checkout/target resolution and planning with settings operations. Candidate CI verifies integration using the same source binary and local artifact bytes. The examples use published v0.7.0; active consumer pins remain on v0.6.1. Current source shares the CLI requirement for a directly configured GitHub.com origin: HTTPS, `git@github.com:owner/repo.git` or full SSH. Rewrite-only aliases and username-less SCP fail metadata before settings execution or success outputs; direct-origin identity survives transport mirror rewrites. Released v0.6.1 keeps its original compatibility. Authenticated plan writes an ephemeral typed report, then a separate credential-free step renders/writes its Summary. A nonzero plan status is preserved; otherwise Summary failure determines failure. Validate/apply also invoke the verified CLI directly; shell retains installation bootstrap and necessary Actions wiring. No command wrapper, terminal-text classification or Python renderer remains in current source, while released v0.6.1 keeps its original implementation. A reviewed full SHA pins that file and therefore the CLI version. Future version-tag calls must agree with the file; release publication rejects a different requested version. Installation downloads only that version's `gh-settings-linux-amd64` and `SHA256SUMS` from fixed upstream HTTPS paths, requires one valid checksum for the exact asset, verifies before execution, and checks exact `gh-settings <version>` output. The SHA pin fixes the version; it does not commit the binary digest. The checksum trust source is the same GitHub release, including its integrity over HTTPS. There is no latest fallback or caller version/URL override.

## Published package and consumer pins

The examples and README snippet use the published [v0.7.0 package](https://github.com/daiksud/gh-settings/releases/tag/v0.7.0), released on 2026-10-17, at `e9edb03f65ac943cd0ebf6a9556c18c485719e66`. It installs CLI v0.7.0 with Linux amd64 SHA-256 `c0ab75712dc7648a6ba9416c468ec9e5f43fd864ab85376fac84aeba3c86a9b0` and uses the Go metadata and typed plan/Summary entries without Python.

The active workflows retain [v0.6.1](https://github.com/daiksud/gh-settings/releases/tag/v0.6.1) at `9d01d3258d9f42239b2daeb1ac26bc532cd56105`. That package installs CLI v0.6.1 with Linux digest `7b93802a64609ed5daf1b3bd02e33ad9384fda26b7776233cd094c8dd85d5874` and needs Python 3. It retains its original origin compatibility, wrappers and quoted CLI-text Summary implementation. Updating examples does not update these operational pins. Existing tags/assets stay immutable; see [historical release limitations](releases.md#historical-release-limitations) for older versions.

## Consumer files and compatibility

The [PR/main examples](../examples/github-actions/README.md) need only these two workflow files plus the repository's `.github/settings.yml`:

| Example source | Consumer destination / purpose |
| --- | --- |
| `examples/github-actions/settings-pr.yml` | `.github/workflows/settings-pr.yml`: PR validate/plan with an explicit read token. |
| `examples/github-actions/settings-main.yml` | `.github/workflows/settings-main.yml`: one approved job with checkout, App token creation and direct apply. |

No caller script, root `action.yml` or copied `action/` directory is required. Both examples use the same v0.7.0 Action pin. Main checks out the event's exact configuration SHA and passes the caller-created token to `command: apply`; it has no prepare job, handoff outputs or separate apply Summary step. The PR plan provides review information without becoming an input or dependency of main apply.

The active [PR](../.github/workflows/settings-pr.yml) and [main](../.github/workflows/settings-main.yml) workflows retain v0.6.1; main keeps its two-job design. Its [operational handoff verifier](../.github/scripts/verify-settings-handoff.sh) checks prepared configuration, target, reviewed Action revision and fixed CLI identity before token minting. Only that operational verifier remains; it is not a consumer file for the simplified examples.

The active workflows' existing path filters include `examples/github-actions/**`, including its README, and their operational verifier. Changes there can run read-only PR plan/main prepare and queue an apply for owner approval. The simplified main example itself watches only `.github/settings.yml` and its adopted workflow path.

## Ordered owner setup

The caller owns Environment selection/approval, App token minting/revocation and concurrency. The Action's token input becomes `GH_TOKEN` only for the plan/apply command step; installation, validation, Summary and metadata/output steps do not receive that input. Composite steps share the caller job, so all code in a credential-bearing job must be trusted and privileged credentials must stay out of PR code.

For a new installation, configure the target's supported Environment controls before adopting the workflows; availability depends on the GitHub plan and repository visibility.[^environment-controls]

1. Install a dedicated selected-repositories App on the target with Administration read/write and automatic Metadata read. The caller's pinned official token Action requests Administration write for only the containing repository through explicit `owner` and `repositories` inputs, retaining default post-job revocation.[^app-token-action]
2. Protect `settings-apply` with the owner reviewer, self-approval allowed, administrator bypass disabled and only a Branch `main` rule. Store Environment variable `SETTINGS_APP_CLIENT_ID` and secret `SETTINGS_APP_PRIVATE_KEY` after protection is configured; an Environment name alone is not a workflow allowlist.
3. Adopt the two YAML files and root configuration through review, retaining exact-event checkout, no persisted credentials, main event/ref guards and concurrency. The entire main job waits for approval before checkout or token issuance. Review the PR plan and exact main configuration before approving it; main creates no additional pre-approval plan.

The examples perform no App/Environment setup themselves. Actual waiting/rejection, branch/tag enforcement and revocation depend on the configured GitHub controls and are not established by local test fixtures.

This repository's operational callers preserve Environment `gh-settings`, App `answer26`, `ANSWER26_CLIENT_ID`, `ANSWER26_PRIVATE_KEY` and their existing declaration. Their main prepare still returns the four Action outputs plus caller-owned Action SHA; apply waits at the existing Environment, checks out the prepared configuration and verifies the handoff before minting a token. The example names do not rename or replace those resources.

## Review a pending run

For the simplified example, review the PR plan Summary and the configuration at the main run's exact commit before approving `settings-apply`. The PR revision and resulting main commit may differ. Main does not depend on PR success or rerun plan before approval, so confirm the approved configuration explicitly; its checkout uses `github.sha`, never a moving main branch.

For this repository's existing two-job workflow, inspect the same run's prepare Summary before approving `gh-settings`. Check original and rerun actors, event/run/attempt, workflow path/ref and revision, target, configuration revision/file link, distinct Action repository/ref, CLI identity and observed before → desired after values. Missing or incomplete review content must be resolved before approval; successful prepare alone does not prove GitHub published its Summary.

A plan is informational. No differences and no managed fields are successful outcomes. Direct apply sends declared values without reading current settings or consuming a saved plan; intervening changes to managed fields can be overwritten, while omitted fields remain outside the request. The simplified main job mints its token before the settings Action installs the CLI and validates configuration; installation or validation failure stops apply and the token Action still performs normal post-job cleanup.

PR concurrency cancels superseded runs. Main uses a repository-scoped group with `cancel-in-progress: false`; GitHub can replace a pending run. Ordinary reruns use GitHub's approval behavior. No custom queue, drift/freshness guard, stale-commit check or reapproval controller is added.

## Failure and recovery

| Situation | Maintainer action |
| --- | --- |
| Missing/invalid config | Correct the declaration and rerun through normal review; validation fails before API access. |
| CLI download/checksum/version error | Inspect the fixed release identity and reviewed pin. Do not execute an unchecked binary or switch to source/latest. |
| Plan read/API permission error | Inspect ordinary read-token access and the target origin. Do not provide administration credentials to PR/prepare. |
| Rejected approval | The main job cannot start. Initiate a new eligible run when appropriate; no rollback is implied. |
| Failed operational prepare | The existing two-job workflow cannot proceed to apply; fix the cause and repeat normal review. |
| Missing App key/client ID or token error | Verify the protected Environment and dedicated installation scope. Have the owner use secure entry; no privileged token fallback. |
| Failed/cancelled apply or output failure | Read target settings before deciding what to do: an update may already have succeeded. The CLI does not promise atomicity or automatic rollback. |
| Token cleanup failure | Inspect the official Action's post-job logs and dedicated installation. Treat revocation as unverified; do not expose the token while investigating. |
| Disable further executions | Disable the adopted settings workflow in Actions and cancel pending/running executions as appropriate; review any update already sent. Disabling does not undo settings. Credential revocation/protection changes remain owner security operations. |

## References

- [GitHub contexts](https://docs.github.com/en/actions/reference/workflows-and-actions/contexts)
- [Environment setup and supported controls](https://docs.github.com/en/actions/how-tos/deploy/configure-and-manage-deployments/manage-environments)
- [Official token Action at the reviewed revision](https://github.com/actions/create-github-app-token/tree/64ab729a6294000f5aa4a4bc0b73d1f32bc45510)

[^composite-metadata]: [GitHub composite metadata syntax](https://docs.github.com/en/actions/reference/workflows-and-actions/metadata-syntax). Inputs and mapped step outputs; runtime checks remain Action-owned.
[^environment-controls]: [GitHub Environment controls](https://docs.github.com/en/actions/how-tos/deploy/configure-and-manage-deployments/manage-environments). Check availability and protection rules for the actual target before owner setup.
[^app-token-action]: [Official token Action at the reviewed revision](https://github.com/actions/create-github-app-token/tree/64ab729a6294000f5aa4a4bc0b73d1f32bc45510). Explicit target/permission inputs and default post-job revocation; actual revocation needs runtime evidence.
