---
type: Feature
title: Export repository settings
description: Specify the user-visible YAML export and its ordinary Go test mapping.
---

## Feature: Export repository settings as YAML

A repository maintainer can capture supported visible About/General controls,
topics, labels and all supported readable setting families without changing
GitHub settings. `export` saves `.github/settings.yml` at the checkout root and
creates a missing `.github` directory. Success has no stdout; unavailable independent
families emit bounded info and independent families continue. First export omits
unavailable families; re-export retains their prior valid declarations where
compatible with positively acquired settings. See the
[automatic settings export contract](readable-settings-export.feature.md).

Existing files are decoded and validated before observation. Invalid/unreadable
files and nonregular/symlink destinations are rejected. Successful encoding uses
guarded atomic replacement for a stable existing snapshot; concurrent file edits
or a newly appearing destination fail instead of being overwritten. Initial-file
IO failure can leave a partial new file. This is not a lock against concurrent
directory/link replacement.

The [Discussions contract](discussions.feature.md) owns its strict read rules and version boundary.

This feature assumes an owner-controlled checkout whose directory and link entries remain stable during export. Concurrent replacement of those entries is outside its scope.

### Rule: Save configuration in the checkout selected by origin

#### Scenario: Export from a subdirectory without a settings directory

- Given the current Git checkout has a GitHub `origin` remote
- And it has no `.github` directory
- When the maintainer runs `gh settings export` from a subdirectory
- Then `.github/settings.yml` at that checkout's root contains the exported settings
- And standard output is empty and standard error contains only applicable info/warnings
- And configuration is not written beneath the subdirectory

#### Scenario: Refresh an existing regular configuration

- Given `.github/settings.yml` is an existing regular file
- When the maintainer successfully runs `gh settings export`
- Then that file contains acquired values and retained unavailable independent intent

#### Scenario: Report a failed write

- Given `.github/settings.yml` is an existing regular file
- When a file write error prevents export from saving the complete configuration
- Then the command fails with a diagnostic on standard error
- And standard output is empty
- And the original existing configuration remains unchanged

#### Scenario: Refuse symlink destinations

- Given `.github` or `.github/settings.yml` is a symlink, including a dangling symlink
- When the maintainer runs `gh settings export`
- Then the command fails with a diagnostic on standard error
- And the symlink and its target remain unchanged

### Rule: Export only the current checkout's supported settings

#### Scenario: Export complete repository settings successfully

- Given the current Git checkout has a GitHub `origin` remote
- And its settings are description "Example repository", homepage "https://example.com", issues enabled, projects disabled, wiki enabled, and Discussions disabled
- When the maintainer runs `gh settings export`
- Then the command succeeds
- And `.github/settings.yml` uses `version: daiksud/gh-settings@v0` with About, Labels and General at their UI locations
- And visible controls retain completely observed typed values while hidden conditional menus are omitted after raw validation
- And topics and complete labels are present, while response metadata and unsupported settings are absent
- And standard error contains only applicable info/normalization warnings
- And standard output is empty
- And no repository settings are changed

#### Scenario: Preserve false and empty string values

- Given the repository's description and homepage are empty strings
- And issues, projects, wiki, and Discussions are disabled
- When the maintainer runs `gh settings export`
- Then the command succeeds
- And supported observed booleans remain present under their UI keys
- And creation-permission and merge-message menus hidden by false parents are omitted
- And the two string values remain empty strings
- And boolean controls retain their observed values, including explicit false

#### Scenario: Export unset strings as empty strings

- Given GitHub returns null for the repository description or homepage
- And the other string is null or an empty string
- When the maintainer runs `gh settings export`
- Then the command succeeds
- And About description and website are present as YAML empty strings, not YAML null values
- And the boolean settings retain their values

#### Scenario: Ignore a repository override

- Given the current checkout's GitHub origin identifies "owner/current"
- And `GH_REPO` identifies a different repository
- When the maintainer runs `gh settings export`
- Then the settings are read from "owner/current"
- And the override does not select another repository

#### Scenario: Require a directly configured GitHub origin in current source

- Given the current checkout's origin directly identifies "owner/current" through HTTPS, `git@github.com:owner/current.git` or a full SSH URL
- Then the settings are read from "owner/current" on GitHub

#### Scenario: Reject an SCP origin without a username in current source

- Given the current checkout's origin is "github.com:owner/current.git"
- Then the command fails with a repository-resolution diagnostic
- And no GitHub API request is made

