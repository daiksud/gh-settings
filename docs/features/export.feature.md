---
type: Feature
title: Export repository settings
description: Specify the user-visible YAML export and its ordinary Go test mapping.
---

## Feature: Export repository settings as YAML

A repository maintainer can capture supported visible About/General controls, topics and labels as configuration without changing GitHub settings. `export` saves `.github/settings.yml` at the current checkout root, creates a missing `.github` directory, and succeeds without stdout. Explicit selection diagnostics can appear on stderr. It always overwrites an existing regular file after completing API reading and YAML serialization. A failed write can leave a truncated or partial file; restoration of tracked configuration belongs to Git. Symlinks at `.github` or `settings.yml` and non-regular destinations are rejected.

The [Discussions contract](discussions.feature.md) owns its strict read rules and version boundary.

This feature assumes an owner-controlled checkout whose directory and link entries remain stable during export. Concurrent replacement of those entries is outside its scope.

### Rule: Save configuration in the checkout selected by origin

#### Scenario: Export from a subdirectory without a settings directory

- Given the current Git checkout has a GitHub `origin` remote
- And it has no `.github` directory
- When the maintainer runs `gh settings export` from a subdirectory
- Then `.github/settings.yml` at that checkout's root contains the exported settings
- And standard output and standard error are empty
- And configuration is not written beneath the subdirectory

#### Scenario: Refresh an existing regular configuration

- Given `.github/settings.yml` is an existing regular file
- When the maintainer successfully runs `gh settings export`
- Then that file contains the complete exported configuration

#### Scenario: Report a failed write

- Given `.github/settings.yml` is an existing regular file
- When a file write error prevents export from saving the complete configuration
- Then the command fails with a diagnostic on standard error
- And standard output is empty
- And the maintainer may need Git to restore a truncated or partial configuration

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
- And standard error is empty
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

#### Scenario: Settings cannot be retrieved

- Given GitHub rejects or cannot complete the settings read
- When the maintainer runs `gh settings export`
- Then the command exits with a nonzero status
- And standard error describes the failure
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

#### Scenario: The response cannot form a complete configuration

- Given GitHub returns malformed data or omits a supported setting
- When the maintainer runs `gh settings export`
- Then the command exits with a nonzero status
- And standard error describes the invalid response
- And standard output is empty

#### Scenario: Standard output is unavailable

- Given standard output cannot be written
- When the maintainer runs `gh settings export`
- Then the command succeeds and saves `.github/settings.yml`
- And standard error is empty

#### Scenario: A command is not implemented

- When the maintainer runs `gh settings unknown`
- Then the command exits with a nonzero status
- And standard error identifies the unsupported command
- And no API request is made

### Test mapping

These scenarios map to ordinary Go tests, [application export tests](../../internal/app/export_test.go), [GitHub read adapter tests](../../internal/github/repository_test.go), [workspace repository tests](../../internal/workspace/repository_test.go), [workspace file tests](../../internal/workspace/config_file_test.go), [export process cases](../../tests/acceptance/scenarios_export.go), [file output cases](../../tests/acceptance/export_file_test.go), [existing-file and worktree cases](../../tests/acceptance/export_workflow_test.go), and the [export-then-validate process test](../../tests/acceptance/export_validate_test.go), without a separate BDD runner. Application tests record target and destination resolution, the request for every supported Repository FieldID, complete typed observations, and the final save without Git or files. Missing required observations stop export before later reads or saving. Workspace tests retain guarded replacement and verify that a prepared saver remains bound to its resolved checkout root. Acceptance invokes the same already-built CLI against a local Unix-socket API fixture, rather than running the Go unit suite again:

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
| Settings cannot be retrieved | `TestReadRepositoryFailures`, `TestExportReadFailure` | `read-failure`, `TestExportExistingFile/api-failure` |
| There is no usable current repository | `TestExportTargetFailureBeforeRead`, `TestExportRootFailureBeforeRead`, `TestCheckoutPrepareSaveOutsideGit` | `repository-failure` |
| Global origin does not replace a missing checkout | `TestCurrentRepositoryRequiresCheckout` | — |
| The response cannot form a complete configuration | `TestReadRepositoryFailures` | `incomplete-response`, `TestExportExistingFile/malformed-response`, `TestExportExistingFile/incomplete-response` |
| Required Repository observations are missing | [TestExportMissingRepositoryObservationPreventsSave](../../internal/app/export_test.go), [TestRepositoryObservationDeclarationRequiresCompleteValues](../../internal/settings/requested_observation_test.go) | — |
| Standard output is unavailable | [TestExportDoesNotWriteStdout](../../internal/cli/run_test.go) | `TestExportExistingFile/replace` |
| A command is not implemented | `TestUnsupportedCommand` | `unsupported-command` |

