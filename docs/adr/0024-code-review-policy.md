---
type: ADR
title: Require focused PRs and explicit breaking-change reviews
description: Record the repository review policy and stricter squash-message requirements.
---

## Status

Accepted. Supersedes the review-time breaking-change notation in [ADR 0016](0016-conventional-pr-titles.md) and resolves the notation question left open in [ADR 0017](0017-pr-based-semver.md).

## Context

PR titles and bodies become durable squash commit messages. Syntax validation cannot determine whether a type describes the actual change, whether independent changes have been combined, or whether existing users must migrate. The previous decision allowed either breaking-change marker, leaving reviewers and release readers with different places to look for compatibility impact.

## Decision

Maintain the review procedure and detailed criteria in the repository's [code-review skill](../../.agents/skills/code-review/SKILL.md), linked from [AGENTS.md](../../AGENTS.md) and the contributor guides.

Require one coherent change purpose, one appropriate type, and at most one meaningful scope per PR. Keep implementation, tests, documentation, and necessary wiring for the same purpose together. Split independent changes rather than hiding them behind a broad type or scope.

Review the title and body as one Conventional Commits message, with an explanation that remains useful in Git history. For every backward-incompatible change, require both `type!: description` or `type(scope)!: description` and an explicit `BREAKING CHANGE:` footer explaining impact and migration, regardless of type or pre-1.0 version.

## Consequences

These are mandatory review criteria. Existing commitlint checks continue to validate syntax and the agreed style constraints; they do not detect semantic type/scope violations or undisclosed incompatibilities, and currently accept either breaking marker alone. Passing CI therefore does not establish compliance with the stricter review policy.

Review relevant product contracts and meaningful verification evidence without requiring unrelated refactoring or speculative safeguards. The skill is the authoritative procedure; this ADR records why it was adopted. Automatic version calculation remains future work under ADR 0017, and release publication remains explicitly initiated.
