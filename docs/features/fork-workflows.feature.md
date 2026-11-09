---
type: Feature
title: Public fork PR workflow approval
description: Explicit contributor approval for workflows from public fork pull requests.
---

## Scope and declaration

The [configuration contract](../settings.md#fork-pr-workflow-policies) owns the public contributor approval choice under `repository.settings.actions.general.approval_for_running_fork_pull_request_workflows_from_contributors`. It accepts the three UI approval policies. Omission is unmanaged; null, unknown keys and wrong types fail locally. This control is distinct from workflow token permissions and approving PR reviews. `pull_request_target` is outside this approval gate. Private fork execution/credential settings and sharing access are [excluded](../settings.md#excluded-private-actions-controls).

## Observation and reconciliation

Explicit `--fork-approval` requires complete typed public-repository observations. Ordinary export adds no request. Inapplicable, denied or incomplete reads preserve the destination. Complete unknown strings retain exact provenance and project to the named first-time-contributor fallback; missing or malformed values never supply policy.

Plan is read-only. Exact raw matching state skips a write; projected equality remains actionable. Apply rereads current policy before a changed PUT and independently verifies the raw desired value afterward. Retained unknown values cannot confirm a fallback match. Narrow payloads, redirect rejection, honest partial failures and no retry/rollback remain required. Summary escapes detailed values and status-only output suppresses policy/provenance.

## Verification boundaries

The [public adapter tests](../../internal/github/ui_actions_public_fork_test.go), [codec tests](../../internal/workspace/ui_actions_test.go), [CLI opt-ins](../../internal/cli/fork_workflows_test.go) and [executable normalization/lifecycle cases](../../tests/acceptance/normalization_fork_approval_test.go) cover this public control using local APIs and dummy credentials. [Excluded-control executable tests](../../tests/acceptance/actions_private_scope_test.go) check rejection before any API call and file preservation. These fixtures do not change live execution policy, credentials, grants or repository settings.
