---
type: Feature
title: Export repository settings
description: Specify the user-visible YAML export and its ordinary Go test mapping.
---

## Feature: Export repository settings as YAML

A repository maintainer can capture the current five supported settings as configuration without changing GitHub settings. `export` saves `.github/settings.yml` at the current checkout root, creates a missing `.github` directory, and succeeds silently. It always overwrites an existing regular file after completing API reading and YAML serialization. A failed write can leave a truncated or partial file; restoration of tracked configuration belongs to Git. Symlinks at `.github` or `settings.yml` and non-regular destinations are rejected.

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

#### Scenario: Export five settings successfully

- Given the current Git checkout has a GitHub `origin` remote
- And its settings are description "Example repository", homepage "https://example.com", issues enabled, projects disabled, and wiki enabled
- When the maintainer runs `gh settings export`
- Then the command succeeds
- And `.github/settings.yml` has root `schemaVersion: 1` and `settings.repository` containing those five fields and values
- And response metadata and other settings are absent
- And standard error is empty
- And standard output is empty
- And no repository settings are changed

#### Scenario: Preserve false and empty string values

- Given the repository's description and homepage are empty strings
- And issues, projects, and wiki are disabled
- When the maintainer runs `gh settings export`
- Then the command succeeds
- And all five keys are present in the YAML output
- And the two string values remain empty strings
- And the three boolean values remain false

#### Scenario: Export unset strings as empty strings

- Given GitHub returns null for the repository description or homepage
- And the other string is null or an empty string
- When the maintainer runs `gh settings export`
- Then the command succeeds
- And description and homepage are present as YAML empty strings, not YAML null values
- And the boolean settings retain their values

#### Scenario: Ignore a repository override

- Given the current checkout's GitHub origin identifies "owner/current"
- And `GH_REPO` identifies a different repository
- When the maintainer runs `gh settings export`
- Then the settings are read from "owner/current"
- And the override does not select another repository

#### Scenario: Resolve an SCP origin without a username

- Given the current checkout's origin is "github.com:owner/current.git"
- When the maintainer runs `gh settings export`
- Then the command succeeds
- And the settings are read from "owner/current" on GitHub

#### Scenario: Keep repository identity when Git rewrites its transport URL

- Given the current checkout's configured origin identifies "owner/current" on GitHub
- And Git `insteadOf` rewrites the transport URL to a non-GitHub mirror
- When the maintainer runs `gh settings export`
- Then the settings are read from "owner/current" on GitHub
- And the rewritten transport URL does not change the selected repository

#### Scenario: Resolve a transport alias to GitHub

- Given the current checkout's origin is a Git alias for "owner/current"
- And Git `insteadOf` rewrites the alias to a GitHub URL
- When the maintainer runs `gh settings export`
- Then the settings are read from "owner/current" on GitHub

#### Scenario: Read an origin from repository-specific Git configuration

- Given the current checkout defines its origin in an included config or worktree config
- And that origin identifies "owner/current" on GitHub
- When the maintainer runs `gh settings export`
- Then the settings are read from "owner/current" on GitHub

### Rule: Exported settings can be validated unchanged

#### Scenario: Save export output and validate it locally

- Given the current checkout has ordinary settings or zero values with an unset string
- When the maintainer runs `gh settings export`
- And runs `gh settings validate` with the same executable
- Then the exported document contains `schemaVersion: 1`, `settings.repository`, and exactly the five supported values
- And validation succeeds without output or changing the saved file bytes
- And export makes one settings read while validation makes no API request

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

These scenarios map to ordinary Go tests, [application export tests](../../internal/app/export_test.go), [GitHub read adapter tests](../../internal/github/repository_test.go), [workspace repository tests](../../internal/workspace/repository_test.go), [workspace file tests](../../internal/workspace/config_file_test.go), [export process cases](../../tests/acceptance/scenarios_export.go), [file output cases](../../tests/acceptance/export_file_test.go), [existing-file and worktree cases](../../tests/acceptance/export_workflow_test.go), and the [export-then-validate process test](../../tests/acceptance/export_validate_test.go), without a separate BDD runner. Acceptance invokes the same already-built CLI against a local Unix-socket API fixture, rather than running the Go unit suite again:

| Scenario | Go test | Process acceptance case |
| --- | --- | --- |
| Export from a subdirectory without a settings directory | — | `TestExportSavesAtCheckoutRoot` |
| Refresh an existing regular configuration | `TestExportStoredSettings/zero_values` | `TestExportExistingFile/replace` |
| Report a failed write | — | `TestExportReportsWriteFailure` |
| Refuse symlink destinations and non-regular files | `TestWriteConfigRejectsUnsafeDestination`, `TestExportStorageFailure` | `TestExportRejectsUnsafeDestination` |
| Save to the linked checkout selected by its worktree origin | — | `TestExportLinkedWorktree` |
| Export five settings successfully | `TestExportStoredSettings/five_values` | `five-settings` |
| Preserve false and empty string values | `TestExportStoredSettings/zero_values` | `zero-values` |
| Export unset strings as empty strings | `TestReadRepositorySettings` | `unset-strings` |
| Ignore a repository override | `TestExportUsesOrigin` | `origin-override` |
| Resolve an SCP origin without a username | `TestRepositoryRemotes` | `origin-scp-without-user` |
| Keep repository identity when Git rewrites its transport URL | `TestExportUsesConfiguredOriginBeforeInsteadOf` | `origin-insteadof` |
| Resolve a transport alias to GitHub | `TestExportUsesAliasRewrittenToGitHub` | `origin-alias` |
| Read an origin from repository-specific Git configuration | `TestExportUsesRepositoryConfigSources` | `origin-worktree` |
| Save export output and validate it locally | — | `TestExportThenValidate/ordinary-values`, `TestExportThenValidate/zero-and-null-strings` |
| Settings cannot be retrieved | `TestReadRepositoryFailures`, `TestExportReadFailure` | `read-failure`, `TestExportExistingFile/api-failure` |
| There is no usable current repository | `TestExportTargetFailureBeforeRead` | `repository-failure` |
| Global origin does not replace a missing checkout | `TestExportRejectsGlobalOriginOutsideCheckout` | — |
| The response cannot form a complete configuration | `TestReadRepositoryFailures` | `incomplete-response`, `TestExportExistingFile/malformed-response`, `TestExportExistingFile/incomplete-response` |
| Standard output is unavailable | [TestExportDoesNotWriteStdout](../../internal/cli/run_test.go) | `TestExportExistingFile/replace` |
| A command is not implemented | `TestUnsupportedCommand` | `unsupported-command` |

Export changes the local configuration but only reads GitHub settings; [apply](../commands.md#apply-behavior) owns GitHub mutation. Only description/homepage nulls returned by the read API are normalized to empty strings. A missing response key is an error, not an unset value. For the initial five input fields, [ADR 0019](../adr/0019-reject-explicit-null.md) rejects explicit YAML null under the update request schema.
