---
type: Feature
title: Actions execution policy
description: Select UI execution permission choices with complete reads and staged API writes.
---

Status: implemented in current source, supported. The [configuration contract](../settings.md#actions-execution-policy) is authoritative for users; this specification records scenarios and test mappings.

Ordinary `export` now follows the [automatic readable-settings contract](readable-settings-export.feature.md): observe all supported families without selectors, retain unavailable declarations with info, and keep plan/apply strict.

## Rule: Validate only explicit native values

- Given `repository.settings.actions.general.actions_permissions` or any individual control is omitted, it remains unmanaged; empty mappings manage nothing.
- Explicit false and an empty patterns array remain declared values. Null, wrong types, unknown keys and unknown policy enums fail before remote work.
- Selected companions within that section are `allow_actions_created_by_github`, `allow_actions_by_marketplace_verified_creators` and `allow_or_block_specified_actions_and_reusable_workflows`: two booleans and at most 1000 nonempty strings. Case, duplicates, order and `!` patterns are retained; GitHub owns matching.
- Companions can accompany only the selected non-owner `policy` choice or an omitted policy. API-shaped aliases, scalar permission sections, flat companions, generic `selection` and a second `enabled` flag are rejected.

Test mapping: [Actions UI codec](../../internal/workspace/ui_actions_test.go), [literal page/section schema](../../internal/workspace/ui_page_sections_test.go), [TestActionsPatternsAreExactSequences](../../internal/settings/actions_policy_test.go), [TestActionsInvalidDeclarationsNeverContactAPI](../../tests/acceptance/actions_policy_test.go).

Public `actions_permissions.policy` accepts `allow_all_actions_and_reusable_workflows`, `disable_actions`, `allow_owner_actions_and_reusable_workflows` or `allow_owner_and_select_non_owner_actions_and_reusable_workflows`. SHA pinning is the independent `require_actions_to_be_pinned_to_a_full_length_commit_sha` checkbox in the same section. Complete typed unknown execution policies follow the [visible/latent normalization contract](../settings.md#execution-policy-observation-normalization); desired unknown choices still fail.

## Rule: Export only complete observable policy

- Ordinary export attempts Actions execution policy with its existing required permissions. Unavailable policy emits info and retains prior intent.
- `export` reads a complete policy. It exports the chosen UI policy and SHA checkbox in `actions_permissions`, with selected companions only when execution is enabled and policy selected. Disabled/other choices omit hidden companions while complete acquired raw state remains internal; no inactive value is invented.
- Export has no selection flags; retired selectors fail before API work.
- Missing/null/wrongly typed fields, permission errors and incomplete selected reads emit info and retain prior execution intent, including when Actions are disabled. Exact supported exported values roundtrip to zero differences; a visible normalized fallback still requires a write.

Test mapping: [TestActionsReadIsCompleteAndSelectedIsConditional](../../internal/github/actions_policy_test.go), [TestExportPolicyFlagsComposeAndRejectDuplicates](../../internal/cli/export_test.go), [TestActionsPolicyExportTransitionAndReconciliation](../../tests/acceptance/actions_policy_test.go).

## Rule: Plan inactive values honestly

- Plan reads only managed resources and never writes.
- A transition from nonselected to selected requires all three selected values. Each inactive before value is unavailable, not an invented false or empty array.
- Partial selected declarations require current selected policy and a complete selected read.
- Normalized execution uses one UI policy decision with private provenance. Exact CLI and Summary entries also use UI choices at `repository.settings.actions.general.actions_permissions.policy`; the private same-version report retains raw identifiers/types. One UI radio can expand to two managed API members. Summary escapes strings and needs no API access.
- Reports reject wrong types/enums, duplicate/no-op changes and incomplete/forged unavailable transitions. A failed later read discards the entire comparison.

Test mapping: [TestActionsPolicyObservationAndTransition and TestActionsReportRejectsInvalidAndForgedUnknownValues](../../internal/settings/actions_policy_test.go), [TestActionsNativePlanAndSummaryUnavailableValues](../../internal/actions/actions_policy_test.go), [TestActionsReadFailuresDiscardPlansAndKeepEarlierWrites](../../tests/acceptance/actions_policy_test.go).

## Rule: Preflight and preserve companions before ordered writes

- Apply reads current state and preflights the whole Actions resource before writing. Incomplete transitions fail without Actions writes.
- A selected allow policy explicitly declares enabled=true plus its policy; Disable declares only false. An omitted policy leaves execution unmanaged. Base PUT echoes freshly read internal `enabled` when needed because the endpoint requires it. Matching optional base fields are omitted. Selected PUT merges omitted companions from a complete active read and is skipped when all selected intent matches.
- Transition policy is written before the selected endpoint that requires it. When the selected policy enables a disabled target, needed stages remain disabled+selected base policy, selected companions, then enabled-only update. Skip an already selected base stage and already matching selected companions; enable only after all required selection work succeeds. Omitted enabling is never inferred.
- Exact raw requested or preserved policy, enabled/SHA booleans and managed selected companions are read back. Complete raw matches perform no PUT. Repeated matching apply reads current state without rewriting it; it is not saved-plan execution or an optimistic concurrency check.
- Writes require 204. HTTP/organization errors, redirects and failed/mismatching readback fail honestly. Earlier policy/resource writes remain applied or uncertain; no rollback or automatic grants occur.

Test mapping: [TestActionsApplyPreservesCompanionsAndOrdersWrites and TestActionsApplyPreflightAndFailures](../../internal/github/actions_policy_test.go), [TestActionsPolicyExportTransitionAndReconciliation and TestActionsReadFailuresDiscardPlansAndKeepEarlierWrites](../../tests/acceptance/actions_policy_test.go).

Test mapping for reconciliation: [raw matching and stage selection](../../internal/github/actions_execution_reconcile_test.go), [honest progress](../../internal/app/actions_execution_reconcile_test.go), [actual supplied-product Action](../../tests/github-actions/actions_execution_acceptance_test.go).

## Evidence boundary

All mutation tests use local transports or Unix-socket HTTP fixtures with dummy credentials. A read-only repository policy observation confirmed the response fields; it did not test live disabled/selected transitions, organization constraints, private-pattern enforcement, token grants or writes. No live policy, workflow, permission, version, release or pin is changed by this implementation.

## Rule: Retain visible fallback and latent raw policy separately

- Given a complete enabled policy with unknown, empty or whitespace `allowed_actions`, project Allow all with exact raw provenance, preserving acquired enabled and SHA values. Pure reads stay silent; export and complete managed policy updates warn for each visible fresh projection.
- A declared policy remains actionable even when its desired choice equals the fallback. One closed radio decision owns one native field for Disable or two for enabling; selected transitions add three unavailable companions, totaling five.
- Given disabled execution and unknown policy, export/plan show known Disable without raw disclosure or a normalized-before label. A matching Disable plan has zero differences. Enabling explicitly selects a canonical policy; hidden unknown policy cannot be activated by enabled-only intent.
- Disable and SHA-only updates preserve exact raw policy. Omitted policy and selected settings remain unmanaged and undisclosed. Selected transitions preserve configure-before-enable order and exact ordered patterns.
- An ignored fallback write, changed unowned raw token or drifted boolean fails fresh verification. Earlier REST effects remain reported; malformed acquisition makes no Actions write and preserves export/source files. Canonical repeated apply performs no PUT. An already disabled target with unknown latent policy also skips a matching Disable request.
- Detailed Summary escapes the same UI decision; private reports reject forged/overlapping entries, while status-only suppresses raw/desired policy values. Other families follow their documented normalization policies.

Test mapping: [complete typed HTTP acquisition](../../internal/github/normalization_actions_execution_read_test.go), [exact HTTP mutations and readback](../../internal/github/normalization_actions_execution_apply_test.go), [closed radio](../../internal/settings/actions_execution_change_test.go), [private report validation](../../internal/actions/normalization_actions_execution_test.go), [export visibility](../../internal/app/normalization_actions_execution_export_test.go), [executable commands](../../tests/acceptance/normalization_actions_execution_test.go).
