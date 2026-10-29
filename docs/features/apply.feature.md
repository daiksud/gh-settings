---
type: Feature
title: Apply repository settings
description: Send declared repository fields directly after local validation.
---

## Feature: Apply declared settings

### Rule: Validate before API access

- Read `.github/settings.yml` from the Git root, including from a subdirectory.
- Reuse the exact current-version UI parser and API conversion and reject invalid or missing configuration before origin, authentication, or API access.
- A valid document with no managed fields prints `No managed repository settings.` and exits 0 without origin, authentication, or API access.
- Otherwise resolve the same GitHub.com origin as export and ignore `GH_REPO`.
- Accept origin values only from checkout-local/worktree scopes, including their included files; system/global/command origin values cannot select the mutation target.

### Rule: Send repository and topics declarations directly

- When ordinary REST-backed About/General scalars are declared, send the declared internal members in the repository PATCH. GraphQL-backed PR/Issue/Sponsor controls use their separate identity-checked mutation; a GraphQL-only declaration sends no PATCH. Topics uses a separate PUT.
- Reject redirects, including those that would convert PATCH into GET.
- Omit the version marker, outer UI mappings, and every undeclared field from API payloads.
- Preserve false and empty strings as explicit values.
- When `repository.settings.general.features.discussions` is declared, verify the PATCH response contains the requested boolean. Missing, null, wrong-type or unequal values fail after the request, can leave repository writes applied, and stop later resources without rollback. Omission keeps the previous response handling.
- Ordinary direct repository/Topics writes do not reconcile a saved comparison; family-specific identity/prerequisite/readback checks still apply. Apply never requires a saved plan.
- Send declared values even if they already match or remote values changed after plan.
- Preserve the configuration file except guarded confirmed-created-ID writeback, and leave omitted scalar fields unmanaged.

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
| Validation before target/writer and local no-op; explicit empty Topics still resolve while empty Labels select nothing | [TestApplyValidatesBeforeWrite, TestApplyNoManagedSettings and TestApplyManagedEmptyCollectionsResolveTarget](../../internal/app/apply_test.go) |
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
- And omitted About topics produce no Topics request.

Test mapping: [TestTopicsApplyOrderAndPartialFailures](../../internal/app/topics_test.go), [TestApplyTopicsPayloadAndErrors](../../internal/github/topics_test.go), [TestTopicsPlanApplyAcceptance](../../tests/acceptance/topics_test.go).

### Rule: Reconcile labels against fresh state

Current unreleased source applies nonempty label selections after repository/topics writes. It preflights ID-selected updates and ID-less creates against complete fresh state, never deleting unlisted objects. Whole supported values include fixed description defaults. Confirmed create IDs are written to the original YAML unless opt-out/Actions disables recording; failures preserve earlier remote writes and recoverable IDs. See the [Labels specification](labels.feature.md).

### Rule: Actions execution policy sequencing

Current source applies repository Actions policy after existing resources. The [Actions policy specification](actions-execution-policy.feature.md#rule-preflight-and-preserve-companions-before-ordered-writes) owns fresh reads, companion preservation, staged enabling, readback and partial failure.

Explicit unarchive runs before other resources. Repository REST updates precede declared PR/Issue/Sponsor GraphQL updates, then Topics, fresh label reconciliation, Rulesets, Environments, Actions execution/workflow/retention/public-fork/private-fork policies, Secret protection/Push protection, Private vulnerability reporting, release immutability and existing-site Pages; final archive runs last. An omitted resource has no request. Current direct repository/topics/execution/workflow paths may resend declared matching values; zero plan is not a promise of no apply requests. Collection reconciliation and the newer policy paths skip freshly matching operations. Current-state prerequisites and full companion readback remain resource-specific; successful reads and skips are not completed writes. Later failure stops work and preserves actual earlier progress without rollback, automatic retry or locking.

| Subset | Existing application and transport/process evidence |
| --- | --- |
| PR/Issue partial writes and identity | [PR access](../../tests/acceptance/pr_access_test.go), [Issue policy](../../tests/acceptance/issue_creation_policy_test.go) |
| Ruleset membership, operation order and checks | [Ruleset app](../../internal/app/rulesets_test.go), [Ruleset process](../../tests/acceptance/rulesets_test.go), [parameter process](../../tests/acceptance/ruleset_parameters_test.go) |
| Workflow permissions and retention | [Workflow app](../../internal/app/workflow_permissions_test.go), [retention app](../../internal/app/retention_test.go), [retention process](../../tests/acceptance/retention_test.go) |
| Secret scanning and public reporting | [Security app](../../internal/app/security_test.go), [reporting app](../../internal/app/private_reporting_test.go), [security specification](security.feature.md) |
| Fork policies and later applicability failure | [Fork app](../../internal/app/fork_workflows_test.go), [fork transport](../../internal/github/fork_workflows_test.go), [fork process](../../tests/acceptance/fork_workflows_test.go) |

These fixtures verify modeled payloads, ordering, readback and uncertainty; they do not establish live grants, enforcement, scan outcomes, deployed workflow safety or transactionality. The dedicated contracts own each transition and prerequisite rather than generalizing one resource's rules to another.

### Rule: Keep public UI intent separate from API protocol

- The current YAML marker and UI fields convert through the prepared family adapters before application work.
- A UI menu sends its complete API pair and an Actions allow radio explicitly declares enabled state plus its policy.
- Old envelopes and API-shaped YAML aliases fail before origin/authentication/network access.
- Root snapshot-bound ID recording targets `repository.labels` and `repository.settings.rulesets`, selecting Rulesets by `ruleset_name`.
- Earlier/uncertain writes and failed local ID persistence remain distinct; conversion does not add rollback or live deployment.

Test mapping: [root composition](../../checkout_config_test.go), [independent Ruleset requests](../../internal/github/ui_config_rulesets_test.go), and supplied-binary family tests. Current-source fixture success does not publish a release or upgrade an Action pin.
