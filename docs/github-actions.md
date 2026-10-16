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

The root [composite Action](../action.yml) implements the [ADR 0023 contract](adr/0023-public-composite-action.md) and [behavior specification](features/actions.feature.md#public-composite-action-contract). Current source selects **unreleased v0.7.0** and invokes one verified Go executable for metadata, settings operations and typed plan/Summary presentation. Shell retains installation bootstrap and Actions wiring; no Python runtime or command wrapper is required by this source. Workflow/configuration, Action and CLI identities remain distinct. Composite inputs/outputs use normal `with` and step-output bindings.[^composite-metadata]

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

Use `steps.<id>.outputs.<name>` only after successful completion. Outputs are published after the command and, for plan, successful Summary rendering/writing. Missing command/token, unsupported prerequisites, install/metadata/output failures and CLI/configuration/API errors fail the Action. CLI failure status is preserved; differences, no differences and no managed fields all exit successfully. Failed calls have no successful output contract. Plan Summary quotes settings/error text and separates workflow, configuration, Action and CLI identities; validate/apply do not render a plan Summary. A caller can record apply's actual step outcome, as the main example does.

Current source selects **unreleased v0.7.0** in [`action/cli-version`](../action/cli-version) for the verified executable’s Go metadata and typed plan/Summary entries, sharing checkout/target resolution and planning with settings operations. Candidate CI supplies the same source binary without requiring unpublished release assets. Published v0.6.1 and active consumer pins remain unchanged; source integration does not publish or adopt v0.7.0. Current source shares the CLI requirement for a directly configured GitHub.com origin: HTTPS, `git@github.com:owner/repo.git` or full SSH. Rewrite-only aliases and username-less SCP fail metadata before settings execution or success outputs; direct-origin identity survives transport mirror rewrites. Released v0.6.1 keeps its original compatibility. Authenticated plan writes an ephemeral typed report, then a separate credential-free step renders/writes its Summary. A nonzero plan status is preserved; otherwise Summary failure determines failure. Validate/apply also invoke the verified CLI directly; shell retains installation bootstrap and necessary Actions wiring. No command wrapper, terminal-text classification or Python renderer remains in current source, while released v0.6.1 keeps its original implementation. A reviewed full SHA pins that file and therefore the CLI version. Future version-tag calls must agree with the file; release publication rejects a different requested version. Installation downloads only that version's `gh-settings-linux-amd64` and `SHA256SUMS` from fixed upstream HTTPS paths, requires one valid checksum for the exact asset, verifies before execution, and checks exact `gh-settings <version>` output. The SHA pin fixes the version; it does not commit the binary digest. The checksum trust source is the same GitHub release, including its integrity over HTTPS. There is no latest fallback or caller version/URL override.

## Published package and consumer pins

The latest published package is [v0.6.1](https://github.com/daiksud/gh-settings/releases/tag/v0.6.1), invoked as `uses: daiksud/gh-settings@9d01d3258d9f42239b2daeb1ac26bc532cd56105`. It installs CLI v0.6.1, Linux digest `7b93802a64609ed5daf1b3bd02e33ad9384fda26b7776233cd094c8dd85d5874`, and needs Python 3 in addition to the supported runner tools. It retains its original origin compatibility, wrappers and quoted CLI-text Summary implementation. Current-source Go entries and local-empty/direct-origin changes are not supplied by that pin.

The active workflows, inert examples and caller verifiers retain that released identity. Candidate CI proves current-source integration using local artifact bytes; it does not supply published v0.7.0 assets or adopt the candidate. Publication and coordinated consumer updates require separate authorization. Existing tags/assets stay immutable; [historical release limitations](releases.md#historical-release-limitations) remain available for older consumers. This guide does not publish a release or Marketplace listing.

## Consumer files and compatibility

The [inert PR/main examples](../examples/github-actions/README.md) use the released public Action. Copy both YAML files and their caller verifier from the same approved source checkout containing the v0.6.1 identities above. The source checkout revision and pinned released Action SHA are separate identities.

| Inert source | Consumer destination / purpose |
| --- | --- |
| `examples/github-actions/settings-pr.yml` | `.github/workflows/settings-pr.yml`: PR validate/plan with explicit read token. |
| `examples/github-actions/settings-main.yml` | `.github/workflows/settings-main.yml`: read-only prepare, then approved direct apply. |
| `examples/github-actions/verify-action-handoff.sh` | Same path: compare prepared configuration/target/Action/CLI identity before token minting. |

Keep the caller verifier at that path in the configuration checkout. Do not copy root `action.yml` or `action/` into the consumer. Prepare exposes the four Action outputs plus separately labeled caller-owned `action-sha`; the latter is not a fifth Action output. Apply checks out prepare's exact configuration commit, verifies the handoff and same reviewed Action pin, then mints its token. Neither job builds source, imports PR executables/artifacts nor replays a saved plan.

The active [PR](../.github/workflows/settings-pr.yml) and [main](../.github/workflows/settings-main.yml) workflows use the released v0.6.1 Action pin. Their operational [handoff verifier](../.github/scripts/verify-settings-handoff.sh) checks the prepared configuration, target, reviewed Action revision and fixed CLI identity before App token minting. The Action then installs/verifies its own CLI; its setup steps do not receive the token input in their environment. Installation failure still fails apply and retains the token Action's normal post-job revocation.

Existing path filters still include `examples/github-actions/**`, including its README; both workflows also include the operational verifier's exact path. Such changes can run read-only PR plan/main prepare and queue an apply for owner approval. The inert examples use the same released Action identity while keeping their independent caller setup names. The [caller verification guide](../examples/github-actions/README.md#caller-verifier-and-validation) describes the retained handoff verifier and current Action/caller coverage. Existing #83–#94 work is completed prior art, not a new setup backlog or evidence for every future installation.

## Ordered owner setup

The Action does not select an Environment, request approval, mint/revoke tokens or configure concurrency. Its token input becomes `GH_TOKEN` only for the plan/apply command step; installation, validation, Summary and metadata/output steps do not receive that token input. This scopes environment exposure, not process isolation: composite steps share the caller job. Trust all code in a credential-bearing job and keep privileged credentials out of untrusted PR code.

For a new installation, the owner must verify supported Environment controls and separately authorize these operations. GitHub's availability varies by plan/visibility; unsupported controls are a stop condition.[^environment-controls]

1. Select the exact target and a dedicated selected-repositories App with Administration read/write and automatic Metadata read, without unrelated access. The caller requests Administration write for just its containing repository using explicit owner/repositories; the installation's persistent grants are the owner's responsibility.
2. Protect the example's `settings-apply` Environment with the owner reviewer, self-approval allowed, administrator bypass disabled and only a Branch `main` rule (no Tag/wildcard rule). Verify protection before securely entering Environment variable `SETTINGS_APP_CLIENT_ID` and secret `SETTINGS_APP_PRIVATE_KEY`; no repository/organization-wide key fallback. An Environment name is not a workflow allowlist.
3. In a separately authorized adoption PR, copy the three files above from the same reviewed source, preserve credential-free checkout and event/ref guards, and inspect trigger/path effects. The main workflow permits only a main-branch push; apply depends on successful prepare and Environment approval before any App mint. The official pinned token Action requests only that repository's Administration write, and retains default post-job revocation. Its token stays within the approved job and is passed only to `command: apply`.[^app-token-action]
4. Stop at owner review of the concrete PR and activation/test/cleanup scope. Merging an adoption PR can trigger read-only prepare and a waiting apply; approve nothing without inspecting the complete Summary. Actual waiting/rejection, branch/tag enforcement and revocation need separately authorized platform checks. Workflow filtering alone does not prove Environment enforcement; do not add temporary probe refs or broaden rules without their own concrete authorization.

This repository's operational callers preserve Environment `gh-settings`, App `answer26`, `ANSWER26_CLIENT_ID`, `ANSWER26_PRIVATE_KEY` and the declaration. The inert names `settings-apply` / `SETTINGS_APP_*` are not instructions to rename or replace existing resources. Main prepare checks the exact triggering checkout and returns the four Action outputs plus caller-owned Action SHA. Apply waits at the existing Environment, checks out that prepared configuration, verifies the handoff, then requests a token for this repository only. Merging a workflow change can start prepare and queue a waiting deployment; the owner must inspect the published Summary before approving it. Neither this guide nor a successful migration check approves a live apply.

## Review a pending run

Open the Actions run Summary before approving its waiting deployment (`gh-settings` here, `settings-apply` in the inert example). Check the original actor and initiating/rerun actor, event, run/attempt, workflow path/ref and exact workflow-source revision, target, configuration revision/file link, distinct Action repository/ref, pinned CLI identity, and observed before → desired after values. The workflow-source SHA and configuration SHA are separate identities; each link uses its displayed revision.

Do not approve a missing or incomplete review Summary. A successful prepare does not prove that GitHub published it. Detectable rendering/write failure fails prepare; no publication polling or additional approval mechanism is provided. Quoted settings are data, including apparent HTML, Markdown headings/fences, and workflow commands.

No differences and no managed fields are informational successful outcomes, not special diff exit codes. Direct apply sends declared values without reading current settings; intervening remote changes to managed fields can be overwritten. Omitted fields are outside the request. Approval does not save the observed plan or guarantee that it stays current.

PR concurrency cancels superseded PR runs. Main uses one repository-scoped group with `cancel-in-progress: false`; GitHub can replace a pending run. Ordinary reruns keep GitHub's approval behavior and show the rerun actor/attempt. There is no custom queue, drift check, freshness timer, stale-commit check, or reapproval controller.

## Failure and recovery

| Situation | Maintainer action |
| --- | --- |
| Missing/invalid config | Correct the declaration and rerun through normal review; validation fails before API access. |
| CLI download/checksum/version error | Inspect the fixed release identity and reviewed pin. Do not execute an unchecked binary or switch to source/latest. |
| Plan read/API permission error | Inspect ordinary read-token access and the target origin. Do not provide administration credentials to PR/prepare. |
| Failed prepare or rejected approval | Apply cannot proceed. Fix the cause or initiate a new eligible run; no rollback is implied. |
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
