---
type: Feature
title: Manual release
description: Specify versioned, tested, and recoverable GitHub CLI extension releases.
---

## Feature: Publish a manually selected version

A maintainer can start a release from `main` with an explicit version. The workflow publishes exactly the binaries accepted for macOS arm64 and Linux amd64.

### Rule: Freeze the source and prove binary identity

#### Scenario: Release two validated native binaries

- Given the maintainer dispatches the workflow from `main` with version `v1.2.3`
- And latest `main` resolves to commit A at workflow start
- And commit A records CLI version `v1.2.3` in `action/cli-version`
- When `main` advances to commit B while the workflow runs
- Then both release binaries are built once from commit A with embedded version `v1.2.3`
- And commit checks pass before native acceptance checks run
- And each acceptance runner executes its own target binary after SHA-256 verification
- And the release job verifies both accepted binary hashes before creating a tag
- And tag `v1.2.3` points to commit A
- And the GitHub Release contains exactly the two raw binaries and `SHA256SUMS`
- And the release notes identify commit A and include changes and installation guidance
- And a supported Ubuntu runner checks out the published tag and verifies it is commit A before executing its root Action
- And credential-free validate reports the requested CLI version and accepted Linux binary digest for a separate configuration-only checkout

#### Scenario: Source version differs from the requested tag

- Given the frozen source records CLI version `v1.2.3`
- When the maintainer requests release `v1.2.4`
- Then the workflow fails before building or creating a tag or release
- And it does not silently publish an Action that installs the older CLI

#### Scenario: Validation fails before the tag

- Given a target binary fails commit checks, acceptance, or hash verification
- When the workflow stops
- Then no tag or release is created by that run

### Rule: Preserve release identity across retries

#### Scenario: Resume an incomplete matching draft

- Given tag `v1.2.3` already points to the frozen commit
- And a draft release records the same version, commit, and tested binary hashes
- And one required asset is absent while the others match their expected bytes
- When the release job resumes with the same verified inputs
- Then it uploads only the missing asset
- And it publishes after all required assets and notes are verified

#### Scenario: Complete release is retried

- Given the published release, tag, manifest, and all assets match the tested inputs
- When the release job is retried
- Then it succeeds without replacing an asset or moving the tag

#### Scenario: Identity conflict stops publication

- Given the existing tag, draft manifest, or asset bytes disagree with the tested inputs
- When the release job inspects existing state
- Then it fails without moving the tag, replacing an asset, or publishing the draft

#### Scenario: Invalid version is rejected

- Given a version does not match `vMAJOR.MINOR.PATCH` with numeric components
- When the maintainer dispatches the workflow
- Then the resolve job fails before any tag or release is created

### Test mapping

The [release mock tests](../../tests/release.sh) exercise first publication, complete and incomplete retries, version rejection, tag conflict, manifest conflict, and asset mismatch without contacting GitHub. The [CLI Go test](../../internal/cli/run_test.go) checks the explicit version command. The native runner and job ordering contracts are declared in the [workflow](../../.github/workflows/release.yml) and require a real Actions run to confirm operationally. That run must only use a release version the maintainer authorizes.
