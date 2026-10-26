---
type: Feature
title: Manual release with calculated metadata
description: Specify automatically numbered, tested, and recoverable GitHub CLI extension releases.
---

## Feature: Publish a manually authorized release

A maintainer explicitly starts publication from `main`. The version is calculated during dispatch from reviewed, committed changes before validation. The workflow publishes exactly the binaries accepted for macOS arm64 and Linux amd64.

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

#### Scenario: Calculate from immutable source without a preparation PR

- Given a published stable baseline and an exact selected main source
- When the Python resolver reads the entire Git interval at workflow dispatch
- Then it emits the next SemVer, published predecessor tag, and predecessor SHA
- And no persistent version-file edit, preparation PR, new tag, or release is required
- And empty commits do not create an increment, while all nonempty source changes do
- But a preparation-named commit with another changed path remains substantive
- And no substantive changes, invalid history, or malformed metadata fail closed

#### Scenario: A published Action selects its matching CLI release

- Given a published version-tag or a full SHA pin to a uniquely tagged published commit
- When the Action installs the CLI
- Then it downloads only that release's expected binary and checksums
- And an untagged SHA, conflicting tag mapping, or failed verification rejects installation
- And local candidate tests use the exact accepted product version, not a release preparation file

### Rule: Freeze the source and prove binary identity

#### Scenario: Dispatch is not from main

- Given the release workflow was manually dispatched from another ref
- When source resolution begins
- Then the run fails explicitly before an API lookup, checkout, build, tag, or release creation
- And the run is not reported as a successful set of skipped jobs

#### Scenario: Main moves before source resolution

- Given the workflow was dispatched for main source A
- When current main is already source B during resolution
- Then resolution stops before checkout, build, tag, or release creation
- And it does not replace the dispatched source with B
- And an optional reviewed source pin is checked separately without a token

#### Scenario: Release two validated native binaries

- Given the maintainer manually dispatches the workflow from main
- And source A is the immutable source for the calculated version
- When main advances to B during validation
- Then both binaries are built once from A with that shared embedded version
- And each product is transferred with a manifest binding A, version, target, asset name, and SHA-256
- And native acceptance executes each verified target binary
- And supported Ubuntu candidate smoke uses A's actual root Action and the already-built Linux artifact for a separate consumer
- And all four consumer/CLI identities match before publication
- And the tag points to A with only the two raw binaries and `SHA256SUMS` attached
- And post-publication verification compares real public downloads byte-for-byte with both accepted manifests
- And the published root Action performs real release downloads for a separate consumer and reports all four identities

#### Scenario: Validation fails before remote state

- Given version resolution, commit checks, native acceptance, candidate smoke, or hash verification fails
- Or a publication prerequisite is skipped or cancelled
- When the workflow evaluates the publication gate
- Then no tag or release is created by that run
- And no override converts the missing validation into success

#### Scenario: Orchestration delegates deterministic product work

- Given the shared workflow freezes one source and resolves allowlisted build and acceptance matrices from that source
- When commit, acceptance, publication, and verification run
- Then the byte-identical workflow delegates each product-specific operation to its local Action
- And the wrappers own pinned artifact upload/download with digest mismatch rejection and exact inventory checks
- And publication obtains binary hashes only from manifests reverified in its own job
- And no build, acceptance, or verification step receives a token environment
- And only publication requests Contents write

### Rule: Preserve release identity across retries

#### Scenario: Resume an incomplete matching draft

- Given a draft marker, tag, calculated version, and frozen source identify the same release
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
- Then breaking labels take precedence over deprecation, feature/performance and fix labels
- And deprecation takes precedence over feature/performance and fix labels
- And the Maintenance catch-all retains every remaining PR, including revert and unlabeled PRs
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

PR/main CI establishes current-source integration, not real published downloads or live native-note rendering. Only an explicitly authorized manual release run establishes operational publication evidence. No successful merge or calculated version alone authorizes that run.
