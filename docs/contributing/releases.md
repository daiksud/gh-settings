# Maintainer releases

The [manual release workflow](../../.github/workflows/release.yml) first checks the latest matching main `ci-push` run. It reuses that successful run's verified Linux and macOS products, or rebuilds and reinspects both products if the artifacts are missing, expired or fail identity checks. A failed, cancelled or unfinished `ci-push` blocks release rather than triggering a rebuild. Merging code neither calculates a persistent version nor authorizes publication. The [README](../../README.md) owns installation; the [Actions guide](../github-actions.md#published-package-and-consumer-pins) owns consumer pins.

## Automatic version calculation

The same standard-library Python [release-version.py](../../.github/scripts/release-version.py) runs in gh-settings and gh-qw. It calculates the next version directly during manually dispatched release resolution, from the last published stable release through the fixed main SHA. There is **no preparation PR, committed release-version update, version input, or automatic publication on merge**. Branch names do not select the version; reviewers must preserve accurate Conventional Commit squash titles and breaking-change footers.

| Committed change | Selected increment |
| --- | --- |
| Header `!` or compatibility footer `BREAKING CHANGE:` / `BREAKING-CHANGE:` | MINOR while v0.x, MAJOR from v1.0.0 |
| `feat`, `perf`, or a `DEPRECATED:` footer | MINOR |
| Any other substantive commit, including maintenance and reverts | PATCH |

The strongest signal wins once; a breaking change never auto-promotes v0.x to v1.0.0. Empty commits are ignored; every tree change counts, including a release-preparation-named commit. If no substantive change remains, calculation fails. The GitHub release list, stable baseline, complete history, and ancestor relationship must validate; drafts, prereleases, unpublished tags, and code-fenced footer examples do not change the result.

The resolver is read-only. It also recognizes an exact immutable gh-settings release retry from its signed-off source/tag/embedded identity marker; the publisher independently checks all binary bytes and retains the original notes. A conflicting tag, missing marker, or new source is not an acceptable retry.

## Start a manual release

After the relevant change PRs are merged and main CI has passed, **only an explicit user release request** authorizes dispatch from main:

```sh
gh workflow run release.yml --repo daiksud/gh-settings --ref main
```

Optionally add `-f source_sha=FULL_40_CHARACTER_SHA` to pin the reviewed main commit. Resolution stops if main moved before it starts, if an explicit pin differs, or if the metadata source differs from the checkout. No workflow dispatch from a non-main ref, issue comment, or PR merge publishes a release.

The workflow requires successful main CI when a matching run exists, calculates the version, and freezes the source SHA and published baseline. It verifies the matching run's artifacts or performs all native builds and acceptance checks again. Before publication it rechecks main and the selected CI result; any failed or incomplete CI blocks release. Source changes after resolution do not enter the release.

## Release assets and validation

| Platform | Go target | Raw executable |
| --- | --- | --- |
| macOS Apple Silicon | `darwin/arm64` | `gh-settings-darwin-arm64` |
| Linux x86_64 | `linux/amd64` | `gh-settings-linux-amd64` |

