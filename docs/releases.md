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

## Initial release flow

1. A maintainer starts the workflow on `main` with an explicit `vMAJOR.MINOR.PATCH` version. The resolve job reads `refs/heads/main` once through GitHub API and records its commit SHA. Every later checkout uses that SHA, even if `main` advances.
2. The commit stage checks the pinned source and builds Linux amd64 once. The macOS arm64 runner builds its binary once after commit checks pass. Both use Go linker `-X main.version=...`; `gh settings --version` verifies the embedded value.[^go-linker]
3. Each native runner downloads its current run's binary and checks the SHA-256 supplied by its build job. The shared [acceptance-stage](../.github/actions/acceptance-stage/action.yml) launches that exact binary against the local API fixture. The release job downloads both binaries again and checks those SHA-256 values before any tag is written.
4. After both acceptance jobs pass, the release job creates a lightweight tag at the frozen SHA, creates a draft with generated change notes and an identity manifest, uploads the two binaries and `SHA256SUMS`, downloads any present assets to verify their bytes, and publishes only when all three required assets and metadata match.

The draft body records version, pinned commit, both binary SHA-256 values, and the checksum file SHA-256 in a machine-readable comment. On retry, the release job finds the draft in the authenticated releases list and reads it by release ID; GitHub's tag lookup only returns published releases.[^github-rest-releases] It then compares the manifest, checks the existing tag target, downloads existing assets to compare their bytes, and uploads only missing assets. Unexpected assets, an altered manifest, hash mismatches, a conflicting tag, or a missing tag for an existing release stop the run without recreating that tag. An already published matching release is a verified no-op. There is no destructive cleanup or clobber. A new workflow run resolves current main, so a retry after main advances will stop at the old tag; the original frozen run remains the source of its SHA and artifacts.

This release pipeline is separate from ordinary main CI. Its stage order is version injection and build, commit-stage validation, acceptance-stage validation, tag, then release publication. The [single-build decision](adr/0012-publish-tested-binaries.md) governs any refactoring of the existing composites.

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

The release tests in [tests/release.sh](../tests/release.sh) use a local `gh` stand-in and create no real tags or releases. They cover a first publication, completed retry, incomplete draft recovery, version rejection, conflicting tag and manifest, and altered asset bytes. A live release is not exercised by these tests. Future automatic versioning remains deferred.

[^github-cli-extensions]: [Creating GitHub CLI extensions](https://docs.github.com/en/github-cli/github-cli/creating-github-cli-extensions), checked on 2026-10-12 for precompiled extension release asset naming.
[^go-linker]: [Go linker command line](https://pkg.go.dev/cmd/link#hdr-Command_Line), checked on 2026-10-12 for the conditional `-X` string-symbol behavior.
[^github-rest-releases]: [REST API endpoints for releases](https://docs.github.com/en/rest/releases/releases), checked on 2026-10-12 for draft visibility in authenticated release listings and published release tag lookup.
