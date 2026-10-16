---
type: Specification
title: Release design
description: Define the manual release pipeline, distribution, recovery, and future version automation.
sources:
  - id: github-cli-extensions
    resource: https://docs.github.com/en/github-cli/github-cli/creating-github-cli-extensions
    title: Creating GitHub CLI extensions
  - id: go-linker
    resource: https://pkg.go.dev/cmd/link#hdr-Command_Line
    title: Go linker command line
  - id: github-rest-releases
    resource: https://docs.github.com/en/rest/releases/releases
    title: REST API endpoints for releases
---

## Status and initial targets

The [manual release workflow](../.github/workflows/release.yml) implements the initial pipeline. The existing [main CI](../.github/workflows/ci-push.yml) validates its own build; the release builds and validates separate platform binaries. Merging the workflow does not trigger a release.

The first releases target only these platforms:

| Platform | Go target | GitHub Release executable |
| --- | --- | --- |
| macOS Apple Silicon | `darwin/arm64` | `gh-settings-darwin-arm64` |
| Linux x86_64 | `linux/amd64` | `gh-settings-linux-amd64` |

GitHub CLI recognizes precompiled extension assets with an OS-architecture suffix.[^github-cli-extensions] The release publishes raw, executable files under those names, a `SHA256SUMS` asset, and release notes with installation guidance and generated changes. The binaries come from `go build`; GoReleaser is not used for this scope. Additional platforms and archives are outside the initial release scope.

## Start a manual release

After the owner chooses the intended version, update `action/cli-version` to it through an ordinary reviewed PR and verify main CI. The Action and CLI share that version: the release workflow rejects any differing input before building, and the publish script checks again before creating a tag or release. Then review the version and current `main` commit and explicitly start the [release workflow](../.github/workflows/release.yml) from `main` with its required `version` input (`vMAJOR.MINOR.PATCH`):

```sh
gh workflow run release.yml --repo daiksud/gh-settings --ref main -f version=v1.2.3
```

The workflow is not scheduled and does not run when code is merged. Its resolve job validates the version and freezes the then-current `main` SHA for every build and check in that run. A later run resolves `main` again; it does not reuse the earlier run's frozen SHA.

## Initial release flow

1. The resolve job reads `refs/heads/main` once through GitHub API and records its commit SHA. Every later checkout uses that SHA, even if `main` advances.
2. The commit stage first requires the pinned source's `action/cli-version` to equal the requested version, then checks that source and builds Linux amd64 once. The macOS arm64 runner builds its binary once after commit checks pass. Both use Go linker `-X main.version=...`; `gh settings --version` verifies the embedded value.[^go-linker]
3. Each native runner downloads its current run's binary and checks the SHA-256 supplied by its build job. The shared [acceptance-stage](../.github/actions/acceptance-stage/action.yml) launches that exact binary against the local API fixture. The release job downloads both binaries again and checks those SHA-256 values before any tag is written.
4. After both acceptance jobs pass, the release job creates a lightweight tag at the frozen SHA if the tag is absent, creates a draft with generated change notes and an identity manifest, uploads the two binaries and `SHA256SUMS`, downloads any present assets to verify their bytes, and publishes only when all three required assets and metadata match.

After publication, `published-action-smoke` checks out the exact published tag into a separate package on supported Ubuntu 24.04. Before executing it, the job requires the tag commit to equal the frozen accepted source and its package version to equal the release input. It invokes that unchanged root Action locally with `command: validate`, no token and no version override, then checks all four outputs against a configuration-only checkout, the release version and the accepted Linux build digest. This verifies the published tag's Action bytes and real released CLI; it is not a dynamic remote `uses` reference.

The smoke runs after publication. A failure leaves an already-published immutable release for diagnosis; do not delete, retag or replace assets to make verification pass. Rerun only the appropriate verification after establishing its cause and the same source/release identity.

## Action version and pre-release checks

A versioned Action installs the CLI version committed in its package, and a strict version-tag invocation must match that file. Reviewed SHA pins also select the committed version, without a mutable latest lookup. The installer verifies the exact release's `SHA256SUMS` before executing the fixed Linux asset and checking its reported version. The digest is trusted from that GitHub release over fixed HTTPS paths rather than embedded in the Action source; this avoids requiring a binary checksum inside the source commit used to build that binary.

