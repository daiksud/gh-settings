---
type: ADR
title: Derive future SemVer bumps from merged PRs
description: Record the agreed precedence for automatic release version selection.
---

## Status

Accepted — future work, not implemented. [ADR 0024](0024-code-review-policy.md) subsequently resolves the breaking-change notation question below; automatic version selection remains unimplemented.

## Context

The initial release still takes an [explicit version](0010-explicit-release-version.md). PR titles and bodies are validated under [ADR 0016](0016-conventional-pr-titles.md). Merged changes can inform a later automatic version proposal.

## Decision

For future automation, determine the next SemVer bump from PRs merged since the previous release: any breaking change wins a major bump; otherwise any new feature yields minor; otherwise use patch.

## Consequences

The calculation examines the full interval since the previous release, not just the latest PR. No special pre-1.0 exception has been decided. The exact title signal for all breaking changes remains open; requiring `!` was only a suggestion, not an accepted rule. Until version calculation is implemented, releases continue to use the explicit input.
