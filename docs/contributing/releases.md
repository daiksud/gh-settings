# Maintainer releases

The [manual release workflow](../../.github/workflows/release.yml) builds and validates the release's own platform binaries. Merging code does not dispatch it, and ordinary main CI artifacts are not substituted for release artifacts. Installation belongs in the [README](../../README.md); current published consumer pins belong in the [Actions guide](../github-actions.md#published-package-and-consumer-pins).

Building the source CLI exposes current-source features, while the public Action installs the release selected by [`action/cli-version`](../../action/cli-version). Version preparation does not publish those assets. Verify the matching release before adopting new features through public Action pins; caller permission changes are a separate [repository automation concern](repository-automation.md#active-settings-workflows).

## Release assets

| Platform | Go target | Raw executable |
| --- | --- | --- |
| macOS Apple Silicon | `darwin/arm64` | `gh-settings-darwin-arm64` |
| Linux x86_64 | `linux/amd64` | `gh-settings-linux-amd64` |

Each release contains those two raw executables, `SHA256SUMS`, and notes identifying the source commit, installation guidance, and generated changes. The executable names follow the [GitHub CLI extension asset convention](https://docs.github.com/en/github-cli/github-cli/creating-github-cli-extensions). The pipeline builds with Go; archives, additional platforms, and GoReleaser are outside the current release scope.

## Start a manual release

1. Have the owner select the intended version. Update [`action/cli-version`](../../action/cli-version) to that version through an ordinary reviewed PR, then verify main CI.
2. Review the intended current `main` commit and version. The Action and CLI share one version; the workflow checks equality before building, and the publisher checks it again before creating a tag or release.
3. Explicitly dispatch the workflow from `main` using the chosen `vMAJOR.MINOR.PATCH` version. For example, after choosing `v1.2.3`:

```sh
gh workflow run release.yml --repo daiksud/gh-settings --ref main -f version=v1.2.3
```

Only the three-component numeric format is accepted, with no leading zeros except a component equal to zero. Prerelease/build suffixes are not accepted. The workflow's resolve job reads the then-current `main` SHA once and freezes it for every build and check in that run.

For a dispatch tied to an already reviewed exact commit, optionally supply `source_sha` as the full 40-character SHA. Resolve requires current `main` to equal it before freezing the source, so an intervening push fails instead of releasing another commit. Omitting it selects current `main` at resolve time. This input does not bypass checks on existing tags, drafts, or assets.

## Release flow

| Stage | Required behavior |
| --- | --- |
| Resolve | Validate the version, optionally match `source_sha`, and freeze one `main` commit |
| Commit/build | Check frozen source and version equality, then build Linux amd64 once; after commit checks, build macOS arm64 once on its native runner |
| Native acceptance | Download each run's corresponding binary, verify its SHA-256 and embedded version, and launch that exact executable against the local API fixture |
| Candidate Action smoke | Use frozen Action source and the already-built Linux artifact on supported Ubuntu; actual credential-free validate must report all four expected identities for a separate configuration-only consumer |
| Publish | Verify both accepted binary hashes, create/validate the tag and draft, upload only missing matching assets, recheck identities, and publish the complete release |
| Published Action smoke | Check out the published tag, verify its source/version, execute its unchanged root Action with real release downloads, and compare outputs with the accepted Linux digest |

Builds inject `main.version` with the Go linker's [`-X` option](https://pkg.go.dev/cmd/link#hdr-Command_Line). Native acceptance uses the shared [acceptance-stage](../../.github/actions/acceptance-stage/action.yml). Its test harness is compiled separately, without rebuilding the accepted product.

`candidate-action-smoke` is a normal required prerequisite of publication alongside both native acceptance jobs. Failure, skipping, or cancellation prevents publication; there is no unconditional or `continue-on-error` publication override. The candidate uses a bounded fixture for download bytes, so its success establishes pre-publication integration rather than real released-download identity. [Release gate tests](../../tests/github-actions/release_test.go) protect that source/version/artifact/dependency contract.

The publisher creates a lightweight tag at the frozen commit if absent and a draft with generated notes plus an identity manifest. It downloads any existing assets to check bytes and publishes only after the two binaries, checksum file, and metadata agree.

Post-publication smoke verifies that the published tag resolves to the frozen accepted source and its committed CLI version equals the release input before executing it. It invokes that root Action locally with credential-free validate on a separate consumer, without a version override, and checks all four outputs against the consumer and accepted release identities. This exercises the actual published tag's Action and real downloaded CLI; it does not use a dynamic remote `uses` reference.

Inspect both the publication job and the post-publication smoke result. A smoke failure occurs after the release is public and leaves that immutable release available for diagnosis.

## Action version and pre-release checks

A versioned Action installs the CLI selected by its committed `action/cli-version`. Strict version-tag invocation must match that file; a reviewed SHA pin selects its committed version without a mutable latest lookup. The installer verifies the exact release's `SHA256SUMS` before executing the fixed Linux asset and checking its reported version.

The checksum is obtained from the same GitHub release over fixed HTTPS paths. It is not embedded into the Action source used to build that binary, which avoids a circular source/checksum dependency. Existing source/tests do not provide a caller version or download-URL override.

PR/main CI builds a candidate once and verifies it through real process acceptance plus the exact-source composite with bounded download/API fixtures. This works before release assets exist. [Testing](testing.md#ci-validation) owns the detailed candidate cases and distinguishes local rendering, hosted integration, and published-download evidence.

The candidate's embedded version alone does not establish published feature availability: current-source bytes can include new fields while the committed version still names a release without them. Prepare an owner-selected release version through the normal process, verify the resulting published package, and update consumer pins separately; do not replace the existing version's immutable assets with the new source build.

## Retry and recovery

The draft's machine-readable manifest records version, pinned source commit, both binary SHA-256 values, and the checksum file SHA-256. The [publisher](../../scripts/publish-release.sh) finds a matching draft through the authenticated releases list and reads it by release ID; GitHub's [tag lookup](https://docs.github.com/en/rest/releases/releases#get-a-release-by-tag-name) returns published releases.

On retry, the publisher checks that manifest, resolves an existing lightweight or annotated tag, downloads and compares existing asset bytes, and uploads only missing assets. It fetches and checks the tag again immediately before publication. Matching source/version alone is insufficient: the tested bytes must match too.

| Existing state | Recovery behavior |
| --- | --- |
| Matching incomplete draft and tag | Verify present assets and upload only missing ones; publish after completeness and identity checks |
| Matching complete published release | Verify it and finish as a no-op |
| Conflicting tag/manifest, altered or unexpected assets, duplicate asset names, or missing tag for an existing release | Stop without moving/recreating the tag or replacing assets |
| Post-publication smoke failed | Diagnose the same source/release identity and rerun appropriate verification; preserve published tags/assets |

A new workflow run resolves current `main` again. If an earlier run already created the tag, a changed source SHA conflicts with that immutable tag. If it failed before creating the tag, a new run can build the same version from a newer `main` commit; review that source before dispatching again.

The original frozen SHA remains in its run logs, and build artifacts are retained for one day. Keep incomplete drafts for inspection instead of automatic deletion or destructive cleanup. Never retag, clobber assets, or replace released bytes to make verification pass.

## Historical release limitations

Published packages retain their original behavior even when current source changes:

- Action v0.6.0 installs CLI v0.5.0, and v0.5.0 has no root Action.
- CLI v0.4.0 and earlier exported to stdout; current source saves the checkout-root file.
- Releases before v0.2.0 used flat YAML; current source uses the versioned/grouped format.

These are historical compatibility facts, not changes to apply retroactively. [ADR 0023](../adr/0023-public-composite-action.md) records the Action release-alignment correction. Use the [consumer pin reference](../github-actions.md#published-package-and-consumer-pins) for the currently adopted package.

## Future automatic versioning

Version selection remains manual. [ADR 0016](../adr/0016-conventional-pr-titles.md) records the enforced Conventional Commits PR-title and body validation. [ADR 0017](../adr/0017-pr-based-semver.md) describes future automatic version calculation from all PRs merged since the previous release.

The agreed future precedence is: any breaking change implies major; otherwise a new feature implies minor; otherwise patch. No special pre-1.0 exception has been decided. [ADR 0024](../adr/0024-code-review-policy.md) resolves the notation question: review requires both `!` in the title and a `BREAKING CHANGE:` footer with impact and migration for every incompatible change. These decisions do not implement automatic version calculation or dispatch.

## Release decisions

| Concern | Decision |
| --- | --- |
| Explicit version and fixed source | [ADR 0010](../adr/0010-explicit-release-version.md), [ADR 0011](../adr/0011-pin-release-source.md) |
| Exact tested binaries | [ADR 0012](../adr/0012-publish-tested-binaries.md) |
| Immutable tags and complete drafts | [ADR 0013](../adr/0013-immutable-release-identity.md), [ADR 0014](../adr/0014-draft-release-publication.md) |
| Raw executable distribution | [ADR 0015](../adr/0015-raw-extension-binaries.md) |
| Enforced PR message validation | [ADR 0016](../adr/0016-conventional-pr-titles.md) |
| Future automatic version calculation | [ADR 0017](../adr/0017-pr-based-semver.md) |
| Immutable published assets | [ADR 0018](../adr/0018-preserve-published-assets.md) |

The [release feature specification](../features/release.feature.md) maps the behavior to tests. [`tests/release.sh`](../../tests/release.sh) uses a local `gh` stand-in for first publication, completed retries, incomplete drafts, conflicts, altered bytes, and tag rechecks. Those tests create no real tags/releases; an authorized workflow run establishes operational publication evidence.
