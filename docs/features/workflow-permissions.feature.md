---
type: Feature
title: Default workflow permissions
description: Select independent UI workflow permission and PR approval controls.
---

Status: implemented in current source, supported. The [configuration contract](../settings.md#default-workflow-permissions) is authoritative.

## Rule: Declare independent optional native values

- Under `repository.settings.actions.general.workflow_permissions`, `default_permissions` accepts `read_and_write_permissions` or `read_repository_contents_and_packages_permissions`; `allow_github_actions_to_create_and_approve_pull_requests` accepts booleans.
- Omitted fields remain unmanaged; false is explicit. Null, unknown/case-varied enums and wrong types fail locally before API access.
- Either field can be declared alone without managing or querying execution policy. Empty declarations remain local no-ops.
- Superseded scalar `workflow_permissions`, flat PR approval and generic `selection` or `value` members are rejected rather than translated.

Test mapping: [Actions UI codec](../../internal/workspace/ui_actions_test.go), [literal page/section schema](../../internal/workspace/ui_page_sections_test.go), [TestWorkflowPermissionsErrorsAndOmission](../../internal/github/workflow_permissions_test.go), [TestWorkflowPermissionsExportPlanApplyAndSummary](../../tests/acceptance/workflow_permissions_test.go).

## Rule: Acquire complete policy only when requested

- Ordinary export independently attempts both workflow values and Actions execution policy; an unavailable family retains prior intent with info.
- Missing/null/malformed members and denied reads fail without invented defaults or replacing the destination. Complete correctly typed out-of-UI permission strings use the defined fallback and retain their raw provenance.
- Supported exact values roundtrip to zero differences. A normalized export does not prove raw equality. Workflow-only plan reads only its own policy endpoint.

Test mapping: [TestWorkflowPermissionsCompleteReads](../../internal/github/workflow_permissions_test.go), [TestWorkflowExportOptionsAndDuplicateRejection](../../internal/cli/export_test.go), [TestWorkflowPermissionsExportPlanApplyAndSummary](../../tests/acceptance/workflow_permissions_test.go).

## Rule: Show permission changes explicitly

- Public YAML choices translate to internal read/write tokens. Normalized comparisons use `repository.settings.actions.general.workflow_permissions.default_permissions` and UI choices, with a warning and write-required note even when the desired choice equals the fallback. Exact observations also use UI choices; API tokens are not YAML aliases. False-to-true remains a native boolean.
- The closed report registry validates both sides. Normalized equality requires consistent acquired raw provenance; unavailable before values and missing/null/malformed normalization metadata are invalid.
- Summary uses existing report data without credentials, origin resolution or API calls.

Test mapping: [TestWorkflowPermissionSummaryTypes](../../internal/actions/workflow_permissions_test.go), [TestWorkflowPermissionsExportPlanApplyAndSummary](../../tests/acceptance/workflow_permissions_test.go).

## Rule: Update only changed declared keys and preserve known progress

- Fresh complete preflight compares declared intent against the raw observation. Matching intent sends no PUT and records an unchanged operation. A normalized fallback is not raw equality.
- PUT contains only changed declared optional keys; no matching key, enabled/allowlist or inferred workflow companion is sent.
- Fresh complete readback verifies the original declaration, including members that initially matched, and preserved raw companions. Repeated matching apply performs only the preflight read.
- Workflow policy is a distinct operation after execution policy; organization conflicts and other failures retain earlier completed progress and never trigger rollback or permission expansion.
- A preflight read failure records a confirmed read failure before any PUT. PUT and readback errors retain uncertain write effects; wrapped errors preserve this distinction.
- Redirects, denied/conflicting writes, missing/malformed readback and mismatches fail honestly. An out-of-UI raw readback never confirms a declared permission, even when its fallback equals the request. An omitted permission is verified against its exact raw preflight companion and is not replayed or printed.

Test mapping: [matching regression](../../internal/github/workflow_permissions_reconcile_test.go), [changed-only payloads and verification failures](../../internal/github/workflow_permissions_test.go), [actual progress and prewrite failures](../../internal/app/workflow_permissions_test.go), [built CLI](../../tests/acceptance/workflow_permissions_test.go), [actual composite Action](../../tests/github-actions/composite_workflow_permissions_acceptance_test.go).

## Evidence boundary

Mutation tests use isolated local transports/HTTP fixtures and dummy credentials. A read-only live observation confirmed the two response fields; no live partial PUT, organization behavior, token/App grants, workflow permissions, Rulesets bypass or security configuration was changed or established by those tests. Publication requires separate authorization; no release or pin change is included.

## Feature: Retain provenance for out-of-UI workflow permissions

Repository maintainers can use a defined UI choice when a successful, complete
read returns a correctly typed permission token outside the UI menu, while
distinguishing that projection from independently observed remote equality.

### Rule: A fallback is a projection of an acquired value

#### Scenario: An unknown permission token is projected with its provenance

- Given a complete workflow policy with permission token `FUTURE` and approval disabled
- When the policy is converted to UI choices
- Then the permission choice is `read_repository_contents_and_packages_permissions`
- And the exact raw token, current YAML field and selected fallback remain available for a warning
- And approval remains disabled

#### Scenario: An exact permission choice does not require a fallback

- Given a complete workflow policy with permission token `write`
- When the policy is converted to UI choices
- Then the permission choice remains `read_and_write_permissions`
- And no normalization provenance is added

### Rule: A fallback cannot establish remote equality

#### Scenario: The requested choice equals the projected fallback

- Given a successful read returns an out-of-UI permission token
- And the requested choice is `read_repository_contents_and_packages_permissions`
- When a plan compares the policy
- Then the output retains the warning and does not claim independently observed zero drift
- When apply runs against isolated API fixtures
- Then it writes the explicitly requested permission and verifies a fresh raw readback
- And an unchanged out-of-UI readback fails with possible write effects retained

#### Scenario: An omitted permission remains unmanaged

- Given an out-of-UI permission token and an explicitly requested approval boolean
- When apply updates the approval boolean
- Then the payload contains only the approval boolean
- And readback verifies the preserved raw permission companion without replaying the fallback

Missing or null members, wrong types, denied requests and failed acquisition do
not trigger this projection. Export and apply warnings use stderr; plan shows the
warning with the comparison. Status-only Action output retains the changed
outcome without exposing policy values or raw diagnostics.

Test mapping: [raw projection and verification](../../internal/github/normalization_test.go),
[complete HTTP acquisition](../../internal/github/normalization_read_test.go),
[HTTP apply and uncertain effects](../../internal/github/normalization_apply_test.go),
[scoped export](../../internal/app/normalization_test.go),
[closed reports and safe presentation](../../internal/actions/normalization_test.go),
and [built-binary commands](../../tests/acceptance/normalization_test.go).
