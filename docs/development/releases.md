# Maintainer releases

The [manual release workflow](../../.github/workflows/release.yml) selects a version for a specific main commit and publishes the Linux and macOS binaries that passed inspection. It can use the [main deployment](../../.github/workflows/push.yml) or run the same builds and inspections itself. See the [README](../../README.md) for installation and the [Actions guide](../guides/github-actions.md#published-package-and-consumer-pins) for consumer pins.

## Automatic version calculation

Main CI and manual release call the same [version workflow](../../.github/workflows/version.yml).
It fixes the source SHA and the last published stable release, then reads the complete
commit messages in that interval. The CLI, the settings YAML it accepts and the
public root Action share one version. The release series started at `v0.1.0`
and publishes stable `vMAJOR.MINOR.PATCH` tags.

| Strongest signal in a substantive commit | Normal increment |
| --- | --- |
| Any type with a header `!` or a formal `BREAKING CHANGE:` or `BREAKING-CHANGE:` footer | MINOR during 0.x, MAJOR from v1 |
| `feat`, user-facing `perf`, or a formal `DEPRECATED:` footer for a public interface | MINOR |
| A change to either shipped CLI executable or the public Action package | PATCH |
| Identical products at the two endpoints | Keep the previous version |

The [classifier](../../.github/scripts/release-version.py) ignores empty commits and
fenced examples of compatibility footers. A `fix`, `refactor`, dependency update
or build change can produce a PATCH. Compare the frozen endpoint trees for the
root `action.yml` and the runtime files under `action/`. The directory guide
is documentation. A change to the installer alone qualifies for a PATCH.
Git-generated revert subjects and other legacy subjects use the same product
comparison. Their formal compatibility and deprecation footers still count.
An empty message on a commit that changes the tree stops resolution.

When a PATCH depends on the CLI, build the previous release and the target SHA
on the same native runner for each supported platform. Each comparison starts
at the frozen main SHA and verifies the previous tag still resolves to the
selected ancestor before running its build recipe. The two diagnostic builds
use runner-local Go caches. For these comparisons,
both builds use the previous release's version and disable Go VCS stamping.
Each checkout supplies its own Go version, dependencies and build recipe. Keep
release compiler and linker options in the [shared build Action](../../.github/actions/build/action.yml)
so comparisons exercise the recipe that builds the shipped product. Compare all
remaining bytes, including Go's build ID derived from its inputs. A difference on either platform selects PATCH. Both comparisons must succeed.
The candidate workflow then builds and inspects the distributable binaries
with the selected version and normal source identity.

Changes confined to documentation, tests or CI that leave both products identical
carry forward to the next release's notes. Main CI still inspects the binaries
for the new SHA with the previous version. A manual release skips publication.
Missing stable baseline, incomplete history, unrelated predecessor or a failed
comparison stops resolution. Drafts, prereleases and unpublished tags cannot
supply the stable baseline.

The **Promote to v1** checkbox defaults to off. Selecting it from a v0.x
baseline with substantive changes chooses `v1.0.1`, including a maintenance-only
interval. The deleted immutable `v1.0.0` release reserved that tag name.
A source already published as v0 needs a new substantive commit to graduate.
An exact immutable v1 retry keeps its recorded version. Once the first v1
release passes public verification, remove the checkbox and its one-time
calculation path in a follow-up PR. [Promotion decision](https://github.com/daiksud/gh-settings/pull/863)

The resolver is read-only. A compatible exact immutable gh-settings release retry retains its recorded source, tag and version using its embedded identity marker; the publisher independently checks all binary bytes and retains the original notes. A conflicting tag, missing marker, or new source is not an acceptable retry.

## Start a manual release

After the relevant change PRs are merged and main CI has passed, **only an explicit user release request** authorizes dispatch from main:

```sh
gh workflow run release.yml --repo daiksud/gh-settings --ref main
```

Optionally add `-f source_sha=FULL_40_CHARACTER_SHA` to pin the reviewed main commit. Resolution stops if main moved before it starts, if an explicit pin differs, or if the metadata source differs from the checkout. No workflow dispatch from a non-main ref, issue comment, or PR merge publishes a release.

Leave **Promote to v1** unchecked for normal calculation. To deliberately select
the initial v1.0.1 promotion, check it in the Actions dispatch form or add
`-f promote_to_v1=true` to the command above. After v1, that choice does not reset
the version or override ordinary SemVer. Dispatch still requires separate
publication authorization.

Release resolution freezes the source SHA and published baseline before selecting artifacts. The release catalog can change between runs even at the same SHA. For example, an opted-in v1.0.1 release must rebuild and inspect both products if main deployed a v0.x candidate. The publisher rechecks the selected main run and current main SHA immediately before publication.

## Release assets and validation

| Platform | Go target | Raw executable |
| --- | --- | --- |
| macOS Apple Silicon | `darwin/arm64` | `gh-settings-darwin-arm64` |
| Linux x86_64 | `linux/amd64` | `gh-settings-linux-amd64` |

Each release contains those two raw executables, `SHA256SUMS`, and notes with source identity, changes, installation, and checksum guidance. Asset names follow the [GitHub CLI extension convention](https://docs.github.com/en/github-cli/github-cli/creating-github-cli-extensions). Additional platforms, archives, and GoReleaser are outside this contract.

| Stage | Required behavior |
| --- | --- |
| Resolve | Require the dispatched main SHA, check the optional source pin and select the version. For a new version or immutable retry, look up the latest exact-source main deployment and validate the two artifacts for reuse. |
| Candidate | If fresh products are needed, invoke the [candidate workflow](../../.github/workflows/candidate.yml) to run Linux screening, build both native targets and inspect the two binaries and candidate root Action. |
| Select | Accept the producer run and two artifact IDs from verified reuse or a successful fresh candidate workflow. |
| Release | Recheck main and the selected CI run. Download both accepted artifacts by ID, verify manifests and bytes, generate notes, validate the tag and draft, upload missing assets and publish. |
| Verify release | Compare the public downloads with the same two accepted artifacts and exercise the published tag's root Action in a separate consumer. |

Builds inject `main.version` with Go's [`-X` option](https://pkg.go.dev/cmd/link#hdr-Command_Line). The [native acceptance harness](../../.github/actions/integration-test/action.yml) compiles separately and runs the supplied product. Manual release runs share a non-cancelling concurrency group.

### Cross-run artifact reuse

The resolver selects the latest `push.yml` run and attempt for this repository, main branch and source SHA. A failed, cancelled or unfinished run, unverifiable provenance or an API error stops release. When no matching run exists, the release runs the complete candidate workflow.

After successful main inspection, `deployment` records the producer run and the IDs and digests of the inspected Linux and macOS artifacts. Candidate artifact names include the producing build attempt, so a retry can use products retained from successful jobs. Release verifies the deployment record for the selected attempt, the archive digest of each product, its two-file inventory and the manifest's source, version, target and binary SHA-256. Missing, expired, corrupt or incompatible artifacts trigger fresh screening and inspection. Publication and verification download from the selected producer run by artifact ID.

The [release gate tests](../../tests/github-actions/release_test.go) and [local Action tests](../../tests/github-actions/release_actions_test.go) cover selection, cancellation, product identity and permission boundaries. Actual public downloads are checked after an authorized release.

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

Compatibility markers must be Conventional Commits headers or footers; mentions in prose, Markdown headings, or quoted/fenced examples do not classify a change. New Feature and Bug Fix describe observable CLI or public Action behavior. Internal development tools and CI/CD changes use their appropriate maintenance types; user-experienced runtime improvements use `perf`. The [code-review policy](../../.agents/skills/code-review/SKILL.md#title-and-type) owns this semantic judgment.

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
subtract its version signal: the classifier still considers breaking, feature, performance and deprecation
signals across the complete interval. PATCH compares the final products. Partial removals must use a clarification explaining what remains rather
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

The resolver recognizes a retry only when the calculated version, release identity marker, and tag identify the exact frozen source; the publisher still verifies every checksum and asset. It does not recalculate a new version or regenerate stored notes for that release. An unrelated release with the same version is not a retry. Publication downloads the two selected artifacts by ID and verifies their per-target manifests.

A new workflow run resolves current main again. Existing tags remain immutable, so changed source cannot reuse a conflicting tag. Before tag creation, a fresh run may validate a newly reviewed source. The run logs record the original SHA, and candidate artifacts are retained for seven days. Keep incomplete drafts for inspection instead of automatically deleting them.

## Action version and availability

A published Action resolves its CLI from its own version tag. Full-SHA pins resolve an exact matching stable tag using public Git refs; untagged commits fail rather than falling back to latest. The installer verifies the release's `SHA256SUMS` and the executable's reported version. When invoking the Action from a local path, set `SETTINGS_CANDIDATE_VERSION` to the exact version under test. Candidate acceptance and post-publication verification supply this version explicitly. There is no public version or download-URL input.

PR CI screens and builds the candidate. Main CI builds release-ready native Linux and macOS products, then inspects those exact products through process acceptance and an exact-source composite with bounded download/API fixtures. This works before publication; [testing](testing.md#ci-validation) distinguishes local/hosted integration from released-download evidence. Source features are not automatically available through an older published Action pin. Publish an explicitly authorized matching package before separately reviewing consumer adoption and permission changes.

## Release decisions

The [versioning and publication decisions](adr/README.md#versioning-and-publication) explain how this repository selects the source, version and artifacts, and protects published identities across retries. Use [release validation](release-validation.md) to run local checks of version precedence, artifact identity, publication gates, note generation and recovery.