Export changes the local configuration but only reads GitHub settings; [apply](../commands.md#apply-behavior) owns GitHub mutation. For the repository response, only description/homepage nulls are normalized to empty strings; nullable label descriptions are covered by the [labels specification](labels.feature.md#rule-observe-complete-identity-and-report-honest-failures). A missing response key is an error, not an unset value. For the initial five input fields, [ADR 0019](../adr/0019-reject-explicit-null.md) rejects explicit YAML null under the update request schema.

### Rule: Export complete topics before writing

- Given repository settings and topic names can be read.
- When export runs.
- Then repository fields and sorted topic names are saved with the exact current version marker and UI paths.
- And a failed topics read leaves an existing destination unchanged.

Test mapping: [TestExportReadFailure/topics](../../internal/app/export_test.go) and [TestTopicsExportAndActionReport](../../tests/acceptance/topics_test.go).

### Rule: Read labels completely

Current source reads every label page after repository settings and topics before saving. Export includes IDs and complete supported settings for each observed object; a failed or incomplete read preserves the destination. Listed objects, not remote collection membership, are owned. The [Labels specification](labels.feature.md#rule-observe-complete-identity-and-report-honest-failures) owns these scenarios.

### Rule: Actions execution policy export opt-in

Current source keeps ordinary export unchanged and adds explicit `--actions`, composing with `--rulesets`. The [Actions policy specification](actions-execution-policy.feature.md#rule-export-only-complete-observable-policy) owns complete active/inactive reads, preserved destinations and test mappings.

Ordinary export reads the non-opt-in repository fields, topics and labels. Each explicit selection adds only its independent resource: `--rulesets`, `--actions`, `--workflow-permissions`, `--retention`, `--fork-approval`, `--private-fork-workflows`, repeatable distinct `--security` families, `--archive`, `--pages` and `--environments`. Duplicate/unknown selections fail before export. Complete all requested observations and serialization before opening the destination; an inapplicable/denied/partial selection fails without plausible defaults or a successful partial save. Unchanged exact supported exports validate and plan with zero differences. Out-of-UI workflow permissions follow the [normalization specification](workflow-permissions.feature.md#feature-retain-provenance-for-out-of-ui-workflow-permissions), which retains warnings and does not treat a fallback match as raw equality. Complete merge/squash pairs and creation policies follow the same distinction under their [menu](../settings.md#commit-message-observation-normalization) and [creation-policy](../settings.md#creation-policy-observation-normalization) contracts. Complete Security status strings follow the [typed boolean normalization policy](../settings.md#secret-scanning-and-push-protection), including inconsistent projection errors without saving. Public-fork approval follows its [raw-aware normalization policy](../settings.md#fork-pr-workflow-policies). Complete Pages modes and legacy folders follow their [raw-source policy](../settings.md#existing-pages-configuration), retaining fallback-equal write intent and preserving unowned source values. Actions execution follows its [visible/latent normalization policy](../settings.md#execution-policy-observation-normalization). Ruleset enforcement, [Required approvals](../settings.md#ruleset-required-approvals-observation-normalization) and [Quality severity](../settings.md#ruleset-quality-severity-observation-normalization) retain [ID-bound raw provenance](../settings.md#ruleset-enforcement-observation-normalization); export warnings follow name order, then enforcement, Required approvals, Allowed merge methods, Team reviewer Approvals (numeric reviewer ID order), Quality and actor-ordered [bypass modes](../settings.md#ruleset-bypass-mode-observation-normalization) per object; fallback-equal selected intent remains actionable. Resource applicability can prevent combining otherwise independently parsed choices; public and private fork choices cannot both apply to a fixed target.

The dedicated [Rulesets](rulesets.feature.md), [workflow permissions](workflow-permissions.feature.md), [retention](retention.feature.md), [security](security.feature.md), [fork workflow](fork-workflows.feature.md), [PR access](pull-request-access.feature.md) and [Issue policy](issue-creation-policy.feature.md) specifications own complete-read and payload details. Existing process coverage includes [Rulesets](../../tests/acceptance/rulesets_test.go), [workflow permissions](../../tests/acceptance/workflow_permissions_test.go), [retention](../../tests/acceptance/retention_test.go), [Secret scanning](../../tests/acceptance/security_test.go), [private reporting](../../tests/acceptance/private_reporting_test.go), [fork workflows](../../tests/acceptance/fork_workflows_test.go) [PR access](../../tests/acceptance/pr_access_test.go) and [Issue policy](../../tests/acceptance/issue_creation_policy_test.go). These test mappings prove modeled current-source behavior; publication evidence remains separate from fixture verification.

### Rule: Convert completely before replacing the destination

- Requested family mappings compose into one current UI document without reimplementing their converters.
- Complete typed Actions execution/workflow permissions, public-fork approval, Security status strings, creation policies, merge/squash menu pairs, complete Pages modes and legacy folders, Ruleset enforcement/Quality severity/bypass-mode strings, Required approvals integers, Allowed merge methods lists and Environment deployment-mode pairs use their named fallback with original provenance and warnings; other unknown/out-of-UI values still fail pending the remaining #480 work. Observations never populate omitted desired intent.
- Raw latent conditional state is not replayed as selected UI controls. Enabled unknown execution policies warn and export the named Allow all fallback; disabled unknown policy exports known Disable without a warning. Execution fallback equality remains actionable, while hidden Disable matches remain unchanged. Missing/null/type failures never supply either choice.
- Complete UI report integration remains #481; an export success does not claim that work is complete.

Test mapping: [UI export aggregation](../../internal/github/ui_export_test.go), [root deferred save](../../checkout_config_test.go), and the independent current-YAML maps/string goldens in the process files above.

### Rule: Preserve Quality rule absence and simultaneous provenance

- Complete typed unknown Quality severity exports Errors with exact raw/ID-bound provenance and a contextual warning. Enforcement and Quality warnings compose in fixed control order within name-sorted objects.
- An absent Quality rule remains absent; missing/null/type or incomplete later policies fail without saving. Neither fallback supplies desired intent.
- A selected fallback-equal exported declaration remains actionable. Ordinary exact canonical policy without explicit write-only dismissal intent stays a no-op; unlisted IDs remain undisclosed and unwritten.

Test mapping: [complete Reader](../../internal/github/ruleset_quality_read_test.go), [export ordering/scope](../../internal/app/normalization_ruleset_quality_export_test.go) and [executable export/plan/apply](../../tests/acceptance/normalization_ruleset_quality_test.go).

### Rule: Export typed Required approvals without inventing PR rules

- Acquire a complete integer count per present PR rule. Out-of-range counts export UI 0 and warn with the original integer and ID/name; canonical 0–10 remain unchanged.
- Warnings follow Ruleset name order then enforcement, Required approvals, Quality. String and integer sinks work independently. An absent PR rule is not created.
- Missing/null/fractional/wrong-type/overflow values and incomplete policies preserve the destination. A later unsupported identity may retain legitimate earlier diagnostics but still prevents saving.
- Selected fallback-equal intent remains actionable. Canonical equality without explicit write-only dismissal intent remains a no-op; unlisted IDs remain private.

Test mapping: [Reader](../../internal/github/ruleset_required_approvals_read_test.go), [typed export](../../internal/app/normalization_ruleset_required_approvals_export_test.go) and [built executable](../../tests/acceptance/normalization_ruleset_required_approvals_test.go).

### Rule: Export normalized Environment modes only after complete acquisition

- A typed equal deployment-policy pair projects No restriction with exact pair provenance; valid null remains exact. Missing/null/wrong-type members, custom protections and failed/incomplete reads never become defaults.
- Raw custom true requires all latent pattern pages even under the fallback. Keep latent IDs/values separate from effective policy and exported YAML; acquire known empty lists explicitly.
- Emit contextual mode warnings in folded-name order after complete collection acquisition. Failed/partial acquisition emits none; a later conversion failure may retain legitimate warnings but preserves the destination.
- Selected fallback-equal mode intent remains write-required. Omitted mode remains API-controlled without copying an observed pair or printing an unowned warning; unlisted names stay private.

Test mapping: [reader completeness](../../internal/github/environment_deployment_read_test.go), [export scope/order](../../internal/app/normalization_environment_deployment_export_test.go) and [built executable](../../tests/acceptance/normalization_environment_deployment_test.go). Full #481 reporting remains pending.

### Rule: Export complete Environment ref-type projections

Given a complete active Selected pattern with an unsupported string type, export warns with its Environment, ID, name and raw/fallback values, and writes the literal `branch` UI choice alongside unchanged canonical companions. IDs and normalization evidence remain outside YAML. Warnings occur only after complete acquisition and before effective materialization.

If distinct raw identities collide after projection, emit scoped warnings and fail without replacing the existing file. If a later type is missing/null or acquisition fails, emit no partial normalization warnings and preserve the file. Patterns under effective No restriction remain latent; only applicable mode diagnostics and effective mode are exported. None of these paths writes repository settings.

Test mapping: [owned export](../../internal/app/normalization_environment_ref_type_export_test.go) and [built executable](../../tests/acceptance/normalization_environment_ref_type_test.go). The [Environment ref-type policy](../settings.md#environment-ref-type-observation-normalization) owns reconciliation and raw verification.

### Rule: Export bypass modes without inventing actors

- Complete acquired out-of-UI mode strings export `always` with exact Ruleset/actor warnings and unchanged canonical companions. No actor, identity, role or permission is synthesized.
- Warnings require complete requested acquisition and preserve name/type/numeric-ID order. Missing/null modes, malformed identities and failed later reads preserve the destination without partial normalization warnings. A later UI conversion failure may retain valid acquisition warnings while still saving nothing.
- Callback mutation cannot change the owned observation or saved policy. Ordinary export does not read Rulesets. Selected fallback equality remains actionable and actual removal requires strict raw confirmation.

Test mapping: [export scope/order/ownership](../../internal/app/normalization_ruleset_bypass_export_test.go), [complete reader](../../internal/github/ruleset_bypass_read_test.go) and [executable export/plan/apply](../../tests/acceptance/normalization_ruleset_bypass_test.go).

### Rule: Export merge methods with whole-list provenance

A complete unique nonempty string list containing an unsupported method exports the full Merge, Squash, Rebase default with the exact raw list in an ID/name-bound warning. Original raw order is retained; canonical companion fields and absent PR rules remain intact. Callback mutations cannot change the exported policy or retained observation. Warnings follow name order, then enforcement, Required approvals, Allowed merge methods, Team reviewer Approvals (numeric reviewer ID order), Quality and bypass actors.

Missing/null/empty/duplicate/malformed lists or incomplete acquisition fail before save and emit no partial projection warnings. Ordinary export does not request Rulesets; pure readers never warn or write. [Export tests](../../internal/app/normalization_ruleset_merge_methods_export_test.go) and [executable commands](../../tests/acceptance/normalization_ruleset_merge_methods_test.go) cover scope, typed provenance, literal UI output and destination preservation. The [Allowed merge methods policy](../settings.md#ruleset-allowed-merge-methods-observation-normalization) owns reconciliation and strict confirmation.

## Rule: Export Team reviewer counts without inventing equality

A complete out-of-range Team approval count exports 0 with an exact integer warning bound to Ruleset ID/name and Team ID. Numeric reviewer-ID order is stable even when the API list is reordered. Canonical counts, repeated/negated pattern order and explicit empty patterns are retained. Absent PR rules and empty reviewer lists stay distinct; malformed members, duplicate identities and later acquisition failures preserve the destination and produce no partial normalization warnings.

The [export use-case tests](../../internal/app/normalization_ruleset_team_approvals_export_test.go) cover independently optional warning sinks, simultaneous records, owned callbacks, canonical silence and failure-before-save behavior. The [executable command sequence](../../tests/acceptance/normalization_ruleset_team_approvals_test.go) verifies literal exported UI YAML and integer warnings, selected write-required plans/reports, detailed/status-only Summary, independently authored whole-object mutations, rejected raw drift, actual reviewer/PR-rule removal, canonical repetition and unlisted-ID privacy. It uses local API fixtures and dummy credentials, not live settings.
