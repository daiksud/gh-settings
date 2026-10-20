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

### Rule: Send repository and topics declarations directly

- When repository fields are declared, send exactly one PATCH to the current repository with only those fields. Topics uses a separate PUT as specified below.
- Reject redirects, including those that would convert PATCH into GET.
- Omit schemaVersion, outer YAML mappings, and every undeclared field.
- Preserve false and empty strings as explicit values.
- When current-source `has_discussions` is declared, verify the PATCH response contains the requested boolean. Missing, null, wrong-type or unequal values fail after the request, can leave repository writes applied, and stop later resources without rollback. Omission keeps the previous response handling. See the [Discussions contract](discussions.feature.md) for the v0.15.0 version boundary.
- For repository fields and topics, do not fetch current values or reconcile differences; apply never requires a saved plan.
- Send declared values even if they already match or remote values changed after plan.
- Preserve the configuration file and leave omitted remote fields unmanaged.

### Rule: Report success and failures

- Successful apply prints `Applied repository settings.` and exits 0.
- File, validation, origin, authentication, API/response, and output failures exit 1.
- Errors appear on standard error; failures before output leave standard output empty.
- Output can fail after an update succeeds; failures provide no automatic rollback.

### Test mapping

Focused application tests record declaration loading, target resolution, resource writes, and fresh label reads using typed values without Git or file fixtures. Workspace tests own checkout resolution and origin rules; GitHub adapter tests use fake transports. Process acceptance uses the already-built executable, isolated Git/configuration directories, a local Unix socket, and a dummy token. No test mutates real GitHub repository settings.

| Behavior | Test |
| --- | --- |
| Exact PATCH payload, omission, original five fields, false/empty strings, escaping | [`TestApplyRepositorySettings`](../../internal/github/apply_test.go) |
| Discussions declared-only true/false writes, verified response, ignored/malformed responses and permission failures | [TestDiscussionsPATCHVerifiesDeclaredValue](../../internal/github/discussions_test.go), [TestApplyAcceptance](../../tests/acceptance/apply_test.go) |
| HTTP/response/transport failures | [`TestApplyRepositoryFailures`](../../internal/github/apply_test.go) |
| System/global/command origins ignored; local/include/worktree selection retained; rewrite-only aliases rejected | [TestCurrentRepositoryIgnoresUnscopedOrigins and TestCurrentRepositoryRejectsRewriteOnlyAlias](../../internal/workspace/repository_test.go) |
| Typed target/declaration, false/empty/omitted values, load → target → write order | [`TestApplyDeclaredSettings`](../../internal/app/apply_test.go) |
| Validation before target/writer and local no-op; explicit empty topics/labels still resolve | [TestApplyValidatesBeforeWrite, TestApplyNoManagedSettings and TestApplyManagedEmptyCollectionsResolveTarget](../../internal/app/apply_test.go) |
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

### Rule: Reconcile labels against fresh state

Current source applies declared labels after repository/topics writes: it reads fresh state, performs needed creates/updates in identity order, then deletes undeclared labels in identity order. A read or write failure leaves earlier updates applied. The [Labels specification](labels.feature.md#rule-reconcile-fresh-state-during-apply) owns these scenarios and test mappings, including association loss on deletion. Published v0.8.0 does not include labels.

### Rule: Actions execution policy sequencing

Current source applies repository Actions policy after existing resources. The [Actions policy specification](actions-execution-policy.feature.md#rule-preflight-and-preserve-companions-before-ordered-writes) owns fresh reads, companion preservation, staged enabling, readback and partial failure.
