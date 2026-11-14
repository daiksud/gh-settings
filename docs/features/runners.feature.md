# Existing repository runner custom labels

The [configuration reference](../settings.md#existing-runner-custom-labels) owns
the supported contract. These scenarios describe existing-runner label management,
not runner lifecycle, credentials, host installation or workflow execution.

```gherkin
Feature: Custom labels on selected existing repository runners
  Scenario: Validate explicit selected sets
    Given runner entries require unique positive IDs and explicit custom-label lists
    Then nulls, unknown fields, invalid names and more than 100 labels fail locally
    And duplicate Unicode simple-fold identities fail without trimming or rewriting
    And omitted runners and an empty runner list make no family requests
    And an empty custom-label list explicitly clears that selected runner's labels

  Scenario: Export complete acquired runner labels independently
    Given a complete stable paginated inventory and matching detail and label reads
    When export acquires all runners
    Then only runner IDs and custom labels replace this family
    And an empty inventory exports an empty list
    But any incomplete, denied, ambiguous or inconsistent observation retains prior intent
    And an initial unavailable family is omitted while readable siblings continue
    And diagnostic output failure prevents saving

  Scenario: Prepare all selected runners before a family write
    Given every selected ID must resolve in the original personal-public repository
    When any desired custom name collides with a read-only label
    Then no runner-label PUT occurs
    And no runner is adopted by name, created, deleted or pruned

  Scenario: Reconcile and independently verify
    Given apply refreshes repository identity, same-ID detail and labels
    When current custom labels match by case-insensitive set
    Then no PUT occurs
    Otherwise PUT sends only the complete declared custom set
    And HTTP 200 acknowledgment and fresh independent GETs must confirm the requested set
    And fresh name, OS and exact readonly names/types/known IDs remain unchanged
    And detail and listing contribute the strongest optional ID evidence
    And busy, status and software metadata are not drift conditions

  Scenario: Preserve honest failures and private evidence
    Given an earlier runner update was independently verified
    When a later fresh read or post-PUT verification fails
    Then confirmed earlier changes remain applied and later work stops
    And pre-write failure is distinct from uncertain attempted effects
    And there is no retry, rollback, ID recovery or input-file writeback
    And detailed plans show selected IDs and custom labels
    But status-only output, raw diagnostics and reports reveal no private snapshots
```

Mapped tests: `TestUIRunnerLabelsRoundTrip`, `TestUIRunnerLabelsBoundary`,
`TestRunnerLabelsOwnershipAndIdentity`, `TestRunnerLabelsCompleteRead`,
`TestRunnerLabelsFreshConditionalWriteAndReadback`,
`TestRunnerInventoryPaginationAndSelectionIsolation`, `TestRunnerPlanAndEmptyOwnership`,
`TestRunnerApplyPreparationAndProgress`, `TestRunnerExportRetentionAndDiagnostics`,
`TestRunnerUnarchiveBeforeFamilyPreparation`, `TestRunnerReportClosedBoundary`,
`TestRunnerReportExactNumericIDsAndOuterShape`,
`TestRunnerLabelsLifecycle`, and `TestBuiltCompositeRunnerLabels`.
