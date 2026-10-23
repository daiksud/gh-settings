# Maintainer releases

The [manual release workflow](../../.github/workflows/release.yml) builds and validates its own platform binaries. Merging code does not dispatch it, and ordinary main CI artifacts are not substituted for release artifacts. Installation belongs in the [README](../../README.md); current published consumer pins belong in the [Actions guide](../github-actions.md#published-package-and-consumer-pins).

Source changes, version preparation, publication, and consumer adoption are separate operations. The public Action installs the release selected by its committed [`action/cli-version`](../../action/cli-version), not an arbitrary current-source build. Only an explicit release request authorizes publication.

## Version preparation

The [release-metadata helper](../../scripts/release-metadata/main.go) calculates the next version from the complete Git interval between the most recently published stable release and one fixed source SHA. Committed squash messages are the authoritative version signal; branch names and mutable PR labels are not. Reviewers keep PR titles, compatibility footers, and labels consistent before merging.

| Strongest change in the interval | Increment |
| --- | --- |
| A Conventional header with `!`, or a `BREAKING CHANGE:` / historical `BREAKING-CHANGE:` footer | MAJOR, including during `0.x` |
| `feat`, `perf`, or a `DEPRECATED:` footer | MINOR |
| Fixes, reverts, maintenance, or legacy messages without a stronger signal | PATCH |

This implements [ADR 0017](../adr/0017-pr-based-semver.md), without importing gh-qw's pre-1.0 exception. A revert does not subtract the classification of another included commit. Fenced examples are not compatibility footers. New incompatible changes still require both signals under the [review policy](../../.agents/skills/code-review/SKILL.md#breaking-changes-require-both-signals).

A nonbreaking `build(release)` or `chore(release)` commit changing only `action/cli-version` is preparation, not another product change. An interval containing nothing else is rejected. A repository with no published stable baseline requires a separately chosen initial release policy; this tool does not invent one.

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

### Request publication from the connected GitHub app

The connected GitHub tool can post an Issue comment but cannot invoke `workflow_dispatch` directly. Use the dedicated [release-command Issue #410](https://github.com/daiksud/gh-settings/issues/410) as an alternate **manual** request surface. After reviewing the intended source and latest main CI, post a **new** owner-authored top-level comment consisting of exactly:

```text
/release FULL_40_CHARACTER_MAIN_SHA
```

Use the **current main SHA** obtained after every preparatory or dispatcher PR is merged, not an earlier source pin. A comment on another Issue, a pull request, an edited comment, or one by anyone other than the repository owner is not authorized. Merely creating the control Issue, merging code, or writing the command in an Issue body does not publish a release.

The [release-request dispatcher](../../.github/workflows/release-request.yml) runs from default-branch trusted code. Its unprivileged authorization job checks the exact Issue, owner identity, event type and command shape. Only on success does a separate job with `actions: write` read current main, verify the requested full SHA, and invoke `workflow_dispatch` for the existing `release.yml` on `main` with that same `source_sha`. It checks out no candidate code, takes no caller-selected workflow name or ref, and uses no App secrets. GitHub [documents both the issue-comment event](https://docs.github.com/en/actions/reference/workflows-and-actions/events-that-trigger-workflows#issue_comment) and the [special `GITHUB_TOKEN` workflow-dispatch exception](https://docs.github.com/en/actions/how-tos/write-workflows/choose-when-workflows-run/trigger-a-workflow#triggering-a-workflow-from-a-workflow).

**Dispatch accepted is not release published.** Follow both the dispatcher and the separate release workflow runs. The release resolver rechecks current main and the committed version, so a movement after the request fails closed; all normal build, native acceptance, candidate Action, publisher identity and post-publication smoke gates still apply. A repeated owner command is a new explicit request and must rely on the existing serialized, immutable retry contract rather than silently overwriting tags or assets. The CLI/API `workflow_dispatch` method above remains available and unchanged.

## Release assets and validation

| Platform | Go target | Raw executable |
| --- | --- | --- |
| macOS Apple Silicon | `darwin/arm64` | `gh-settings-darwin-arm64` |
| Linux x86_64 | `linux/amd64` | `gh-settings-linux-amd64` |

Each release contains those two raw executables, `SHA256SUMS`, and notes with source identity, changes, installation, and checksum guidance. Asset names follow the [GitHub CLI extension convention](https://docs.github.com/en/github-cli/github-cli/creating-github-cli-extensions). Additional platforms, archives, and GoReleaser are outside this contract.

| Stage | Required behavior |
| --- | --- |
| Resolve | Freeze main, derive the version or recognize an exact retry, verify committed version, and freeze the predecessor for a new release |
| Commit/build | Check frozen source/version equality, build Linux once, and then build macOS once on its native runner |
| Native acceptance | Download each run's binary, verify SHA-256 and embedded version, and execute that exact binary against the local API fixture |
| Candidate Action smoke | Use frozen Action source and the already-built Linux artifact; actual credential-free validate reports all four expected identities for a separate consumer |
| Publish | Verify accepted hashes, generate new notes, create/validate the tag and draft, upload only missing matching assets, recheck identity, and publish |
| Published Action smoke | Verify the published tag/source/version, run its unchanged root Action with real release downloads, and compare outputs with the accepted Linux digest |

Builds inject `main.version` with Go's [`-X` option](https://pkg.go.dev/cmd/link#hdr-Command_Line). The shared [acceptance-stage](../../.github/actions/acceptance-stage/action.yml) compiles its harness separately, without rebuilding the accepted product. All manual versions share one non-cancelling concurrency group; this does not promise FIFO queue delivery.

Both native acceptance jobs and `candidate-action-smoke` must succeed before publication. Failure, skipping, or cancellation cannot be overridden. The candidate's bounded download fixture establishes pre-publication integration, not published-download identity. [Release gate tests](../../tests/github-actions/release_test.go) protect the source/version/artifact and dependency bindings.

## GitHub-generated release notes

New notes use GitHub's [generate-notes API](https://docs.github.com/en/rest/releases/releases#generate-release-notes-content-for-a-release), configured by [`.github/release.yml`](../../.github/release.yml). The old categorized Git-log renderer has been removed. The request supplies the proposed tag, frozen target SHA, explicit previous published tag, and configuration path. The Contents-write publication token performs generation only after validation; the read-only resolver does not generate notes or create state.

| Reviewed PR labels | Native heading |
| --- | --- |
| `BREAKING CHANGE` | `🚨 BREAKING CHANGE` |
| `feat`, `perf`, `DEPRECATED`, without breaking | `✨ New Feature` |
| `fix`, `revert`, without a stronger category | `🐛 Bug Fix` |
| Everything not already categorized | `🧹 Chore` |

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

The resolver recognizes a retry only when the committed version, release identity marker, and tag identify the exact frozen source; the publisher still verifies every checksum and asset. It does not recalculate a new version or regenerate stored notes for that release. An unrelated release with the same version is not a retry.

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
