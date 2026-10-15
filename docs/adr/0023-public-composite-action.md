---
type: ADR
title: Public composite Action contract
description: Expose a small verified CLI wrapper while keeping approval and credentials in the caller.
sources:
  - id: composite-metadata
    resource: https://docs.github.com/en/actions/reference/workflows-and-actions/metadata-syntax
  - id: cli-release
    resource: https://github.com/daiksud/gh-settings/releases/tag/v0.5.0
  - id: cli-release-v060
    resource: https://github.com/daiksud/gh-settings/releases/tag/v0.6.0
---

## Status

Accepted — implemented through the focused [#108–#115 sequence](https://github.com/daiksud/gh-settings/issues/107). See the [usage guide](../github-actions.md) for reviewed Action pins and the separate operational-adoption boundary.

Narrowly supersedes the reusable-Action non-goal in [ADR 0022](0022-owner-approved-actions.md). Its owner approval, direct-apply, concurrency, and credential decisions remain in force. The [behavior specification](../features/actions.feature.md#public-composite-action-contract) defines the public inputs, outputs, and failure contract.

## Context

Consumers should be able to invoke the existing CLI from a reviewed root Action without copying this repository's example scripts. The published `v0.5.0` is a CLI release; its tree has no root `action.yml` and cannot serve as the proposed root Action. Existing immutable CLI tags and assets remain unchanged. No future release version or publication date is promised.[^cli-release]

## Decision

Provide a root `action.yml` with `runs.using: composite` for `uses: daiksud/gh-settings@<approved full commit SHA>`. That SHA selects the Action package and its committed `action/cli-version`. Future release tags must use that same version; a strict version-tag invocation rejects disagreement. Composite metadata will declare inputs and map outputs to actual step outputs, use explicit Bash shells, and resolve packaged helpers through `github.action_path`. Runtime validation must enforce the command enum and conditional token requirement; metadata alone does not enforce them.[^composite-metadata]

The first version supports only a fresh GitHub-hosted Ubuntu 24.04 x64 job, a caller checkout rooted at `GITHUB_WORKSPACE`, and `.github/settings.yml` at that checkout root. Preserve the CLI's existing checkout-scoped GitHub `origin` resolution. The Action does not add configuration-path, target, CLI-version, download-URL, or arbitrary-argument inputs.

Record an explicit CLI release in `action/cli-version`, currently the already-published `v0.6.0`. Fetch only that version's `gh-settings-linux-amd64` and `SHA256SUMS` from fixed upstream HTTPS release paths. Require one valid checksum for the exact asset, verify bytes before execution, and require `--version` to report the selected version. A SHA pin fixes the version; the checksum trust source is the same GitHub release over HTTPS, rather than a digest embedded in the Action source. This avoids checksum/source circularity and runtime release discovery. No latest fallback or caller version/URL override is allowed. Use a per-invocation runner-temp executable; do not execute caller helper files or search for an installed CLI. The Action owns installation, bounded command execution, the plan Summary, and four non-secret success outputs.[^cli-release-v060]

The initial implementation separately pinned CLI v0.5.0, including in published Action v0.6.0. The [release-alignment correction](https://github.com/daiksud/gh-settings/issues/127) replaces that decision for current source: release publication must match the frozen source's committed CLI version before any tag/release creation. Published tags/assets, including the v0.6.0 limitation, remain immutable. The historical examples retain their reviewed v0.5.0 identity. Future release preparation changes the file through review after the owner explicitly chooses a version; no new release is authorized by this correction.

The caller owns checkout without persisted credentials, event/ref guards, Environment and owner approval, token minting/revocation, and concurrency. `validate` needs no authentication; `plan` requires an explicitly supplied read-only caller token; `apply` requires a caller-created App installation token limited to the approved target with Administration write. No implicit `github.token` or ambient authentication fallback is part of the contract.

Bind the token input as `GH_TOKEN` only for the plan/apply command step, never installation, Summary, metadata/output steps, `GITHUB_ENV`, files, logs, artifacts, or public outputs. This scopes environment exposure; a composite Action shares the caller's job and is not a process or credential sandbox. Callers must trust all code in a credential-bearing job and must not expose privileged credentials to untrusted PR code.

Active workflows retain Environment `gh-settings`, App `answer26`, `ANSWER26_CLIENT_ID`, and `ANSWER26_PRIVATE_KEY`. The inert examples retain their documented `settings-apply` Environment until a separately authorized adoption change. This contract does not migrate or activate workflows, modify Environments/Apps, issue credentials, or authorize live apply.

## Consequences and implementation order

The Summary identifies configuration commit, workflow revision, Action identity/ref, and CLI version/checksum separately. A diff remains informational: [direct apply](0003-direct-apply.md) can overwrite intervening changes to declared remote fields. There is no saved-plan, drift, atomicity, rollback, or custom reapproval guarantee.

Implement one Issue per PR in this order: [#107 contract](https://github.com/daiksud/gh-settings/issues/107), [#108 verified installer](https://github.com/daiksud/gh-settings/issues/108), [#109 bounded execution](https://github.com/daiksud/gh-settings/issues/109), [#110 escaped Summary](https://github.com/daiksud/gh-settings/issues/110), [#111 root metadata and outputs](https://github.com/daiksud/gh-settings/issues/111), [#112 hosted validate CI](https://github.com/daiksud/gh-settings/issues/112), [#113 inert PR example](https://github.com/daiksud/gh-settings/issues/113), [#114 inert main example](https://github.com/daiksud/gh-settings/issues/114), [#115 integration verification](https://github.com/daiksud/gh-settings/issues/115), and [#116 usage and bounded adoption guidance](https://github.com/daiksud/gh-settings/issues/116). Each implementation PR includes its focused tests; reassess remaining scope and dependencies after each merge. Operational migration and releases remain separate.

Revisit this bounded contract only when an actual consumer requirement changes supported runners, configuration placement, authentication ownership, or CLI capabilities; do not add a generic framework in anticipation.

[^cli-release]: Published CLI v0.5.0 release and Linux amd64 asset digest, checked against GitHub release metadata and the existing example installer on 2026-10-15.
[^composite-metadata]: GitHub composite metadata syntax, checked for input handling, composite output mappings, packaged script paths, and explicit shells on 2026-10-15.

[^cli-release-v060]: Published CLI v0.6.0 release, manifest, SHA256SUMS and Linux amd64 asset digest checked on 2026-10-15. The release tag retains the earlier Action installer; this decision changes current source only.