Pre-release PR/main CI builds one current-source candidate with the committed CLI version. Acceptance and actual local composite validate, plan and direct apply use that same binary and the producer's exact source SHA through a bounded same-run download fixture, without a copied version override or old-release fallback. Plan uses a required Unix-socket API and dummy read token, including representative read/config failures and actual Summary-byte checks. Direct apply uses a separate dummy write token and exact received PATCH assertions, with representative write failure, invalid configuration and empty no-op cases; every apply case preserves the consumer and writes no new Settings review. This proves source integration and rendering before publication without requiring unpublished assets; it does not prove Summary publication or live API behavior. See the [CI validation boundary](development.md#ci-validation). Offline fixtures separately check committed-version selection, strict tag mismatch, checksum failures and the publication guard.

Published v0.6.0 predates this synchronization and its immutable Action still installs CLI v0.5.0. Retain its tag and assets with accurate limitation guidance; changing main cannot repair that revision. A reviewed SHA containing the fix can use the already-published v0.6.0 CLI. Publishing a corrected version requires explicit owner authorization.

## Retry and recovery

The draft body records version, pinned commit, both binary SHA-256 values, and the checksum file SHA-256 in a machine-readable comment. On retry, the release job finds the draft in the authenticated releases list and reads it by release ID; GitHub's tag lookup only returns published releases.[^github-rest-releases] It then compares the manifest, checks the existing lightweight or annotated tag target, downloads existing assets to compare their bytes, and uploads only missing assets. Immediately before publication it fetches and checks the tag again. Unexpected assets, an altered manifest, hash mismatches, a conflicting tag, or a missing tag for an existing release stop the run without recreating or moving that tag. An already published matching release is a verified no-op; published assets are never replaced.

A new workflow run resolves current `main`. If the earlier run created the tag, a different SHA stops at that existing tag. If the earlier run failed before tag creation, a new run can publish the same version from the newly resolved SHA; review that commit and confirm it is the intended release source before dispatching again. The original run's frozen SHA remains available in its logs; its artifacts are retained for one day. Keep incomplete drafts for inspection rather than deleting or replacing them automatically. There is no destructive cleanup or clobber.

This release pipeline is separate from ordinary main CI. Source resolution precedes the commit-stage validation and Linux build. The macOS build and Linux acceptance can then run concurrently; macOS acceptance waits for its build. Publication waits for both acceptance jobs before creating or checking the tag and release. The [single-build decision](adr/0012-publish-tested-binaries.md) governs any refactoring of the existing composites.

## Future automatic versioning

The initial workflow continues to require a manually supplied version. A later version may derive SemVer from strict Conventional Commits PR titles merged since the previous release. Breaking changes take precedence for a major bump; otherwise a new feature causes a minor bump; otherwise the bump is patch. No special pre-1.0 exception has been decided.

The exact title signal used to identify every breaking change remains unresolved. Requiring `!` in PR titles was suggested but not accepted. Do not treat that suggestion as a current title rule. Automatic version calculation is not part of the initial release workflow.

## Decisions and open mechanics

- [ADR 0010: Require an explicit version for initial releases](adr/0010-explicit-release-version.md)
- [ADR 0011: Pin each release to one main commit](adr/0011-pin-release-source.md)
- [ADR 0012: Publish the exact tested binaries](adr/0012-publish-tested-binaries.md)
- [ADR 0013: Never overwrite an existing release tag](adr/0013-immutable-release-identity.md)
- [ADR 0014: Publish only a complete draft release](adr/0014-draft-release-publication.md)
- [ADR 0015: Distribute GitHub CLI compatible raw executables](adr/0015-raw-extension-binaries.md)
- [ADR 0016: Use strict Conventional Commits PR titles](adr/0016-conventional-pr-titles.md)
- [ADR 0017: Derive future SemVer bumps from merged PRs](adr/0017-pr-based-semver.md)
- [ADR 0018: Never overwrite published release assets](adr/0018-preserve-published-assets.md)

The release tests in [tests/release.sh](../tests/release.sh) use a local `gh` stand-in and create no real tags or releases. They cover a first publication, completed retry, incomplete draft recovery, version rejection, conflicting tag and manifest, altered asset bytes, a tag change before publication, and annotated tag resolution at both checks. A live release is not exercised by these tests. Future automatic versioning remains deferred.

[^github-cli-extensions]: [Creating GitHub CLI extensions](https://docs.github.com/en/github-cli/github-cli/creating-github-cli-extensions), checked on 2026-10-12 for precompiled extension release asset naming.
[^go-linker]: [Go linker command line](https://pkg.go.dev/cmd/link#hdr-Command_Line), checked on 2026-10-12 for the conditional `-X` string-symbol behavior.
[^github-rest-releases]: [REST API endpoints for releases](https://docs.github.com/en/rest/releases/releases), checked on 2026-10-12 for draft visibility in authenticated release listings and published release tag lookup.
