---
type: Feature
title: Export repository settings
description: Specify the user-visible YAML export and its ordinary Go test mapping.
---

## Feature: Export repository settings as YAML

A repository maintainer can capture the current five supported settings as configuration without changing GitHub settings. `export` writes to standard output; creating `.github/settings.yml` is an explicit shell redirection, and an existing configuration file is not required.

### Rule: Export only the current checkout's supported settings

#### Scenario: Export five settings successfully

- Given the current Git checkout has a GitHub `origin` remote
- And its settings are description "Example repository", homepage "https://example.com", issues enabled, projects disabled, and wiki enabled
- When the maintainer runs `gh settings export`
- Then the command succeeds
- And standard output has root `schemaVersion: 1` and `settings.repository` containing those five fields and values
- And response metadata and other settings are absent
- And standard error is empty
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
- When the maintainer saves `gh settings export` output as `.github/settings.yml`
- And runs `gh settings validate` with the same executable
- Then the exported document contains `schemaVersion: 1`, `settings.repository`, and exactly the five supported values
- And validation succeeds without output or changing the saved file bytes
- And export makes one settings read while validation makes no API request

### Rule: Report failures without emitting a successful configuration

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

- Given the output destination rejects the exported data
- When the maintainer runs `gh settings export`
- Then the command exits with a nonzero status
- And standard error describes the output failure

#### Scenario: A command is not implemented

- When the maintainer runs `gh settings apply`
- Then the command exits with a nonzero status
- And standard error identifies the unsupported command
- And no API request is made

### Test mapping

These scenarios map to ordinary Go tests, [export process cases](../../tests/acceptance/scenarios_export.go), and the [export-then-validate process test](../../tests/acceptance/export_validate_test.go), without a separate BDD runner. Acceptance invokes the same already-built CLI against a local Unix-socket API fixture, rather than running the Go unit suite again:

| Scenario | Go test | Process acceptance case |
| --- | --- | --- |
| Export five settings successfully | `TestExportFiveSettings` | `five-settings` |
| Preserve false and empty string values | `TestExportPreservesZeroValues` | `zero-values` |
| Export unset strings as empty strings | `TestExportNormalizesUnsetStrings` | `unset-strings` |
| Ignore a repository override | `TestExportUsesOrigin` | `origin-override` |
| Resolve an SCP origin without a username | `TestRepositoryRemotes` | `origin-scp-without-user` |
| Keep repository identity when Git rewrites its transport URL | `TestExportUsesConfiguredOriginBeforeInsteadOf` | `origin-insteadof` |
| Resolve a transport alias to GitHub | `TestExportUsesAliasRewrittenToGitHub` | `origin-alias` |
| Read an origin from repository-specific Git configuration | `TestExportUsesRepositoryConfigSources` | `origin-worktree` |
| Save export output and validate it locally | — | `TestExportThenValidate/ordinary-values`, `TestExportThenValidate/zero-and-null-strings` |
| Settings cannot be retrieved | `TestExportReadFailure`, `TestExportNetworkFailure` | `read-failure` |
| There is no usable current repository | `TestExportRepositoryFailure` | `repository-failure` |
| Global origin does not replace a missing checkout | `TestExportRejectsGlobalOriginOutsideCheckout` | — |
| The response cannot form a complete configuration | `TestExportRejectsIncompleteResponse` | `incomplete-response` |
| Standard output is unavailable | `TestExportOutputFailure` | `output-failure` |
| A command is not implemented | `TestUnsupportedCommand` | `unsupported-command` |

The read-only export contract is separate from the future [mutation commands](../commands.md). Only description/homepage nulls returned by the read API are normalized to empty strings. A missing response key is an error, not an unset value. For the initial five input fields, [ADR 0019](../adr/0019-reject-explicit-null.md) rejects explicit YAML null under the update request schema.
