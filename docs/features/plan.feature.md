---
type: Feature
title: Plan repository settings
description: Observe differences for declared repository fields without mutation.
---

## Feature: Plan declared settings

### Rule: Validate before reading GitHub

- Read `.github/settings.yml` from the Git root, including from a subdirectory.
- Reject missing files and invalid configuration before API access.
- For declared fields, resolve the same GitHub.com origin as export and ignore `GH_REPO`.
- When any REST-backed repository field is declared, read the complete six-field REST state, including `has_discussions`, using the existing GET normalization and validation. Merge-related fields use the selective [GraphQL read](merge-methods.feature.md); topics and labels use their own reads below.
- Never send a mutation request or change the configuration file.

### Rule: Empty declarations are local no-ops

A valid document without managed fields succeeds before origin resolution, authentication or API access. Invalid documents still fail validation; explicit false and empty strings remain managed values. The [Action's metadata and explicit-token requirements](actions.feature.md#public-composite-action-contract) remain an independent audit contract.

#### Scenario: A version-only declaration without origin or credentials

- Given a Git checkout with `.github/settings.yml` containing only `schemaVersion: 1`
- And no origin or authentication
- When the maintainer runs plan
- Then it prints `No managed repository settings.` and exits 0
- And it makes no API calls or configuration changes

#### Scenario: Empty settings without origin or credentials

- Given a Git checkout with a version-1 declaration and `settings: {}`
- And no origin or authentication
- When the maintainer runs plan
- Then it prints `No managed repository settings.` and exits 0
- And it makes no API calls or configuration changes

#### Scenario: An empty repository declaration without origin or credentials

- Given a Git checkout with a version-1 declaration and `settings.repository: {}`
- And no origin or authentication
- When the maintainer runs plan
- Then it prints `No managed repository settings.` and exits 0
- And it makes no API calls or configuration changes

### Rule: Compare declared fields only

- Omitted fields are unmanaged and produce no difference.
- Explicit false and empty strings are managed values.
- Show changed REST fields in description, homepage, has_issues, has_projects, has_wiki, has_discussions order, followed by the [merge-related fields](../commands.md#plan-behavior), topics and labels. Discussions is supported since v0.15.0; v0.14.0 retains the five-field REST state.
- Each line uses `settings.repository.FIELD: CURRENT → DESIRED`.
- Strings are double-quoted with Go string escaping; booleans use `true` or `false`.
- Equal managed values produce `No differences for managed repository settings.`
- A valid document without managed fields produces `No managed repository settings.`

### Rule: Report success and failures

- Successful comparison exits 0 whether differences exist or not.
- File, validation, origin, authentication, API, decoding, and output failures exit 1.
- Errors appear on standard error; failures before output leave standard output empty.

### Test mapping

Focused application, comparison and CLI tests cover target selection, declaration semantics, exact output, and failures. Process acceptance uses the already-built executable with an isolated local API fixture, checks exact exit codes and output, observes GET-only requests, and verifies configuration preservation.

| Behavior | Test |
| --- | --- |
| Original five-field order, omission, false, empty strings, escaping, equality, no managed fields | [TestCompareRepositoryChanges and TestCompareRepositoryDeclaredFields](../../internal/settings/comparison_test.go), [TestRenderRepositoryComparison](../../internal/cli/comparison_test.go) |
| Discussions omission/false/equality/order after wiki and REST-only selective read | [TestDiscussionsComparison](../../internal/settings/discussions_test.go), [TestPlanDiscussionsOnlyReadsREST](../../internal/app/discussions_test.go), [TestPlanAcceptance](../../tests/acceptance/plan_test.go) |
| CLI operation selection and output failure | [TestRunSelectsOperation and TestPlanOutputFailure](../../internal/cli/run_test.go) |
| API, decoding, transport failures and GET-only access | [TestReadRepositorySettings and TestReadRepositoryFailures](../../internal/github/repository_test.go) |
| Typed target/comparison, changes/no differences, unchanged configuration | [TestPlanResults](../../internal/app/plan_test.go) |
| All three empty forms succeed before origin/read, zero result and unchanged file | [TestPlanNoManagedSettings](../../internal/app/plan_test.go) |
| Omitted fields versus each explicit false/empty field | [TestRepositoryDeclarationEmpty](../../internal/settings/repository_test.go) |
| Validation before target/read, target failure before read, read failures | [TestPlanValidatesBeforeRead, TestPlanTargetFailureBeforeRead and TestPlanReadFailure](../../internal/app/plan_test.go) |
| File/subdirectory, managed false/empty values, GET nulls, API/output failures, no PATCH; three empty forms without origin/auth/GET | [TestPlanAcceptance](../../tests/acceptance/plan_test.go) |
| Export → edit → validate → plan | `TestExportEditValidatePlan` |

### Rule: Compare topics as a stable set

- Given declared topics and observed topics contain the same names in different orders.
- When plan runs.
- Then no topic difference is reported.
- And omitted resources are not read.
- Given the topic sets differ.
- Then one settings.topics.names change shows sorted arrays with native string values.

Test mapping: [TestTopicsSetComparison](../../internal/settings/topics_test.go), [TestTopicsOnlyPlanAndApply](../../internal/app/topics_test.go), [TestTopicsPlanApplyAcceptance](../../tests/acceptance/topics_test.go).

### Rule: Show label collection operations

Current source reads a declared label collection and reports typed create/update/delete operations. An explicit empty list counts as one managed resource, and omitted label attributes remain unmanaged. The [Labels specification](labels.feature.md#rule-complete-reads-before-export-or-reconciliation) owns label comparison scenarios and test mappings. Published v0.8.0 does not include labels.
