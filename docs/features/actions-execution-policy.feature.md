---
type: Feature
title: Actions execution policy
description: Select UI execution permission choices with complete reads and staged API writes.
---

Status: implemented in current source, supported. The [configuration contract](../settings.md#actions-execution-policy) is authoritative for users; this specification records scenarios and test mappings.

## Rule: Validate only explicit native values

- Given `repository.settings.actions.general` or any individual control is omitted, it remains unmanaged; empty mappings manage nothing.
- Explicit false and an empty patterns array remain declared values. Null, wrong types, unknown keys and unknown policy enums fail before remote work.
- Selected companions are `allow_actions_created_by_github`, `allow_actions_by_marketplace_verified_creators` and `allow_or_block_specified_actions_and_reusable_workflows`: two booleans and at most 1000 nonempty strings. Case, duplicates, order and `!` patterns are retained; GitHub owns matching.
- Companions can accompany only the selected non-owner Actions radio or an omitted radio. API-shaped aliases are rejected.

Test mapping: [Actions UI codec](../../internal/workspace/ui_actions_test.go), [TestActionsPatternsAreExactSequences](../../internal/settings/actions_policy_test.go), [TestActionsInvalidDeclarationsNeverContactAPI](../../tests/acceptance/actions_policy_test.go).

Public `actions_permissions` accepts `allow_all_actions_and_reusable_workflows`, `disable_actions`, `allow_owner_actions_and_reusable_workflows` or `allow_owner_and_select_non_owner_actions_and_reusable_workflows`. SHA pinning is the independent `require_actions_to_be_pinned_to_a_full_length_commit_sha` checkbox. Observation normalization/default warnings are separate #480 work; unknown choices currently fail.

## Rule: Export only complete observable policy

- Ordinary export does not instantiate an Actions client or require additional Administration read.
- `export --actions` reads a complete policy. It exports the chosen UI radio and SHA checkbox, with selected companions only when execution is enabled and policy selected. Disabled/other choices omit hidden companions while complete acquired raw state remains internal; no inactive value is invented.
- Options compose with `--rulesets` in either order. Duplicate/unknown flags fail before export.
- Missing/null/unsupported fields, permission errors and incomplete selected reads fail before saving, including when Actions are disabled. Exported values roundtrip to zero differences.

Test mapping: [TestActionsReadIsCompleteAndSelectedIsConditional](../../internal/github/actions_policy_test.go), [TestExportPolicyFlagsComposeAndRejectDuplicates](../../internal/cli/actions_policy_test.go), [TestActionsPolicyExportTransitionAndReconciliation](../../tests/acceptance/actions_policy_test.go).

## Rule: Plan inactive values honestly

- Plan reads only managed resources and never writes.
- A transition from nonselected to selected requires all three selected values. Each inactive before value is unavailable, not an invented false or empty array.
- Partial selected declarations require current selected policy and a complete selected read.
- CLI and Summary retain native component comparisons pending #481; the private same-version report retains raw identifiers/types. One UI radio can expand to two managed API members. Summary escapes strings and needs no API access.
- Reports reject wrong types/enums, duplicate/no-op changes and incomplete/forged unavailable transitions. A failed later read discards the entire comparison.

Test mapping: [TestActionsPolicyObservationAndTransition and TestActionsReportRejectsInvalidAndForgedUnknownValues](../../internal/settings/actions_policy_test.go), [TestActionsNativePlanAndSummaryUnavailableValues](../../internal/actions/actions_policy_test.go), [TestActionsReadFailuresDiscardPlansAndKeepEarlierWrites](../../tests/acceptance/actions_policy_test.go).

## Rule: Preflight and preserve companions before ordered writes

- Apply reads current state and preflights the whole Actions resource before writing. Incomplete transitions fail without Actions writes.
- A selected allow radio explicitly declares enabled=true plus its policy; Disable declares only false. An omitted radio leaves execution unmanaged. Base PUT echoes freshly read internal `enabled` when needed because the endpoint requires it. Other base fields are declared only. Selected PUT merges omitted companions from a complete active read.
- Transition policy is written before the selected endpoint that requires it. When the selected radio enables a disabled target, the exact stages are disabled+selected base policy, selected companions, then enabled-only update. Omitted enabling is never inferred.
- Requested values and selected companions sent are read back. Direct repeated apply still writes; it is not saved-plan execution or an optimistic concurrency check.
- Writes require 204. HTTP/organization errors, redirects and failed/mismatching readback fail honestly. Earlier policy/resource writes remain applied or uncertain; no rollback or automatic grants occur.

Test mapping: [TestActionsApplyPreservesCompanionsAndOrdersWrites and TestActionsApplyPreflightAndFailures](../../internal/github/actions_policy_test.go), [TestActionsPolicyExportTransitionAndReconciliation and TestActionsReadFailuresDiscardPlansAndKeepEarlierWrites](../../tests/acceptance/actions_policy_test.go).

## Evidence boundary

All mutation tests use local transports or Unix-socket HTTP fixtures with dummy credentials. A read-only repository policy observation confirmed the response fields; it did not test live disabled/selected transitions, organization constraints, private-pattern enforcement, token grants or writes. No live policy, workflow, permission, version, release or pin is changed by this implementation.
