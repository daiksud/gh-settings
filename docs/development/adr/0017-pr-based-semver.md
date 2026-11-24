---
type: ADR
title: Derive SemVer from reviewed changes
description: Derive release versions and note categories from durable squash messages in a fixed Git interval.
---

# ADR 0017. Derive SemVer from reviewed changes

## Status

Accepted — implemented. Supersedes [ADR 0010](0010-explicit-release-version.md)'s manual version input, not manual publication or the fixed-source/artifact contracts. [ADR 0024](0024-code-review-policy.md) defines the required review signals for breaking changes.

## Context

PR titles and bodies become durable squash messages. Classification can be corrected during review without renaming branches. Mutable PR metadata can change after merge, so it cannot reliably describe an already-committed release interval. Automatic selection must not change source after validation or imply automatic publication.

## Decision

Use the complete previous-published-stable-to-frozen-source Git interval. Classify committed Conventional Commits headers and compatibility footers. Compare the shipped products when a PATCH is possible.

The strongest included change selects the increment:

- Any breaking header or breaking-change footer selects MINOR while the previous stable version is `0.x`, and MAJOR from `1.0.0` onward. It never graduates the project automatically from `0.x` to `1.0.0`.
- Otherwise `feat`, `perf`, or a `DEPRECATED:` footer selects MINOR.
- Otherwise select PATCH if either native CLI executable or the public Action package differs between the endpoints. A documentation, test or CI change can carry forward to the next release when the products match. A refactor or dependency update that changes a binary qualifies.

Ignore empty commits when classifying the interval. A later revert can restore
identical products for a PATCH decision. Earlier breaking or MINOR signals
still apply. The release series started at `v0.1.0` and requires an established
published stable baseline. An explicit manual promotion from v0 selects
`v1.0.1` for any substantive interval.

The shared version workflow compares each platform on a single native runner
using the same embedded version and disabled Go VCS stamps. Each checkout
provides its own build recipe and toolchain. Comparison binaries serve only
to calculate the version. Candidate builds embed the final version before
inspection and publication.

The published composite Action selects its CLI by the invoking version tag or exact SHA-matching release tag. Existing matching drafts/releases retain their original version, source, notes, and verified bytes on retry.

Generate notes locally from the same committed interval. A reviewed `Release-withdraws:` footer can explicitly move a whole earlier entry from the same interval to Withdrawn before release, retaining its links and reason while suppressing its obsolete migration instructions. `Release-note:` footers add prominent final-state or upgrade clarifications. Neither changes version signals or rewrites retained retry notes. Otherwise each commit enters one category, in priority order: breaking header or formal breaking footer, formal deprecation footer, `feat`, `fix`, `perf`, then Maintenance. Performance has its own heading. Entries link to exact commits; escaped breaking footer details and a fixed-SHA comparison retain compatibility guidance and traceability. Current PR titles, bodies, and GitHub's native note-generation API are not inputs.

## Consequences

Reviewers must reconcile the proposed squash title, body, and compatibility footers with the actual user impact before merging. Preserve that reviewed message in the squash commit. Later PR edits cannot alter either version calculation or the release notes.

Version calculation does not authorize publication. Computation, API, history, or immutable-identity mismatches fail before remote tag/release creation. Existing publication gates, immutable artifacts, and matching-draft recovery remain required. See the [maintainer release contract](../releases.md) for executable steps and evidence limits.
