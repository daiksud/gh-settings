---
type: Specification
title: Release design
description: Define the proposed release pipeline, distribution, recovery, and future version automation.
sources:
  - id: github-cli-extensions
    resource: https://docs.github.com/en/github-cli/github-cli/creating-github-cli-extensions
    title: Creating GitHub CLI extensions
  - id: go-linker
    resource: https://pkg.go.dev/cmd/link#hdr-Command_Line
    title: Go linker command line
---

## Status and initial targets

This page specifies a future release pipeline. No release workflow or publication mechanism is implemented by this documentation change. The existing [main CI](../.github/workflows/ci-push.yml) validates its own build; a release must build and validate its own platform binaries rather than reuse that CI artifact.

The first releases target only these platforms:

| Platform | Go target | GitHub Release executable |
| --- | --- | --- |
| macOS Apple Silicon | `darwin/arm64` | `gh-settings-darwin-arm64` |
| Linux x86_64 | `linux/amd64` | `gh-settings-linux-amd64` |

GitHub CLI recognizes precompiled extension assets with an OS-architecture suffix.[^github-cli-extensions] Publish raw, executable files under those names, plus SHA-256 checksums for the published binaries and release notes. The distribution contract follows common Go CLI release practice; adopting GoReleaser itself has not been decided. Additional platforms and archives are outside the initial release scope.

## Initial release flow

1. A maintainer starts a release workflow manually and supplies an explicit version. The workflow records the latest `main` commit once at its start. Every checkout, build, test, tag, and release operation uses that recorded commit, even if `main` advances during the run.
2. Inject the supplied version while building each target binary, then complete the commit stage. Go's linker supports setting a string symbol with `-X` when its declaration is suitable; the exact symbol and build wiring remain implementation details.[^go-linker] Build each platform binary only once in this release pipeline.
3. Run the acceptance stage against each exact binary intended for publication, on a compatible platform, and verify its SHA-256 across stage transfers. The existing [commit-stage](../.github/actions/commit-stage/action.yml) and [acceptance-stage](../.github/actions/acceptance-stage/action.yml) are intended for reuse or refactoring, but their current product build must not cause a second release build. A matching source commit and version alone do not prove that two binaries have identical bytes.
4. Only after both targets pass, create the tag at the recorded commit and stage a draft GitHub Release for the supplied version. Upload the tested executables and their checksums, and prepare release notes. Verify the expected assets and metadata before publishing the draft.

Do not overwrite an existing tag or a published release asset. A conflicting version, tag target, asset name, or artifact identity stops publication. After an interruption, inspect the tag, draft, and uploaded artifacts; resume unfinished steps only when the version, commit, and tested artifact bytes can be verified as identical. The recovery mechanism and evidence storage still need implementation design. An incomplete draft must not be presented as a published release.

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

Workflow implementation, the exact version and tag spelling, checksum file name, release-note generation, compatible acceptance runners, and recovery evidence storage remain open mechanics. GoReleaser adoption is undecided. These details must preserve the contracts above.

[^github-cli-extensions]: [Creating GitHub CLI extensions](https://docs.github.com/en/github-cli/github-cli/creating-github-cli-extensions), checked on 2026-10-12 for precompiled extension release asset naming.
[^go-linker]: [Go linker command line](https://pkg.go.dev/cmd/link#hdr-Command_Line), checked on 2026-10-12 for the conditional `-X` string-symbol behavior.
