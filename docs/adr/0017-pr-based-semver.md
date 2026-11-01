---
type: ADR
title: Derive SemVer from reviewed changes
description: Derive release versions and note categories from durable squash messages in a fixed Git interval.
---

## Status

Accepted — implemented. Supersedes [ADR 0010](0010-explicit-release-version.md)'s manual version input, not manual publication or the fixed-source/artifact contracts. [ADR 0024](0024-code-review-policy.md) defines the required review signals for breaking changes.

## Context

PR titles and bodies become durable squash messages. Classification can be corrected during review without renaming branches. Mutable PR metadata can change after merge, so it cannot reliably describe an already-committed release interval. Automatic selection must not change source after validation or imply automatic publication.

## Decision

Use the complete previous-published-stable-to-frozen-source Git interval. Actual committed Conventional Commits headers and compatibility footers determine versions and release-note categories. Labels remain independent organization metadata, with no automatic synchronization, type-agreement requirement, or release check.

The strongest included change selects the increment:

- Any breaking header or breaking-change footer selects MINOR while the previous stable version is `0.x`, and MAJOR from `1.0.0` onward. It never graduates the project automatically from `0.x` to `1.0.0`.
- Otherwise `feat`, `perf`, or a `DEPRECATED:` footer selects MINOR.
- Otherwise fixes, reverts, maintenance, and legacy messages select PATCH when publication is explicitly requested.

A revert does not automatically cancel another included change's signal. Ignore empty commits only; every changed tree, including any release-preparation-named commit, remains substantive. Reject an empty remaining interval. Initial publication without an established stable baseline requires a separate policy decision.

The manually dispatched release workflow uses the same Python version calculator as gh-qw at the frozen main SHA. No version-preparation PR or persisted version update is necessary. The published composite Action selects its CLI by the invoking version tag or exact SHA-matching release tag. Existing matching drafts/releases retain their original version, source, notes, and verified bytes on retry.

Generate notes locally from the same committed interval. Each commit enters one category, in priority order: breaking header or formal breaking footer, formal deprecation footer, `feat`, `fix`, `perf`, then Maintenance. Performance has its own heading. Entries link to exact commits; escaped breaking footer details and a fixed-SHA comparison retain compatibility guidance and traceability. Current PR titles, bodies, labels, and GitHub's native note-generation API are not inputs.

## Consequences

Reviewers must reconcile the proposed squash title, body, and compatibility footers with the actual user impact before merging. Preserve that reviewed message in the squash commit. Later PR edits and relabeling cannot alter either version calculation or the release notes. Existing path-based and Dependabot labels can continue independently; PRLint remains commitlint-only.

Version calculation does not authorize publication. Computation, API, history, or immutable-identity mismatches fail before remote tag/release creation. Existing publication gates, immutable artifacts, and matching-draft recovery remain required. See the [maintainer release contract](../contributing/releases.md) for executable steps and evidence limits.
