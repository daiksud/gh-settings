---
type: Feature
title: Apply repository settings
description: Send declared repository fields directly after local validation.
---

## Feature: Apply declared settings

### Rule: Validate before API access

- Read `.github/settings.yml` from the Git root, including from a subdirectory.
- Reuse the schemaVersion 1 typed parser and reject invalid or missing configuration before origin, authentication, or API access.
- A valid document with no managed fields prints `No managed repository settings.` and exits 0 without origin, authentication, or API access.
- Otherwise resolve the same GitHub.com origin as export and ignore `GH_REPO`.
- Accept origin values only from checkout-local/worktree scopes, including their included files; system/global/command origin values cannot select the mutation target.

### Rule: Send the declaration directly

- When repository fields are declared, send exactly one PATCH to the current repository with only those fields. Topics uses a separate PUT as specified below.
- Reject redirects, including those that would convert PATCH into GET.
- Omit schemaVersion, outer YAML mappings, and every undeclared field.
- Preserve false and empty strings as explicit values.
- Do not fetch current settings, reconcile differences, or require a saved plan.
- Send declared values even if they already match or remote values changed after plan.
- Preserve the configuration file and leave omitted remote fields unmanaged.

### Rule: Report success and failures

- Successful apply prints `Applied repository settings.` and exits 0.
- File, validation, origin, authentication, API/response, and output failures exit 1.
- Errors appear on standard error; failures before output leave standard output empty.
- Output can fail after an update succeeds; failures provide no automatic rollback.

### Test mapping

Application tests use real temporary checkouts/files and a writer spy; GitHub adapter tests use fake transports. Process acceptance uses the already-built executable, isolated Git/configuration directories, a local Unix socket, and a dummy token. No test mutates real GitHub repository settings.

| Behavior | Test |
| --- | --- |
| Exact PATCH payload, omission, five fields, false/empty strings, escaping | [`TestApplyRepositorySettings`](../../internal/github/apply_test.go) |
| HTTP/response/transport failures | [`TestApplyRepositoryFailures`](../../internal/github/apply_test.go) |
| System/global/command origins ignored; local/include/worktree selection retained; rewrite-only aliases rejected | [TestCurrentRepositoryIgnoresUnscopedOrigins and TestCurrentRepositoryRejectsRewriteOnlyAlias](../../internal/workspace/repository_test.go) |
| Typed target/declaration, false/empty/omitted values, one write and unchanged file | [`TestApplyDeclaredSettings`](../../internal/app/apply_test.go) |
| Validation before target/writer and local no-op | [TestApplyValidatesBeforeWrite and TestApplyNoManagedSettings](../../internal/app/apply_test.go) |
| Target failure before write and write failure without applied status | [TestApplyTargetFailureBeforeWrite and TestApplyWriteFailure](../../internal/app/apply_test.go) |
| Command dispatch, arguments, typed no-op message and output failures | [TestRunSelectsOperation, TestUnsupportedCommand, TestApplyNoManagedResult and TestApplyOutputFailure](../../internal/cli/run_test.go) |
| Built product: exact PATCH, subdirectory/origin resolution, unscoped-origin isolation, no GET, invalid YAML, no-op, auth/API/redirect/output failures, failure status/stderr/empty stdout, unchanged file | [`TestApplyAcceptance`](../../tests/acceptance/apply_test.go) |
| Export → edit → validate → plan → apply; changed remote state, omitted values preserved | `TestExportEditValidatePlanApply` |

### Rule: Apply declared resources independently

- Given repository fields and topic names are declared.
- When apply runs.
- Then repository PATCH precedes topics PUT without a preliminary GET.
- And the PUT contains only the sorted names array, including an explicit empty array.
- And a failure stops later operations without rolling back successful writes.
- And omitted names produce no topics request.

Test mapping: [TestTopicsApplyOrderAndPartialFailures](../../internal/app/topics_test.go), [TestApplyTopicsPayloadAndErrors](../../internal/github/topics_test.go), [TestTopicsPlanApplyAcceptance](../../tests/acceptance/topics_test.go).