#### Scenario: Keep repository identity when Git rewrites its transport URL

- Given the current checkout's configured origin identifies "owner/current" on GitHub
- And Git `insteadOf` rewrites the transport URL to a non-GitHub mirror
- When the maintainer runs `gh settings export`
- Then the settings are read from "owner/current" on GitHub
- And the rewritten transport URL does not change the selected repository

#### Scenario: Reject a rewrite-only transport alias in current source

- Given the current checkout's origin is a Git alias for "owner/current"
- And Git `insteadOf` rewrites the alias to a GitHub URL
- Then the command fails with a repository-resolution diagnostic
- And no GitHub API request is made

These origin-support changes are supported and do not change settings YAML syntax.

#### Scenario: Read an origin from repository-specific Git configuration

- Given the current checkout defines its origin in an included config or worktree config
- And that origin identifies "owner/current" on GitHub
- When the maintainer runs `gh settings export`
- Then the settings are read from "owner/current" on GitHub

### Rule: Exported settings can be validated unchanged

#### Scenario: Save export output and validate it locally

- Given the current checkout has ordinary settings or explicit false/empty values with a valid existing default branch
- When the maintainer runs `gh settings export`
- And runs `gh settings validate` with the same executable
- Then the document contains the exact current version, supported visible General controls, direct `repository.about.topics`, and complete `repository.labels`
- And validation succeeds without output or changing the saved file bytes
- And export reads repository settings, topics, and all label pages while validation makes no additional API request

### Rule: Report export failures

#### Scenario: About and basic General settings cannot be retrieved

- Given GitHub rejects or cannot complete the About/basic General settings read
- When the maintainer runs `gh settings export`
- Then independent readable groups are still acquired and saved
- And unavailable fields are omitted initially or retain compatible valid prior intent
- And standard error contains bounded family information without upstream details
- And standard output is empty

#### Scenario: There is no usable current repository

- Given the current directory has no GitHub origin remote
- When the maintainer runs `gh settings export`
- Then the command exits with a nonzero status
- And standard error explains the repository resolution failure
- And standard output is empty

#### Scenario: Global origin does not replace a missing checkout

- Given the current directory is outside a Git checkout
- And global Git configuration defines a GitHub origin
- When the maintainer runs `gh settings export`
- Then the command exits with a nonzero status
- And no GitHub API request is made

#### Scenario: One repository response cannot form a complete observation

- Given GitHub returns malformed repository data or omits a required group member
- When the maintainer runs `gh settings export`
- Then no partial value from that response is adopted
- And other independent groups continue with bounded information for the failed group
- And the resulting partial declaration validates locally
- And successful output does not claim that unobserved settings or repository existence were verified

#### Scenario: Standard output is unavailable

- Given standard output cannot be written
- When the maintainer runs `gh settings export`
- Then the command succeeds and saves `.github/settings.yml`
- And standard error contains only applicable info/normalization warnings

#### Scenario: A command is not implemented

- When the maintainer runs `gh settings unknown`
- Then the command exits with a nonzero status
- And standard error identifies the unsupported command
- And no API request is made

### Test mapping

These scenarios map to ordinary Go tests, [application export tests](../../internal/app/export_test.go), [GitHub read adapter tests](../../internal/github/repository_test.go), [workspace repository tests](../../internal/workspace/repository_test.go), [workspace file tests](../../internal/workspace/config_file_test.go), [export process cases](../../tests/acceptance/scenarios_export.go), [file output cases](../../tests/acceptance/export_file_test.go), [existing-file and worktree cases](../../tests/acceptance/export_workflow_test.go), and the [export-then-validate process test](../../tests/acceptance/export_validate_test.go), without a separate BDD runner. Application tests record target and destination resolution, the request for every supported Repository FieldID, complete typed observations, and the final save without Git or files. The five About/General observation groups are independently complete; unavailable groups emit info and retain compatible prior intent while successful siblings are saved. The [partial-export process regressions](../../tests/acceptance/repository_export_isolation_test.go) verify retained partial declarations and positively observed dependency suppression. Workspace tests retain guarded replacement and verify that a prepared saver remains bound to its resolved checkout root. Acceptance invokes the same already-built CLI against a local Unix-socket API fixture, rather than running the Go unit suite again:

