# Feature: Set approval policy for workflows from public forks

## Rule: Verify the configured contributor approval policy

### Scenario: An unknown API value projects to the declared choice

- Given a public repository returns an unknown approval policy and the declaration selects `require_approval_for_first_time_contributors`
- When the maintainer runs `gh settings plan`
- Then the plan reports the original value and a required update despite the matching UI projection
- And no policy is changed by the plan

## Scope and declaration

The [configuration contract](../../references/settings/actions/general.md#fork-pr-workflow-policies) owns the public contributor approval choice under `repository.settings.actions.general.approval_for_running_fork_pull_request_workflows_from_contributors`. It accepts the three UI approval policies. Omission is unmanaged; null, unknown keys and wrong types fail locally. This control is distinct from workflow token permissions and approving PR reviews. `pull_request_target` is outside this approval gate. Private fork execution/credential settings and sharing access are [excluded](../../references/settings/actions/general.md#excluded-private-actions-controls).

## Observation and reconciliation

Ordinary `export` attempts this supported family automatically under the [readable-settings contract](readable-settings-export.feature.md). Complete observations replace prior values; unavailable/incomplete/unsupported observations emit info and retain prior valid declarations. Plan/apply keep their strict declared-state and mutation contracts.

Plan is read-only. Exact raw matching state skips a write; projected equality remains actionable. Apply rereads current policy before a changed PUT and independently verifies the raw desired value afterward. Retained unknown values cannot confirm a fallback match. Narrow payloads, redirect rejection, honest partial failures and no retry/rollback remain required. Summary escapes detailed values and status-only output suppresses policy/provenance.

## Verification boundaries

The [public adapter tests](../../../internal/github/ui_actions_public_fork_test.go), [codec tests](../../../internal/workspace/ui_actions_test.go), [Default export interface](../../../internal/cli/export_test.go) and [executable normalization/lifecycle cases](../../../tests/acceptance/normalization_fork_approval_test.go) cover this public control using local APIs and dummy credentials. [Excluded-control executable tests](../../../tests/acceptance/actions_private_scope_test.go) check rejection before any API call and file preservation. These fixtures do not change live execution policy, credentials, grants or repository settings.
