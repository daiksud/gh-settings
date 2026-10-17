---
type: ADR
title: Owner-approved settings workflows
description: Keep PR plans read-only and gate direct main apply with owner approval.
---

## Status

Accepted for owner approval, direct apply, concurrency and credential ownership. [ADR 0023](0023-public-composite-action.md) supersedes the original example-only delivery boundary with a public composite Action; operational callers now use that released package. The decision below records the original example delivery. Its shell wrappers and historical fixture combinations are not permanent compatibility obligations under the [v0 simplification contract](../concept.md#v0-simplification-contract). Operational setup, publication, adoption and deployment approval were separate operations for this original design. Active workflows later adopted a [trusted PR Environment guard](../github-actions.md#active-repository-environment-guard), requiring the guard's `check` in the branch Ruleset before disabling the reviewer gate described below.

## Context

The CLI already validates, observes differences, and applies declared values directly. Actions integration should compose those commands without changing their semantics or enabling this repository to manage itself merely by merging examples.

## Decision

Deliver two workflow examples and their small support files under `examples/github-actions/`, outside `.github/workflows`. Use fresh GitHub-hosted Linux runners for every job; shared persistent self-hosted runners are outside this setup.

The PR example uses `pull_request`, checkout with unpersisted credentials, and explicit read permissions. It validates before GET-only plan and never references the protected Environment or App credentials. Per-PR concurrency cancels superseded runs.

The main example triggers only on `push` to Branch `main`, with explicit event/ref guards. Prepare checks out the triggering configuration commit, validates, plans with the ordinary read token, and writes the review Summary. Apply depends on successful prepare and waits at the fixed `settings-apply` Environment before obtaining its private key. Both jobs check out that same configuration commit and verify the same committed fixed CLI version, asset, and SHA-256. External Actions use full commit SHAs. No PR binary or plan artifact is promoted.

The dedicated App is installed only on selected approved repositories with Administration write and unavoidable basic permissions. Its official token Action requests one containing target repository and Administration write, retains normal post-job revocation, and exposes the masked step token only to apply. Neither key nor token is a job/workflow output.

The Environment requires the owner as reviewer, permits self-approval, disables administrator bypass, and allows only a Branch rule named `main`, with no Tag rule. An Environment is not a workflow allowlist: all eligible workflows, helpers, Actions, and CLI pins require review. Setup, credential entry, activation, and live apply remain separate authorized operations.

The Summary shows original and triggering/rerun actors, event, run/attempt, workflow path/ref and its own exact workflow SHA, target repository, configuration SHA and exact file link, CLI version/checksum, and observed before → desired after values. Workflow SHA and configuration SHA are distinct identities. Values are untrusted text. The owner must not approve missing or incomplete Summary content; successful prepare alone does not prove GitHub published it. Detectable render/write failure fails prepare; no publication polling is added.

Main uses one repository-scoped concurrency group with `cancel-in-progress: false`. Ordinary GitHub pending-run replacement applies. Ordinary reruns display their actor and attempt and follow platform approval behavior. No custom queue, stale-commit, freshness, or reapproval mechanism is added.

## Consequences

Failed or cancelled prepare blocks apply. Waiting or rejected approval makes no update. Apply failures propagate; success is reported only for a successful CLI invocation. There is no atomicity or automatic rollback guarantee.

The observed diff is informational. [Direct apply](0003-direct-apply.md) can overwrite intervening remote changes to declared fields. Saved plans, drift checks, additional approval machinery, a reusable Action framework, and releases are outside scope. See the [behavior specification](../features/actions.feature.md).
