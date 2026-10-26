# Maintainer releases

The [manual release workflow](../../.github/workflows/release.yml) builds and validates its own platform binaries. Merging code does not dispatch it, and ordinary main CI artifacts are not substituted for release artifacts. Installation belongs in the [README](../../README.md); current published consumer pins belong in the [Actions guide](../github-actions.md#published-package-and-consumer-pins).

Source changes, version preparation, publication, and consumer adoption are separate operations. The public Action installs the release selected by its committed [`action/cli-version`](../../action/cli-version), not an arbitrary current-source build. Only an explicit release request authorizes publication.

## Version preparation

The [release-metadata helper](../../scripts/release-metadata/main.go) calculates the next version from the complete Git interval between the most recently published stable release and one fixed source SHA. Committed squash messages are the authoritative version signal; branch names and mutable PR labels are not. Reviewers keep PR titles, compatibility footers, and labels consistent before merging.

| Strongest change in the interval | Increment |
| --- | --- |
| A Conventional header with `!`, or a `BREAKING CHANGE:` / historical `BREAKING-CHANGE:` footer | MINOR during `0.x`; MAJOR from `1.0.0` onward |
| `feat`, `perf`, or a `DEPRECATED:` footer | MINOR |
| Fixes, reverts, maintenance, or legacy messages without a stronger signal | PATCH |

This implements [ADR 0017](../adr/0017-pr-based-semver.md). A pre-1.0 breaking signal never graduates the project automatically to `1.0.0`; that milestone remains an explicit project decision. A revert does not subtract the classification of another included commit. Fenced examples are not compatibility footers. Either breaking marker suffices under [Conventional Commits](../../.agents/skills/code-review/SKILL.md#breaking-changes-follow-conventional-commits).

A nonbreaking `build(release)` or `chore(release)` commit changing only `action/cli-version` is preparation, not another substantive change. A similarly named commit that changes any other path remains substantive and permits a PATCH. An interval containing nothing else is rejected. A repository with no published stable baseline requires a separately chosen initial release policy; this tool does not invent one.

On a clean topic branch based on current main, with complete history and tags:

```sh
go run ./scripts/release-metadata/main.go prepare daiksud/gh-settings
```

The command reads GitHub metadata through the authenticated `gh` CLI and updates only `action/cli-version`. It does not create a branch, PR, commit, tag, draft, or release. Review that one-file preparation in a normal PR, using `build(release): prepare the next version`, and verify main CI after merging it. The operator never needs to calculate or enter a version manually.

Preparation preserves existing tracked changes by refusing a dirty tracked worktree. Missing/malformed release metadata, failed reads, incomplete history, invalid version components, and unresolvable or unrelated predecessor tags stop calculation. If new changes or a new published baseline alter the computed version before dispatch, prepare and review the corrected file rather than overriding validation.

## Start a manual release

After the preparation PR is merged and the owner explicitly requests publication, dispatch from `main`:

```sh
gh workflow run release.yml --repo daiksud/gh-settings --ref main
```

Optionally add `-f source_sha=FULL_40_CHARACTER_SHA` to require an already reviewed exact main commit. Resolve reads current main once and fails if it differs from that pin. There is no version input and no push/tag publication trigger.

The read-only resolver checks out complete history at the frozen SHA, computes the version, and requires the committed Action version to match before product builds begin. It freezes the previous release's tag and resolved SHA as well. Only numeric `vMAJOR.MINOR.PATCH` versions without leading zeros or prerelease/build suffixes are supported.

Issue and pull-request comments do not authorize or dispatch a release. `release.yml`'s direct `workflow_dispatch` event is the only release entrypoint. After dispatch, follow the release workflow itself through native acceptance, publication, and the published Action smoke test. The resolver rechecks current main and the committed version, and the serialized immutable retry contract still applies to repeated direct dispatches.

## Release assets and validation

| Platform | Go target | Raw executable |
| --- | --- | --- |
| macOS Apple Silicon | `darwin/arm64` | `gh-settings-darwin-arm64` |
| Linux x86_64 | `linux/amd64` | `gh-settings-linux-amd64` |

Each release contains those two raw executables, `SHA256SUMS`, and notes with source identity, changes, installation, and checksum guidance. Asset names follow the [GitHub CLI extension convention](https://docs.github.com/en/github-cli/github-cli/creating-github-cli-extensions). Additional platforms, archives, and GoReleaser are outside this contract.

| Stage | Required behavior |
| --- | --- |
| Resolve | Require the dispatched main commit to remain current, check an optional reviewed source pin, derive the version or recognize an exact retry, verify the committed version, freeze the predecessor, and emit allowlisted build/acceptance matrices from that trusted source |
| Commit stage | Run source checks in the Linux lane, build Linux and macOS once on their native runners, and upload each executable with a source/version/asset/SHA-256 manifest |
| Acceptance stage | Download only this run's manifested products, verify exact inventory and bytes, run both native harnesses, and exercise the candidate root Action against a separate consumer |
| Release | Redownload and reverify both accepted manifests, generate new notes, create/validate the tag and draft, upload only missing matching assets, recheck identity, and publish |
| Verify release | Check out the published tag separately, compare actual public downloads byte-for-byte with accepted manifests, and run the published root Action with real downloads for a separate consumer |

Builds inject `main.version` with Go's [`-X` option](https://pkg.go.dev/cmd/link#hdr-Command_Line). The release workflow delegates product-specific work to five local Actions while retaining one byte-identical orchestration contract shared with gh-qw: metadata, build, acceptance, publish, and verify. Those wrappers own the pinned artifact transfer and manifest checks; no matrix job output supplies a product hash to publication. The shared [acceptance-stage](../../.github/actions/acceptance-stage/action.yml) compiles its harness separately, without rebuilding the accepted product. All manual versions share one non-cancelling concurrency group; this does not promise FIFO queue delivery.

Both native acceptance lanes and the `candidate-action` lane must succeed before publication. Failure, skipping, or cancellation cannot be overridden. The candidate's bounded download fixture establishes pre-publication integration, not published-download identity. Post-publication verification independently establishes the tag and public-download identity. [Release gate tests](../../tests/github-actions/release_test.go) and [local Action tests](../../tests/github-actions/release_actions_test.go) protect orchestration bytes, source/version/manifests, dependency bindings, and tamper rejection. Fixture and PR validation do not execute a hosted release.

## GitHub-generated release notes

New notes use GitHub's [generate-notes API](https://docs.github.com/en/rest/releases/releases#generate-release-notes-content-for-a-release), configured by [`.github/release.yml`](../../.github/release.yml). The old categorized Git-log renderer has been removed. The request supplies the proposed tag, frozen target SHA, explicit previous published tag, and configuration path. The Contents-write publication token performs generation only after validation; the read-only resolver does not generate notes or create state.

| Reviewed PR labels | Native heading |
| --- | --- |
| `BREAKING CHANGE` | `💥 BREAKING CHANGE` |
| `DEPRECATED`, without breaking | `⚠️ DEPRECATED` |
| `feat` or `perf`, without breaking or deprecation | `✨ New Feature` |
| `fix`, without a stronger category | `🐛 Bug Fix` |
| Everything not already categorized, including `revert` and unlabeled PRs | `🛠️ Maintenance` |

The catch-all retains unlabeled PRs. Before the first release using this configuration, review labels on already-merged, unreleased PRs; do not infer their meaning from branch names. Later label edits affect a newly generated presentation, not the durable version calculation. Reconcile disagreements before publication rather than maintaining two version engines.

GitHub lists merged PRs, contributors, and a changelog link; it does not promise to copy PR bodies or compatibility footers. A small supplement preserves breaking-change impact/migration footers from the same frozen Git interval, escaping them as text. A breaking header without a footer produces an explicit instruction to inspect that change. This is migration guidance, not another categorized change list. A fixed-SHA comparison also covers direct commits that are not separate native PR entries.

The predecessor is the stable numeric-tag release with the latest publication timestamp, with release ID breaking ties. Drafts, prereleases, and unpublished tags do not establish that boundary. Full local history, predecessor ancestry, and the resolver's previous tag/SHA must agree; a changed publication boundary stops generation before remote mutations. GitHub receives the frozen target, and a conflicting existing target tag is rejected. New main commits cannot enter the frozen interval.

Generation must return a nonempty body. API or decoding failures stop before tag/draft creation. The publisher appends installation/upgrade/checksum guidance and its existing hidden version/source/binary identity manifest. A matching draft or completed release retains its original body on retry, even if labels or later publications change.

## Publication and recovery

The [Go publisher](../../scripts/release-publisher/main.go) verifies the committed/requested version, source, expected hashes, repository, and both binary files before any GitHub lookup. It prepares `SHA256SUMS` and the identity marker, checks existing release metadata and lightweight/annotated tags through `gh`, and owns publication, draft recovery, and verified no-ops. The [shell entry point](../../scripts/publish-release.sh) builds only this internal tool; accepted product binaries are never rebuilt.

For a new release, the complete notes are prepared before creating any missing tag or draft. The publisher checks asset names and downloaded bytes, uploads only missing assets, then rechecks assets, draft state, marker, and tag before publishing. A fresh release read must confirm `draft=false`. Final API operations are not transactional: failure can leave an inspectable tag or incomplete draft.

| Existing state | Recovery behavior |
| --- | --- |
| Matching incomplete draft and tag | Reuse its version/source, verify present bytes, upload only missing assets, and retain stored notes |
| Matching complete published release | Verify all identities and bytes, then finish as a no-op |
| Conflicting tag/manifest, altered or unexpected assets, duplicate names, or missing tag for an existing release | Stop without retagging, replacing assets, or recreating the missing tag |
| Post-publication smoke failed | Diagnose the same immutable release and rerun verification; do not replace published bytes |

The resolver recognizes a retry only when the committed version, release identity marker, and tag identify the exact frozen source; the publisher still verifies every checksum and asset. It does not recalculate a new version or regenerate stored notes for that release. An unrelated release with the same version is not a retry. Publication consumes only products downloaded from this run's build artifacts and verified against their per-target manifests; it never rebuilds an accepted executable.

A new workflow run resolves current main again. Existing tags remain immutable, so changed source cannot reuse a conflicting tag. Before tag creation, a fresh run may validate a newly reviewed source. The original SHA remains in run logs; artifacts are retained for one day. Keep incomplete drafts for inspection instead of automatically deleting them.

## Action version and availability

A versioned Action installs the CLI selected by its committed file. Strict version-tag invocation must match that file; SHA pins select their committed version without a mutable latest lookup. The installer verifies the fixed version's `SHA256SUMS` over HTTPS and checks the installed version. There is no caller version/download-URL override.

PR/main CI builds one candidate and exercises it through process acceptance and an exact-source composite with bounded download/API fixtures. This works before publication; [testing](testing.md#ci-validation) distinguishes local/hosted integration from released-download evidence. Source features are not automatically available through an older published Action pin. Publish an explicitly authorized matching package before separately reviewing consumer adoption and permission changes.

### Historical release limitations

Historical packages retain their original behavior: Action v0.6.0 installs CLI v0.5.0; v0.5.0 has no root Action; CLI v0.4.0 and earlier export to stdout; releases before v0.2.0 use flat YAML. Do not repair those immutable versions by changing their tags or assets. [ADR 0023](../adr/0023-public-composite-action.md) records the Action/CLI alignment correction.

## Release decisions

| Concern | Decision |
| --- | --- |
| Automatic selection, superseding the original explicit input | [ADR 0017](../adr/0017-pr-based-semver.md), historical [ADR 0010](../adr/0010-explicit-release-version.md) |
| Frozen source and exact tested binaries | [ADR 0011](../adr/0011-pin-release-source.md), [ADR 0012](../adr/0012-publish-tested-binaries.md) |
| Immutable identity and complete drafts | [ADR 0013](../adr/0013-immutable-release-identity.md), [ADR 0014](../adr/0014-draft-release-publication.md), [ADR 0018](../adr/0018-preserve-published-assets.md) |
| Raw executables and reviewed messages | [ADR 0015](../adr/0015-raw-extension-binaries.md), [ADR 0016](../adr/0016-conventional-pr-titles.md), [ADR 0024](../adr/0024-code-review-policy.md) |

The [release specification](../features/release.feature.md) maps behavior to tests. [Metadata tests](../../scripts/release-metadata/main_test.go) use temporary Git repositories and an injected GitHub boundary for version precedence, exact ranges, preparation, retries, generation requests, migration details, and failures. [Release fixtures](../../tests/release.sh) exercise new publication and recovery with a local `gh` stand-in, retaining notes and validating artifact identity. These tests neither publish a release nor prove GitHub's live rendering; only an explicitly authorized release run establishes that operational evidence.
