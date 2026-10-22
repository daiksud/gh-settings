---
type: ADR
title: Derive SemVer from reviewed changes
description: Calculate release versions from durable squash messages and use GitHub-native release notes.
---

## Status

Accepted — implemented. Supersedes [ADR 0010](0010-explicit-release-version.md)'s manual version input, not manual publication or the fixed-source/artifact contracts. [ADR 0024](0024-code-review-policy.md) defines the required review signals for breaking changes.

## Context

PR titles and bodies become durable squash messages. Classification can be corrected during review without renaming branches, while labels provide useful release-note categories. Automatic selection must not change source after validation or imply automatic publication.

## Decision

Use the complete previous-published-stable-to-frozen-source Git interval. Committed Conventional Commits headers and compatibility footers are the authoritative version signal; mutable labels and branch names are not.

The strongest included change selects the increment:

- Any breaking header or breaking-change footer selects MAJOR, including before 1.0.
- Otherwise `feat`, `perf`, or a `DEPRECATED:` footer selects MINOR.
- Otherwise fixes, reverts, maintenance, and legacy messages select PATCH.

A revert does not automatically cancel another included change's signal. Ignore only nonbreaking `build(release)` or `chore(release)` commits whose entire file change is `action/cli-version`. Reject an empty remaining interval. Initial publication without an established stable baseline requires a separate policy decision.

A local preparation command computes the version and updates the committed Action/CLI version through a reviewed PR. The manually dispatched workflow recomputes and verifies it at a fixed main SHA before builds; operators do not supply a version. Existing matching drafts/releases retain their original version, source, notes, and verified bytes on retry.

New release notes use GitHub's native generation with `.github/release.yml`. Reviewed labels classify PR entries; a minimal supplement retains committed breaking migration footers and a fixed-SHA comparison. Do not retain the superseded categorized Git-log renderer or implement a second version engine from labels.

## Consequences

Reviewers must reconcile titles, footers, and labels before merging, including already-merged unreleased PRs during adoption. A later label edit can affect new notes but not change the committed version signal. Native PR lists and contributors differ from the former per-commit changelog; direct commits remain visible through the frozen comparison.

Version preparation remains separate from authorization to publish. Computation, API, history, or preparation mismatches fail before remote tag/release creation. Existing publication gates, immutable artifacts, and matching-draft recovery remain required. See the [maintainer release contract](../contributing/releases.md) for executable steps and evidence limits.