Each release contains those two raw executables, `SHA256SUMS`, and notes with source identity, changes, installation, and checksum guidance. Asset names follow the [GitHub CLI extension convention](https://docs.github.com/en/github-cli/github-cli/creating-github-cli-extensions). Additional platforms, archives, and GoReleaser are outside this contract.

| Stage | Required behavior |
| --- | --- |
| Resolve | Require the dispatched main commit to remain current, check an optional reviewed source pin, derive the version or recognize an exact retry, freeze the predecessor, and emit allowlisted build/acceptance matrices from that trusted source |
| Reuse | Look up the exact-source main `ci-push` result; require success, verify both artifact archives and contained source/version/target/SHA-256 manifests, then upload verified bytes to the current release run. Missing/expired products or digest/version mismatches warn and select full fallback; API and authentication failures abort | 
| Commit stage (fallback only) | Run source checks in the Linux lane, build Linux and macOS on native runners, and upload source/version/asset/SHA-256 manifests |
| Acceptance stage (fallback only) | Download only this run's manifested products, verify exact inventory and bytes, run both native harnesses, and exercise the candidate root Action against a separate consumer |
| Delivery gate | Require either fully verified reused artifacts with build and Inspection skipped, or complete successful fallback Screening and Inspection; no failed/skipped work may be represented as acceptance |
| Release | Recheck exact main and `ci-push` status; redownload and reverify both release-run manifests, generate notes, create/validate the tag and draft, upload only missing matching assets, recheck identity, and publish |
| Verify release | Check out the published tag separately, compare actual public downloads byte-for-byte with accepted manifests, and run the published root Action with real downloads for a separate consumer |

Builds inject `main.version` with Go's [`-X` option](https://pkg.go.dev/cmd/link#hdr-Command_Line). The release workflow delegates metadata, artifact reuse, build, acceptance, publish and verification to focused local Actions. Only the version-resolution logic remains shared with gh-qw; gh-settings uses its own release gates and manifests. Those wrappers own the pinned artifact transfer and manifest checks; no matrix job output supplies a product hash to publication. The shared [acceptance-stage](../../.github/actions/acceptance-stage/action.yml) compiles its harness separately, without rebuilding the accepted product. All manual versions share one non-cancelling concurrency group; this does not promise FIFO queue delivery.

All three main-CI inspection lanes must succeed before their products can be reused. Otherwise all fallback native and candidate-Action acceptance lanes must succeed before publication. Failure, unexpected skipping or cancellation cannot be overridden. The candidate's bounded download fixture establishes pre-publication integration, not published-download identity. Post-publication verification independently establishes the tag and public-download identity. [Release gate tests](../../tests/github-actions/release_test.go) and [local Action tests](../../tests/github-actions/release_actions_test.go) protect orchestration bytes, source/version/manifests, dependency bindings, and tamper rejection. Fixture and PR validation do not execute a hosted release.

## Release notes from committed messages

The [notes generator](../../scripts/release-metadata/main.go) reads the actual committed messages in the fixed `PREVIOUS_SHA..SOURCE_SHA` interval. Normal squash merging records each PR's reviewed title and body in one main commit. It follows main's first-parent history and skips commits with no tree change, keeping substantive main squash commits as the release entries. Notes use those messages directly, without querying current PR titles or bodies. Their generation does not call GitHub's generate-notes API or use a release-category configuration file.

Each included commit appears once under the first matching category (explicit withdrawals take precedence):

| Committed message signal, in priority order | Heading |
| --- | --- |
| A later formal `Release-withdraws:` footer naming this commit | `↩️ Withdrawn before release` |
| Header `!` or formal `BREAKING CHANGE:` / `BREAKING-CHANGE:` footer | `💥 BREAKING CHANGE` |
| Formal `DEPRECATED:` footer | `⚠️ DEPRECATED` |
| `feat` | `✨ New Feature` |
| `fix` | `🐛 Bug Fix` |
| `perf` | `⚡ Performance` |
| Any remaining message, including internal maintenance, reverts, and legacy headers | `🛠️ Maintenance` |

Compatibility markers must be Conventional Commits headers or footers; mentions in prose, Markdown headings, or quoted/fenced examples do not classify a change. New Feature and Bug Fix describe observable CLI or public Action behavior. Internal development tools and CI/CD changes use their appropriate maintenance types; user-experienced runtime improvements use `perf`. The [code-review policy](../../.agents/skills/code-review/SKILL.md#title-and-type) owns this semantic judgment. PRLint remains commitlint-only.

Entries link to their exact commits and render message text safely. Migration guidance preserves breaking-change footer details from the same interval as escaped text. A breaking header without a footer produces an instruction to inspect that change. A fixed-SHA comparison links the complete interval, including any direct commits. Later PR edits cannot change the committed classification or text.

### Explicit withdrawals and release clarifications

When a later change fully withdraws an earlier entry before publication, record
that fact in the reviewed squash body with a formal footer:

```text
Release-withdraws: FULL_40_CHARACTER_COMMIT_SHA Reason the whole entry is unavailable.
Release-note: Explain the final supported behavior and any actual upgrade impact.
Continuation lines preserve additional detail.
```

Use one `Release-withdraws:` footer per fully withdrawn entry. The target must be
an earlier substantive first-parent commit in the same frozen release interval.
Its entry moves out of feature/fix/compatibility categories into Withdrawn before
release, retaining its original commit link, the withdrawal reason, and a link to
the annotating commit. Its obsolete migration and clarification footers are omitted. This does not
subtract its version signal: the Python resolver still considers every committed
change. Partial removals must use a clarification explaining what remains rather
than claiming the entire entry was withdrawn. The generator does not infer
withdrawals from matching subjects, Git diffs, PR labels, or `revert` headers.

`Release-note:` footers appear prominently before the categorized history, with
the annotating commit link and escaped multiline text. They supplement existing
breaking-change guidance; they do not change compatibility classification or
replace a breaking marker. Use them to distinguish features removed during the
unreleased interval from capabilities already available in the previous release
that users must stop declaring. Later PR edits cannot amend old committed bodies;
a new substantive commit can record the correction. Empty commits remain excluded.

These footers use the same formal footer boundaries and fence exclusion as
compatibility footers. Multiple notes are retained in history order. Empty notes,
malformed withdrawal targets or missing reasons, targets outside the interval,
self/forward references, duplicate withdrawals, and withdrawals of a withdrawing
entry fail generation before any tag or draft mutation. New annotations do not
rewrite notes retained by an existing draft or published-release retry.

The predecessor is the stable numeric-tag release with the latest publication timestamp, with release ID breaking ties. Drafts, prereleases, and unpublished tags do not establish that boundary. Full local history, predecessor ancestry, and the resolver's previous tag/SHA must agree; a changed publication boundary stops generation before remote mutations. A conflicting existing target tag is rejected. New main commits cannot enter the frozen interval.

Catalog, history, or generation failures stop before tag/draft creation. The publisher appends installation/upgrade/checksum guidance and its existing hidden version/source/binary identity manifest. A matching draft or completed release retains its original body on retry, even after later publications.

## Publication and recovery

The [Go publisher](../../scripts/release-publisher/main.go) verifies the resolved release version, source, expected hashes, repository, and both binary files before any GitHub lookup. It prepares `SHA256SUMS` and the identity marker, checks existing release metadata and lightweight/annotated tags through `gh`, and owns publication, draft recovery, and verified no-ops. The [shell entry point](../../scripts/publish-release.sh) builds only this internal tool; accepted product binaries are never rebuilt.

For a new release, the complete notes are prepared before creating any missing tag or draft. The publisher checks asset names and downloaded bytes, uploads only missing assets, then rechecks assets, draft state, marker, and tag before publishing. A fresh release read must confirm `draft=false`. Final API operations are not transactional: failure can leave an inspectable tag or incomplete draft.

| Existing state | Recovery behavior |
| --- | --- |
| Matching incomplete draft and tag | Reuse its version/source, verify present bytes, upload only missing assets, and retain stored notes |
| Matching complete published release | Verify all identities and bytes, then finish as a no-op |
| Conflicting tag/manifest, altered or unexpected assets, duplicate names, or missing tag for an existing release | Stop without retagging, replacing assets, or recreating the missing tag |
| Post-publication smoke failed | Diagnose the same immutable release and rerun verification; do not replace published bytes |

The resolver recognizes a retry only when the calculated version, release identity marker, and tag identify the exact frozen source; the publisher still verifies every checksum and asset. It does not recalculate a new version or regenerate stored notes for that release. An unrelated release with the same version is not a retry. Publication consumes only products downloaded from this run's build artifacts and verified against their per-target manifests; it never rebuilds an accepted executable.

A new workflow run resolves current main again. Existing tags remain immutable, so changed source cannot reuse a conflicting tag. Before tag creation, a fresh run may validate a newly reviewed source. The original SHA remains in run logs; artifacts are retained for one day. Keep incomplete drafts for inspection instead of automatically deleting them.

## Action version and availability

A published Action resolves its CLI from its own version tag. Full-SHA pins resolve an exact matching stable tag using public Git refs; untagged commits fail rather than falling back to latest. The installer verifies the release's `SHA256SUMS` and the executable's reported version. The committed `action/cli-version` file is only the local development candidate default; it does not control a published release. Local candidate/verification Actions use the exact accepted binary version. There is no public version or download-URL input.

PR/main CI builds one candidate and exercises it through process acceptance and an exact-source composite with bounded download/API fixtures. This works before publication; [testing](testing.md#ci-validation) distinguishes local/hosted integration from released-download evidence. Source features are not automatically available through an older published Action pin. Publish an explicitly authorized matching package before separately reviewing consumer adoption and permission changes.

## Release decisions

| Concern | Decision |
| --- | --- |
| Automatic selection, superseding the original explicit input | [ADR 0017](../adr/0017-pr-based-semver.md), [ADR 0010](../adr/0010-explicit-release-version.md) |
| Frozen source and exact tested binaries | [ADR 0011](../adr/0011-pin-release-source.md), [ADR 0012](../adr/0012-publish-tested-binaries.md) |
| Immutable identity and complete drafts | [ADR 0013](../adr/0013-immutable-release-identity.md), [ADR 0014](../adr/0014-draft-release-publication.md), [ADR 0018](../adr/0018-preserve-published-assets.md) |
| Raw executables and reviewed messages | [ADR 0015](../adr/0015-raw-extension-binaries.md), [ADR 0016](../adr/0016-conventional-pr-titles.md), [ADR 0024](../adr/0024-code-review-policy.md) |

The [release specification](../features/release.feature.md) maps behavior to tests. Identical [Python version tests](../../.github/scripts/test_release_version.py) use temporary Git repositories for version precedence, exact ranges, metadata failures and retry identity. [Go notes tests](../../scripts/release-metadata/main_test.go) cover committed category precedence, formal footers, escaping, fixed ranges, and failures. [Release fixtures](../../tests/release.sh) exercise the generated body through new publication and recovery with a local `gh` stand-in, retaining notes and validating artifact identity. These tests neither publish a release nor prove GitHub's live rendering; only an explicitly authorized release run establishes that operational evidence.

### Cross-run artifact reuse

The release resolver queries the source commit's latest `ci-push.yml` push run on `main`. A failed, cancelled, timed-out or pending result stops release immediately; older successful attempts are not substitutes. If no run exists, fresh Screening and Inspection are required. GitHub API/authentication or provenance errors also stop release.

For successful CI, the reuse step requires unique, unexpired Linux and macOS artifacts. Each archive's SHA-256 is compared with the recorded GitHub artifact digest, then the exact two-file inventory and source/version/target/binary SHA-256 manifest are checked. Missing, expired, mismatched-version or corrupted artifacts are rejected with an Actions warning and Job Summary entry; they trigger full validation, not publication. Both accepted products are transferred into the release run using the original release artifact names. Publication and post-publication verification continue to consume these same-run artifacts, whether they came from CI reuse or a fresh build. The release job rechecks main and CI success immediately before publication.
