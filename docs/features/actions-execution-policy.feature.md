# Actions execution policy

Status: implemented in current source, published in v0.17.0. Published v0.16.0 and older CLI/Action packages reject the group and export option. The [configuration contract](../settings.md#actions-execution-policy) is authoritative for users; this specification records scenarios and test mappings.

## Rule: Validate only explicit native values

- Given `settings.actions` or any individual field is omitted, it remains unmanaged; empty mappings manage nothing.
- Explicit false and an empty patterns array remain declared values. Null, wrong types, unknown keys and unknown policy enums fail before remote work.
- The selected mapping accepts only the two booleans and a string sequence of at most 1000 nonempty entries. Case, duplicates, order and `!` patterns are retained; GitHub owns matching.
- A selected declaration cannot accompany an explicit all/local-only policy.

Test mapping: [TestActionsYAMLStrictnessAndRoundtrip](../../internal/workspace/actions_policy_test.go), [TestActionsPatternsAreExactSequences](../../internal/settings/actions_policy_test.go), [TestActionsInvalidDeclarationsNeverContactAPI](../../tests/acceptance/actions_policy_test.go).

## Rule: Export only complete observable policy

- Ordinary export does not instantiate an Actions client or require additional Administration read.
- `export --actions` reads a complete policy. It includes selected settings only when selected; it never substitutes false or empty for unobservable inactive settings.
- Options compose with `--rulesets` in either order. Duplicate/unknown flags fail before export.
- Missing/null/unsupported fields, permission errors and incomplete selected reads fail before saving, including when Actions are disabled. Exported values roundtrip to zero differences.

Test mapping: [TestActionsReadIsCompleteAndSelectedIsConditional](../../internal/github/actions_policy_test.go), [TestExportPolicyFlagsComposeAndRejectDuplicates](../../internal/cli/actions_policy_test.go), [TestActionsPolicyExportTransitionAndReconciliation](../../tests/acceptance/actions_policy_test.go).

## Rule: Plan inactive values honestly

- Plan reads only managed resources and never writes.
- A transition from nonselected to selected requires all three selected values. Each inactive before value is unavailable, not an invented false or empty array.
- Partial selected declarations require current selected policy and a complete selected read.
- CLI and native Action reports show each managed field. Summary uses the same comparison lines, escapes strings and needs no API access.
- Reports reject wrong types/enums, duplicate/no-op changes and incomplete/forged unavailable transitions. A failed later read discards the entire comparison.

Test mapping: [TestActionsPolicyObservationAndTransition and TestActionsReportRejectsInvalidAndForgedUnknownValues](../../internal/settings/actions_policy_test.go), [TestActionsNativePlanAndSummaryUnavailableValues](../../internal/actions/actions_policy_test.go), [TestActionsReadFailuresDiscardPlansAndKeepEarlierWrites](../../tests/acceptance/actions_policy_test.go).

## Rule: Preflight and preserve companions before ordered writes

- Apply reads current state and preflights the whole Actions resource before writing. Incomplete transitions fail without Actions writes.
- Base PUT echoes freshly read `enabled` when it is omitted, because the endpoint requires it. Other base fields are declared only. Selected PUT merges omitted companions from a complete active read.
- Transition policy is written before the selected endpoint that requires it. A currently disabled repository is enabled only after requested selection values are installed; omitted enabling is never inferred.
- Requested values and selected companions sent are read back. Direct repeated apply still writes; it is not saved-plan execution or an optimistic concurrency check.
- Writes require 204. HTTP/organization errors, redirects and failed/mismatching readback fail honestly. Earlier policy/resource writes remain applied or uncertain; no rollback or automatic grants occur.

Test mapping: [TestActionsApplyPreservesCompanionsAndOrdersWrites and TestActionsApplyPreflightAndFailures](../../internal/github/actions_policy_test.go), [TestActionsPolicyExportTransitionAndReconciliation and TestActionsReadFailuresDiscardPlansAndKeepEarlierWrites](../../tests/acceptance/actions_policy_test.go).

## Evidence boundary

All mutation tests use local transports or Unix-socket HTTP fixtures with dummy credentials. A read-only repository policy observation confirmed the response fields; it did not test live disabled/selected transitions, organization constraints, private-pattern enforcement, token grants or writes. No live policy, workflow, permission, version, release or pin is changed by this implementation.
