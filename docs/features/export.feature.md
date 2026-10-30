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

Ordinary export reads the non-opt-in repository fields, topics and labels. Each explicit selection adds only its independent resource: `--rulesets`, `--actions`, `--workflow-permissions`, `--retention`, `--fork-approval`, `--private-fork-workflows`, repeatable distinct `--security` families, `--archive`, `--pages` and `--environments`. Duplicate/unknown selections fail before export. Complete all requested observations and serialization before opening the destination; an inapplicable/denied/partial selection fails without plausible defaults or a successful partial save. Unchanged exact supported exports validate and plan with zero differences. Out-of-UI workflow permissions follow the [normalization specification](workflow-permissions.feature.md#feature-retain-provenance-for-out-of-ui-workflow-permissions), which retains warnings and does not treat a fallback match as raw equality. Complete merge/squash pairs and creation policies follow the same distinction under their [menu](../settings.md#commit-message-observation-normalization) and [creation-policy](../settings.md#creation-policy-observation-normalization) contracts. Resource applicability can prevent combining otherwise independently parsed choices; public and private fork choices cannot both apply to a fixed target.

The dedicated [Rulesets](rulesets.feature.md), [workflow permissions](workflow-permissions.feature.md), [retention](retention.feature.md), [security](security.feature.md), [fork workflow](fork-workflows.feature.md), [PR access](pull-request-access.feature.md) and [Issue policy](issue-creation-policy.feature.md) specifications own complete-read and payload details. Existing process coverage includes [Rulesets](../../tests/acceptance/rulesets_test.go), [workflow permissions](../../tests/acceptance/workflow_permissions_test.go), [retention](../../tests/acceptance/retention_test.go), [Secret scanning](../../tests/acceptance/security_test.go), [private reporting](../../tests/acceptance/private_reporting_test.go), [fork workflows](../../tests/acceptance/fork_workflows_test.go) [PR access](../../tests/acceptance/pr_access_test.go) and [Issue policy](../../tests/acceptance/issue_creation_policy_test.go). These test mappings prove modeled current-source behavior; publication evidence remains separate from fixture verification.

### Rule: Convert completely before replacing the destination

- Requested family mappings compose into one current UI document without reimplementing their converters.
- Complete typed workflow permissions, creation policies and merge/squash menu pairs use their named fallback with original provenance and warnings; other unknown/out-of-UI values still fail pending the remaining #480 work. Observations never populate omitted desired intent.
- Raw latent conditional state is not replayed as selected UI controls.
- Complete UI report integration remains #481; an export success does not claim that work is complete.

Test mapping: [UI export aggregation](../../internal/github/ui_export_test.go), [root deferred save](../../checkout_config_test.go), and the independent current-YAML maps/string goldens in the process files above.
