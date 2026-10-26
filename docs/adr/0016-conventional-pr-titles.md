---
type: ADR
title: Validate PR squash messages as Conventional Commits
description: Record the PR squash message format independently of version calculation.
sources:
  - id: conventional-commits
    resource: https://www.conventionalcommits.org/en/v1.0.0/
    title: Conventional Commits 1.0.0
---

## Status

Accepted — enforced in PRLint. Either Conventional Commits breaking-change marker is sufficient.

## Context

Future release automation needs structured change descriptions. Conventional Commits defines a type, optional scope, and description in the title line.[^conventional-commits] The repository uses squash merges and retains the PR title and body as the final commit message.

## Decision

Validate the PR title and optional body as one proposed Conventional Commits message with commitlint's unmodified `@commitlint/config-conventional` preset, independently of release-version calculation. Do not add overrides or separate message-policy tests.

## Consequences

The independent `pr-message` check validates the proposed squash message on open, update, and edit; the code-validation `pass` status is separate. Local commits are not checked. The merge dialog can still change the final message, so reviewers must confirm it before merging. This decision does not require `!` in every title; either `!` or a `BREAKING CHANGE:` footer can signal a breaking change. [ADR 0017](0017-pr-based-semver.md) records version calculation separately.

[^conventional-commits]: [Conventional Commits 1.0.0](https://www.conventionalcommits.org/en/v1.0.0/), checked on 2026-10-12 for title structure and breaking-change notation.