| Scenario | Go test | Process acceptance case |
| --- | --- | --- |
| Export from a subdirectory without a settings directory | `TestCheckoutPrepareSaveBindsResolvedRoot` | `TestExportSavesAtCheckoutRoot` |
| Refresh an existing regular configuration | `TestWriteConfigAtRoot/replace`, `TestCheckoutPrepareSaveBindsResolvedRoot` | `TestExportExistingFile/replace` |
| Report a failed write | `TestExportStorageFailure` | `TestExportReportsWriteFailure` |
| Refuse symlink destinations and non-regular files | `TestWriteConfigRejectsUnsafeDestination`, `TestExportStorageFailure` | `TestExportRejectsUnsafeDestination` |
| Save to the linked checkout selected by its worktree origin | — | `TestExportLinkedWorktree` |
| Export complete repository settings successfully | `TestExportStoredSettings/five_values` | `five-settings` |
| Export observed Discussions true/false after complete reads | [TestExportStoredSettings](../../internal/app/export_test.go), [TestDiscussionsReadRequiresBoolean](../../internal/github/discussions_test.go) | `TestExportThenValidate/ordinary-values`, `TestExportThenValidate/zero-and-null-strings` |
| Preserve false and empty string values | `TestExportStoredSettings/zero_values` | `zero-values` |
| Export unset strings as empty strings | `TestReadRepositorySettings` | `unset-strings` |
| Ignore a repository override | `TestCurrentRepositoryIgnoresOverride` | `origin-override` |
| Require direct origins and reject username-less SCP | `TestCurrentRepositoryFormats` | `origin-scp-without-user` |
| Keep repository identity when Git rewrites its transport URL | `TestCurrentRepositoryKeepsDirectOriginBeforeTransportRewrite` | `origin-insteadof` |
| Reject a rewrite-only transport alias | `TestCurrentRepositoryRejectsRewriteOnlyAlias` | `origin-alias` |
| Read an origin from repository-specific Git configuration | `TestCurrentRepositoryIgnoresUnscopedOrigins` | `origin-worktree` |
| Save export output and validate it locally | — | `TestExportThenValidate/ordinary-values`, `TestExportThenValidate/zero-and-null-strings` |
| Unavailable base settings still save readable siblings | `TestReadRepositoryFailures`, [TestExportRepositoryUnavailableGroupsRetainOnlyPriorIntent](../../internal/app/export_repository_test.go) | `read-failure`, `TestExportExistingFile/api-failure` |
| There is no usable current repository | `TestExportTargetFailureBeforeRead`, `TestExportRootFailureBeforeRead`, `TestCheckoutPrepareSaveOutsideGit` | `repository-failure` |
| Global origin does not replace a missing checkout | `TestCurrentRepositoryRequiresCheckout` | — |
| The response cannot form a complete configuration | `TestReadRepositoryFailures` | `incomplete-response`, `TestExportExistingFile/malformed-response`, `TestExportExistingFile/incomplete-response` |
| Missing repository-group observations retain compatible prior intent | [TestExportRepositoryUnavailableGroupsRetainOnlyPriorIntent](../../internal/app/export_repository_test.go), [strict all-fields projection](../../internal/settings/requested_observation_test.go) | `TestRepositoryExportIndependentObservationsAcceptance` |
| Standard output is unavailable | [TestExportDoesNotWriteStdout](../../internal/cli/run_test.go) | `TestExportExistingFile/replace` |
| A command is not implemented | `TestUnsupportedCommand` | `unsupported-command` |

