# Architecture and design

This page explains the design for contributors. The [settings reference](../settings.md), [command reference](../commands.md), and [GitHub Actions guide](../github-actions.md) define the interfaces users work with; [development](development.md) describes how to extend them.

## Purpose and product boundary

gh-settings lets repository maintainers record intended settings in version control, review differences, and apply declared values. It is inspired by [babarot/gh-infra](https://github.com/babarot/gh-infra), [noirbizarre/gh-settings](https://github.com/noirbizarre/gh-settings), and [the GitHub Terraform provider](https://github.com/integrations/terraform-provider-github), rather than being a fork of those implementations.

Configuration lives at the current checkout's root `.github/settings.yml` and targets only that checkout's GitHub.com repository. Current resources are repository fields, topics, labels, and the unreleased repository-owned branch/tag Rulesets subset. The [published package and consumer pins reference](../github-actions.md#published-package-and-consumer-pins) owns supported-version and adoption guidance. Organization settings, inherited organization rulesets, and management of multiple repositories are outside this product boundary.

Written fields are managed and authoritative; omitted fields remain unmanaged. Declaring a labels list additionally owns its collection membership, while omitted attributes of retained labels stay unmanaged. Configuration keys and value types follow the corresponding update request schema, within the explicit versioned resource grouping. A repository GET response does not define the writable configuration schema. The [settings reference](../settings.md#configuration-semantics) owns exact field, omission, null, and grouping behavior.

Observation and mutation are separate operations. Plan compares declarations with observed state. Apply sends repository fields and topics directly; labels alone reads fresh state to determine the required collection operations. A plan is informational, is never apply input, and cannot guarantee what later remote state will be. Failures remain visible, without transactional or automatic recovery guarantees.

## Implementation ownership

One Go implementation serves thin CLI and Actions adapters. Keep declaration, observed state, and comparison distinct; the [glossary](glossary.md) defines their types and meanings.

| Owner | Responsibility | Main implementation |
| --- | --- | --- |
| Settings model | Optional declarations, complete observed values, topic sets, label identity/membership, and ordered native comparisons; no I/O or YAML/JSON/presentation tags | [Repository model](../../internal/settings/repository.go), [resources/topics](../../internal/settings/settings.go), [comparison](../../internal/settings/comparison.go), [labels](../../internal/settings/labels.go) |
| Workspace YAML | Decode and encode the versioned/grouped contract through private YAML mappings | [Codec](../../internal/workspace/config.go), [labels codec](../../internal/workspace/labels.go) |
| Workspace files and Git | Fixed configuration path, file reads/writes, checkout root, commit, and checkout-scoped target resolution | [Checkout](../../internal/workspace/checkout.go), [files](../../internal/workspace/config_file.go), [Git](../../internal/workspace/repository.go) |
| GitHub adapter | Client configuration, complete read normalization, declared-only repository PATCH, topics GET/PUT, paginated labels reads/CRUD, and write redirect handling | [Repository reads](../../internal/github/repository.go), [repository writes](../../internal/github/apply.go), [topics](../../internal/github/topics.go), [labels](../../internal/github/labels.go), [individual label requests](../../internal/github/label_operations.go) |
| Shared application | Validation, resource selection, operation sequencing, and typed results | [Application operations](../../internal/app/), [deferred readers/writers](../../internal/app/resources.go) |
| CLI adapter | Arguments, help/version, terminal field/operation presentation, diagnostics, and process exit status | [Dispatch](../../internal/cli/run.go), [comparison rendering](../../internal/cli/comparison.go) |
| Shared presentation | Plain-text label operation wording and quoting shared by CLI and Summary; no HTML or I/O | [Label formatting](../../internal/presentation/labels.go) |
| Actions adapter | Same-version metadata, ephemeral field/label reports, validated Summary rendering, and status | [Dispatch](../../internal/actions/run.go), [reports](../../internal/actions/plan.go), [label reports](../../internal/actions/labels.go), [Summary](../../internal/actions/summary.go) |
| Composite Action | Verified installation bootstrap and necessary Actions input/token/status/output wiring | [Root Action](../../action.yml), [installer](../../action/install-cli.sh) |

The [root entry point](../../main.go) binds both adapters to shared application operations and concrete deferred GitHub functions. A needed read or write constructs its API client; local validation, help/version, and empty CLI plan/apply do not eagerly configure authentication. Keep use-case sequencing in the application rather than moving it into root dispatch, shell, or presentation code.

### Workspace and GitHub boundaries

The workspace codec returns a typed declaration. `workspace.Checkout.Declaration` resolves the Git root for its explicit directory and loads the fixed configuration path with the existing diagnostic context. The root entry point supplies this deferred loader to validation, CLI/Action planning, and apply; constructing a Checkout does not resolve Git or read files. CLI/Action planning and CLI apply also receive the same Checkout's deferred `Repository` resolver, called by the application only after loading a managed declaration. Both Git commands used for resolution run in that Checkout's directory. Export receives the Checkout target resolver and `PrepareSave` operation. `PrepareSave` resolves the Git root without changing files and returns a saver bound to that root; the application invokes it only after complete reads. The saver encodes the declaration completely before applying the existing guarded file-writing policy. Workspace does not contact GitHub. Git resolution uses the checkout's local/worktree origin, including repository-specific configuration, ignores `GH_REPO` and unscoped origins, and retains the direct repository identity even when Git rewrites transport URLs.

Plan requests only declared scalar Repository `FieldID` values through `github.ObserveRepository`; the GitHub adapter chooses and lazily creates the necessary REST and GraphQL clients, in REST-first order. Its result retains which complete observation groups were acquired. Shared comparison rejects missing required observations rather than treating unread fields as false or empty, and preserves canonical field order. Export requests every supported Repository FieldID through the same observation boundary and converts the result to a complete declaration only after all required values have been acquired. The application and root composition no longer select REST versus GraphQL reads.

The GitHub read adapter requires complete repository values, normalizes permitted null strings to empty strings, and rejects missing fields or null booleans. Repository/topics writes map declarations to private request DTOs without preliminary reads. The labels adapter completes and validates every page before exposing observed state. Read and write clients retain separate 30-second timeouts; writes reject redirects. Labels reads during apply use the write-client policy too, rejecting redirects. A network error does not establish whether a remote update occurred.

### Shared operation flow

| Operation | Application behavior |
| --- | --- |
| Validate | Invoke the declaration loader and return its typed declaration or error; no origin, authentication, API access, or output formatting |
| Plan | Validate; return a local no-op for an empty declaration; otherwise resolve the target, read only managed resources, and return native field changes and typed label operations |
| Apply | Validate; return a local no-op for an empty declaration; otherwise send declared repository settings, then topics, then read/reconcile declared labels, stopping on the first failure |
| Export | Resolve the checkout/target; read complete repository settings, topics, and every labels page; construct a complete declaration, serialize it, then write the configuration |

`app.PlanError` preserves the original error and identifies validation, target, or read/comparison failure; read failures carry the resolved target. A failed later resource does not expose an earlier partial comparison.

[`app.ApplyResult`](../../internal/app/apply_result.go) retains an internal outcome, resolved target, and ordered operation records. Repository and Topics updates and a declared labels read are known before execution; individual label operations are added only after complete fresh observations determine them. Failures retain earlier completed operations and later known operations that were not executed. Failed writes have an unknown remote effect, rather than an unapplied status; failed reads are separate. Partial failure means at least one earlier write met the existing success criteria. Completion does not establish readback agreement or a transaction. A matching labels-only apply is distinct from an unmanaged declaration, while `Completed()` selects the same existing CLI success message for both completed managed outcomes. Errors and later output failures retain their existing presentation and do not undo updates. [Progress tests](../../internal/app/apply_progress_test.go) establish these distinctions without additional reads or recovery attempts.

### Repository merge-method read path

The v0.10.0 [merge-method extension](../features/merge-methods.feature.md) remains part of `settings.repository`. Its three optional declaration booleans use the existing repository PATCH. v0.11.0 adds the [`delete_branch_on_merge` boolean](../features/delete-branch-on-merge.feature.md) to this same merge-related state, fixed query and deferred read; there is no extra API call or new resource family. A separate complete `settings.MergeMethodsState` and deferred `app.Reader.MergeMethods` callback isolate the GraphQL observation from the six-value REST state (including `has_discussions` in v0.15.0 source). Plan composes their ordered comparisons only for declared fields; a merge-method-only declaration skips the base REST read. Export performs both reads before serialization. No new YAML resource family or generic backend registry is introduced.

The v0.12.0 squash commit defaults extend the same state/query/callback with two validated string enums. The settings domain checks message-to-title dependencies after YAML scalar validation; direct apply never reads or synthesizes a companion title. The v0.12.0 observation contains four booleans and two enums; v0.13.0 adds `autoMergeAllowed` and `allowUpdateBranch` for six booleans and two squash enums; v0.14.0 adds two normal merge enums, while comparison and PATCH retain per-field omission.

The narrow [GraphQL adapter](../../internal/github/merge_methods.go) uses the existing go-gh dependency and a fixed ten-field query with origin-derived owner/name variables. Missing/null fields and GraphQL partial data/errors fail rather than becoming false. [Read-only hosted evidence](https://github.com/daiksud/gh-settings/actions/runs/39086857913) establishes ordinary Contents-read token access without expanding PR permissions. The linked initial experiment covered the original three merge-method booleans; each later extension records its own hosted proof. The temporary diagnostic workflow is excluded from the final implementation; local fixtures own product/error coverage. Apply never uses this read path.

### Labels reconciliation

[`settings.CompareLabels`](../../internal/settings/labels.go) owns authoritative membership and attribute-presence comparison. A nil labels pointer is unmanaged; a present list, including an empty list, manages the entire collection. Label identity uses Unicode simple case folding without trimming or rename inference. Ambiguous names fail, and a case-only declaration change retains the remote spelling.

`settings.LabelDeclarations` validates names, attributes, and unique identities at construction, normalizes colors, and owns copies of the slice and optional attribute values. YAML decoding and export construct this collection; comparison consumes its guarantees without revalidating declarations. Retrieved values and resulting operations contain independent attribute copies, preserving the collection's invariants. Its empty value is a managed empty collection when present in a declaration.

`settings.LabelObservation` validates complete observed values and unique identities directly, owns its entries, and returns independent copies. The GitHub adapter constructs it only after every page and response type has been checked. A zero observation represents a missing read and cannot expose values; a constructed empty observation represents a completed read with no labels. Comparison and export require a completed observation, so a failed or missing read cannot become an empty current collection. Comparison no longer converts observations into declarations for validation.

The comparison produces closed create/update/delete variants in a `settings.LabelChanges` sequence with unique identities. Each variant owns its required prior state or declared attributes; constructors reject renames, noncanonical colors, and empty or unchanged updates. Visitors expose independent attribute copies to GitHub writes, CLI rendering, and report conversion. External report DTOs retain strict shape/type validation before constructing the same operations and sequence.

Creates and updates run together in identity order, followed by deletes in identity order. Create requests contain the name and declared attributes; updates contain only changed declared attributes. Omitted attributes of existing labels preserve remote values, and no default attributes are synthesized for new labels. Deleting an undeclared label removes its issue/PR associations; recreating its name does not restore them. The [labels contract](../features/labels.feature.md) defines the full behavior.

Application apply performs this fresh comparison only after any declared repository PATCH and topics PUT have succeeded. It reads all labels pages using the apply credential, then executes the required individual operations in comparison order. A read, comparison, or write failure stops subsequent work while earlier resource/label writes remain applied. No saved plan, drift protection, transaction, rollback, or automatic retry is introduced.

The GitHub adapter's `CreateLabel`, `UpdateLabel`, and `DeleteLabel` methods own individual request payloads and HTTP dispatch from typed operations. `app.Apply` owns the entire resource and label sequence, stopping at the first failure and adding the existing label-operation error context. Deferred writer callbacks bind each individual API operation. Individual requests are verified in the API adapter; ordering and partial-failure behavior are verified in the application.

### Adapter boundaries

The CLI owns terminal field quoting, arrows, messages, and exit codes. Shared presentation owns label-operation wording and quoting for both CLI and Summary; Summary applies its own HTML and newline escaping to that plain text. Actions metadata resolves consumer identity and hashes the executing binary without a settings operation. Plan-report generation invokes shared planning once; a separate credential-free Summary entry renders its typed result without planning again.

Reports preserve native field values and carry label operations separately, retaining omitted declared attributes and explicit create/update/delete names. A declared labels collection counts as one managed item regardless of its number of operations. Categorical review diagnostics stay separate from original stderr errors. Apply never consumes a report. Failed plans remain nonzero after report/Summary output, and the composite exposes success outputs only after required work succeeds. See [Actions internals](actions-internals.md) for those entry contracts and [testing](testing.md) for their verification boundaries.

Summary validation converts the JSON report DTO to a private validated presentation input, retaining native comparisons, label operations, and checked report state. Typed visitors render the retained values; JSON type assertions and operation reconstruction remain at the input boundary. This preserves credential-free rendering and validates all input and metadata before opening the Summary destination.

The public Action installs the released CLI selected by its committed [`action/cli-version`](../../action/cli-version). See the [published package and consumer pins reference](../github-actions.md#published-package-and-consumer-pins) for released capabilities and the [labels permission guidance](../github-actions.md#labels-and-permissions) for caller requirements.

## v0 simplification contract

The completed [source-refactoring series](https://github.com/daiksud/gh-settings/issues/133) established shared Go operations and thin adapters while retaining the product requirements below.

- Keep schema versioning, resource grouping, partial declarations, false/empty values, accepted YAML aliases and merges, strict validation, and checkout/target isolation.
- Validate before remote work and omit unmanaged fields from updates. Repository/topics apply remains direct; labels adds only the fresh collection reconciliation described above. No resource consumes a saved plan or gains drift protection, automatic rollback, or retries.
- Preserve explicit credential ownership, release artifact verification, and the distinction among consumer, Action, and CLI identities.
- Put tests with the owner of a current requirement. Retired wrappers, internal interfaces, and historical fixture combinations do not require compatibility shims merely because they once existed.
- Extend useful concrete structure when a resource is requested; avoid speculative registries or a package for every concept.

Two deliberate behavior simplifications remain current: empty CLI plan/apply succeeds before target/authentication work, and origins must directly identify GitHub.com through supported HTTPS, `git@` SCP, or full SSH forms. Rewrite-only aliases and username-less SCP have no fallback mode. Exact user behavior belongs in the [command reference](../commands.md).

Topics and labels were added as separate concrete resources after that refactoring. Topics retains its omitted-pointer/explicit-empty-set distinction and sorted comparisons/payloads; labels has the [collection reconciliation](#labels-reconciliation) described above. These additions retain checkout isolation, explicit ownership, and partial-failure visibility without introducing a generic resource framework.

Published tags/assets remain immutable when source changes. Candidate-source tests and published-download verification have different subjects. [Releases](releases.md), [consumer adoption](../github-actions.md#published-package-and-consumer-pins), and [this repository's automation](repository-automation.md) have separate operational procedures.

## Design decisions

ADRs record the context and consequences of individual decisions. Read their status and superseding links when a historical consequence differs from today's contract.

| Concern | Decisions |
| --- | --- |
| Delivery and implementation | [GitHub CLI extension](../adr/0001-github-cli-extension.md), [Go](../adr/0007-go-implementation.md), [official go-gh library](../adr/0008-official-go-gh-library.md), [YAML v4](../adr/0009-yaml-v4-library.md) |
| Managed settings | [Declared-field ownership](../adr/0002-managed-fields.md), [update request schema](../adr/0006-update-api-schema.md), [explicit null](../adr/0019-reject-explicit-null.md), [resource grouping](../adr/0020-group-settings-by-resource.md), [schema version](../adr/0021-require-schema-version.md) |
| Mutation | [Direct apply](../adr/0003-direct-apply.md), [strict validation](../adr/0004-strict-validation.md), [no automatic rollback](../adr/0005-no-automatic-rollback.md) |
| Actions | [Caller approval and credential ownership](../adr/0022-owner-approved-actions.md), [public composite Action](../adr/0023-public-composite-action.md) |
| Distribution | [Release decisions and future versioning](releases.md#release-decisions) |

Prefer the Go standard library where sufficient; no command framework is mandated. The selected YAML implementation is maintained by the YAML organization, not the Go team. Its release-candidate dependency is an explicit choice recorded in ADR 0009; current dependency versions belong in [go.mod](../../go.mod).

## Pull request access transport

The unreleased [PR access feature](../features/pull-request-access.feature.md) adds a third complete scalar observation group in `internal/settings`, alongside base REST and merge-related GraphQL values. Existing field IDs, requested observation routing and native report presentation retain omission and stable order. Export requests all twenty-two scalar fields; selective plan skips clients for unrelated groups.

The [PR access adapter](../../internal/github/repository_access.go) uses GraphQL query variables for complete policy reads and an independently validated repository-ID lookup before a declared-only `updateRepository` mutation. The root composition retains existing REST writes, skips an empty PATCH for PR-only input, and then invokes PR access before later resources. Mutation response values and identity are verified; mixed-resource errors preserve evidence of earlier writes. This does not add a generic backend registry, saved plans, retries, or permission expansion.

## Issue creation policy

The unreleased [Issue creation policy](../features/issue-creation-policy.feature.md) has its own complete observation group and query, so PR-only reads do not acquire unrelated Issue data. `RepositoryObservation.WithIssueCreationPolicy` adds the independently acquired enum without mutating an earlier observation. The static field definitions retain order, native values and omission; full source export acquires twenty-two scalars.

The shared [repository access adapter](../../internal/github/repository_access.go) retains the narrow PR query and adds a separate Issue query. Both use `followRenames: false`. Its closed mutation selects only declared Issue/PR fields and performs one validated ID lookup and mutation for combined input. The root composition excludes these fields from REST payloads and preserves existing REST-first partial-failure behavior. `has_issues` keeps its existing REST ownership and is never synthesized by an Issue policy declaration.

## Actions execution policy

The unreleased [Actions policy feature](../features/actions-execution-policy.feature.md) is an independent resource, not a repository scalar observation group. `internal/settings` owns partial declarations, complete active observations, transition preflight and six bounded native comparisons. Inactive selected before values are explicitly unavailable. Workspace owns strict YAML; the Actions API adapter owns complete conditional reads, redirect rejection, fresh companion merge, policy-before-selection writes, delayed enabling and readback. Shared app operations defer the client until needed; export opts in, plan remains read-only and apply places this resource after existing ones. CLI and Summary share the presentation formatter; Summary reconstructs and validates the native comparisons without network access. No workflow, grant or pin is changed.

## Default workflow permissions

The unreleased [workflow-permission feature](../features/workflow-permissions.feature.md) extends the actions declaration with two independent fields. `ExecutionEmpty` isolates the existing six-field execution path; `WorkflowPermissionsEmpty` selects a separate complete observation and callback. Shared app sequencing records workflow updates after execution policy as a distinct operation, preserving known partial progress. An API-neutral `PreWriteReadFailure` marker preserves a prerequisite read failure across the adapter/use-case boundary; only PUT/readback failures are recorded as unknown writes. The dedicated API adapter sends declared-only optional PUT keys and verifies complete fresh state; no execution endpoint is needed for workflow-only declarations. Explicit export opt-in preserves existing resource choices. Three composable choices use a small CLI `ExportOptions` value; native comparisons reuse the closed eight-field Actions registry and existing formatter.

## Future direction

Pages and Actions settings remain outside current support. Rulesets lifecycle and PR/check rules are implemented in current source, unreleased. Label definitions are implemented in current source; individual issue/PR assignment management, renaming, and archive settings are not part of that feature.

[ADR 0025](../adr/0025-declarative-repository-rulesets.md) accepts label-style membership and retained-attribute semantics for repository-owned branch/tag Rulesets. The [Rulesets specification](../features/rulesets.feature.md) owns exact-name-to-ID matching, nested identity, deletion, complete observations, export opt-in and staged acceptance criteria. Inherited/push Rulesets and legacy branch protection remain outside that managed collection; legacy protection implementation is deferred. Published v0.16.0 and older CLI/Action packages do not support this source-only feature.

Other future resources still require their own null, output, creation and identity decisions; the Rulesets decision does not resolve them. [Automatic version selection](releases.md#future-automatic-versioning) is also future work.

[ADR 0026](../adr/0026-explicit-security-observations.md) accepts explicit per-family security observations and export selection. The [Security specification](../features/security.feature.md) records planned boolean declarations, permissions/availability, dependencies, failure semantics, and gates for #344–#347. This design has no executable support; ambiguous Dependabot disabled reads and distinct paid-product adoption require the documented evidence before implementation.
