---
type: Feature
title: Repository retention policy
description: Configured days, complete read-only upper bounds, native comparisons and nontransactional updates.
---

## Status and scope

Implemented in current source, **supported**, for [#341](https://github.com/daiksud/gh-settings/issues/341). The [public configuration contract](../settings.md#check-run-status-artifact-and-log-retention) owns usage. No version, pin, grant or live setting changes with this feature.

The repository policy applies to new checks (suites/runs/annotations), workflow runs, third-party commit statuses, artifacts and logs. It is independent of Actions execution and does not enable it. Individual data deletion, workflow `retention-days`, managing organization/enterprise configuration and retroactive cleanup are outside scope. The [official API](https://docs.github.com/en/rest/actions/permissions#set-artifact-and-log-retention-settings-for-a-repository) specifies only `days` in PUT; GET additionally requires `maximum_allowed_days`. The [repository guide](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/enabling-features-for-your-repository/managing-github-actions-settings-for-a-repository) documents public/private ranges, upper policy, the October 2026 expansion and new-object-only application.

## Declaration and complete observation

`repository.settings.actions.general.check_workflow_run_status_artifact_and_log_retention` is an optional native integer in 1–400. Omission/empty mapping is unmanaged; zero, negative, null, float, string and other types fail locally. A lower current maximum narrows the valid desired range; read-only metadata cannot be declared or exported as intent. Ordinary export and existing opt-ins retain their requests; `--retention` composes independently and completes both integer reads before saving.

A zero observation is unacquired, not default 90/400 or zero. Both fields must be present with exact keys and integer types; incomplete, denied, malformed and redirected responses fail. Complete configured days may exceed a newly lowered maximum; observation does not invent a correlation or clamp the value. Desired comparisons/preflight reject above-cap intent, including matching values. Export fails when the current value cannot form a valid declaration rather than promising a zero-plan round-trip. An allowed lower declaration can reconcile a known over-cap current value.

Test mapping: [TestRetentionAcquisitionBoundsAndReadOnlyMaximum](../../internal/settings/retention_test.go), [Actions UI codec](../../internal/workspace/ui_actions_test.go), [TestRetentionReadsRequireBothIntegers](../../internal/github/retention_test.go), [TestRetentionExportOptInAndComposition](../../internal/cli/retention_test.go).

## Read-only plan and direct apply

Plan reads only this resource for a retention-only declaration and displays known native integers. Apply follows workflow permissions, performs complete read/current-cap preflight, then a fresh adapter read before its sole days-only PUT if needed. Matching state performs no mutation. Later resources stop on failure; no saved plan, transaction, lock, retry or rollback is introduced.

Post-write verification requires desired days and complete applicable upper metadata. A maximum changing independently is allowed if desired days remain within it; a newly restrictive/unknown maximum or wrong retained days fails verification with uncertain write effects. Pre-write read/preflight failures carry confirmed no-write certainty. Successful reads and matching/no-write callbacks do not create false prior-write progress; earlier actual resource writes remain confirmed on a later failure.

Test mapping: [TestRetentionFreshCapNarrowPUTAndReadback](../../internal/github/retention_test.go), [TestRetentionIndependentReadAndMatchingApply](../../internal/app/retention_test.go), [TestRetentionFailureDoesNotInventEarlierWrites](../../internal/app/retention_test.go).

## Native reports and process evidence

The private comparison protocol retains internal `actions.artifact_and_log_retention_days`, not a public YAML alias. Report reconstruction normalizes bounded whole JSON numbers to native integers; null/unavailable before values, fractions, strings, wrong fields and no-op changes are rejected. Other field/type decoding is unchanged. CLI and credential-free Summary share unquoted numeric output, while identifiers/bounds retain the existing private protocol. Reports remain presentation data and never enter apply.

Test mapping: [TestRetentionNativeReportNumbersAreClosed](../../internal/settings/retention_test.go), [TestRetentionSummaryHasNativeIntegerValues](../../internal/actions/retention_test.go), [TestRetentionExportPlanApplyAndSummary](../../tests/acceptance/retention_test.go). The process case covers ordinary/selected export→zero plan→edit→apply, upper-cap reduction, over-cap correction, policy refusal/unchanged readback, file preservation, GET/PUT redirect rejection, choice composition and retained earlier workflow writes. API owner fixtures separately permit a changing, still-compatible maximum.

All mutation tests use local modeled responses and dummy credentials. A read-only GET on this public repository returned days=90 and maximum_allowed_days=90. No actual PUT, data deletion, policy/grant change or release was performed; these tests do not establish live cleanup timing, mutation compatibility or other token grants.
