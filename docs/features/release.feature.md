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
- And a read-only supported Ubuntu job validates a configuration-only consumer through commit A's actual root Action using the already-built Linux artifact
- And its four outputs match the frozen Action source, consumer commit/target and accepted CLI version/digest before publication
- And the release job verifies both accepted binary hashes before creating a tag
- And tag `v1.2.3` points to commit A
- And the GitHub Release contains exactly the two raw binaries and `SHA256SUMS`
- And the release notes identify commit A and present grouped changes before installation, upgrade, and checksum guidance
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

#### Scenario: Candidate validation does not succeed

- Given the mandatory pre-publication candidate job fails, is skipped or is cancelled
- When the workflow evaluates the publication prerequisites
- Then publish does not run and this run creates no tag or release
- And no override treats the missing candidate validation as successful

### Rule: Preserve release identity across retries

#### Scenario: Resume an incomplete matching draft

- Given tag `v1.2.3` already points to the frozen commit
- And a draft release records the same version, commit, and tested binary hashes
- And one required asset is absent while the others match their expected bytes
- And another release has since been published
- When the release job resumes with the same verified inputs
- Then it uploads only the missing asset
- And it preserves the existing notes and their original comparison range without regenerating them
- And it publishes after all required assets and notes are verified

#### Scenario: Complete release is retried

- Given the published release, tag, manifest, and all assets match the tested inputs
- When the release job is retried
- Then it succeeds without replacing an asset or moving the tag
- And it preserves the published notes without regenerating them

#### Scenario: Identity conflict stops publication

- Given the existing tag, draft manifest, or asset bytes disagree with the tested inputs
- When the release job inspects existing state
- Then it fails without moving the tag, replacing an asset, or publishing the draft

#### Scenario: Invalid version is rejected

- Given a version does not match `vMAJOR.MINOR.PATCH` with numeric components
- When the maintainer dispatches the workflow
- Then the resolve job fails before any tag or release is created

### Rule: Generate release notes from the frozen Git history

#### Scenario: Select the previous published release

- Given the repository has published stable releases with numeric `vMAJOR.MINOR.PATCH` tags
- And it also has unpublished tags, draft releases, and prereleases
- When notes are prepared for a new release
- Then the previous release is the qualifying release with the latest publication timestamp, excluding the requested version
- And version-number ordering, unpublished tags, drafts, and prereleases do not select the boundary

#### Scenario: Include only the released commit range

- Given the previous published release resolves to commit P
- And the current release freezes descendant commit S
- And `main` or `HEAD` later points to a newer commit
- When notes are generated
- Then they contain the commits selected by `git log P..S`
- And each entry links to its commit
- And the comparison link uses P and S as immutable commit SHAs

#### Scenario: Generate the first release's notes

- Given there is no qualifying previous published release
- When notes are generated for frozen commit S
- Then they include all history reachable from S
- And an unpublished tag does not truncate that history

#### Scenario: Unverifiable history stops publication

- Given the checkout is shallow, the selected predecessor tag is missing locally, or its commit is not an ancestor of the frozen source
- When notes are prepared for a new release
- Then generation fails before creating a remote tag or release
- And it does not silently select an older predecessor or treat the release as the first release

#### Scenario: Classify complete Conventional Commits messages

- Given the released range contains Conventional Commits and legacy messages
- When the notes are rendered
- Then headers with `!` or uppercase `BREAKING CHANGE:` or `BREAKING-CHANGE:` footers go under `## 🚨 BREAKING CHANGE`
- And breaking entries retain their footer's impact and migration description without a duplicate entry in another section
- And nonbreaking `feat` and `perf` commits go under `## ✨ New Feature`
- And nonbreaking `fix` commits go under `## 🐛 Bug Fix`
- And other types and unclassified legacy messages go under `## 🧹 Chore`
- And empty sections are omitted

### Test mapping

The [release-notes helper tests](../../scripts/release-notes/main_test.go) use temporary Git repositories to exercise published predecessor selection, frozen ranges, first releases, invalid history, breaking-footer handling, type classification, and Markdown rendering. The [release mock tests](../../tests/release.sh) exercise first publication, note-generation failure before remote changes, preservation of existing notes, complete and incomplete retries, version rejection, tag conflict, manifest conflict, and asset mismatch without contacting GitHub.

The [CLI Go test](../../internal/cli/run_test.go) checks the explicit version command. The [candidate gate tests](../../tests/github-actions/release_test.go) inspect the actual workflow's frozen-source/artifact/version/output mappings and ordinary prerequisite-success dependency. They also execute the unchanged candidate helper with matching identities and separate source/version/checksum mismatches, checking exact local asset/manifest/version bytes and rejection before PATH publication without downloads. Regular PR/main CI executes the same-source candidate composite; these source/fixture checks do not dispatch the manual release or prove published downloads. The native runner and job ordering contracts are declared in the [workflow](../../.github/workflows/release.yml) and require a real Actions run to confirm operationally. That run must only use a release version the maintainer authorizes.
