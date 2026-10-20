# Default workflow token and approval permissions

Status: implemented in current source, unreleased. Published v0.16.0 and older CLI/Action packages reject the fields and export option. The [configuration contract](../settings.md#default-workflow-permissions-unreleased) is authoritative.

## Rule: Declare independent optional native values

- `settings.actions.default_workflow_permissions` accepts exact read/write strings; `can_approve_pull_request_reviews` accepts booleans.
- Omitted fields remain unmanaged; false is explicit. Null, unknown/case-varied enums and wrong types fail locally before API access.
- Either field can be declared alone without managing or querying execution policy. Empty declarations remain local no-ops.

Test mapping: [TestWorkflowPermissionsStrictYAMLRoundtrip](../../internal/workspace/workflow_permissions_test.go), [TestWorkflowPermissionsErrorsAndOmission](../../internal/github/workflow_permissions_test.go), [TestWorkflowPermissionsExportPlanApplyAndSummary](../../tests/acceptance/workflow_permissions_test.go).

## Rule: Acquire complete policy only when requested

- Ordinary export and `--actions` retain their existing scope. `--workflow-permissions` independently includes both complete values and composes with other opt-ins.
- Missing/null/unknown fields and denied reads fail without invented defaults or replacing the destination.
- Exported values roundtrip to zero differences. Workflow-only plan reads only its own policy endpoint.

Test mapping: [TestWorkflowPermissionsCompleteReads](../../internal/github/workflow_permissions_test.go), [TestWorkflowExportOptionsAndDuplicateRejection](../../internal/cli/actions_policy_test.go), [TestWorkflowPermissionsExportPlanApplyAndSummary](../../tests/acceptance/workflow_permissions_test.go).

## Rule: Show permission changes explicitly

- Read-to-write and false-to-true changes remain native enums and booleans in CLI/report/Summary.
- The closed report registry validates both sides; workflow values cannot use unavailable before values.
- Summary uses existing report data without credentials, origin resolution or API calls.

Test mapping: [TestWorkflowPermissionSummaryTypes](../../internal/actions/workflow_permissions_test.go), [TestWorkflowPermissionsExportPlanApplyAndSummary](../../tests/acceptance/workflow_permissions_test.go).

## Rule: Update only declared keys and preserve known progress

- Fresh complete preflight precedes a PUT containing only declared optional keys; no enabled/allowlist or inferred workflow companion is sent.
- Fresh complete readback verifies desired and preserved companion values. Direct repeated apply still writes.
- Workflow policy is a distinct operation after execution policy; organization conflicts and other failures retain earlier completed progress and never trigger rollback or permission expansion.
- A preflight read failure records a confirmed read failure before any PUT. PUT and readback errors retain uncertain write effects; wrapped errors preserve this distinction.
- Redirects, denied/conflicting writes, unknown/missing readback and mismatches fail honestly.

Test mapping: [TestWorkflowPermissionsDeclaredOnlyPUTAndCompanionVerification and TestWorkflowPermissionsErrorsAndOmission](../../internal/github/workflow_permissions_test.go), [TestWorkflowPermissionFailureRetainsEarlierExecutionProgress and TestWorkflowPreflightReadFailureHasNoUncertainWrite](../../internal/app/workflow_permissions_test.go), [TestWorkflowPermissionsExportPlanApplyAndSummary](../../tests/acceptance/workflow_permissions_test.go).

## Evidence boundary

Mutation tests use isolated local transports/HTTP fixtures and dummy credentials. A read-only live observation confirmed the two response fields; no live partial PUT, organization behavior, token/App grants, workflow permissions, Rulesets bypass or security configuration was changed or established by those tests. Publication requires separate authorization; no release or pin change is included.
