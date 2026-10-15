---
type: ADR
title: Public composite Action contract
description: Expose a small verified CLI wrapper while keeping approval and credentials in the caller.
sources:
  - id: composite-metadata
    resource: https://docs.github.com/en/actions/reference/workflows-and-actions/metadata-syntax
  - id: cli-release
    resource: https://github.com/daiksud/gh-settings/releases/tag/v0.5.0
---

## Status

Accepted — contract only; implementation follows in separate PRs.

Narrowly supersedes the reusable-Action non-goal in [ADR 0022](0022-owner-approved-actions.md). Its owner approval, direct-apply, concurrency, and credential decisions remain in force. The [behavior specification](../features/actions.feature.md#feature-use-the-public-composite-action) defines the public inputs, outputs, and failure contract.

## Context

Consumers should be able to invoke the existing CLI from a reviewed root Action without copying this repository's example scripts. The published `v0.5.0` is a CLI release; its tree has no root `action.yml` and cannot serve as the proposed root Action. Existing immutable CLI tags and assets remain unchanged. No future release version or publication date is promised.[^cli-release]

## Decision

Provide a root `action.yml` with `runs.using: composite` for `uses: daiksud/gh-settings@<approved full commit SHA>`. That SHA selects the Action package, independently of the CLI version it installs. Composite metadata will declare inputs and map outputs to actual step outputs, use explicit Bash shells, and resolve packaged helpers through `github.action_path`. Runtime validation must enforce the command enum and conditional token requirement; metadata alone does not enforce them.[^composite-metadata]

The first version supports only a fresh GitHub-hosted Ubuntu 24.04 x64 job, a caller checkout rooted at `GITHUB_WORKSPACE`, and `.github/settings.yml` at that checkout root. Preserve the CLI's existing checkout-scoped GitHub `origin` resolution. The Action does not add configuration-path, target, CLI-version, download-URL, or arbitrary-argument inputs.

Pin CLI `v0.5.0`, asset `gh-settings-linux-amd64`, and SHA-256 `b4a63eb8912657deabc5ea8909a7e6198d375c94cb0a9cd83f7d2179f7bbf5b4`. Verify downloaded bytes before execution and require `--version` to report `gh-settings v0.5.0`. Use a per-invocation runner-temp executable; do not execute caller helper files or search for an installed CLI. The Action owns installation, bounded command execution, the plan Summary, and four non-secret success outputs.[^cli-release]

The caller owns checkout without persisted credentials, event/ref guards, Environment and owner approval, token minting/revocation, and concurrency. `validate` needs no authentication; `plan` requires an explicitly supplied read-only caller token; `apply` requires a caller-created App installation token limited to the approved target with Administration write. No implicit `github.token` or ambient authentication fallback is part of the contract.

Bind the token input as `GH_TOKEN` only for the plan/apply command step, never installation, Summary, metadata/output steps, `GITHUB_ENV`, files, logs, artifacts, or public outputs. This scopes environment exposure; a composite Action shares the caller's job and is not a process or credential sandbox. Callers must trust all code in a credential-bearing job and must not expose privileged credentials to untrusted PR code.

Active workflows retain Environment `gh-settings`, App `answer26`, `ANSWER26_CLIENT_ID`, and `ANSWER26_PRIVATE_KEY`. The inert examples retain their documented `settings-apply` Environment until a separately authorized adoption change. This contract does not migrate or activate workflows, modify Environments/Apps, issue credentials, or authorize live apply.

## Consequences and implementation order

The Summary identifies configuration commit, workflow revision, Action identity/ref, and CLI version/checksum separately. A diff remains informational: [direct apply](0003-direct-apply.md) can overwrite intervening changes to declared remote fields. There is no saved-plan, drift, atomicity, rollback, or custom reapproval guarantee.

Implement one Issue per PR in this order: [#107 contract](https://github.com/daiksud/gh-settings/issues/107), [#108 verified installer](https://github.com/daiksud/gh-settings/issues/108), [#109 bounded execution](https://github.com/daiksud/gh-settings/issues/109), [#110 escaped Summary](https://github.com/daiksud/gh-settings/issues/110), [#111 root metadata and outputs](https://github.com/daiksud/gh-settings/issues/111), [#112 hosted validate CI](https://github.com/daiksud/gh-settings/issues/112), [#113 inert PR example](https://github.com/daiksud/gh-settings/issues/113), [#114 inert main example](https://github.com/daiksud/gh-settings/issues/114), [#115 integration verification](https://github.com/daiksud/gh-settings/issues/115), and [#116 usage and bounded adoption guidance](https://github.com/daiksud/gh-settings/issues/116). Each implementation PR includes its focused tests; reassess remaining scope and dependencies after each merge. Operational migration and releases remain separate.

Revisit this bounded contract only when an actual consumer requirement changes supported runners, configuration placement, authentication ownership, or CLI capabilities; do not add a generic framework in anticipation.

[^cli-release]: Published CLI v0.5.0 release and Linux amd64 asset digest, checked against GitHub release metadata and the existing example installer on 2026-10-15.
[^composite-metadata]: GitHub composite metadata syntax, checked for input handling, composite output mappings, packaged script paths, and explicit shells on 2026-10-15.
