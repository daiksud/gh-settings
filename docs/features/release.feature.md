---
type: Feature
title: Manual release with calculated metadata
description: Specify automatically numbered, tested, and recoverable GitHub CLI extension releases.
---

## Feature: Publish a manually authorized release

A maintainer explicitly starts publication from `main`. The version is calculated from reviewed, committed changes and prepared in source before validation. The workflow publishes exactly the binaries accepted for macOS arm64 and Linux amd64.

### Rule: Derive one version from durable reviewed changes

#### Scenario: Select the strongest signal in the complete interval

- Given a published stable baseline and a fixed descendant source
- When all committed messages in the interval are classified
- Then any breaking header or breaking-change footer selects MINOR during `0.x` and MAJOR from `1.0.0` onward
- And a breaking signal does not automatically graduate `0.x` to `1.0.0`
- Otherwise `feat`, `perf`, or `DEPRECATED:` selects MINOR
- Otherwise fixes, reverts, maintenance, and legacy messages select PATCH
- And later main commits and mutable branch names or labels do not affect that calculation
- And a revert does not subtract another included commit's signal

#### Scenario: Prepare a shared Action and CLI version

- Given a clean tracked checkout with complete history and a stable release baseline
- When the preparation command computes the next version
- Then it updates only `action/cli-version` for a reviewed preparation PR
- And it does not create a commit, branch, tag, draft, or release
- And a preparation-only commit does not create another version increment
- But a preparation-named commit that changes another path remains a substantive PATCH signal
- And an interval containing no other changes is rejected

#### Scenario: Prepared source no longer agrees

- Given the frozen source records a different version from the calculated result
- When the release resolver validates that source
- Then it stops before building or creating a tag or release
- And it does not silently rewrite source or publish an Action that installs another CLI

### Rule: Freeze the source and prove binary identity

#### Scenario: Release two validated native binaries

- Given the maintainer manually dispatches the workflow from main
- And source A contains the correctly prepared calculated version
- When main advances to B during validation
- Then both binaries are built once from A with that shared embedded version
- And native acceptance executes each verified target binary
- And supported Ubuntu candidate smoke uses A's actual root Action and the already-built Linux artifact for a separate consumer
- And all four consumer/CLI identities match before publication
- And the tag points to A with only the two raw binaries and `SHA256SUMS` attached
- And post-publication smoke verifies A's tag/source/version before real release downloads

#### Scenario: Validation fails before remote state

- Given preparation, commit checks, native acceptance, candidate smoke, or hash verification fails
- Or a publication prerequisite is skipped or cancelled
- When the workflow evaluates the publication gate
- Then no tag or release is created by that run
- And no override converts the missing validation into success

### Rule: Preserve release identity across retries

#### Scenario: Resume an incomplete matching draft

- Given a draft marker, tag, committed version, and frozen source identify the same release
- And one required asset is absent while the others match their expected bytes
- When the release run is retried
- Then it retains that version rather than recalculating another increment
- And the publisher uploads only the missing asset after checking the tested inputs
- And it preserves the existing notes and comparison range even if another release was published

#### Scenario: Complete release is retried

- Given the published release, tag, marker, and every asset match the tested inputs
- When the release job is retried
- Then it succeeds without replacing assets, moving the tag, or regenerating notes

#### Scenario: A conflicting existing identity is not a retry

- Given a tag, manifest, source, checksum, or asset differs
- When the workflow inspects existing state
- Then it fails without repairing an immutable release or fabricating a match
- And missing/malformed release metadata is not interpreted as an empty catalog

### Rule: Generate native release notes at a fixed boundary

#### Scenario: Select the previous published stable release

- Given stable numeric-tag releases, drafts, prereleases, and unpublished tags
- When preparing a new release
- Then the latest stable publication timestamp selects the predecessor, with ID breaking ties
- And its tag and resolved commit are frozen before validation
- And a shallow checkout, missing tag, unrelated predecessor, or changed boundary stops publication

#### Scenario: Generate GitHub categories and preserve migration guidance

- Given reviewed labels on the merged PRs included by GitHub
- When the publisher requests native notes with the version, frozen target SHA, previous tag, and `.github/release.yml`
- Then breaking labels take precedence over feature/performance/deprecation and fix/revert labels
- And the Chore catch-all retains uncategorized PRs
- And a minimal supplement preserves committed breaking migration footers as escaped text
- And an immutable comparison covers the complete source interval, including direct commits
- And native notes are not claimed to contain every PR body or every Git commit as an entry

#### Scenario: Generation fails before tag creation

- Given the note API fails or returns missing, malformed, or empty content
- When new notes are prepared
- Then the publisher stops before creating a remote tag or draft
- And it does not silently use a second local categorized renderer

### Test mapping

The [release-metadata tests](../../scripts/release-metadata/main_test.go) use temporary Git repositories and an injected API boundary to exercise version precedence, complete frozen ranges, preparation writes, immutable retries, invalid catalogs/history, exact native generation arguments, preserved migration details, and failures. They do not execute GitHub's renderer or mutate GitHub. The repository already has published stable releases; automatic first-release baseline selection is deliberately unsupported.

The [release fixtures](../../tests/release.sh) invoke the real publisher with a local `gh` stand-in. They retain first publication, failed generation before remote state, complete/incomplete retries, note preservation, source/version conflicts, missing or altered assets, annotated tags, and final tag rechecks. The full expected body combines independently supplied native fixture content, migration details, and the unchanged installation/checksum/identity template, including long content. No fixture creates a GitHub tag or release.

The [workflow gate tests](../../tests/github-actions/release_test.go) protect frozen source/version/artifact/output mappings and ordinary prerequisite-success dependencies. `TestReleaseOptionalSourcePin` executes the actual source step with Bash and an owned `gh` stub: omitted/matching pins succeed, while invalid/mismatched pins, invalid responses, and API failures emit no source output. The fixture requires Bash 5 like hosted Ubuntu; structural checks remain available on older local shells. [Metadata wiring tests](../../tests/github-actions/release_metadata_test.go) protect the manual-only trigger, read-only resolver, frozen predecessor outputs, and native category precedence/catch-all.

The existing [publisher preflight](../../scripts/release-publisher/main_test.go), [lookup/tag](../../scripts/release-publisher/lookup_test.go), [state](../../scripts/release-publisher/state_test.go), [asset](../../scripts/release-publisher/assets_test.go), and [controller](../../scripts/release-publisher/publish_test.go) tests retain validation order, exact checksum/manifest bytes, duplicate/unknown assets, downloaded hashes, missing-only uploads, immutable published no-ops, and fresh confirmation. The [CLI test](../../internal/cli/run_test.go) still owns the explicit version command.

PR/main CI establishes current-source integration, not real published downloads or live native-note rendering. Only an explicitly authorized manual release run establishes operational publication evidence. No version preparation or successful merge alone authorizes that run.
