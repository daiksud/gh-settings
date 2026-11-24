---
type: ADR
title: Require focused PRs and review of change impact
description: Record scope and semantic-review requirements without extra breaking-message syntax.
---

# ADR 0024. Require focused PRs and review of change impact

## Status

Accepted — the code-review skill owns the current review procedure.

## Context

The PR title and body become a durable squash commit message. Commitlint
validates Conventional Commits format, but cannot judge user impact, review
scope, or whether the selected type reflects the actual change.

## Decision

Require one coherent purpose with one Conventional Commits type and at most
one meaningful scope. Keep implementation, tests, and necessary documentation
for that purpose in one PR; split unrelated changes.

Follow [Conventional Commits 1.0.0](https://www.conventionalcommits.org/en/v1.0.0/):
**either** a `!` header marker **or** a `BREAKING CHANGE:` footer is sufficient
for a breaking change. Do not require both. During v0.x, do not require
obsolete compatibility paths or historical migration guides.
Reviewers assess actual impact. The `commitlint` check validates message syntax and rejects HTML comments in the PR body.

## Consequences

Review changes against the current public contract and relevant evidence.
Reviewers verify the PR body describes the user impact and the checks actually performed. Publication requires an explicit release request.