Export changes the local configuration but only reads GitHub settings; [apply](../commands.md#apply-behavior) owns GitHub mutation. For the repository response, only description/homepage nulls are normalized to empty strings; nullable label descriptions are covered by the [labels specification](labels.feature.md#rule-observe-complete-identity-and-report-honest-failures). A missing response key is an error, not an unset value. For the initial five input fields, [ADR 0019](../adr/0019-reject-explicit-null.md) rejects explicit YAML null under the update request schema.

### Rule: Export complete topics before writing

- Given repository settings and topic names can be read.
- When export runs.
- Then repository fields and sorted topic names are saved with the exact current version marker and UI paths.
- And a failed topics read emits info, retains prior topics and continues independent reads.

Test mapping: [TestReadableExportIndependentFailuresRetainIntent/Topics](../../internal/app/export_test.go) and [TestTopicsExportAndActionReport](../../tests/acceptance/topics_test.go).

### Rule: Read labels completely

Current source reads every label page after repository settings and topics before saving. Export includes IDs and complete supported settings for each observed object; a failed or incomplete read retains prior Labels with info while independent reads continue. Listed objects, not remote collection membership, are owned. The [Labels specification](labels.feature.md#rule-observe-complete-identity-and-report-honest-failures) owns these scenarios.

### Rule: Acquire supported setting families independently

Ordinary export attempts every supported family without selectors. Complete
observations replace prior intent. Unavailable or unsupported observations emit
bounded info and retain prior declarations, while other reads continue. This
includes complete object/variable inventories and independently read Actions
siblings. See the [automatic readable-settings contract](readable-settings-export.feature.md).
Existing explicit normalization policies still own raw/effective projections;
retained intent is never described as verified remote state. Plan/apply remain
strict and require complete observations for declared settings.

### Rule: Convert completely before replacing the destination

- Requested family mappings compose into one current UI document without reimplementing their converters.
- Complete typed Actions execution/workflow permissions, public-fork approval, Security status strings, creation policies, merge/squash menu pairs, complete Pages modes and legacy folders, Ruleset enforcement/bypass-mode/scanning-threshold strings, Required approvals integers, Allowed merge methods lists and Environment deployment-mode pairs/ref-type strings use their named fallback with original provenance and warnings; unsupported identities and incomplete/malformed observations remain errors. Acquired Ruleset `evaluate` enforcement and unsupported protection fail instead of being normalized or omitted. Observations never populate omitted desired intent.
- Raw latent conditional state is not replayed as selected UI controls. Enabled unknown execution policies warn and export the named Allow all fallback; disabled unknown policy exports known Disable without a warning. Execution fallback equality remains actionable, while hidden Disable matches remain unchanged. Missing/null/type failures never supply either choice.
- Export success does not prove future drift protection or writable access.

Test mapping: [UI export aggregation](../../internal/github/ui_export_test.go), [root deferred save](../../checkout_config_test.go), and the independent current-YAML maps/string goldens in the process files above.

### Rule: Export typed Required approvals without inventing PR rules

- Acquire a complete integer count per present PR rule. Out-of-range counts export UI 0 and warn with the original integer and ID/name; canonical 0–10 remain unchanged.
- Warnings follow Ruleset name order then enforcement then Required approvals. String and integer sinks work independently. An absent PR rule is not created.
- Missing/null/fractional/wrong-type/overflow values and incomplete policies preserve the destination. A later unsupported identity may retain legitimate earlier diagnostics but still prevents saving.
- Selected fallback-equal intent remains actionable. Canonical equality remains a no-op; unlisted IDs remain private.

Test mapping: [Reader](../../internal/github/ruleset_required_approvals_read_test.go), [typed export](../../internal/app/normalization_ruleset_required_approvals_export_test.go) and [built executable](../../tests/acceptance/normalization_ruleset_required_approvals_test.go).

### Rule: Export normalized Environment modes only after complete acquisition

- A typed equal deployment-policy pair projects No restriction with exact pair provenance; valid null remains exact. Missing/null/wrong-type members, custom protections and failed/incomplete reads never become defaults.
- Raw custom true requires all latent pattern pages even under the fallback. Keep latent IDs/values separate from effective policy and exported YAML; acquire known empty lists explicitly.
- Emit contextual mode warnings in folded-name order after complete collection acquisition. Failed/partial acquisition emits none; a later conversion failure may retain legitimate warnings but preserves the destination.
- Selected fallback-equal mode intent remains write-required. Omitted mode remains API-controlled without copying an observed pair or printing an unowned warning; unlisted names stay private.

Test mapping: [reader completeness](../../internal/github/environment_deployment_read_test.go), [export scope/order](../../internal/app/normalization_environment_deployment_export_test.go) and [built executable](../../tests/acceptance/normalization_environment_deployment_test.go).

### Rule: Export complete Environment ref-type projections

Given a complete active Selected pattern with an unsupported string type, export warns with its Environment, ID, name and raw/fallback values, and writes the literal `branch` UI choice alongside unchanged canonical companions. IDs and normalization evidence remain outside YAML. Warnings occur only after complete acquisition and before effective materialization.

If distinct raw identities collide after projection, emit scoped warnings and info, retaining the prior Environment declaration while other families continue. If a later type is missing/null or acquisition fails, emit info without partial normalization warnings and retain prior family intent. Patterns under effective No restriction remain latent; only applicable mode diagnostics and effective mode are exported. None of these paths writes repository settings.

Test mapping: [owned export](../../internal/app/normalization_environment_ref_type_export_test.go) and [built executable](../../tests/acceptance/normalization_environment_ref_type_test.go). The [Environment ref-type policy](../settings.md#environment-ref-type-observation-normalization) owns reconciliation and raw verification.

### Rule: Export bypass modes without inventing actors

- Complete acquired out-of-UI mode strings export `always` with exact Ruleset/actor warnings and unchanged canonical companions. No actor, identity, role or permission is synthesized.
- Warnings require complete requested acquisition and preserve name/type/numeric-ID order. Missing/null modes, malformed identities and failed later reads preserve the destination without partial normalization warnings. A later UI conversion failure may retain valid acquisition warnings while still saving nothing.
- Callback mutation cannot change the owned observation or saved policy. Ordinary export attempts Rulesets. Selected fallback equality remains actionable and actual removal requires strict raw confirmation.

Test mapping: [export scope/order/ownership](../../internal/app/normalization_ruleset_bypass_export_test.go), [complete reader](../../internal/github/ruleset_bypass_read_test.go) and [executable export/plan/apply](../../tests/acceptance/normalization_ruleset_bypass_test.go).

### Rule: Export merge methods with whole-list provenance

A complete unique nonempty string list containing an unsupported method exports the full Merge, Squash, Rebase default with the exact raw list in an ID/name-bound warning. Original raw order is retained; canonical companion fields and absent PR rules remain intact. Callback mutations cannot change the exported policy or retained observation. Warnings follow name order, then enforcement, Required approvals, Allowed merge methods, scanning Alerts and Security alerts (each in tool-name order), bypass actors.

Missing/null/empty/duplicate/malformed lists or incomplete acquisition emit info and retain prior family intent without partial projection warnings. Ordinary export attempts Rulesets; pure readers never warn or write. [Export tests](../../internal/app/normalization_ruleset_merge_methods_export_test.go) and [executable commands](../../tests/acceptance/normalization_ruleset_merge_methods_test.go) cover scope, typed provenance, literal UI output and destination preservation. The [Allowed merge methods policy](../settings.md#ruleset-allowed-merge-methods-observation-normalization) owns reconciliation and strict confirmation.

## Rule: Export scanning Alerts with exact tool identity

A complete out-of-UI Alerts string exports Errors and an exact raw-string warning scoped to Ruleset ID/name and tool name. Tool identity remains unchanged; names sort exactly, canonical thresholds and Security alerts remain intact. Absent rules and empty tool lists stay distinct. Missing/null/non-string values, duplicate or invalid tools and incomplete acquisition produce no plausible partial export or normalization evidence.

The [export tests](../../internal/app/normalization_ruleset_scanning_alerts_export_test.go) verify the independently optional warning sink, canonical silence, owned callbacks and failure-before-save behavior. The [executable sequence](../../tests/acceptance/normalization_ruleset_scanning_alerts_test.go) checks literal UI YAML, contextual warnings, write-required plans and reports, detailed/status-only Summary, independently authored mutations, rejected raw drift, actual tool/rule removal, canonical repetition and unlisted-ID privacy. These fixtures use local APIs and dummy credentials.

## Rule: Export scanning Security alerts independently

Complete unsupported Security alerts strings export High or higher with exact raw evidence and Ruleset/tool context. Alerts records and canonical companion values remain independent. Each threshold's warnings use exact tool-name order; Security alerts follow Alerts. Invalid identities, missing/null/non-string members, duplicates and incomplete reads preserve the destination without invented defaults.

[Export use-case tests](../../internal/app/normalization_ruleset_scanning_security_alerts_export_test.go) verify the independent sink, owned values, canonical silence and failure boundaries. [Executable tests](../../tests/acceptance/normalization_ruleset_scanning_security_alerts_test.go) verify literal UI export, normalized write-required plans/reports, detailed/status-only Summary, independent whole-policy payloads, rejected raw drift, canonical repetition, actual removal and simultaneous Alerts/Security alerts evidence. They use local APIs and dummy credentials and make no live settings changes.

### Rule: Automatic Security export preserves unread prior intent

The executable [automatic export cases](../../tests/acceptance/readable_security_export_test.go)
cover independent continuation, omission versus preservation, known false values,
last-family success, bounded diagnostics and invalid-existing-file rejection.
[Snapshot tests](../../internal/workspace/export_snapshot_test.go) cover concurrent
edits and a file appearing after first-export preparation. Plan/apply retain their
strict observation failures; all supported families now use automatic export.
